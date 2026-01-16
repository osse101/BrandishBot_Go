package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Service defines the interface for stats operations
type Service interface {
	RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata map[string]interface{}) error
	GetUserStats(ctx context.Context, userID string, period string) (*domain.StatsSummary, error)
	GetUserCurrentStreak(ctx context.Context, userID string) (int, error)
	GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error)
	GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error)
}

// service implements the Service interface
type service struct {
	repo repository.Stats
}

// NewService creates a new stats service
func NewService(repo repository.Stats) Service {
	return &service{
		repo: repo,
	}
}

// RecordUserEvent records a user event with the provided metadata
func (s *service) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata map[string]interface{}) error {
	log := logger.FromContext(ctx)

	if userID == "" {
		return fmt.Errorf(ErrMsgUserIDRequired)
	}

	event := &domain.StatsEvent{
		UserID:    userID,
		EventType: eventType,
		EventData: metadata,
		CreatedAt: time.Now(),
	}

	if err := s.repo.RecordEvent(ctx, event); err != nil {
		log.Error(LogMsgFailedToRecordEvent, "error", err, "user_id", userID, "event_type", eventType)
		return fmt.Errorf(ErrMsgRecordEventFailed, err)
	}

	log.Debug(LogMsgEventRecorded, "event_id", event.EventID, "user_id", userID, "event_type", eventType)

	// Check for daily streak
	if eventType != domain.EventDailyStreak {
		if err := s.checkDailyStreak(ctx, userID); err != nil {
			log.Warn(LogMsgFailedToCheckDailyStreak, "error", err, "user_id", userID)
		}
	}

	return nil
}

// checkDailyStreak calculates and records daily login streak
func (s *service) checkDailyStreak(ctx context.Context, userID string) error {
	// Get the last streak event
	events, err := s.repo.GetUserEventsByType(ctx, userID, domain.EventDailyStreak, StreakEventQueryLimit)
	if err != nil {
		return fmt.Errorf(ErrMsgGetStreakEventsFailed, err)
	}

	var lastStreak int
	var lastStreakTime time.Time

	if len(events) > 0 {
		lastStreakTime = events[0].CreatedAt
		// Extract streak from metadata
		if streakVal, ok := events[0].EventData[MetadataKeyStreak]; ok {
			// Handle float64 (JSON default) or int
			switch v := streakVal.(type) {
			case float64:
				lastStreak = int(v)
			case int:
				lastStreak = v
			case int64:
				lastStreak = int(v)
			}
		}
	}

	now := time.Now()
	// Compare dates (UTC)
	y1, m1, d1 := lastStreakTime.UTC().Date()
	y2, m2, d2 := now.UTC().Date()

	// If already recorded today, do nothing
	if y1 == y2 && m1 == m2 && d1 == d2 {
		return nil
	}

	// Check if it was yesterday
	yesterday := now.UTC().AddDate(0, 0, DayOffsetYesterday)
	y3, m3, d3 := yesterday.Date()

	newStreak := 1
	// If last streak was yesterday, increment
	if y1 == y3 && m1 == m3 && d1 == d3 {
		newStreak = lastStreak + 1
	}

	// Record new streak
	meta := map[string]interface{}{
		MetadataKeyStreak: newStreak,
	}

	// Use RecordUserEvent but with EventDailyStreak type (which will be skipped by the check above)
	// Triggers "STREAK_INCREASED" if streak > 1? The client can handle that based on event.
	if err := s.RecordUserEvent(ctx, userID, domain.EventDailyStreak, meta); err != nil {
		return fmt.Errorf(ErrMsgRecordStreakEventFailed, err)
	}

	return nil
}

// GetUserCurrentStreak retrieves the current daily login streak for a user
func (s *service) GetUserCurrentStreak(ctx context.Context, userID string) (int, error) {
	// Get the last streak event
	events, err := s.repo.GetUserEventsByType(ctx, userID, domain.EventDailyStreak, StreakEventQueryLimit)
	if err != nil {
		return 0, fmt.Errorf(ErrMsgGetStreakEventsFailed, err)
	}

	if len(events) == 0 {
		return 0, nil
	}

	lastEvent := events[0]
	lastStreakTime := lastEvent.CreatedAt
	var streak int

	// Extract streak from metadata
	if streakVal, ok := lastEvent.EventData[MetadataKeyStreak]; ok {
		switch v := streakVal.(type) {
		case float64:
			streak = int(v)
		case int:
			streak = v
		case int64:
			streak = int(v)
		}
	}

	now := time.Now()
	// Compare dates (UTC)
	y1, m1, d1 := lastStreakTime.UTC().Date()
	y2, m2, d2 := now.UTC().Date()

	// If today, valid
	if y1 == y2 && m1 == m2 && d1 == d2 {
		return streak, nil
	}

	// If yesterday, valid
	yesterday := now.UTC().AddDate(0, 0, DayOffsetYesterday)
	y3, m3, d3 := yesterday.Date()
	if y1 == y3 && m1 == m3 && d1 == d3 {
		return streak, nil
	}

	// Otherwise, streak is broken (return 0)
	return 0, nil
}

// GetUserStats retrieves statistics for a specific user within a time period
func (s *service) GetUserStats(ctx context.Context, userID string, period string) (*domain.StatsSummary, error) {
	log := logger.FromContext(ctx)

	if userID == "" {
		return nil, fmt.Errorf(ErrMsgUserIDRequired)
	}

	startTime, endTime := getPeriodRange(period)

	eventCounts, err := s.repo.GetUserEventCounts(ctx, userID, startTime, endTime)
	if err != nil {
		log.Error(LogMsgFailedToGetUserEventCounts, "error", err, "user_id", userID)
		return nil, fmt.Errorf(ErrMsgGetUserEventCountsFailed, err)
	}

	totalEvents := 0
	for _, count := range eventCounts {
		totalEvents += count
	}

	summary := &domain.StatsSummary{
		Period:      period,
		StartTime:   startTime,
		EndTime:     endTime,
		TotalEvents: totalEvents,
		EventCounts: eventCounts,
	}

	log.Debug(LogMsgRetrievedUserStats, "user_id", userID, "period", period, "total_events", totalEvents)
	return summary, nil
}

// GetSystemStats retrieves system-wide statistics for a time period
func (s *service) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) {
	log := logger.FromContext(ctx)

	startTime, endTime := getPeriodRange(period)

	totalEvents, err := s.repo.GetTotalEventCount(ctx, startTime, endTime)
	if err != nil {
		log.Error(LogMsgFailedToGetTotalEventCount, "error", err)
		return nil, fmt.Errorf(ErrMsgGetTotalEventCountFailed, err)
	}

	eventCounts, err := s.repo.GetEventCounts(ctx, startTime, endTime)
	if err != nil {
		log.Error(LogMsgFailedToGetEventCounts, "error", err)
		return nil, fmt.Errorf(ErrMsgGetEventCountsFailed, err)
	}

	summary := &domain.StatsSummary{
		Period:      period,
		StartTime:   startTime,
		EndTime:     endTime,
		TotalEvents: totalEvents,
		EventCounts: eventCounts,
	}

	log.Debug(LogMsgRetrievedSystemStats, "period", period, "total_events", totalEvents)
	return summary, nil
}

// GetLeaderboard retrieves the leaderboard for a specific event type and time period
func (s *service) GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error) {
	log := logger.FromContext(ctx)

	if limit <= 0 {
		limit = DefaultLeaderboardLimit
	}

	startTime, endTime := getPeriodRange(period)

	entries, err := s.repo.GetTopUsers(ctx, eventType, startTime, endTime, limit)
	if err != nil {
		log.Error(LogMsgFailedToGetLeaderboard, "error", err, "event_type", eventType)
		return nil, fmt.Errorf(ErrMsgGetLeaderboardFailed, err)
	}

	log.Debug(LogMsgRetrievedLeaderboard, "event_type", eventType, "period", period, "entries", len(entries))
	return entries, nil
}

// getPeriodRange calculates the start and end time for a given period
func getPeriodRange(period string) (startTime, endTime time.Time) {
	now := time.Now()
	endTime = now

	switch period {
	case PeriodHourly:
		startTime = now.Add(-1 * time.Hour)
	case PeriodDaily:
		startTime = now.AddDate(0, 0, -1)
	case PeriodWeekly:
		startTime = now.AddDate(0, 0, -7)
	case PeriodMonthly:
		startTime = now.AddDate(0, -1, 0)
	case PeriodYearly:
		startTime = now.AddDate(-1, 0, 0)
	case PeriodAll:
		// Set to a very old date for "all time"
		startTime = time.Date(AllTimeStartYear, AllTimeStartMonth, AllTimeStartDay, 0, 0, 0, 0, time.UTC)
	default:
		// Default to daily
		startTime = now.AddDate(0, 0, -1)
	}

	return startTime, endTime
}
