package worker

import (
	"context"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// DailyResetWorker handles scheduled daily resets for job XP caps at 00:00 UTC+7
type DailyResetWorker struct {
	jobService job.Service
	publisher  *event.ResilientPublisher
	timer      *time.Timer
	shutdown   chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex
}

// NewDailyResetWorker creates a new DailyResetWorker
func NewDailyResetWorker(jobService job.Service, publisher *event.ResilientPublisher) *DailyResetWorker {
	return &DailyResetWorker{
		jobService: jobService,
		publisher:  publisher,
		shutdown:   make(chan struct{}),
	}
}

// Start initializes the worker and schedules the first reset
func (w *DailyResetWorker) Start() {
	w.scheduleNext()
}

// scheduleNext calculates the time until next reset (00:00 UTC+7) and schedules the reset
func (w *DailyResetWorker) scheduleNext() {
	duration := timeUntilNextReset()
	log := logger.FromContext(context.Background())

	w.mu.Lock()
	if w.timer != nil {
		w.timer.Stop()
	}

	// Two-stage scheduling to prevent "tight loop" rescheduling caused by early triggers
	if duration > 1*time.Hour {
		// Stage 1: Long-range (Standby). Wake up 45 minutes before reset.
		waitDuration := duration - 45*time.Minute
		w.timer = time.AfterFunc(waitDuration, func() {
			w.scheduleNext()
		})
		w.mu.Unlock()

		nextCheck := time.Now().UTC().Add(waitDuration)
		log.Info(LogMsgDailyResetStandby, "next_check_at", nextCheck)
		return
	}

	// Stage 2: Final approach. Schedule the actual reset.
	w.timer = time.AfterFunc(duration, func() {
		select {
		case <-w.shutdown:
			return
		default:
		}

		// Jitter protection: if the timer triggered early (jitter > 10s),
		// simply reschedule for the remaining time.
		// If duration is > 23h, it means we are actually on time or slightly LATE.
		rem := timeUntilNextReset()
		if rem > 10*time.Second && rem < 23*time.Hour {
			w.scheduleNext()
			return
		}

		w.executeReset()
		w.scheduleNext() // This will now calculate ~24h and jump back to Stage 1
	})
	w.mu.Unlock()

	nextReset := time.Now().UTC().Add(duration)
	log.Info(LogMsgDailyResetApproach, "next_reset_at", nextReset)
}

// executeReset performs the daily reset in a tracked goroutine
func (w *DailyResetWorker) executeReset() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		ctx := context.Background()
		log := logger.FromContext(ctx)
		log.Info(LogMsgDailyResetStarting)

		recordsAffected, err := w.jobService.ResetDailyJobXP(ctx)
		if err != nil {
			log.Error(LogMsgDailyResetFailed, "error", err)
			return
		}

		log.Info(LogMsgDailyResetCompleted, "records_affected", recordsAffected)

		// Publish event (ResilientPublisher will handle retry)
		if w.publisher != nil {
			evt := event.Event{
				Version: "1.0",
				Type:    event.Type(domain.EventTypeDailyResetComplete),
				Payload: map[string]interface{}{
					"reset_time":       time.Now().UTC(),
					"records_affected": recordsAffected,
				},
			}

			w.publisher.PublishWithRetry(ctx, evt)
		}
	}()
}

// Shutdown gracefully shuts down the daily reset worker
// Cancels the pending timer and waits for any in-flight resets to complete
func (w *DailyResetWorker) Shutdown(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("Shutting down daily reset worker")

	// Signal shutdown to timer callback (safe to close once)
	select {
	case <-w.shutdown:
		// Already closed, nothing to do
	default:
		close(w.shutdown)
	}

	// Cancel pending timer
	w.mu.Lock()
	if w.timer != nil {
		w.timer.Stop()
		log.Info("Cancelled pending daily reset")
	}
	w.mu.Unlock()

	// Wait for any in-flight resets to complete
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("Daily reset worker shutdown complete")
		return nil
	case <-ctx.Done():
		log.Warn("Daily reset worker shutdown timeout, some resets may still be running")
		return ctx.Err()
	}
}

// timeUntilNextReset calculates the duration until the next 00:00 UTC+7
func timeUntilNextReset() time.Duration {
	location := time.FixedZone("UTC+7", 7*60*60)
	now := time.Now().In(location)
	nextReset := time.Date(
		now.Year(), now.Month(), now.Day(),
		0, 0, 0, 0, location,
	)
	if !nextReset.After(now) {
		nextReset = nextReset.AddDate(0, 0, 1)
	}
	return nextReset.Sub(now)
}
