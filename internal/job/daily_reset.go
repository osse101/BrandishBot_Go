package job

import (
	"context"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// ResetDailyJobXP resets the daily XP counters for all users
// Returns the number of records affected
func (s *service) ResetDailyJobXP(ctx context.Context) (int64, error) {
	log := logger.FromContext(ctx)
	log.Info("Starting daily job XP reset")

	recordsAffected, err := s.repo.ResetDailyJobXP(ctx)
	if err != nil {
		log.Error("Daily XP reset failed", "error", err)
		return 0, err
	}

	// Update the reset state in the database
	now := time.Now().UTC()
	if err := s.repo.UpdateDailyResetTime(ctx, now, recordsAffected); err != nil {
		log.Warn("Failed to update reset state", "error", err)
		// Don't fail the reset operation itself, just warn
	} else {
		// Update cache
		s.resetCacheMu.Lock()
		s.resetCache = &domain.DailyResetStatus{
			LastResetTime:   now,
			RecordsAffected: recordsAffected,
		}
		s.resetCacheMu.Unlock()
	}

	log.Info("Daily XP reset completed", "records_affected", recordsAffected)

	// Publish event
	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.NewDailyResetCompleteEvent(time.Now().UTC(), recordsAffected))
	}

	return recordsAffected, nil
}

// GetDailyResetStatus returns information about the daily reset state
func (s *service) GetDailyResetStatus(ctx context.Context) (*domain.DailyResetStatus, error) {
	s.resetCacheMu.RLock()
	cache := s.resetCache
	s.resetCacheMu.RUnlock()

	var lastReset time.Time
	var recordsAffected int64

	if cache != nil {
		lastReset = cache.LastResetTime
		recordsAffected = cache.RecordsAffected
	} else {
		var err error
		lastReset, recordsAffected, err = s.repo.GetLastDailyResetTime(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get reset status: %w", err)
		}

		// Fill cache
		s.resetCacheMu.Lock()
		s.resetCache = &domain.DailyResetStatus{
			LastResetTime:   lastReset,
			RecordsAffected: recordsAffected,
		}
		s.resetCacheMu.Unlock()
	}

	// Calculate next reset time (00:00 UTC+7)
	// UTC+7 is 7 hours ahead of UTC. 00:00 UTC+7 is 17:00 UTC of previous day.
	location := time.FixedZone("UTC+7", 7*60*60)
	now := time.Now().In(location)
	nextReset := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	if !nextReset.After(now) {
		nextReset = nextReset.AddDate(0, 0, 1)
	}

	return &domain.DailyResetStatus{
		LastResetTime:   lastReset,
		NextResetTime:   nextReset.UTC(),
		RecordsAffected: recordsAffected,
	}, nil
}
