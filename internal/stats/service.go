package stats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Service defines the interface for stats operations
type Service interface {
	RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata interface{}) error
	GetUserStats(ctx context.Context, userID string, period string) (*domain.StatsSummary, error)
	GetUserCurrentStreak(ctx context.Context, userID string) (int, error)
	GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error)
	GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error)
	// Slots-specific stats
	GetUserSlotsStats(ctx context.Context, userID string, period string) (*domain.SlotsStats, error)
	GetSlotsLeaderboardByProfit(ctx context.Context, period string, limit int) ([]domain.SlotsStats, error)
	GetSlotsLeaderboardByWinRate(ctx context.Context, period string, minSpins, limit int) ([]domain.SlotsStats, error)
	GetSlotsLeaderboardByMegaJackpots(ctx context.Context, period string, limit int) ([]domain.SlotsStats, error)
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
func (s *service) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata interface{}) error {
	log := logger.FromContext(ctx)

	if userID == "" {
		return errors.New(ErrMsgUserIDRequired)
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
	if eventType != domain.StatsEventDailyStreak {
		if err := s.checkDailyStreak(ctx, userID); err != nil {
			log.Warn(LogMsgFailedToCheckDailyStreak, "error", err, "user_id", userID)
		}
	}

	return nil
}

// checkDailyStreak calculates and records daily login streak
func (s *service) checkDailyStreak(ctx context.Context, userID string) error {
	// Get the last streak event
	events, err := s.repo.GetUserEventsByType(ctx, userID, domain.StatsEventDailyStreak, StreakEventQueryLimit)
	if err != nil {
		return fmt.Errorf(ErrMsgGetStreakEventsFailed, err)
	}

	var lastStreak int
	var lastStreakTime time.Time

	if len(events) > 0 {
		lastStreakTime = events[0].CreatedAt
		// Extract streak from metadata using typed decoder
		if meta, err := event.DecodePayload[domain.StreakMetadata](events[0].EventData); err == nil {
			lastStreak = meta.Streak
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
	meta := domain.StreakMetadata{
		Streak: newStreak,
	}

	// Use RecordUserEvent but with EventDailyStreak type (which will be skipped by the check above)
	// Triggers "STREAK_INCREASED" if streak > 1? The client can handle that based on event.
	if err := s.RecordUserEvent(ctx, userID, domain.StatsEventDailyStreak, meta); err != nil {
		return fmt.Errorf(ErrMsgRecordStreakEventFailed, err)
	}

	return nil
}

// GetUserCurrentStreak retrieves the current daily login streak for a user
func (s *service) GetUserCurrentStreak(ctx context.Context, userID string) (int, error) {
	// Get the last streak event
	events, err := s.repo.GetUserEventsByType(ctx, userID, domain.StatsEventDailyStreak, StreakEventQueryLimit)
	if err != nil {
		return 0, fmt.Errorf(ErrMsgGetStreakEventsFailed, err)
	}

	if len(events) == 0 {
		return 0, nil
	}

	lastEvent := events[0]
	lastStreakTime := lastEvent.CreatedAt
	var streak int

	// Extract streak from metadata using typed decoder
	if meta, err := event.DecodePayload[domain.StreakMetadata](lastEvent.EventData); err == nil {
		streak = meta.Streak
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
		return nil, errors.New(ErrMsgUserIDRequired)
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

// GetUserSlotsStats retrieves slots statistics for a specific user
func (s *service) GetUserSlotsStats(ctx context.Context, userID string, period string) (*domain.SlotsStats, error) {
	log := logger.FromContext(ctx)

	if userID == "" {
		return nil, errors.New(ErrMsgUserIDRequired)
	}

	startTime, endTime := getPeriodRange(period)

	stats, err := s.repo.GetUserSlotsStats(ctx, userID, startTime, endTime)
	if err != nil {
		log.Error("Failed to get user slots stats", "error", err, "user_id", userID)
		return nil, fmt.Errorf("failed to get user slots stats: %w", err)
	}

	stats.Period = period
	log.Debug("Retrieved user slots stats", "user_id", userID, "period", period, "total_spins", stats.TotalSpins)
	return stats, nil
}

// GetSlotsLeaderboardByProfit retrieves the slots leaderboard ranked by net profit
func (s *service) GetSlotsLeaderboardByProfit(ctx context.Context, period string, limit int) ([]domain.SlotsStats, error) {
	return s.getSlotsLeaderboard(ctx, period, limit, s.repo.GetSlotsLeaderboardByProfit, "profit")
}

// GetSlotsLeaderboardByWinRate retrieves the slots leaderboard ranked by win rate
func (s *service) GetSlotsLeaderboardByWinRate(ctx context.Context, period string, minSpins, limit int) ([]domain.SlotsStats, error) {
	log := logger.FromContext(ctx)

	if limit <= 0 {
		limit = DefaultLeaderboardLimit
	}

	if minSpins <= 0 {
		minSpins = 10 // Default minimum spins to qualify
	}

	startTime, endTime := getPeriodRange(period)

	stats, err := s.repo.GetSlotsLeaderboardByWinRate(ctx, startTime, endTime, minSpins, limit)
	if err != nil {
		log.Error("Failed to get slots leaderboard by win rate", "error", err)
		return nil, fmt.Errorf("failed to get slots leaderboard by win rate: %w", err)
	}

	// Set period on all entries
	for i := range stats {
		stats[i].Period = period
	}

	log.Debug("Retrieved slots leaderboard by win rate", "period", period, "entries", len(stats))
	return stats, nil
}

// GetSlotsLeaderboardByMegaJackpots retrieves the slots leaderboard ranked by mega jackpots hit
func (s *service) GetSlotsLeaderboardByMegaJackpots(ctx context.Context, period string, limit int) ([]domain.SlotsStats, error) {
	return s.getSlotsLeaderboard(ctx, period, limit, s.repo.GetSlotsLeaderboardByMegaJackpots, "mega jackpots")
}

func (s *service) getSlotsLeaderboard(
	ctx context.Context,
	period string,
	limit int,
	fetchFn func(context.Context, time.Time, time.Time, int) ([]domain.SlotsStats, error),
	metricName string,
) ([]domain.SlotsStats, error) {
	log := logger.FromContext(ctx)

	if limit <= 0 {
		limit = DefaultLeaderboardLimit
	}

	startTime, endTime := getPeriodRange(period)

	stats, err := fetchFn(ctx, startTime, endTime, limit)
	if err != nil {
		log.Error("Failed to get slots leaderboard", "metric", metricName, "error", err)
		return nil, fmt.Errorf("failed to get slots leaderboard by %s: %w", metricName, err)
	}

	// Set period on all entries
	for i := range stats {
		stats[i].Period = period
	}

	log.Debug("Retrieved slots leaderboard", "metric", metricName, "period", period, "entries", len(stats))
	return stats, nil
}
