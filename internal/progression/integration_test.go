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
	var winnerNodeID int
	for _, opt := range session.Options {
		if opt.NodeDetails.NodeKey == "item_lootbox0" {
			lootboxKey = opt.NodeDetails.NodeKey
			winnerNodeID = opt.NodeID
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
	node, _ := repo.GetNodeByID(ctx, winnerNodeID)
	unlockCost := node.UnlockCost
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
	var lootboxNodeID int
	for _, opt := range session1.Options {
		if opt.NodeDetails.NodeKey == "item_lootbox0" {
			lootboxKey = opt.NodeDetails.NodeKey
			lootboxNodeID = opt.NodeID
			break
		}
	}
	if lootboxKey == "" {
		t.Fatal("lootbox0 not found in session options")
	}

	service.VoteForUnlock(ctx, "discord", "user1", lootboxKey)
	service.EndVoting(ctx)

	progress, _ := repo.GetActiveUnlockProgress(ctx)
	node, _ := repo.GetNodeByID(ctx, lootboxNodeID)
	cost := node.UnlockCost
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

	// Pre-unlock all other paths to isolate the cooldown node
	repo.UnlockNode(ctx, 2, 1, "test", 0)  // money
	repo.UnlockNode(ctx, 3, 1, "test", 0)  // economy
	repo.UnlockNode(ctx, 6, 1, "test", 0)  // buy
	repo.UnlockNode(ctx, 7, 1, "test", 0)  // sell
	repo.UnlockNode(ctx, 4, 1, "test", 0)  // lootbox0
	repo.UnlockNode(ctx, 8, 1, "test", 0)  // upgrade
	repo.UnlockNode(ctx, 9, 1, "test", 0)  // disassemble
	repo.UnlockNode(ctx, 10, 1, "test", 0) // search

	// Only Cooldown level 1 is available now. Start session.
	service.StartVotingSession(ctx, nil)

	// Vote for cooldown level 1
	var cooldownKey string
	session, _ := repo.GetActiveSession(ctx)
	for _, opt := range session.Options {
		if opt.NodeID == 5 {
			cooldownKey = opt.NodeDetails.NodeKey
			break
		}
	}

	assert.NotEmpty(t, cooldownKey, "Cooldown level 1 should be available")
	service.VoteForUnlock(ctx, "discord", "user1", cooldownKey)
	service.EndVoting(ctx)

	// Complete unlock of Cooldown L1
	progress, _ := repo.GetActiveUnlockProgress(ctx)
	node, _ := repo.GetNodeByID(ctx, 5)
	needed := node.UnlockCost - progress.ContributionsAccumulated
	service.AddContribution(ctx, needed)
	service.CheckAndUnlockNode(ctx)

	// Wait for async transition
	time.Sleep(100 * time.Millisecond)

	// Since all other nodes are unlocked, Cooldown level 2 MUST be the new target.
	newProgress, _ := repo.GetActiveUnlockProgress(ctx)
	assert.NotNil(t, newProgress)
	assert.NotNil(t, newProgress.NodeID)
	assert.Equal(t, 5, *newProgress.NodeID)
	assert.Equal(t, 2, *newProgress.TargetLevel, "Next target should be exactly cooldown level 2")
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
	moneyID := 2
	node, _ := repo.GetNodeByID(ctx, moneyID)
	cost := node.UnlockCost

	repo.SetUnlockTarget(ctx, progressID, moneyID, 1, 1)

	// Add cost-100 points
	repo.AddContribution(ctx, progressID, cost-100)

	// Add 250 more (150 excess if cost-100 was 400, but let's be dynamic)
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
	node, _ := repo.GetNodeByID(ctx, session.Options[0].NodeID)
	unlockCost := node.UnlockCost
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
	assert.NotNil(t, newProgress)
	assert.NotEqual(t, progress.ID, newProgress.ID)
	assert.Equal(t, 10, newProgress.ContributionsAccumulated)
}
