package stats

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// mockStatsRepository implements Repository interface for testing
type mockStatsRepository struct {
	events                  []domain.StatsEvent
	recordEventError        error
	getUserEventsByTypeCalls int
}

func (m *mockStatsRepository) RecordEvent(ctx context.Context, event *domain.StatsEvent) error {
	if m.recordEventError != nil {
		return m.recordEventError
	}
	event.EventID = int64(len(m.events) + 1)
	m.events = append(m.events, *event)
	return nil
}

func (m *mockStatsRepository) GetEventsByUser(ctx context.Context, userID string, startTime, endTime time.Time) ([]domain.StatsEvent, error) {
	var filtered []domain.StatsEvent
	for _, event := range m.events {
		if event.UserID == userID && event.CreatedAt.After(startTime) && event.CreatedAt.Before(endTime) {
			filtered = append(filtered, event)
		}
	}
	return filtered, nil
}

func (m *mockStatsRepository) GetEventsByType(ctx context.Context, eventType domain.EventType, startTime, endTime time.Time) ([]domain.StatsEvent, error) {
	var filtered []domain.StatsEvent
	for _, event := range m.events {
		if event.EventType == eventType && event.CreatedAt.After(startTime) && event.CreatedAt.Before(endTime) {
			filtered = append(filtered, event)
		}
	}
	return filtered, nil
}

func (m *mockStatsRepository) GetUserEventsByType(ctx context.Context, userID string, eventType domain.EventType, limit int) ([]domain.StatsEvent, error) {
	m.getUserEventsByTypeCalls++
	var filtered []domain.StatsEvent
	for _, event := range m.events {
		if event.UserID == userID && event.EventType == eventType {
			filtered = append(filtered, event)
		}
	}
	// Sort DESC
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

func (m *mockStatsRepository) GetTopUsers(ctx context.Context, eventType domain.EventType, startTime, endTime time.Time, limit int) ([]domain.LeaderboardEntry, error) {
	counts := make(map[string]int)
	for _, event := range m.events {
		if event.EventType == eventType && event.CreatedAt.After(startTime) && event.CreatedAt.Before(endTime) {
			counts[event.UserID]++
		}
	}

	var entries []domain.LeaderboardEntry
	for userID, count := range counts {
		entries = append(entries, domain.LeaderboardEntry{
			UserID:    userID,
			Count:     count,
			EventType: string(eventType),
		})
	}

	return entries, nil
}

func (m *mockStatsRepository) GetEventCounts(ctx context.Context, startTime, endTime time.Time) (map[domain.EventType]int, error) {
	counts := make(map[domain.EventType]int)
	for _, event := range m.events {
		if event.CreatedAt.After(startTime) && event.CreatedAt.Before(endTime) {
			counts[event.EventType]++
		}
	}
	return counts, nil
}

func (m *mockStatsRepository) GetUserEventCounts(ctx context.Context, userID string, startTime, endTime time.Time) (map[domain.EventType]int, error) {
	counts := make(map[domain.EventType]int)
	for _, event := range m.events {
		if event.UserID == userID && event.CreatedAt.After(startTime) && event.CreatedAt.Before(endTime) {
			counts[event.EventType]++
		}
	}
	return counts, nil
}

func (m *mockStatsRepository) GetTotalEventCount(ctx context.Context, startTime, endTime time.Time) (int, error) {
	count := 0
	for _, event := range m.events {
		if event.CreatedAt.After(startTime) && event.CreatedAt.Before(endTime) {
			count++
		}
	}
	return count, nil
}

func TestRecordUserEvent(t *testing.T) {
	repo := &mockStatsRepository{}
	svc := NewService(repo)

	ctx := context.Background()
	userID := "test-user-123"
	eventType := domain.EventItemAdded
	metadata := map[string]interface{}{
		"item":     "sword",
		"quantity": 5,
	}

	err := svc.RecordUserEvent(ctx, userID, eventType, metadata)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should be 2 events now: item_added and daily_streak
	if len(repo.events) != 2 {
		t.Fatalf("Expected 2 events (item + streak), got %d", len(repo.events))
	}

	event := repo.events[0]
	if event.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, event.UserID)
	}
	if event.EventType != eventType {
		t.Errorf("Expected event type %s, got %s", eventType, event.EventType)
	}
}

func TestRecordUserEvent_DailyStreak(t *testing.T) {
	repo := &mockStatsRepository{}
	svc := NewService(repo)
	ctx := context.Background()
	userID := "user-streak"

	// 1. First event - should trigger streak 1
	svc.RecordUserEvent(ctx, userID, domain.EventItemAdded, nil)

	// Check if streak event was recorded
	if len(repo.events) != 2 { // 1 item_added, 1 daily_streak
		t.Errorf("Expected 2 events, got %d", len(repo.events))
	}
	streakEvent := repo.events[1]
	if streakEvent.EventType != domain.EventDailyStreak {
		t.Errorf("Expected daily_streak event, got %s", streakEvent.EventType)
	}
	if s, ok := streakEvent.EventData["streak"].(int); !ok || s != 1 {
		t.Errorf("Expected streak 1, got %v", streakEvent.EventData["streak"])
	}

	// 2. Second event same day - should NOT trigger new streak event
	svc.RecordUserEvent(ctx, userID, domain.EventItemAdded, nil)
	if len(repo.events) != 3 { // previous 2 + 1 new item_added. No new streak.
		t.Errorf("Expected 3 events (no new streak), got %d", len(repo.events))
	}

	// 3. Simulate yesterday event
	// Manually insert a streak event for yesterday
	yesterday := time.Now().AddDate(0, 0, -1)
	repo.events = []domain.StatsEvent{
		{
			EventID:   10,
			UserID:    userID,
			EventType: domain.EventDailyStreak,
			EventData: map[string]interface{}{"streak": 5},
			CreatedAt: yesterday,
		},
	}

	// Clear cache for this test scenario because we manually inserted an event
	// but the service cache thinks we already checked today (from step 2)
	// We need a fresh service to simulate a new day or clear the cache
	// Since cache is internal, we'll create a new service instance
	svc2 := NewService(repo)

	// Record event today - should increment streak to 6
	svc2.RecordUserEvent(ctx, userID, domain.EventItemAdded, nil)

	// repo.events should have: yesterday streak (preserved), today item_added, today streak
	// Note: `RecordUserEvent` appends.

	// Check the last event
	lastEvent := repo.events[len(repo.events)-1]
	if lastEvent.EventType != domain.EventDailyStreak {
		t.Errorf("Expected streak event")
	}
	if s, ok := lastEvent.EventData["streak"].(int); !ok || s != 6 {
		t.Errorf("Expected streak 6, got %v", lastEvent.EventData["streak"])
	}

	// 4. Simulate break in streak (2 days ago)
	twoDaysAgo := time.Now().AddDate(0, 0, -2)
	repo.events = []domain.StatsEvent{
		{
			EventID:   20,
			UserID:    userID,
			EventType: domain.EventDailyStreak,
			EventData: map[string]interface{}{"streak": 10},
			CreatedAt: twoDaysAgo,
		},
	}
	svc3 := NewService(repo)

	// Record event today - should reset streak to 1
	svc3.RecordUserEvent(ctx, userID, domain.EventItemAdded, nil)

	lastEvent = repo.events[len(repo.events)-1]
	if s, ok := lastEvent.EventData["streak"].(int); !ok || s != 1 {
		t.Errorf("Expected streak 1 (reset), got %v", lastEvent.EventData["streak"])
	}
}

func TestDailyStreakCache(t *testing.T) {
	repo := &mockStatsRepository{}
	svc := NewService(repo)
	ctx := context.Background()
	userID := "user-cache"

	// 1. First event - call DB
	if err := svc.RecordUserEvent(ctx, userID, domain.EventItemAdded, nil); err != nil {
		t.Fatalf("Failed: %v", err)
	}

	// Check that GetUserEventsByType was called
	// We expect 1 call: to check the last streak
	if repo.getUserEventsByTypeCalls != 1 {
		t.Errorf("Expected 1 DB call, got %d", repo.getUserEventsByTypeCalls)
	}

	// 2. Second event - should hit cache
	if err := svc.RecordUserEvent(ctx, userID, domain.EventItemSold, nil); err != nil {
		t.Fatalf("Failed: %v", err)
	}

	// Should still be 1 call
	if repo.getUserEventsByTypeCalls != 1 {
		t.Errorf("Expected 1 DB call (cached), got %d", repo.getUserEventsByTypeCalls)
	}

	// 3. Different user - should call DB
	if err := svc.RecordUserEvent(ctx, "other-user", domain.EventItemAdded, nil); err != nil {
		t.Fatalf("Failed: %v", err)
	}

	if repo.getUserEventsByTypeCalls != 2 {
		t.Errorf("Expected 2 DB calls (new user), got %d", repo.getUserEventsByTypeCalls)
	}
}

func TestRecordUserEventEmptyUserID(t *testing.T) {
	repo := &mockStatsRepository{}
	svc := NewService(repo)

	ctx := context.Background()
	err := svc.RecordUserEvent(ctx, "", domain.EventItemAdded, nil)
	if err == nil {
		t.Fatal("Expected error for empty user ID, got nil")
	}
}

func TestGetUserStats(t *testing.T) {
	repo := &mockStatsRepository{
		events: []domain.StatsEvent{
			{
				EventID:   1,
				UserID:    "user-123",
				EventType: domain.EventItemAdded,
				CreatedAt: time.Now().Add(-1 * time.Hour),
			},
			{
				EventID:   2,
				UserID:    "user-123",
				EventType: domain.EventItemSold,
				CreatedAt: time.Now().Add(-30 * time.Minute),
			},
			{
				EventID:   3,
				UserID:    "user-456",
				EventType: domain.EventItemAdded,
				CreatedAt: time.Now().Add(-20 * time.Minute),
			},
		},
	}

	svc := NewService(repo)
	ctx := context.Background()

	summary, err := svc.GetUserStats(ctx, "user-123", "daily")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if summary.TotalEvents != 2 {
		t.Errorf("Expected 2 events, got %d", summary.TotalEvents)
	}
}

func TestGetSystemStats(t *testing.T) {
	repo := &mockStatsRepository{
		events: []domain.StatsEvent{
			{EventID: 1, UserID: "user-1", EventType: domain.EventItemAdded, CreatedAt: time.Now().Add(-1 * time.Hour)},
			{EventID: 2, UserID: "user-2", EventType: domain.EventItemSold, CreatedAt: time.Now().Add(-30 * time.Minute)},
			{EventID: 3, UserID: "user-3", EventType: domain.EventItemAdded, CreatedAt: time.Now().Add(-20 * time.Minute)},
		},
	}

	svc := NewService(repo)
	ctx := context.Background()

	summary, err := svc.GetSystemStats(ctx, "daily")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if summary.TotalEvents != 3 {
		t.Errorf("Expected 3 events, got %d", summary.TotalEvents)
	}

	if summary.EventCounts[domain.EventItemAdded] != 2 {
		t.Errorf("Expected 2 item_added events, got %d", summary.EventCounts[domain.EventItemAdded])
	}
}

func TestGetLeaderboard(t *testing.T) {
	repo := &mockStatsRepository{
		events: []domain.StatsEvent{
			{EventID: 1, UserID: "user-1", EventType: domain.EventItemSold, CreatedAt: time.Now().Add(-1 * time.Hour)},
			{EventID: 2, UserID: "user-1", EventType: domain.EventItemSold, CreatedAt: time.Now().Add(-50 * time.Minute)},
			{EventID: 3, UserID: "user-2", EventType: domain.EventItemSold, CreatedAt: time.Now().Add(-30 * time.Minute)},
		},
	}

	svc := NewService(repo)
	ctx := context.Background()

	leaderboard, err := svc.GetLeaderboard(ctx, domain.EventItemSold, "daily", 10)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(leaderboard) != 2 {
		t.Fatalf("Expected 2 leaderboard entries, got %d", len(leaderboard))
	}

	// Check that users are counted correctly
	userCounts := make(map[string]int)
	for _, entry := range leaderboard {
		userCounts[entry.UserID] = entry.Count
	}

	if userCounts["user-1"] != 2 {
		t.Errorf("Expected user-1 to have 2 events, got %d", userCounts["user-1"])
	}
	if userCounts["user-2"] != 1 {
		t.Errorf("Expected user-2 to have 1 event, got %d", userCounts["user-2"])
	}
}
