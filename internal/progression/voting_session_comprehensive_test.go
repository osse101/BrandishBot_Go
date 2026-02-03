package progression

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// StartVotingSession Tests

func TestStartVotingSession_Success(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	// Should create session with available nodes
	err := service.StartVotingSession(ctx, nil)
	assert.NoError(t, err)

	// Verify session was created
	session, err := repo.GetActiveSession(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, "voting", session.Status)

	// Should have 2 options (money and lootbox0 are available)
	assert.Len(t, session.Options, 2)
}

func TestStartVotingSession_FewerThan4Available(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	// Unlock both money and lootbox0 to reduce available options
	repo.UnlockNode(ctx, 2, 1, "test", 0) // money
	repo.UnlockNode(ctx, 4, 1, "test", 0) // lootbox0

	err := service.StartVotingSession(ctx, nil)
	assert.NoError(t, err)

	session, _ := repo.GetActiveSession(ctx)
	// Should have 3 options (economy, upgrade, disassemble, search are available)
	// But we select max 4, so should get 3-4
	assert.LessOrEqual(t, len(session.Options), 4)
	assert.GreaterOrEqual(t, len(session.Options), 1)
}

func TestStartVotingSession_NoAvailableNodes(t *testing.T) {
	repo := NewMockRepository()
	// Don't setup tree - no nodes available
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	err := service.StartVotingSession(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no nodes available")
}

func TestStartVotingSession_MultiLevelNode(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	// Unlock economy to make cooldown_reduction available
	repo.UnlockNode(ctx, 2, 1, "test", 0) // money
	repo.UnlockNode(ctx, 3, 1, "test", 0) // economy

	// Unlock cooldown level 1
	repo.UnlockNode(ctx, 5, 1, "test", 0)

	err := service.StartVotingSession(ctx, nil)
	assert.NoError(t, err)

	session, _ := repo.GetActiveSession(ctx)

	// Find cooldown option
	var cooldownOption *domain.ProgressionVotingOption
	for i := range session.Options {
		if session.Options[i].NodeID == 5 {
			cooldownOption = &session.Options[i]
			break
		}
	}

	// Should target level 2 (next level)
	if cooldownOption != nil {
		assert.Equal(t, 2, cooldownOption.TargetLevel)
	}
}

// VoteForUnlock Tests

func TestVoteForUnlock_Success(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	// Start session
	service.StartVotingSession(ctx, nil)
	session, _ := repo.GetActiveSession(ctx)

	// Vote for first option
	nodeKey := session.Options[0].NodeDetails.NodeKey
	err := service.VoteForUnlock(ctx, domain.PlatformDiscord, "user1", "user1", nodeKey)
	assert.NoError(t, err)

	// Verify vote was recorded
	hasVoted, _ := repo.HasUserVotedInSession(ctx, "test-user-1", session.ID)
	assert.True(t, hasVoted)

	// Verify vote count incremented
	updatedSession, _ := repo.GetActiveSession(ctx)
	assert.Equal(t, 1, updatedSession.Options[0].VoteCount)
}

func TestVoteForUnlock_NoActiveSession(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	// No session started
	err := service.VoteForUnlock(ctx, domain.PlatformDiscord, "user1", "user1", "item_money")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active voting session")
}

func TestVoteForUnlock_SessionNotVoting(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	// Create and end session
	service.StartVotingSession(ctx, nil)
	session, _ := repo.GetActiveSession(ctx)
	optionID := session.Options[0].ID
	repo.EndVotingSession(ctx, session.ID, &optionID)

	err := service.VoteForUnlock(ctx, domain.PlatformDiscord, "user1", "user1", "item_money")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active voting session")
}

func TestVoteForUnlock_NodeNotInOptions(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	service.StartVotingSession(ctx, nil)

	// Try to vote for node not in options
	err := service.VoteForUnlock(ctx, domain.PlatformDiscord, "user1", "user1", "feature_economy")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in current voting options")
}

func TestVoteForUnlock_UserAlreadyVoted(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	service.StartVotingSession(ctx, nil)
	session, _ := repo.GetActiveSession(ctx)
	nodeKey := session.Options[0].NodeDetails.NodeKey

	// First vote succeeds
	err := service.VoteForUnlock(ctx, domain.PlatformDiscord, "user1", "user1", nodeKey)
	assert.NoError(t, err)

	// Second vote fails
	err = service.VoteForUnlock(ctx, domain.PlatformDiscord, "user1", "user1", nodeKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already voted")
}

// EndVoting Tests

func TestEndVoting_Success(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	service.StartVotingSession(ctx, nil)
	session, _ := repo.GetActiveSession(ctx)

	// Cast some votes
	service.VoteForUnlock(ctx, domain.PlatformDiscord, "user1", "user1", session.Options[0].NodeDetails.NodeKey)
	service.VoteForUnlock(ctx, domain.PlatformDiscord, "user2", "user2", session.Options[0].NodeDetails.NodeKey)

	winner, err := service.EndVoting(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, winner)
	assert.Equal(t, 2, winner.VoteCount)

	// Verify session ended
	updatedSession, _ := repo.GetSessionByID(ctx, session.ID)
	assert.Equal(t, "ended", updatedSession.Status)
}

func TestEndVoting_TieBreaker(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	service.StartVotingSession(ctx, nil)
	session, _ := repo.GetActiveSession(ctx)

	// Create tie - 1 vote each
	service.VoteForUnlock(ctx, domain.PlatformDiscord, "user1", "user1", session.Options[0].NodeDetails.NodeKey)
	service.VoteForUnlock(ctx, domain.PlatformDiscord, "user2", "user2", session.Options[1].NodeDetails.NodeKey)

	winner, err := service.EndVoting(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, winner)
	// Winner should be the one that reached highest vote first (LastHighestVoteAt)
	assert.Equal(t, 1, winner.VoteCount)
}

func TestEndVoting_ZeroVotes(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	service.StartVotingSession(ctx, nil)

	// No votes cast
	winner, err := service.EndVoting(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, winner)
	// Should randomly select one
	assert.Equal(t, 0, winner.VoteCount)
}

func TestEndVoting_AwardsContributions(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	service.StartVotingSession(ctx, nil)
	session, _ := repo.GetActiveSession(ctx)

	// Cast votes
	service.VoteForUnlock(ctx, domain.PlatformDiscord, "user1", "user1", session.Options[0].NodeDetails.NodeKey)
	service.VoteForUnlock(ctx, domain.PlatformDiscord, "user2", "user2", session.Options[0].NodeDetails.NodeKey)

	_, err := service.EndVoting(ctx)
	assert.NoError(t, err)

	// Verify engagement metrics recorded
	user1Engagement, _ := service.GetUserEngagement(ctx, domain.PlatformDiscord, "user1")
	assert.Greater(t, user1Engagement.TotalScore, 0)
}

func TestEndVoting_NoActiveSession(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	_, err := service.EndVoting(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active voting session")
}

func TestEndVoting_AlreadyEnded(t *testing.T) {
	repo := NewMockRepository()
	setupTestTree(repo)
	service := NewService(repo, NewMockUser(), nil, nil, nil)
	ctx := context.Background()

	service.StartVotingSession(ctx, nil)

	// End it once
	service.EndVoting(ctx)

	// Try to end again
	_, err := service.EndVoting(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active voting session")
}
