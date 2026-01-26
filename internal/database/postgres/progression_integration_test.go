package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/osse101/BrandishBot_Go/internal/database"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/progression"
)

func TestProgressionRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start Postgres container
	var pgContainer *postgres.PostgresContainer
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Skipf("Skipping integration test due to panic (likely Docker issue): %v", r)
			}
		}()
		pgContainer, err = postgres.Run(ctx,
			"postgres:15-alpine",
			postgres.WithDatabase("testdb"),
			postgres.WithUsername("testuser"),
			postgres.WithPassword("testpass"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(5*time.Second)),
		)
	}()

	if pgContainer == nil {
		if err != nil {
			t.Fatalf("failed to start postgres container: %v", err)
		}
		return
	}
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %v", err)
		}
	}()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Connect to database
	pool, err := database.NewPool(connStr, 10, 30*time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Apply migrations
	if err := applyMigrations(ctx, t, pool, "../../../migrations"); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}

	repo := NewProgressionRepository(pool, nil) // nil bus for tests

	// Run all the test sub-tests
	t.Run("NodeOperations", func(t *testing.T) {
		node, err := repo.GetNodeByKey(ctx, progression.FeatureProgressionSystem)
		if err != nil {
			t.Fatalf("GetNodeByKey failed: %v", err)
		}
		if node == nil {
			t.Fatal("Expected root node to exist")
		}

		nodes, err := repo.GetAllNodes(ctx)
		if err != nil {
			t.Fatalf("GetAllNodes failed: %v", err)
		}
		if len(nodes) == 0 {
			t.Error("Expected at least one node from migration")
		}
	})

	t.Run("UnlockFlow", func(t *testing.T) {
		root, err := repo.GetNodeByKey(ctx, progression.FeatureProgressionSystem)
		if err != nil || root == nil {
			t.Fatal("Failed to get root node")
		}

		// Unlock root
		err = repo.UnlockNode(ctx, root.ID, 1, "integration_test", 0)
		if err != nil {
			t.Fatalf("UnlockNode failed: %v", err)
		}

		// Verify unlock
		unlocked, err := repo.IsNodeUnlocked(ctx, progression.FeatureProgressionSystem, 1)
		if err != nil {
			t.Fatalf("IsNodeUnlocked failed: %v", err)
		}
		if !unlocked {
			t.Error("Root node should be unlocked")
		}

		// Get unlock details (root is auto-unlocked by migration)
		unlock, err := repo.GetUnlock(ctx, root.ID, 1)
		if err != nil {
			t.Fatalf("GetUnlock failed: %v", err)
		}
		if unlock == nil {
			t.Fatal("Expected unlock to exist")
		}
		if unlock.UnlockedBy != "auto" {
			t.Errorf("Expected unlocked_by 'auto' (from migration), got '%s'", unlock.UnlockedBy)
		}

		// Test relock
		money, _ := repo.GetNodeByKey(ctx, progression.ItemMoney)
		if money != nil {
			repo.UnlockNode(ctx, money.ID, 1, "test", 0)
			err = repo.RelockNode(ctx, money.ID, 1)
			if err != nil {
				t.Fatalf("RelockNode failed: %v", err)
			}
			unlocked, _ = repo.IsNodeUnlocked(ctx, progression.ItemMoney, 1)
			if unlocked {
				t.Error("Money should be locked after relock")
			}
		}
	})

	t.Run("VotingFlow", func(t *testing.T) {
		money, err := repo.GetNodeByKey(ctx, progression.ItemMoney)
		if err != nil || money == nil {
			t.Skip("Money node not found")
		}

		// Create voting session
		sessionID, err := repo.CreateVotingSession(ctx)
		if err != nil {
			t.Fatalf("CreateVotingSession failed: %v", err)
		}

		// Add voting option
		err = repo.AddVotingOption(ctx, sessionID, money.ID, 1)
		if err != nil {
			t.Fatalf("AddVotingOption failed: %v", err)
		}

		// Get active session
		session, err := repo.GetActiveSession(ctx)
		if err != nil {
			t.Fatalf("GetActiveSession failed: %v", err)
		}
		if session == nil || session.ID != sessionID {
			t.Error("Expected active session to match created session")
		}

		// Record user vote
		userID := "integration_user"
		if len(session.Options) > 0 {
			optionID := session.Options[0].ID
			err = repo.RecordUserSessionVote(ctx, userID, sessionID, optionID, money.ID)
			if err != nil {
				t.Fatalf("RecordUserSessionVote failed: %v", err)
			}

			// Verify vote recorded
			hasVoted, err := repo.HasUserVotedInSession(ctx, userID, sessionID)
			if err != nil || !hasVoted {
				t.Error("User vote should be recorded in session")
			}

			// Increment vote
			err = repo.IncrementOptionVote(ctx, optionID)
			if err != nil {
				t.Fatalf("IncrementOptionVote failed: %v", err)
			}
		}

		// End voting session
		if len(session.Options) > 0 {
			winningOptionID := session.Options[0].ID
			err = repo.EndVotingSession(ctx, sessionID, &winningOptionID)
			if err != nil {
				t.Fatalf("EndVotingSession failed: %v", err)
			}
		}
	})

	t.Run("EngagementTracking", func(t *testing.T) {
		metric := &domain.EngagementMetric{
			UserID:      "integration_user",
			MetricType:  "message",
			MetricValue: 10,
			RecordedAt:  time.Now(),
		}

		err := repo.RecordEngagement(ctx, metric)
		if err != nil {
			t.Fatalf("RecordEngagement failed: %v", err)
		}

		// Get user engagement
		breakdown, err := repo.GetUserEngagement(ctx, "integration_user")
		if err != nil {
			t.Fatalf("GetUserEngagement failed: %v", err)
		}
		if breakdown.MessagesSent < 10 {
			t.Errorf("Expected at least 10 messages, got %d", breakdown.MessagesSent)
		}

		// Get total score
		since := time.Now().Add(-1 * time.Hour)
		score, err := repo.GetEngagementScore(ctx, &since)
		if err != nil {
			t.Fatalf("GetEngagementScore failed: %v", err)
		}
		if score < 0 {
			t.Error("Expected non-negative score")
		}
	})

	t.Run("UserProgression", func(t *testing.T) {
		userID := "progression_user"
		recipeKey := "recipe_test"

		err := repo.UnlockUserProgression(ctx, userID, "recipe", recipeKey, nil)
		if err != nil {
			t.Fatalf("UnlockUserProgression failed: %v", err)
		}

		unlocked, err := repo.IsUserProgressionUnlocked(ctx, userID, "recipe", recipeKey)
		if err != nil || !unlocked {
			t.Error("Recipe should be unlocked")
		}

		progressions, err := repo.GetUserProgressions(ctx, userID, "recipe")
		if err != nil || len(progressions) == 0 {
			t.Error("Expected at least one progression")
		}
	})

	t.Run("TreeReset", func(t *testing.T) {
		// Unlock some nodes
		root, _ := repo.GetNodeByKey(ctx, progression.FeatureProgressionSystem)
		money, _ := repo.GetNodeByKey(ctx, progression.ItemMoney)

		if root != nil {
			repo.UnlockNode(ctx, root.ID, 1, "test", 0)
		}
		if money != nil {
			repo.UnlockNode(ctx, money.ID, 1, "test", 100)
		}

		// Reset tree
		err := repo.ResetTree(ctx, "admin", "integration test reset", false)
		if err != nil {
			t.Fatalf("ResetTree failed: %v", err)
		}

		// Root should still be unlocked
		if root != nil {
			unlocked, _ := repo.IsNodeUnlocked(ctx, progression.FeatureProgressionSystem, 1)
			if !unlocked {
				t.Error("Root should remain unlocked after reset")
			}
		}

		// Other nodes should be locked
		if money != nil {
			unlocked, _ := repo.IsNodeUnlocked(ctx, progression.ItemMoney, 1)
			if unlocked {
				t.Error("Money should be locked after reset")
			}
		}
	})

	t.Run("ContributionLeaderboard", func(t *testing.T) {
		// Clear any existing metrics for clean test
		_, err := pool.Exec(ctx, "DELETE FROM engagement_metrics")
		if err != nil {
			t.Logf("Warning: Could not clear engagement metrics: %v", err)
		}

		// Test empty leaderboard
		leaderboard, err := repo.GetContributionLeaderboard(ctx, 10)
		if err != nil {
			t.Fatalf("GetContributionLeaderboard failed: %v", err)
		}
		if len(leaderboard) != 0 {
			t.Errorf("Expected empty leaderboard, got %d entries", len(leaderboard))
		}

		// Insert test engagement metrics for multiple users
		testData := []struct {
			userID      string
			metricType  string
			metricValue int
		}{
			{"user1", "message", 100},      // Total: 100
			{"user1", "command", 50},       // Total: 150
			{"user2", "message", 200},      // Total: 200
			{"user2", "item_used", 25},     // Total: 225
			{"user3", "message", 150},      // Total: 150 (tie with user1)
			{"user4", "command", 10},       // Total: 10
			{"user5", "item_crafted", 300}, // Total: 300 (highest)
		}

		for _, td := range testData {
			metric := &domain.EngagementMetric{
				UserID:      td.userID,
				MetricType:  td.metricType,
				MetricValue: td.metricValue,
				RecordedAt:  time.Now(),
			}
			if err := repo.RecordEngagement(ctx, metric); err != nil {
				t.Fatalf("RecordEngagement failed: %v", err)
			}
		}

		// Test leaderboard with various limits
		t.Run("Top3", func(t *testing.T) {
			leaderboard, err := repo.GetContributionLeaderboard(ctx, 3)
			if err != nil {
				t.Fatalf("GetContributionLeaderboard failed: %v", err)
			}

			if len(leaderboard) != 3 {
				t.Errorf("Expected 3 entries, got %d", len(leaderboard))
			}

			// Verify order (highest to lowest)
			if len(leaderboard) >= 3 {
				if leaderboard[0].UserID != "user5" || leaderboard[0].Contribution != 300 {
					t.Errorf("Expected user5 with 300 at rank 1, got %s with %d",
						leaderboard[0].UserID, leaderboard[0].Contribution)
				}
				if leaderboard[0].Rank != 1 {
					t.Errorf("Expected rank 1 for top user, got %d", leaderboard[0].Rank)
				}

				if leaderboard[1].UserID != "user2" || leaderboard[1].Contribution != 225 {
					t.Errorf("Expected user2 with 225 at rank 2, got %s with %d",
						leaderboard[1].UserID, leaderboard[1].Contribution)
				}

				// Rank 3 could be either user1 or user3 (both have 150)
				if leaderboard[2].Contribution != 150 {
					t.Errorf("Expected 150 at rank 3, got %d", leaderboard[2].Contribution)
				}
			}
		})

		t.Run("AllUsers", func(t *testing.T) {
			leaderboard, err := repo.GetContributionLeaderboard(ctx, 10)
			if err != nil {
				t.Fatalf("GetContributionLeaderboard failed: %v", err)
			}

			// Should have all 5 users
			if len(leaderboard) != 5 {
				t.Errorf("Expected 5 entries, got %d", len(leaderboard))
			}

			// Verify descending order
			for i := 1; i < len(leaderboard); i++ {
				if leaderboard[i].Contribution > leaderboard[i-1].Contribution {
					t.Errorf("Leaderboard not in descending order at index %d", i)
				}
			}

			// Verify ranks are sequential
			for i, entry := range leaderboard {
				expectedRank := i + 1
				if entry.Rank != expectedRank {
					t.Errorf("Expected rank %d at index %d, got %d", expectedRank, i, entry.Rank)
				}
			}
		})

		t.Run("LimitEnforcement", func(t *testing.T) {
			leaderboard, err := repo.GetContributionLeaderboard(ctx, 2)
			if err != nil {
				t.Fatalf("GetContributionLeaderboard failed: %v", err)
			}

			if len(leaderboard) != 2 {
				t.Errorf("Expected exactly 2 entries with limit=2, got %d", len(leaderboard))
			}
		})
	})

	t.Run("DailyEngagementTotals", func(t *testing.T) {
		// Clear metrics and weights to be sure (optional, but good for isolation)
		_, err := pool.Exec(ctx, "DELETE FROM engagement_metrics")
		if err != nil {
			t.Logf("Warning: Could not clear engagement metrics: %v", err)
		}

		// Ensure weights exist
		_, err = pool.Exec(ctx, `
			INSERT INTO engagement_weights (metric_type, weight, description)
			VALUES 
				('vote_cast', 5.0, 'Test Vote'),
				('message', 1.0, 'Test Message')
			ON CONFLICT (metric_type) DO UPDATE SET weight = EXCLUDED.weight
		`)
		if err != nil {
			t.Fatalf("Failed to insert weights: %v", err)
		}

		userID := "user_velocity_test"

		// Use fixed dates to avoid timezone/boundary issues
		// Day 1
		day1 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
		// Day 2 (Skip a day to ensure separation even with extreme timezones)
		day2 := time.Date(2025, 1, 3, 12, 0, 0, 0, time.UTC)

		// Insert metrics
		// vote_cast (5.0) * 20 = 100
		metric1 := &domain.EngagementMetric{
			UserID:      userID,
			MetricType:  "vote_cast",
			MetricValue: 20,
			RecordedAt:  day2, // Newer
		}
		err = repo.RecordEngagement(ctx, metric1)
		if err != nil {
			t.Fatalf("RecordEngagement failed: %v", err)
		}

		// message (1.0) * 50 = 50
		metric2 := &domain.EngagementMetric{
			UserID:      userID,
			MetricType:  "message",
			MetricValue: 50,
			RecordedAt:  day1, // Older
		}
		err = repo.RecordEngagement(ctx, metric2)
		if err != nil {
			t.Fatalf("RecordEngagement failed: %v", err)
		}

		// Act
		// Since day1 (inclusive)
		totals, err := repo.GetDailyEngagementTotals(ctx, day1)
		if err != nil {
			t.Fatalf("GetDailyEngagementTotals failed: %v", err)
		}

		// Assert
		// We can't predict exact keys due to TZ, but sum should be 150
		totalPoints := 0
		for _, v := range totals {
			totalPoints += v
		}

		if totalPoints != 150 {
			t.Errorf("Expected 150 total points, got %d", totalPoints)
		}
		if len(totals) != 2 {
			t.Errorf("Expected 2 days of data, got %d", len(totals))
		}
	})
}
