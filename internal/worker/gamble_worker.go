package worker

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/gamble"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// GambleWorker checks for expired gambles and executes them
type GambleWorker struct {
	BaseWorker
	service gamble.Service
}

// NewGambleWorker creates a new GambleWorker
func NewGambleWorker(service gamble.Service) *GambleWorker {
	w := &GambleWorker{
		service: service,
	}
	w.init()
	return w
}

// Start checks for any existing active gamble on startup and schedules it
func (w *GambleWorker) Start() {
	ctx := context.Background()
	log := logger.FromContext(ctx)

	active, err := w.service.GetActiveGamble(ctx)
	if err != nil {
		log.Error(LogMsgFailedToCheckActiveGambleOnStartup, "error", err)
		return
	}

	if active != nil && active.State == domain.GambleStateJoining {
		w.scheduleExecution(active)
	}
}

// Subscribe subscribes the worker to relevant events
func (w *GambleWorker) Subscribe(bus event.Bus) {
	bus.Subscribe(event.Type(domain.EventGambleStarted), w.handleGambleStarted)
}

func (w *GambleWorker) handleGambleStarted(ctx context.Context, e event.Event) error {
	gamble, ok := e.Payload.(*domain.Gamble)
	if !ok {
		return nil
	}
	w.scheduleExecution(gamble)
	return nil
}

//nolint:dupl
func (w *GambleWorker) scheduleExecution(g *domain.Gamble) {
	duration := time.Until(g.JoinDeadline)

	log := logger.FromContext(context.Background())
	log.Info(LogMsgSchedulingGambleExecution, "gambleID", g.ID, "duration", duration)

	// If deadline has already passed, execute immediately in a goroutine
	if duration <= 0 {
		w.executeGamble(g.ID)
		return
	}

	// Stop existing timer if one exists for this gamble
	w.stopTimer(g.ID)

	// Schedule for future execution
	timer := time.AfterFunc(duration, func() {
		// Check if shutting down
		select {
		case <-w.shutdown:
			return
		default:
		}

		// Execute the gamble
		w.executeGamble(g.ID)

		// Clean up timer reference
		w.removeTimer(g.ID)
	})

	w.registerTimer(g.ID, timer)
}

// executeGamble executes a gamble in a tracked goroutine
func (w *GambleWorker) executeGamble(gambleID uuid.UUID) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		ctx := context.Background()
		log := logger.FromContext(ctx)
		log.Info(LogMsgExecutingScheduledGamble, "gambleID", gambleID)

		if _, err := w.service.ExecuteGamble(ctx, gambleID); err != nil {
			log.Error(LogMsgFailedToExecuteGamble, "gambleID", gambleID, "error", err)
		}
	}()
}

// Shutdown gracefully shuts down the gamble worker, canceling all pending timers
// and waiting for any in-flight gamble executions to complete
func (w *GambleWorker) Shutdown(ctx context.Context) error {
	return w.shutdownInternal(ctx, "gamble worker")
}
