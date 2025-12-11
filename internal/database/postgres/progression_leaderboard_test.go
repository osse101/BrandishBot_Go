package postgres

import (
	"context"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetContributionLeaderboard(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	defer cleanupTestDB(t, pool)

	repo := NewProgressionRepository(pool)
	ctx := context.Background()

	// Insert test engagement metrics
	metrics := []struct {
		userID string
		value  int
	}{
		{"user1", 100},
		{"user1", 50},  // total: 150
		{"user2", 200},
		{"user3", 75},
		{"user3", 25},  // total: 100
		{"user4", 300},
	}

	for _, m := range metrics {
		err := repo.RecordEngagement(ctx, &domain.EngagementMetric{
			UserID:      m.userID,
			MetricType:  "test_contribution",
			MetricValue: m.value,
		})
		require.NoError(t, err)
	}

	// Test leaderboard
	t.Run("top 3 contributors", func(t *testing.T) {
		leaderboard, err := repo.GetContributionLeaderboard(ctx, 3)
		require.NoError(t, err)
		assert.Len(t, leaderboard, 3)

		// Verify order: user4 (300), user2 (200), user1 (150)
		assert.Equal(t, "user4", leaderboard[0].UserID)
		assert.Equal(t, 300, leaderboard[0].Contribution)
		assert.Equal(t, 1, leaderboard[0].Rank)

		assert.Equal(t, "user2", leaderboard[1].UserID)
		assert.Equal(t, 200, leaderboard[1].Contribution)
		assert.Equal(t, 2, leaderboard[1].Rank)

		assert.Equal(t, "user1", leaderboard[2].UserID)
		assert.Equal(t, 150, leaderboard[2].Contribution)
		assert.Equal(t, 3, leaderboard[2].Rank)
	})

	t.Run("all contributors", func(t *testing.T) {
		leaderboard, err := repo.GetContributionLeaderboard(ctx, 10)
		require.NoError(t, err)
		assert.Len(t, leaderboard, 4) // Only 4 unique users

		// Verify last entry
		assert.Equal(t, "user3", leaderboard[3].UserID)
		assert.Equal(t, 100, leaderboard[3].Contribution)
		assert.Equal(t, 4, leaderboard[3].Rank)
	})

	t.Run("limit 1", func(t *testing.T) {
		leaderboard, err := repo.GetContributionLeaderboard(ctx, 1)
		require.NoError(t, err)
		assert.Len(t, leaderboard, 1)
		assert.Equal(t, "user4", leaderboard[0].UserID)
	})
}

func TestVotingSessionWithDeadline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	pool := setupTestDB(t)
	defer cleanupTestDB(t, pool)

	repo := NewProgressionRepository(pool)
	ctx := context.Background()

	// Create a voting session
	sessionID, err := repo.CreateVotingSession(ctx)
	require.NoError(t, err)

	// Retrieve session and verify deadline is set
	session, err := repo.GetSessionByID(ctx, sessionID)
	require.NoError(t, err)
	assert.NotNil(t, session)
	assert.False(t, session.VotingDeadline.IsZero(), "voting deadline should be set")

	// Deadline should be approximately 24 hours from now
	expectedDeadline := session.StartedAt.Add(24 * time.Hour)
	timeDiff := session.VotingDeadline.Sub(expectedDeadline)
	assert.Less(t, timeDiff.Abs().Seconds(), 5.0, "deadline should be ~24h from start")
}
