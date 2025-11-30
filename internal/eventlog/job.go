package eventlog

import (
	"context"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// CleanupJob is a job that cleans up old events
type CleanupJob struct {
	service       Service
	retentionDays int
}

// NewCleanupJob creates a new cleanup job
func NewCleanupJob(service Service, retentionDays int) *CleanupJob {
	return &CleanupJob{
		service:       service,
		retentionDays: retentionDays,
	}
}

// Process executes the cleanup job
func (j *CleanupJob) Process(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("Starting event log cleanup job", "retentionDays", j.retentionDays)

	start := time.Now()
	count, err := j.service.CleanupOldEvents(ctx, j.retentionDays)
	duration := time.Since(start)

	if err != nil {
		log.Error("Event log cleanup failed", "error", err, "duration", duration)
		return err
	}

	log.Info("Event log cleanup completed", "deletedCount", count, "duration", duration)
	return nil
}
