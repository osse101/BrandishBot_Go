package progression

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestVotingFlow_Complete tests the full voting and unlock cycle
func TestVotingFlow_Complete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Step 1: Start voting session
	err := service.StartVotingSession(ctx, nil)
	assert.NoError(t, err)

	session, _ := repo.GetActiveSession(ctx)
	assert.NotNil(t, session)
	assert.Len(t, session.Options, 2) // money and lootbox0 available

	// Step 2: Multiple users vote for lootbox0 specifically (deterministic test)
	// Find lootbox0 option
	var lootboxKey string
	for _, opt := range session.Options {
		if opt.NodeDetails.NodeKey == "item_lootbox0" {
			lootboxKey = opt.NodeDetails.NodeKey
			break
		}
	}
	if lootboxKey == "" {
		t.Fatal("lootbox0 not found in session options")
	}

	service.VoteForUnlock(ctx, "discord", "user1", lootboxKey)
	service.VoteForUnlock(ctx, "discord", "user2", lootboxKey)
	service.VoteForUnlock(ctx, "discord", "user3", lootboxKey)

	// Step 3: End voting
	winner, err := service.EndVoting(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, winner)
	assert.Equal(t, 3, winner.VoteCount)

	// Step 4: Verify unlock progress target is set
	progress, _ := repo.GetActiveUnlockProgress(ctx)
	assert.NotNil(t, progress)
	assert.NotNil(t, progress.NodeID)
	assert.Equal(t, winner.NodeID, *progress.NodeID)

	// Step 5: Add contributions to meet threshold
	unlockCost := winner.NodeDetails.UnlockCost
	currentContrib := progress.ContributionsAccumulated
	needed := unlockCost - currentContrib

	err = service.AddContribution(ctx, needed)
	assert.NoError(t, err)

	// Step 6: Trigger unlock check
	unlock, err := service.CheckAndUnlockNode(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, unlock)
	assert.Equal(t, winner.NodeID, unlock.NodeID)

	// Step 7: Verify node is unlocked
	isUnlocked, _ := repo.IsNodeUnlocked(ctx, winner.NodeDetails.NodeKey, 1)
	assert.True(t, isUnlocked)

	// Step 8: Verify new session is started
	// After unlocking lootbox0, 4 options become available:
	// - money (root child, still available)
	// - upgrade, disassemble, search (lootbox0 children, now unlocked)
	// Since 4 options remain (≥2), a new voting session SHOULD be created.
	time.Sleep(100 * time.Millisecond)
	newProgress, _ := repo.GetActiveUnlockProgress(ctx)
	assert.NotNil(t, newProgress)
	assert.NotEqual(t, progress.ID, newProgress.ID, "New progress should be created after unlock")

	// Verify a new session was created
	newSession, _ := repo.GetActiveSession(ctx)
	assert.NotNil(t, newSession, "A new voting session should be created (4 options available)")
	assert.NotEqual(t, session.ID, newSession.ID, "Should be a different session")
}

// TestVotingFlow_MultipleVoters verifies multi-user voting scenarios
func TestVotingFlow_MultipleVoters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	service.StartVotingSession(ctx, nil)
	session, _ := repo.GetActiveSession(ctx)

	// Multiple users vote for different options
	service.VoteForUnlock(ctx, "discord", "user1", session.Options[0].NodeDetails.NodeKey)
	service.VoteForUnlock(ctx, "discord", "user2", session.Options[0].NodeDetails.NodeKey)
	service.VoteForUnlock(ctx, "discord", "user3", session.Options[1].NodeDetails.NodeKey)

	winner, _ := service.EndVoting(ctx)

	// Option 0 should win with 2 votes
	assert.Equal(t, session.Options[0].NodeID, winner.NodeID)
	assert.Equal(t, 2, winner.VoteCount)
}

// TestVotingFlow_AutoNextSession verifies automatic target selection after unlock
func TestVotingFlow_AutoNextSession(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Complete first unlock cycle - vote for lootbox0 specifically
	service.StartVotingSession(ctx, nil)
	session1, _ := repo.GetActiveSession(ctx)

	// Find lootbox0 option
	var lootboxKey string
	for _, opt := range session1.Options {
		if opt.NodeDetails.NodeKey == "item_lootbox0" {
			lootboxKey = opt.NodeDetails.NodeKey
			break
		}
	}
	if lootboxKey == "" {
		t.Fatal("lootbox0 not found in session options")
	}

	service.VoteForUnlock(ctx, "discord", "user1", lootboxKey)
	service.EndVoting(ctx)

	progress, _ := repo.GetActiveUnlockProgress(ctx)
	cost := session1.Options[0].NodeDetails.UnlockCost
	service.AddContribution(ctx, cost-progress.ContributionsAccumulated)

	service.CheckAndUnlockNode(ctx)

	// Wait for async transition
	time.Sleep(100 * time.Millisecond)

	// After unlocking lootbox0, 4 options become available:
	// - money (root child, still available)
	// - upgrade, disassemble, search (lootbox0 children, now unlocked)
	// Since 4 options remain (≥2), a voting session SHOULD be created.
	newProgress, _ := repo.GetActiveUnlockProgress(ctx)
	assert.NotNil(t, newProgress)
	assert.NotEqual(t, progress.ID, newProgress.ID, "New progress should be created")

	// Verify a new session was created
	session2, _ := repo.GetActiveSession(ctx)
	assert.NotNil(t, session2, "A new voting session should be created (4 options available)")
	assert.NotEqual(t, session1.ID, session2.ID, "Should be a different session")
	assert.Equal(t, "voting", session2.Status, "New session should be in voting status")
}

// TestMultiLevel_Progressive tests unlocking multiple levels of same node
func TestMultiLevel_Progressive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Unlock prerequisites first
	repo.UnlockNode(ctx, 2, 1, "test", 0) // money
	repo.UnlockNode(ctx, 3, 1, "test", 0) // economy

	// Level 1 of cooldown_reduction
	repo.UnlockNode(ctx, 5, 1, "test", 0)

	// Start session - should include cooldown level 2
	service.StartVotingSession(ctx, nil)
	session, _ := repo.GetActiveSession(ctx)

	// Find cooldown option
	var cooldownOption *int
	for i := range session.Options {
		if session.Options[i].NodeID == 5 {
			cooldownOption = &i
			break
		}
	}

	if assert.NotNil(t, cooldownOption, "Cooldown level 2 should be in options") {
		assert.Equal(t, 2, session.Options[*cooldownOption].TargetLevel)
	}
}

// TestMultiLevel_SessionTargeting verifies next level targeting
func TestMultiLevel_SessionTargeting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Unlock money and economy to access cooldown
	repo.UnlockNode(ctx, 2, 1, "test", 0)
	repo.UnlockNode(ctx, 3, 1, "test", 0)

	// Start session
	service.StartVotingSession(ctx, nil)
	session, _ := repo.GetActiveSession(ctx)

	// Vote for cooldown level 1
	var cooldownKey string
	for _, opt := range session.Options {
		if opt.NodeID == 5 {
			cooldownKey = opt.NodeDetails.NodeKey
			break
		}
	}

	assert.NotEmpty(t, cooldownKey, "Cooldown level 1 should be in the initial session options")

	service.VoteForUnlock(ctx, "discord", "user1", cooldownKey)
	service.EndVoting(ctx)

	// Complete unlock
	progress, _ := repo.GetActiveUnlockProgress(ctx)
	needed := 1500 - progress.ContributionsAccumulated
	service.AddContribution(ctx, needed)
	service.CheckAndUnlockNode(ctx)

	// Wait for new session
	time.Sleep(100 * time.Millisecond)

	// After unlocking cooldown level 1, we have 3-4 options available:
	// - cooldown_reduction level 2 (multi-level node)
	// - buy (requires economy, which is unlocked)
	// - sell (requires economy, which is unlocked)
	// - item_lootbox0 (root child, always available)
	// Therefore, a target MUST be set and a voting session SHOULD be created (if 2+ options remain).
	
	newProgress, _ := repo.GetActiveUnlockProgress(ctx)
	isTarget := newProgress != nil && newProgress.NodeID != nil && *newProgress.NodeID == 5 && *newProgress.TargetLevel == 2
	
	newSession, _ := repo.GetActiveSession(ctx)
	hasInOptions := false
	if newSession != nil {
		for _, opt := range newSession.Options {
			if opt.NodeID == 5 && opt.TargetLevel == 2 {
				hasInOptions = true
				break
			}
		}
	}
	
	assert.True(t, isTarget || hasInOptions, "Next cycle should target cooldown level 2 (either as current target or as voting option)")
}

// TestRollover_ExcessPoints verifies excess contributions carry over
func TestRollover_ExcessPoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Setup progress with target
	progressID, _ := repo.CreateUnlockProgress(ctx)
	moneyID := 2 // cost 500
	repo.SetUnlockTarget(ctx, progressID, moneyID, 1, 1)

	// Add 400 points (100 short)
	repo.AddContribution(ctx, progressID, 400)

	// Add 250 more (150 excess)
	repo.AddContribution(ctx, progressID, 250)

	// Unlock
	unlock, _ := service.CheckAndUnlockNode(ctx)
	assert.NotNil(t, unlock)

	// Verify new progress has 150 rollover
	newProgress, _ := repo.GetActiveUnlockProgress(ctx)
	assert.NotNil(t, newProgress)
	assert.NotEqual(t, progressID, newProgress.ID)
	assert.Equal(t, 150, newProgress.ContributionsAccumulated)
}

// TestCache_ThresholdDetection verifies instant unlock cache mechanism
func TestCache_ThresholdDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil)
	ctx := context.Background()

	// Complete voting to set cache
	service.StartVotingSession(ctx, nil)
	session, _ := repo.GetActiveSession(ctx)
	service.VoteForUnlock(ctx, "discord", "user1", session.Options[0].NodeDetails.NodeKey)
	service.EndVoting(ctx)

	progress, _ := repo.GetActiveUnlockProgress(ctx)
	unlockCost := session.Options[0].NodeDetails.UnlockCost
	current := progress.ContributionsAccumulated

	// Add partial contribution (no instant unlock)
	halfNeeded := (unlockCost - current) / 2
	service.AddContribution(ctx, halfNeeded)

	// Verify not yet unlocked
	isUnlocked, _ := repo.IsNodeUnlocked(ctx, session.Options[0].NodeDetails.NodeKey, 1)
	assert.False(t, isUnlocked)

	// Add remaining (should trigger instant unlock via cache)
	remaining := unlockCost - current - halfNeeded + 10 // Add 10 extra
	service.AddContribution(ctx, remaining)

	// Wait for async unlock
	time.Sleep(100 * time.Millisecond)

	// Verify unlocked
	isUnlocked, _ = repo.IsNodeUnlocked(ctx, session.Options[0].NodeDetails.NodeKey, 1)
	assert.True(t, isUnlocked)

	// Verify new progress created with rollover
	newProgress, _ := repo.GetActiveUnlockProgress(ctx)
	assert.NotEqual(t, progress.ID, newProgress.ID)
	assert.Equal(t, 10, newProgress.ContributionsAccumulated)
}
