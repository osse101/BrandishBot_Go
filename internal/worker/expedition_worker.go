package worker

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/expedition"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// ExpeditionWorker schedules expedition execution after the join deadline
type ExpeditionWorker struct {
	service  expedition.Service
	mu       sync.Mutex
	timers   map[uuid.UUID]*time.Timer
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// NewExpeditionWorker creates a new ExpeditionWorker
func NewExpeditionWorker(service expedition.Service) *ExpeditionWorker {
	return &ExpeditionWorker{
		service:  service,
		timers:   make(map[uuid.UUID]*time.Timer),
		shutdown: make(chan struct{}),
	}
}

// Start checks for any existing active expedition on startup and schedules it
func (w *ExpeditionWorker) Start() {
	ctx := context.Background()
	log := logger.FromContext(ctx)

	active, err := w.service.GetActiveExpedition(ctx)
	if err != nil {
		log.Error(LogMsgFailedToCheckActiveExpeditionOnStartup, "error", err)
		return
	}

	if active != nil && active.Expedition.State == domain.ExpeditionStateRecruiting {
		w.scheduleExecution(&active.Expedition)
	}
}

// Subscribe subscribes the worker to relevant events
func (w *ExpeditionWorker) Subscribe(bus event.Bus) {
	bus.Subscribe(event.Type(domain.EventExpeditionStarted), w.handleExpeditionStarted)
}

func (w *ExpeditionWorker) handleExpeditionStarted(_ context.Context, e event.Event) error {
	exp, ok := e.Payload.(*domain.Expedition)
	if !ok {
		return nil
	}
	w.scheduleExecution(exp)
	return nil
}

func (w *ExpeditionWorker) scheduleExecution(exp *domain.Expedition) {
	duration := time.Until(exp.JoinDeadline)

	log := logger.FromContext(context.Background())
	log.Info(LogMsgSchedulingExpeditionExecution, "expeditionID", exp.ID, "duration", duration)

	// If deadline has already passed, execute immediately
	if duration <= 0 {
		w.executeExpedition(exp.ID)
		return
	}

	// Stop existing timer if one exists
	w.mu.Lock()
	if existingTimer, ok := w.timers[exp.ID]; ok {
		existingTimer.Stop()
		delete(w.timers, exp.ID)
	}

	// Schedule for future execution
	timer := time.AfterFunc(duration, func() {
		select {
		case <-w.shutdown:
			return
		default:
		}

		w.executeExpedition(exp.ID)

		w.mu.Lock()
		delete(w.timers, exp.ID)
		w.mu.Unlock()
	})

	w.timers[exp.ID] = timer
	w.mu.Unlock()
}

func (w *ExpeditionWorker) executeExpedition(expeditionID uuid.UUID) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		ctx := context.Background()
		log := logger.FromContext(ctx)
		log.Info(LogMsgExecutingScheduledExpedition, "expeditionID", expeditionID)

		if err := w.service.ExecuteExpedition(ctx, expeditionID); err != nil {
			log.Error(LogMsgFailedToExecuteExpedition, "expeditionID", expeditionID, "error", err)
		}
	}()
}

// Shutdown gracefully shuts down the expedition worker
func (w *ExpeditionWorker) Shutdown(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("Shutting down expedition worker")

	close(w.shutdown)

	// Cancel all pending timers
	w.mu.Lock()
	for expeditionID, timer := range w.timers {
		timer.Stop()
		log.Info("Cancelled pending expedition execution", "expeditionID", expeditionID)
	}
	w.timers = make(map[uuid.UUID]*time.Timer)
	w.mu.Unlock()

	// Wait for in-flight executions
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("Expedition worker shutdown complete")
		return nil
	case <-ctx.Done():
		log.Warn("Expedition worker shutdown timeout")
		return ctx.Err()
	}
}
