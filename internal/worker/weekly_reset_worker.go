package worker

import (
	"context"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/quest"
)

type WeeklyResetWorker struct {
	questService quest.Service
	timer        *time.Timer
	shutdown     chan struct{}
	wg           sync.WaitGroup
	mu           sync.Mutex
}

func NewWeeklyResetWorker(questService quest.Service) *WeeklyResetWorker {
	return &WeeklyResetWorker{
		questService: questService,
		shutdown:     make(chan struct{}),
	}
}

func (w *WeeklyResetWorker) Start() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.scheduleNext()
	}()
}

func (w *WeeklyResetWorker) scheduleNext() {
	duration := timeUntilNextWeeklyReset()

	w.mu.Lock()
	w.timer = time.AfterFunc(duration, func() {
		w.wg.Add(1)
		go w.executeReset()
	})
	w.mu.Unlock()
}

func (w *WeeklyResetWorker) executeReset() {
	defer w.wg.Done()

	ctx := context.Background()
	log := logger.FromContext(ctx)

	log.Info("Starting weekly quest reset")

	if err := w.questService.ResetWeeklyQuests(ctx); err != nil {
		log.Error("Weekly quest reset failed", "error", err)
	} else {
		log.Info("Weekly quest reset completed successfully")
	}

	// Schedule next reset
	w.scheduleNext()
}

// timeUntilNextWeeklyReset calculates time until next Monday 00:00 UTC
func timeUntilNextWeeklyReset() time.Duration {
	now := time.Now().UTC()

	// Next Monday at 00:00 UTC
	// Monday is day 1 in Go's time.Weekday
	daysUntilMonday := (8 - int(now.Weekday())) % 7
	if daysUntilMonday == 0 && now.Hour() == 0 && now.Minute() == 0 && now.Second() == 0 {
		// It's exactly Monday 00:00:00, go to next Monday
		daysUntilMonday = 7
	} else if daysUntilMonday == 0 && (now.Hour() > 0 || now.Minute() > 0 || now.Second() > 0) {
		// It's Monday but past midnight, go to next Monday
		daysUntilMonday = 7
	}

	nextReset := time.Date(
		now.Year(), now.Month(), now.Day(),
		0, 0, 0, 0, time.UTC,
	).AddDate(0, 0, daysUntilMonday)

	duration := nextReset.Sub(now)

	log := logger.FromContext(context.Background())
	log.Info("Next weekly reset scheduled",
		"next_reset", nextReset.Format(time.RFC3339),
		"duration", duration.String())

	return duration
}

func (w *WeeklyResetWorker) Shutdown(ctx context.Context) error {
	close(w.shutdown)

	w.mu.Lock()
	if w.timer != nil {
		w.timer.Stop()
	}
	w.mu.Unlock()

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
