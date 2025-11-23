package stats

import (
	"context"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// mockStatsRepository implements Repository interface for testing
type mockStatsRepository struct {
	events           []domain.StatsEvent
	recordEventError error
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
		"item": "sword",
		"quantity": 5,
	}
	
	err := svc.RecordUserEvent(ctx, userID, eventType, metadata)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(repo.events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(repo.events))
	}
	
	event := repo.events[0]
	if event.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, event.UserID)
	}
	if event.EventType != eventType {
		t.Errorf("Expected event type %s, got %s", eventType, event.EventType)
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
