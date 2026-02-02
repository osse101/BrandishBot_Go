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
	log.Info(LogMsgCleanupJobStarting, LogFieldRetentionDays, j.retentionDays)

	start := time.Now()
	count, err := j.service.CleanupOldEvents(ctx, j.retentionDays)
	duration := time.Since(start)

	if err != nil {
		log.Error(LogMsgCleanupJobFailed, LogFieldError, err, LogFieldDuration, duration)
		return err
	}

	log.Info(LogMsgCleanupJobCompleted, LogFieldDeletedCount, count, LogFieldDuration, duration)
	return nil
}
