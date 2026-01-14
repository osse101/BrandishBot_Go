package stats

import (
	"context"
	"fmt"
	"sync"
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

	// streakCheckCache tracks when a user's streak was last checked/updated.
	// Key: userID, Value: Time of last successful check
	streakCheckCache map[string]time.Time
	cacheMu          sync.RWMutex
}

// NewService creates a new stats service
func NewService(repo repository.Stats) Service {
	return &service{
		repo:             repo,
		streakCheckCache: make(map[string]time.Time),
	}
}

// RecordUserEvent records a user event with the provided metadata
func (s *service) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata map[string]interface{}) error {
	log := logger.FromContext(ctx)

	if userID == "" {
		return fmt.Errorf("user ID is required")
	}

	event := &domain.StatsEvent{
		UserID:    userID,
		EventType: eventType,
		EventData: metadata,
		CreatedAt: time.Now(),
	}

	if err := s.repo.RecordEvent(ctx, event); err != nil {
		log.Error("Failed to record event", "error", err, "user_id", userID, "event_type", eventType)
		return fmt.Errorf("failed to record event: %w", err)
	}

	log.Debug("Event recorded", "event_id", event.EventID, "user_id", userID, "event_type", eventType)

	// Check for daily streak
	if eventType != domain.EventDailyStreak {
		if err := s.checkDailyStreak(ctx, userID); err != nil {
			log.Warn("Failed to check daily streak", "error", err, "user_id", userID)
		}
	}

	return nil
}

// checkDailyStreak calculates and records daily login streak
func (s *service) checkDailyStreak(ctx context.Context, userID string) error {
	now := time.Now()
	y2, m2, d2 := now.UTC().Date()

	// Optimization: Check in-memory cache first
	s.cacheMu.RLock()
	lastChecked, found := s.streakCheckCache[userID]
	s.cacheMu.RUnlock()

	if found {
		y1, m1, d1 := lastChecked.UTC().Date()
		// If already checked today, we can skip the database query
		if y1 == y2 && m1 == m2 && d1 == d2 {
			return nil
		}
	}

	// Get the last streak event from DB
	events, err := s.repo.GetUserEventsByType(ctx, userID, domain.EventDailyStreak, 1)
	if err != nil {
		return fmt.Errorf("failed to get streak events: %w", err)
	}

	var lastStreak int
	var lastStreakTime time.Time

	if len(events) > 0 {
		lastStreakTime = events[0].CreatedAt
		// Extract streak from metadata
		if streakVal, ok := events[0].EventData["streak"]; ok {
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

	// Compare dates (UTC)
	yLast, mLast, dLast := lastStreakTime.UTC().Date()

	// Update cache immediately if we confirm it's already done today
	if yLast == y2 && mLast == m2 && dLast == d2 {
		s.updateCache(userID, now)
		return nil
	}

	// Check if it was yesterday
	yesterday := now.UTC().AddDate(0, 0, -1)
	y3, m3, d3 := yesterday.Date()

	newStreak := 1
	// If last streak was yesterday, increment
	if yLast == y3 && mLast == m3 && dLast == d3 {
		newStreak = lastStreak + 1
	}

	// Record new streak
	meta := map[string]interface{}{
		"streak": newStreak,
	}

	// Use RecordUserEvent but with EventDailyStreak type (which will be skipped by the check above)
	// Triggers "STREAK_INCREASED" if streak > 1? The client can handle that based on event.
	if err := s.RecordUserEvent(ctx, userID, domain.EventDailyStreak, meta); err != nil {
		return fmt.Errorf("failed to record streak event: %w", err)
	}

	// Update cache after successful record
	s.updateCache(userID, now)

	return nil
}

func (s *service) updateCache(userID string, timestamp time.Time) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	// Prevent memory leak: Reset cache if it gets too large
	if len(s.streakCheckCache) > 5000 {
		s.streakCheckCache = make(map[string]time.Time)
	}
	s.streakCheckCache[userID] = timestamp
}

// GetUserCurrentStreak retrieves the current daily login streak for a user
func (s *service) GetUserCurrentStreak(ctx context.Context, userID string) (int, error) {
	// Get the last streak event
	events, err := s.repo.GetUserEventsByType(ctx, userID, domain.EventDailyStreak, 1)
	if err != nil {
		return 0, fmt.Errorf("failed to get streak events: %w", err)
	}

	if len(events) == 0 {
		return 0, nil
	}

	lastEvent := events[0]
	lastStreakTime := lastEvent.CreatedAt
	var streak int

	// Extract streak from metadata
	if streakVal, ok := lastEvent.EventData["streak"]; ok {
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
	yesterday := now.UTC().AddDate(0, 0, -1)
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
		return nil, fmt.Errorf("user ID is required")
	}

	startTime, endTime := getPeriodRange(period)

	eventCounts, err := s.repo.GetUserEventCounts(ctx, userID, startTime, endTime)
	if err != nil {
		log.Error("Failed to get user event counts", "error", err, "user_id", userID)
		return nil, fmt.Errorf("failed to get user event counts: %w", err)
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

	log.Debug("Retrieved user stats", "user_id", userID, "period", period, "total_events", totalEvents)
	return summary, nil
}

// GetSystemStats retrieves system-wide statistics for a time period
func (s *service) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) {
	log := logger.FromContext(ctx)

	startTime, endTime := getPeriodRange(period)

	totalEvents, err := s.repo.GetTotalEventCount(ctx, startTime, endTime)
	if err != nil {
		log.Error("Failed to get total event count", "error", err)
		return nil, fmt.Errorf("failed to get total event count: %w", err)
	}

	eventCounts, err := s.repo.GetEventCounts(ctx, startTime, endTime)
	if err != nil {
		log.Error("Failed to get event counts", "error", err)
		return nil, fmt.Errorf("failed to get event counts: %w", err)
	}

	summary := &domain.StatsSummary{
		Period:      period,
		StartTime:   startTime,
		EndTime:     endTime,
		TotalEvents: totalEvents,
		EventCounts: eventCounts,
	}

	log.Debug("Retrieved system stats", "period", period, "total_events", totalEvents)
	return summary, nil
}

// GetLeaderboard retrieves the leaderboard for a specific event type and time period
func (s *service) GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error) {
	log := logger.FromContext(ctx)

	if limit <= 0 {
		limit = 10 // Default to top 10
	}

	startTime, endTime := getPeriodRange(period)

	entries, err := s.repo.GetTopUsers(ctx, eventType, startTime, endTime, limit)
	if err != nil {
		log.Error("Failed to get leaderboard", "error", err, "event_type", eventType)
		return nil, fmt.Errorf("failed to get leaderboard: %w", err)
	}

	log.Debug("Retrieved leaderboard", "event_type", eventType, "period", period, "entries", len(entries))
	return entries, nil
}

// getPeriodRange calculates the start and end time for a given period
func getPeriodRange(period string) (startTime, endTime time.Time) {
	now := time.Now()
	endTime = now

	switch period {
	case "hourly":
		startTime = now.Add(-1 * time.Hour)
	case "daily":
		startTime = now.AddDate(0, 0, -1)
	case "weekly":
		startTime = now.AddDate(0, 0, -7)
	case "monthly":
		startTime = now.AddDate(0, -1, 0)
	case "yearly":
		startTime = now.AddDate(-1, 0, 0)
	case "all":
		// Set to a very old date for "all time"
		startTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	default:
		// Default to daily
		startTime = now.AddDate(0, 0, -1)
	}

	return startTime, endTime
}
