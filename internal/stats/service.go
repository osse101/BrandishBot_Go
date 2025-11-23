package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// Repository defines the interface for stats persistence
type Repository interface {
	RecordEvent(ctx context.Context, event *domain.StatsEvent) error
	GetEventsByUser(ctx context.Context, userID string, startTime, endTime time.Time) ([]domain.StatsEvent, error)
	GetEventsByType(ctx context.Context, eventType domain.EventType, startTime, endTime time.Time) ([]domain.StatsEvent, error)
	GetTopUsers(ctx context.Context, eventType domain.EventType, startTime, endTime time.Time, limit int) ([]domain.LeaderboardEntry, error)
	GetEventCounts(ctx context.Context, startTime, endTime time.Time) (map[domain.EventType]int, error)
	GetUserEventCounts(ctx context.Context, userID string, startTime, endTime time.Time) (map[domain.EventType]int, error)
	GetTotalEventCount(ctx context.Context, startTime, endTime time.Time) (int, error)
}

// Service defines the interface for stats operations
type Service interface {
	RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata map[string]interface{}) error
	GetUserStats(ctx context.Context, userID string, period string) (*domain.StatsSummary, error)
	GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error)
	GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error)
}

// service implements the Service interface
type service struct {
	repo Repository
}

// NewService creates a new stats service
func NewService(repo Repository) Service {
	return &service{
		repo: repo,
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
	return nil
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
