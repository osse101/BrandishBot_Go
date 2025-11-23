package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/database"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestStatsRepository_Integration(t *testing.T) {
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
	pool, err := database.NewPool(connStr)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Apply migrations
	if err := applyMigrations(ctx, pool, "../../../migrations"); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}

	// Create test user first (required for foreign key constraints)
	userRepo := NewUserRepository(pool)
	testUser := &domain.User{
		Username: "test_stats_user",
		TwitchID: "test123",
	}
	if err := userRepo.UpsertUser(ctx, testUser); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	statsRepo := NewStatsRepository(pool)

	t.Run("RecordEvent", func(t *testing.T) {
		event := &domain.StatsEvent{
			UserID:    testUser.ID,
			EventType: domain.EventItemAdded,
			EventData: map[string]interface{}{
				"item_id":  1,
				"quantity": 10,
			},
			CreatedAt: time.Now(),
		}

		err := statsRepo.RecordEvent(ctx, event)
		if err != nil {
			t.Fatalf("RecordEvent failed: %v", err)
		}

		if event.EventID == 0 {
			t.Error("expected event ID to be set")
		}
	})

	t.Run("GetEventsByUser", func(t *testing.T) {
		// Record some test events
		now := time.Now()
		events := []domain.StatsEvent{
			{
				UserID:    testUser.ID,
				EventType: domain.EventItemAdded,
				EventData: map[string]interface{}{"item": "sword"},
				CreatedAt: now.Add(-1 * time.Hour),
			},
			{
				UserID:    testUser.ID,
				EventType: domain.EventItemUsed,
				EventData: map[string]interface{}{"item": "potion"},
				CreatedAt: now.Add(-30 * time.Minute),
			},
		}

		for i := range events {
			if err := statsRepo.RecordEvent(ctx, &events[i]); err != nil {
				t.Fatalf("failed to record event: %v", err)
			}
		}

		// Query events
		startTime := now.Add(-2 * time.Hour)
		endTime := now
		retrieved, err := statsRepo.GetEventsByUser(ctx, testUser.ID, startTime, endTime)
		if err != nil {
			t.Fatalf("GetEventsByUser failed: %v", err)
		}

		if len(retrieved) < 2 {
			t.Errorf("expected at least 2 events, got %d", len(retrieved))
		}

		// Verify events are ordered by created_at DESC
		for i := 0; i < len(retrieved)-1; i++ {
			if retrieved[i].CreatedAt.Before(retrieved[i+1].CreatedAt) {
				t.Error("events are not ordered by created_at DESC")
				break
			}
		}
	})

	t.Run("GetEventsByType", func(t *testing.T) {
		now := time.Now()
		
		// Record events of different types
		eventTypes := []domain.EventType{
			domain.EventItemAdded,
			domain.EventItemAdded,
			domain.EventItemUsed,
		}

		for _, eventType := range eventTypes {
			event := &domain.StatsEvent{
				UserID:    testUser.ID,
				EventType: eventType,
				EventData: map[string]interface{}{"test": true},
				CreatedAt: now,
			}
			if err := statsRepo.RecordEvent(ctx, event); err != nil {
				t.Fatalf("failed to record event: %v", err)
			}
		}

		// Query by type
		startTime := now.Add(-1 * time.Hour)
		endTime := now.Add(1 * time.Hour)
		retrieved, err := statsRepo.GetEventsByType(ctx, domain.EventItemAdded, startTime, endTime)
		if err != nil {
			t.Fatalf("GetEventsByType failed: %v", err)
		}

		// Should have at least the 2 we just added
		if len(retrieved) < 2 {
			t.Errorf("expected at least 2 events of type item_added, got %d", len(retrieved))
		}

		// Verify all are of correct type
		for _, event := range retrieved {
			if event.EventType != domain.EventItemAdded {
				t.Errorf("expected event type %s, got %s", domain.EventItemAdded, event.EventType)
			}
		}
	})

	t.Run("GetTopUsers", func(t *testing.T) {
		// Create another user
		anotherUser := &domain.User{
			Username: "another_stats_user",
			TwitchID: "test456",
		}
		if err := userRepo.UpsertUser(ctx, anotherUser); err != nil {
			t.Fatalf("failed to create another user: %v", err)
		}

		now := time.Now()
		
		// Record 3 events for testUser
		for i := 0; i < 3; i++ {
			event := &domain.StatsEvent{
				UserID:    testUser.ID,
				EventType: domain.EventMessageReceived,
				CreatedAt: now,
			}
			if err := statsRepo.RecordEvent(ctx, event); err != nil {
				t.Fatalf("failed to record event: %v", err)
			}
		}

		// Record 1 event for anotherUser
		event := &domain.StatsEvent{
			UserID:    anotherUser.ID,
			EventType: domain.EventMessageReceived,
			CreatedAt: now,
		}
		if err := statsRepo.RecordEvent(ctx, event); err != nil {
			t.Fatalf("failed to record event: %v", err)
		}

		// Get top users
		startTime := now.Add(-1 * time.Hour)
		endTime := now.Add(1 * time.Hour)
		topUsers, err := statsRepo.GetTopUsers(ctx, domain.EventMessageReceived, startTime, endTime, 10)
		if err != nil {
			t.Fatalf("GetTopUsers failed: %v", err)
		}

		if len(topUsers) < 2 {
			t.Errorf("expected at least 2 users, got %d", len(topUsers))
		}

		// First user should be testUser with more events
		if topUsers[0].UserID != testUser.ID {
			t.Errorf("expected top user to be %s, got %s", testUser.ID, topUsers[0].UserID)
		}

		if topUsers[0].Count < 3 {
			t.Errorf("expected top user to have at least 3 events, got %d", topUsers[0].Count)
		}
	})

	t.Run("GetEventCounts", func(t *testing.T) {
		now := time.Now()
		
		// Record various events
		eventTypes := []domain.EventType{
			domain.EventItemSold,
			domain.EventItemSold,
			domain.EventItemBought,
		}

		for _, eventType := range eventTypes {
			event := &domain.StatsEvent{
				UserID:    testUser.ID,
				EventType: eventType,
				CreatedAt: now,
			}
			if err := statsRepo.RecordEvent(ctx, event); err != nil {
				t.Fatalf("failed to record event: %v", err)
			}
		}

		// Get counts
		startTime := now.Add(-1 * time.Hour)
		endTime := now.Add(1 * time.Hour)
		counts, err := statsRepo.GetEventCounts(ctx, startTime, endTime)
		if err != nil {
			t.Fatalf("GetEventCounts failed: %v", err)
		}

		if counts[domain.EventItemSold] < 2 {
			t.Errorf("expected at least 2 item_sold events, got %d", counts[domain.EventItemSold])
		}

		if counts[domain.EventItemBought] < 1 {
			t.Errorf("expected at least 1 item_bought event, got %d", counts[domain.EventItemBought])
		}
	})

	t.Run("GetUserEventCounts", func(t *testing.T) {
		now := time.Now()
		
		// Record various events for specific user
		eventTypes := []domain.EventType{
			domain.EventItemTransferred,
			domain.EventItemTransferred,
			domain.EventItemTransferred,
			domain.EventItemRemoved,
		}

		for _, eventType := range eventTypes {
			event := &domain.StatsEvent{
				UserID:    testUser.ID,
				EventType: eventType,
				CreatedAt: now,
			}
			if err := statsRepo.RecordEvent(ctx, event); err != nil {
				t.Fatalf("failed to record event: %v", err)
			}
		}

		// Get user-specific counts
		startTime := now.Add(-1 * time.Hour)
		endTime := now.Add(1 * time.Hour)
		counts, err := statsRepo.GetUserEventCounts(ctx, testUser.ID, startTime, endTime)
		if err != nil {
			t.Fatalf("GetUserEventCounts failed: %v", err)
		}

		if counts[domain.EventItemTransferred] < 3 {
			t.Errorf("expected at least 3 item_transferred events, got %d", counts[domain.EventItemTransferred])
		}

		if counts[domain.EventItemRemoved] < 1 {
			t.Errorf("expected at least 1 item_removed event, got %d", counts[domain.EventItemRemoved])
		}
	})

	t.Run("GetTotalEventCount", func(t *testing.T) {
		now := time.Now()
		
		// Record some events
		for i := 0; i < 5; i++ {
			event := &domain.StatsEvent{
				UserID:    testUser.ID,
				EventType: domain.EventUserRegistered,
				CreatedAt: now,
			}
			if err := statsRepo.RecordEvent(ctx, event); err != nil {
				t.Fatalf("failed to record event: %v", err)
			}
		}

		// Get total count
		startTime := now.Add(-1 * time.Hour)
		endTime := now.Add(1 * time.Hour)
		count, err := statsRepo.GetTotalEventCount(ctx, startTime, endTime)
		if err != nil {
			t.Fatalf("GetTotalEventCount failed: %v", err)
		}

		// Should be at least 5 from this test
		if count < 5 {
			t.Errorf("expected at least 5 events, got %d", count)
		}
	})

	t.Run("TimeRangeFiltering", func(t *testing.T) {
		// Test that events outside time range are not returned
		// Create a specific user for this test to avoid conflicts with other tests
		isolatedUser := &domain.User{
			Username: "time_filter_test_user",
			TwitchID: "time_test_123",
		}
		if err := userRepo.UpsertUser(ctx, isolatedUser); err != nil {
			t.Fatalf("failed to create isolated user: %v", err)
		}

		pastTime := time.Now().Add(-48 * time.Hour)

		// Record event in the past
		pastEvent := &domain.StatsEvent{
			UserID:    isolatedUser.ID,
			EventType: domain.EventItemAdded,
			CreatedAt: pastTime,
		}
		if err := statsRepo.RecordEvent(ctx, pastEvent); err != nil {
			t.Fatalf("failed to record past event: %v", err)
		}

		// Query for events only in the last hour
		startTime := time.Now().Add(-1 * time.Hour)
		endTime := time.Now()
		events, err := statsRepo.GetEventsByUser(ctx, isolatedUser.ID, startTime, endTime)
		if err != nil {
			t.Fatalf("GetEventsByUser failed: %v", err)
		}

		// Past event should not be in results
		for _, event := range events {
			if event.CreatedAt.Before(startTime) {
				t.Error("event outside time range was returned")
			}
		}

		// Should have no events for this user in the recent time range
		if len(events) != 0 {
			t.Errorf("expected 0 events in recent time range, got %d", len(events))
		}
	})
}
