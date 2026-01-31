package worker

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/gamble"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// GambleWorker checks for expired gambles and executes them
type GambleWorker struct {
	service  gamble.Service
	mu       sync.Mutex
	timers   map[uuid.UUID]*time.Timer // gambleID -> timer
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// NewGambleWorker creates a new GambleWorker
func NewGambleWorker(service gamble.Service) *GambleWorker {
	return &GambleWorker{
		service:  service,
		timers:   make(map[uuid.UUID]*time.Timer),
		shutdown: make(chan struct{}),
	}
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
	w.mu.Lock()
	if existingTimer, ok := w.timers[g.ID]; ok {
		existingTimer.Stop()
		delete(w.timers, g.ID)
	}

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
		w.mu.Lock()
		delete(w.timers, g.ID)
		w.mu.Unlock()
	})

	w.timers[g.ID] = timer
	w.mu.Unlock()
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
	log := logger.FromContext(ctx)
	log.Info("Shutting down gamble worker")

	// Signal shutdown to all timer callbacks
	close(w.shutdown)

	// Cancel all pending timers
	w.mu.Lock()
	for gambleID, timer := range w.timers {
		timer.Stop()
		log.Info("Cancelled pending gamble execution", "gambleID", gambleID)
	}
	w.timers = make(map[uuid.UUID]*time.Timer)
	w.mu.Unlock()

	// Wait for any in-flight executions to complete
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("Gamble worker shutdown complete")
		return nil
	case <-ctx.Done():
		log.Warn("Gamble worker shutdown timeout, some executions may still be running")
		return ctx.Err()
	}
}
