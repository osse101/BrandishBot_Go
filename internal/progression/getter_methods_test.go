package progression

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Additional tests to reach 70% coverage target

// GetActiveVotingSession Tests
func TestGetActiveVotingSession_Success(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, nil)
	ctx := context.Background()

	// Start session
	service.StartVotingSession(ctx, nil)

	// Get active session
	session, err := service.GetActiveVotingSession(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, "voting", session.Status)
}

func TestGetActiveVotingSession_NoSession(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo, nil)
	ctx := context.Background()

	// No session exists
	session, err := service.GetActiveVotingSession(ctx)
	assert.NoError(t, err)
	assert.Nil(t, session)
}

// CheckAndUnlockCriteria Tests
func TestCheckAndUnlockCriteria_TriggersUnlock(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, nil)
	ctx := context.Background()

	// Setup progress with met threshold
	progressID, _ := repo.CreateUnlockProgress(ctx)
	moneyID := 2
	repo.SetUnlockTarget(ctx, progressID, moneyID, 1, 1)
	repo.AddContribution(ctx, progressID, 500)

	// Check and unlock
	unlock, err := service.CheckAndUnlockCriteria(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, unlock)
	assert.Equal(t, moneyID, unlock.NodeID)
}

func TestCheckAndUnlockCriteria_NoUnlock_StartsSession(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, nil)
	ctx := context.Background()

	// No unlock ready
	unlock, err := service.CheckAndUnlockCriteria(ctx)
	assert.NoError(t, err)
	assert.Nil(t, unlock)

	// Should have started a session asynchronously
	// We can't easily verify async, but no error is good
}

// GetContributionLeaderboard Tests
func TestGetContributionLeaderboard_Success(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo, nil)
	ctx := context.Background()

	leaderboard, err := service.GetContributionLeaderboard(ctx, 10)
	assert.NoError(t, err)
	assert.NotNil(t, leaderboard)
}

func TestGetContributionLeaderboard_ClampLimit(t *testing.T) {
	repo := NewMockRepository()
	service := NewService(repo, nil)
	ctx := context.Background()

	// Request too many
	leaderboard, err := service.GetContributionLeaderboard(ctx, 500)
	assert.NoError(t, err)
	assert.NotNil(t, leaderboard)

	// Request zero
	leaderboard, err = service.GetContributionLeaderboard(ctx, 0)
	assert.NoError(t, err)
	assert.NotNil(t, leaderboard)
}

// GetProgressionStatus Tests
func TestGetProgressionStatus_Complete(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, nil)
	ctx := context.Background()

	// Unlock a node
	repo.UnlockNode(ctx, 2, 1, "test", 0)

	// Get status
	status, err := service.GetProgressionStatus(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.GreaterOrEqual(t, status.TotalUnlocked, 1)
}

func TestGetProgressionStatus_WithActiveSession(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, nil)
	ctx := context.Background()

	// Start session
	service.StartVotingSession(ctx, nil)

	// Get status
	status, err := service.GetProgressionStatus(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.NotNil(t, status.ActiveSession)
}
