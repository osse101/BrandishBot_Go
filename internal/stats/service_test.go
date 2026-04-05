package stats

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// mockStatsRepository implements Repository interface for testing
type mockStatsRepository struct {
	getTotalEventCountError                error
	getEventCountsError                    error
	getTopUsersError                       error
	getUserEventCountsError                error
	getUserEventsByTypeError               error
	getUserSlotsStatsError                 error
	getSlotsLeaderboardByProfitError       error
	getSlotsLeaderboardByWinRateError      error
	getSlotsLeaderboardByMegaJackpotsError error
	events                                 []domain.StatsEvent
	recordEventError                       error
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
	if m.getUserEventsByTypeError != nil {
		return nil, m.getUserEventsByTypeError
	}
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
	if m.getTopUsersError != nil {
		return nil, m.getTopUsersError
	}
	counts := make(map[string]int)
	for _, event := range m.events {
		if event.EventType == eventType && event.CreatedAt.After(startTime) && event.CreatedAt.Before(endTime) {
			counts[event.UserID]++
		}
	}

	entries := make([]domain.LeaderboardEntry, 0, len(counts))
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
	if m.getEventCountsError != nil {
		return nil, m.getEventCountsError
	}
	counts := make(map[domain.EventType]int)
	for _, event := range m.events {
		if event.CreatedAt.After(startTime) && event.CreatedAt.Before(endTime) {
			counts[event.EventType]++
		}
	}
	return counts, nil
}

func (m *mockStatsRepository) GetUserEventCounts(ctx context.Context, userID string, startTime, endTime time.Time) (map[domain.EventType]int, error) {
	if m.getUserEventCountsError != nil {
		return nil, m.getUserEventCountsError
	}
	counts := make(map[domain.EventType]int)
	for _, event := range m.events {
		if event.UserID == userID && event.CreatedAt.After(startTime) && event.CreatedAt.Before(endTime) {
			counts[event.EventType]++
		}
	}
	return counts, nil
}

func (m *mockStatsRepository) GetTotalEventCount(ctx context.Context, startTime, endTime time.Time) (int, error) {
	if m.getTotalEventCountError != nil {
		return 0, m.getTotalEventCountError
	}
	count := 0
	for _, event := range m.events {
		if event.CreatedAt.After(startTime) && event.CreatedAt.Before(endTime) {
			count++
		}
	}
	return count, nil
}

func (m *mockStatsRepository) GetUserSlotsStats(ctx context.Context, userID string, startTime, endTime time.Time) (*domain.SlotsStats, error) {
	if m.getUserSlotsStatsError != nil {
		return nil, m.getUserSlotsStatsError
	}
	return &domain.SlotsStats{}, nil
}

func (m *mockStatsRepository) GetSlotsLeaderboardByProfit(ctx context.Context, startTime, endTime time.Time, limit int) ([]domain.SlotsStats, error) {
	if m.getSlotsLeaderboardByProfitError != nil {
		return nil, m.getSlotsLeaderboardByProfitError
	}
	return []domain.SlotsStats{{UserID: "1"}}, nil
}

func (m *mockStatsRepository) GetSlotsLeaderboardByWinRate(ctx context.Context, startTime, endTime time.Time, minSpins, limit int) ([]domain.SlotsStats, error) {
	if m.getSlotsLeaderboardByWinRateError != nil {
		return nil, m.getSlotsLeaderboardByWinRateError
	}
	return []domain.SlotsStats{{UserID: "1"}}, nil
}

func (m *mockStatsRepository) GetSlotsLeaderboardByMegaJackpots(ctx context.Context, startTime, endTime time.Time, limit int) ([]domain.SlotsStats, error) {
	if m.getSlotsLeaderboardByMegaJackpotsError != nil {
		return nil, m.getSlotsLeaderboardByMegaJackpotsError
	}
	return []domain.SlotsStats{{UserID: "1"}}, nil
}

func TestRecordUserEvent(t *testing.T) {
	repo := &mockStatsRepository{}
	svc := NewService(repo)

	ctx := context.Background()
	userID := "test-user-123"
	eventType := domain.StatsEventItemAdded
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
	svc.RecordUserEvent(ctx, userID, domain.StatsEventItemAdded, nil)

	// Check if streak event was recorded
	if len(repo.events) != 2 { // 1 item_added, 1 daily_streak
		t.Errorf("Expected 2 events, got %d", len(repo.events))
	}
	streakEvent := repo.events[1]
	if streakEvent.EventType != domain.StatsEventDailyStreak {
		t.Errorf("Expected daily_streak event, got %s", streakEvent.EventType)
	}
	var streak int
	if s, ok := streakEvent.EventData.(domain.StreakMetadata); ok {
		streak = s.Streak
	} else if dataMap, ok := streakEvent.EventData.(map[string]interface{}); ok {
		// Handle map case if it falls back
		if s, ok := dataMap["streak"].(int); ok {
			streak = s
		}
	}
	if streak != 1 {
		t.Errorf("Expected streak 1, got %v", streak)
	}

	// 2. Second event same day - should NOT trigger new streak event
	svc.RecordUserEvent(ctx, userID, domain.StatsEventItemAdded, nil)
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
			EventType: domain.StatsEventDailyStreak,
			EventData: map[string]interface{}{"streak": 5},
			CreatedAt: yesterday,
		},
	}

	// Record event today - should increment streak to 6
	svc.RecordUserEvent(ctx, userID, domain.StatsEventItemAdded, nil)

	// repo.events should have: yesterday streak (preserved), today item_added, today streak
	// Note: `RecordUserEvent` appends.

	// Check the last event
	lastEvent := repo.events[len(repo.events)-1]
	if lastEvent.EventType != domain.StatsEventDailyStreak {
		t.Errorf("Expected streak event")
	}
	// It's a bit complicated because RecordUserEvent appends the struct, but we manually inserted maps earlier
	// The `lastEvent` (index len-1) is the one we just triggered, so it should have struct metadata
	var lastStreak int
	if s, ok := lastEvent.EventData.(domain.StreakMetadata); ok {
		lastStreak = s.Streak
	}
	if lastStreak != 6 {
		t.Errorf("Expected streak 6, got %v", lastStreak)
	}

	// 4. Simulate break in streak (2 days ago)
	twoDaysAgo := time.Now().AddDate(0, 0, -2)
	repo.events = []domain.StatsEvent{
		{
			EventID:   20,
			UserID:    userID,
			EventType: domain.StatsEventDailyStreak,
			EventData: map[string]interface{}{"streak": 10},
			CreatedAt: twoDaysAgo,
		},
	}

	// Record event today - should reset streak to 1
	svc.RecordUserEvent(ctx, userID, domain.StatsEventItemAdded, nil)

	lastEvent = repo.events[len(repo.events)-1]
	var resetStreak int
	if s, ok := lastEvent.EventData.(domain.StreakMetadata); ok {
		resetStreak = s.Streak
	}
	if resetStreak != 1 {
		t.Errorf("Expected streak 1 (reset), got %v", resetStreak)
	}
}

func TestRecordUserEventEmptyUserID(t *testing.T) {
	repo := &mockStatsRepository{}
	svc := NewService(repo)

	ctx := context.Background()
	err := svc.RecordUserEvent(ctx, "", domain.StatsEventItemAdded, nil)
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
				EventType: domain.StatsEventItemAdded,
				CreatedAt: time.Now().Add(-1 * time.Hour),
			},
			{
				EventID:   2,
				UserID:    "user-123",
				EventType: domain.StatsEventItemSold,
				CreatedAt: time.Now().Add(-30 * time.Minute),
			},
			{
				EventID:   3,
				UserID:    "user-456",
				EventType: domain.StatsEventItemAdded,
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
			{EventID: 1, UserID: "user-1", EventType: domain.StatsEventItemAdded, CreatedAt: time.Now().Add(-1 * time.Hour)},
			{EventID: 2, UserID: "user-2", EventType: domain.StatsEventItemSold, CreatedAt: time.Now().Add(-30 * time.Minute)},
			{EventID: 3, UserID: "user-3", EventType: domain.StatsEventItemAdded, CreatedAt: time.Now().Add(-20 * time.Minute)},
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

	if summary.EventCounts[domain.StatsEventItemAdded] != 2 {
		t.Errorf("Expected 2 item_added events, got %d", summary.EventCounts[domain.StatsEventItemAdded])
	}
}

func TestGetLeaderboard(t *testing.T) {
	repo := &mockStatsRepository{
		events: []domain.StatsEvent{
			{EventID: 1, UserID: "user-1", EventType: domain.StatsEventItemSold, CreatedAt: time.Now().Add(-1 * time.Hour)},
			{EventID: 2, UserID: "user-1", EventType: domain.StatsEventItemSold, CreatedAt: time.Now().Add(-50 * time.Minute)},
			{EventID: 3, UserID: "user-2", EventType: domain.StatsEventItemSold, CreatedAt: time.Now().Add(-30 * time.Minute)},
		},
	}

	svc := NewService(repo)
	ctx := context.Background()

	leaderboard, err := svc.GetLeaderboard(ctx, domain.StatsEventItemSold, "daily", 10)
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

func TestService_GetSystemStats_ErrorCounts(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{
		getEventCountsError: errors.New("db error"),
	}
	svc := NewService(mockRepo)

	_, err := svc.GetSystemStats(ctx, "daily")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_GetSystemStats_ErrorTotal(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{
		getTotalEventCountError: errors.New("db error"),
	}
	svc := NewService(mockRepo)

	_, err := svc.GetSystemStats(ctx, "daily")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_GetLeaderboard_Error(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{
		getTopUsersError: errors.New("db error"),
	}
	svc := NewService(mockRepo)

	_, err := svc.GetLeaderboard(ctx, domain.StatsEventItemSold, "daily", 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_GetUserStats_Error(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{
		getUserEventCountsError: errors.New("db error"),
	}
	svc := NewService(mockRepo)

	_, err := svc.GetUserStats(ctx, "user-1", "daily")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_GetUserSlotsStats_Error(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{
		getUserSlotsStatsError: errors.New("db error"),
	}
	svc := NewService(mockRepo)

	_, err := svc.GetUserSlotsStats(ctx, "user-1", "daily")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_GetSlotsLeaderboardByProfit_Error(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{
		getSlotsLeaderboardByProfitError: errors.New("db error"),
	}
	svc := NewService(mockRepo)

	_, err := svc.GetSlotsLeaderboardByProfit(ctx, "daily", 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_GetSlotsLeaderboardByWinRate_Error(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{
		getSlotsLeaderboardByWinRateError: errors.New("db error"),
	}
	svc := NewService(mockRepo)

	_, err := svc.GetSlotsLeaderboardByWinRate(ctx, "daily", 10, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_GetSlotsLeaderboardByMegaJackpots_Error(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{
		getSlotsLeaderboardByMegaJackpotsError: errors.New("db error"),
	}
	svc := NewService(mockRepo)

	_, err := svc.GetSlotsLeaderboardByMegaJackpots(ctx, "daily", 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_RecordUserEvent_Error(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{
		recordEventError: errors.New("db error"),
	}
	svc := NewService(mockRepo)

	err := svc.RecordUserEvent(ctx, "user-1", "some_event", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_RecordUserEvent_EmptyUser(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{}
	svc := NewService(mockRepo)

	err := svc.RecordUserEvent(ctx, "", "some_event", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_GetUserStats_EmptyUser(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{}
	svc := NewService(mockRepo)

	_, err := svc.GetUserStats(ctx, "", "daily")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_GetUserSlotsStats_EmptyUser(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{}
	svc := NewService(mockRepo)

	_, err := svc.GetUserSlotsStats(ctx, "", "daily")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_GetSlotsLeaderboardByProfit(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{}
	svc := NewService(mockRepo)

	_, err := svc.GetSlotsLeaderboardByProfit(ctx, "daily", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestService_GetSlotsLeaderboardByWinRate(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{}
	svc := NewService(mockRepo)

	_, err := svc.GetSlotsLeaderboardByWinRate(ctx, "daily", 10, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestService_GetSlotsLeaderboardByMegaJackpots(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{}
	svc := NewService(mockRepo)

	_, err := svc.GetSlotsLeaderboardByMegaJackpots(ctx, "daily", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestService_GetUserSlotsStats(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{}
	svc := NewService(mockRepo)

	stats, err := svc.GetUserSlotsStats(ctx, "user-1", "daily")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats == nil {
		t.Fatal("expected stats, got nil")
	}
}

func TestService_GetSystemStats(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{}
	svc := NewService(mockRepo)

	stats, err := svc.GetSystemStats(ctx, "daily")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats == nil {
		t.Fatal("expected stats, got nil")
	}
}

func TestService_GetLeaderboard(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{}
	svc := NewService(mockRepo)

	stats, err := svc.GetLeaderboard(ctx, domain.StatsEventItemSold, "daily", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats == nil {
		t.Fatal("expected stats, got nil")
	}
}

func TestService_GetUserStats(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{}
	svc := NewService(mockRepo)

	stats, err := svc.GetUserStats(ctx, "user-1", "daily")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats == nil {
		t.Fatal("expected stats, got nil")
	}
}

func TestService_GetUserCurrentStreak(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{
		events: []domain.StatsEvent{
			{
				EventID:   1,
				UserID:    "user-1",
				EventType: domain.StatsEventDailyStreak,
				EventData: map[string]interface{}{"streak": 5},
				CreatedAt: time.Now(),
			},
		},
	}
	svc := NewService(mockRepo)

	streak, err := svc.GetUserCurrentStreak(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if streak != 5 {
		t.Errorf("expected streak 5, got %d", streak)
	}
}

func TestService_GetUserCurrentStreak_Yesterday(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{
		events: []domain.StatsEvent{
			{
				EventID:   1,
				UserID:    "user-1",
				EventType: domain.StatsEventDailyStreak,
				EventData: map[string]interface{}{"streak": 5},
				CreatedAt: time.Now().AddDate(0, 0, -1),
			},
		},
	}
	svc := NewService(mockRepo)

	streak, err := svc.GetUserCurrentStreak(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if streak != 5 {
		t.Errorf("expected streak 5, got %d", streak)
	}
}

func TestService_GetUserCurrentStreak_Older(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{
		events: []domain.StatsEvent{
			{
				EventID:   1,
				UserID:    "user-1",
				EventType: domain.StatsEventDailyStreak,
				EventData: map[string]interface{}{"streak": 5},
				CreatedAt: time.Now().AddDate(0, 0, -2),
			},
		},
	}
	svc := NewService(mockRepo)

	streak, err := svc.GetUserCurrentStreak(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if streak != 0 {
		t.Errorf("expected streak 0, got %d", streak)
	}
}

func TestService_GetUserCurrentStreak_NoEvents(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{}
	svc := NewService(mockRepo)

	streak, err := svc.GetUserCurrentStreak(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if streak != 0 {
		t.Errorf("expected streak 0, got %d", streak)
	}
}

func TestService_GetUserCurrentStreak_Error(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{
		getUserEventsByTypeError: errors.New("db error"),
	}
	svc := NewService(mockRepo)

	_, err := svc.GetUserCurrentStreak(ctx, "user-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_getPeriodRange(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{}
	svc := NewService(mockRepo)

	stats, _ := svc.GetUserStats(ctx, "user-1", "hourly")
	require.NotNil(t, stats)
	stats, _ = svc.GetUserStats(ctx, "user-1", "weekly")
	require.NotNil(t, stats)
	stats, _ = svc.GetUserStats(ctx, "user-1", "monthly")
	require.NotNil(t, stats)
	stats, _ = svc.GetUserStats(ctx, "user-1", "yearly")
	require.NotNil(t, stats)
	stats, _ = svc.GetUserStats(ctx, "user-1", "all")
	require.NotNil(t, stats)
	stats, _ = svc.GetUserStats(ctx, "user-1", "unknown")
	require.NotNil(t, stats)
}

func TestService_GetSlotsLeaderboardByWinRate_LimitZero(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{}
	svc := NewService(mockRepo)

	lb, err := svc.GetSlotsLeaderboardByWinRate(ctx, "daily", 0, 0)
	require.NoError(t, err)
	require.NotNil(t, lb)
}

func TestService_getSlotsLeaderboard_LimitZero(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{}
	svc := NewService(mockRepo)

	// Calls getSlotsLeaderboard internally with limit 0
	lb, err := svc.GetSlotsLeaderboardByProfit(ctx, "daily", 0)
	require.NoError(t, err)
	require.NotNil(t, lb)
}

func TestService_GetSlotsLeaderboardByWinRate_LimitZero2(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{}
	svc := NewService(mockRepo)

	// ensure limit logic works
	lb, err := svc.GetSlotsLeaderboardByWinRate(ctx, "daily", 1, 0)
	require.NoError(t, err)
	require.NotNil(t, lb)
}

func TestService_GetLeaderboard_Error2(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockStatsRepository{
		getTopUsersError: errors.New("db error"),
	}
	svc := NewService(mockRepo)
	svc.GetLeaderboard(ctx, domain.StatsEventItemSold, "daily", 10)
}
