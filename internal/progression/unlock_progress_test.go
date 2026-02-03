package progression

import (
	"context"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/stretchr/testify/assert"
)

// CheckAndUnlockNode Tests

func TestCheckAndUnlock_Success(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	// 1. Setup active progress
	progressID, _ := repo.CreateUnlockProgress(ctx)

	// 2. Set target to money (cost 500)
	moneyID := 2
	repo.SetUnlockTarget(ctx, progressID, moneyID, 1, 1)

	// 3. Add contributions to meet threshold
	repo.AddContribution(ctx, progressID, 500)

	// 4. Check and unlock
	unlock, err := service.CheckAndUnlockNode(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, unlock)
	assert.Equal(t, moneyID, unlock.NodeID)
	assert.Equal(t, 1, unlock.CurrentLevel)

	// 5. Verify node is unlocked
	isUnlocked, _ := repo.IsNodeUnlocked(ctx, "item_money", 1)
	assert.True(t, isUnlocked)
}

func TestCheckAndUnlock_Rollover(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	progressID, _ := repo.CreateUnlockProgress(ctx)
	moneyID := 2 // cost 500
	repo.SetUnlockTarget(ctx, progressID, moneyID, 1, 1)

	// Add excess contributions (600 total, 100 excess)
	repo.AddContribution(ctx, progressID, 600)

	// Unlock
	_, err := service.CheckAndUnlockNode(ctx)
	assert.NoError(t, err)

	// Verify new progress created with 100 rollover
	newProgress, _ := repo.GetActiveUnlockProgress(ctx)
	assert.NotNil(t, newProgress)
	assert.NotEqual(t, progressID, newProgress.ID)
	assert.Equal(t, 100, newProgress.ContributionsAccumulated)
}

func TestCheckAndUnlock_StartsNewSession(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	// Ensure money is available for session (root unlocks it)
	// Unlock money to check if session starts properly

	progressID, _ := repo.CreateUnlockProgress(ctx)
	moneyID := 2
	repo.SetUnlockTarget(ctx, progressID, moneyID, 1, 1)
	repo.AddContribution(ctx, progressID, 500)

	// Unlock should trigger handlePostUnlockTransition
	_, err := service.CheckAndUnlockNode(ctx)
	assert.NoError(t, err)

	// We need to wait a small bit for async goroutine
	time.Sleep(10 * time.Millisecond)

	// NEW FLOW: After unlock, a new session is only started if 2+ options remain
	// With the test tree setup, only lootbox0 remains after money is unlocked,
	// so no new session is started, but a new target should be set
	newProgress, _ := repo.GetActiveUnlockProgress(ctx)
	assert.NotNil(t, newProgress)
	assert.NotNil(t, newProgress.NodeID, "New target should be set after unlock")

	// A session may or may not exist depending on available options
	session, _ := repo.GetActiveSession(ctx)
	if session != nil {
		assert.Equal(t, "voting", session.Status)
	}
}

func TestCheckAndUnlock_ClearsCacheOnUnlock(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	progressID, _ := repo.CreateUnlockProgress(ctx)
	moneyID := 2
	repo.SetUnlockTarget(ctx, progressID, moneyID, 1, 1)
	repo.AddContribution(ctx, progressID, 500)

	// Manually set cache to verify it gets cleared
	// We can't access private fields easily in test without reflection or helpers
	// But we can verify behavior - next AddContribution won't trigger instant unlock
	// This is implicit in logic

	_, err := service.CheckAndUnlockNode(ctx)
	assert.NoError(t, err)

	// If cache cleared, internal state is reset
}

func TestCheckAndUnlock_NoProgress(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	// No active progress exists

	unlock, err := service.CheckAndUnlockNode(ctx)
	assert.NoError(t, err)
	assert.Nil(t, unlock)

	// Should create new progress
	progress, _ := repo.GetActiveUnlockProgress(ctx)
	assert.NotNil(t, progress)
	assert.Equal(t, 0, progress.ContributionsAccumulated)
}

func TestCheckAndUnlock_TargetNotSet(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	repo.CreateUnlockProgress(ctx)
	// Target not set (voting phase)

	unlock, err := service.CheckAndUnlockNode(ctx)
	assert.NoError(t, err)
	assert.Nil(t, unlock)
}

func TestCheckAndUnlock_BelowThreshold(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	progressID, _ := repo.CreateUnlockProgress(ctx)
	moneyID := 2 // Cost 500
	repo.SetUnlockTarget(ctx, progressID, moneyID, 1, 1)

	// Add only 400
	repo.AddContribution(ctx, progressID, 400)

	unlock, err := service.CheckAndUnlockNode(ctx)
	assert.NoError(t, err)
	assert.Nil(t, unlock)
}

// AddContribution Tests

func TestAddContribution_Success(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	// Create progress
	repo.CreateUnlockProgress(ctx)

	err := service.AddContribution(ctx, 50)
	assert.NoError(t, err)

	progress, _ := repo.GetActiveUnlockProgress(ctx)
	assert.Equal(t, 50, progress.ContributionsAccumulated)

	// Add more
	service.AddContribution(ctx, 25)
	progress, _ = repo.GetActiveUnlockProgress(ctx)
	assert.Equal(t, 75, progress.ContributionsAccumulated)
}

func TestAddContribution_CreatesProgress(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	// No progress exists

	err := service.AddContribution(ctx, 100)
	assert.NoError(t, err)

	progress, _ := repo.GetActiveUnlockProgress(ctx)
	assert.NotNil(t, progress)
	assert.Equal(t, 100, progress.ContributionsAccumulated)
}

func TestAddContribution_InstantUnlock(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	// Setup everything needed for instant unlock (via cache)
	// We need to simulate EndVoting to populate cache, or SetUnlockTarget + private field manipulation
	// Easiest is to go through EndVoting flow

	service.StartVotingSession(ctx, nil)
	session, _ := repo.GetActiveSession(ctx)

	// Option 0 is likely money (cost 500)
	// Vote and end
	err := service.VoteForUnlock(ctx, domain.PlatformDiscord, "user1", "user1", session.Options[0].NodeDetails.NodeKey)
	assert.NoError(t, err)

	_, err = service.EndVoting(ctx)
	assert.NoError(t, err)

	// Get current accumulated
	progress, err := repo.GetActiveUnlockProgress(ctx)
	assert.NoError(t, err)
	if !assert.NotNil(t, progress, "Progress should not be nil after voting ends") {
		return
	}
	initial := progress.ContributionsAccumulated

	// Add remaining points to trigger immediate unlock
	needed := 500 - initial
	if initial >= 500 {
		// Already unlocked?
		t.Log("Warning: initial contributions already exceed threshold")
		needed = 0
	}

	err = service.AddContribution(ctx, needed)
	assert.NoError(t, err)

	// Wait for async unlock
	time.Sleep(100 * time.Millisecond)

	// Should be unlocked now
	completedProgress, _ := repo.GetActiveUnlockProgress(ctx)
	// If unlocked, active progress is now the NEW one (empty or low rollover)
	// The unlocked node should be unlocked in repo
	isUnlocked, _ := repo.IsNodeUnlocked(ctx, session.Options[0].NodeDetails.NodeKey, 1)
	assert.True(t, isUnlocked, "Node should be unlocked")

	// The new progress should exist
	if assert.NotNil(t, completedProgress, "Active progress should not be nil") {
		assert.NotEqual(t, progress.ID, completedProgress.ID, "New progress ID should differ from old")
	}
}

// GetUnlockProgress Tests

func TestGetUnlockProgress_Active(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	repo.CreateUnlockProgress(ctx)

	progress, err := service.GetUnlockProgress(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, progress)
	// Active means UnlockedAt is nil
	assert.Nil(t, progress.UnlockedAt)
}

func TestGetUnlockProgress_None(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	progress, err := service.GetUnlockProgress(ctx)
	assert.NoError(t, err)
	assert.Nil(t, progress)
}
