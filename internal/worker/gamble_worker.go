package worker

import (
	"context"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/gamble"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// GambleWorker checks for expired gambles and executes them
type GambleWorker struct {
	service gamble.Service
}

// NewGambleWorker creates a new GambleWorker
func NewGambleWorker(service gamble.Service) *GambleWorker {
	return &GambleWorker{
		service: service,
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
	if duration < 0 {
		duration = 0
	}

	log := logger.FromContext(context.Background())
	log.Info(LogMsgSchedulingGambleExecution, "gambleID", g.ID, "duration", duration)

	time.AfterFunc(duration, func() {
		ctx := context.Background()
		log.Info(LogMsgExecutingScheduledGamble, "gambleID", g.ID)
		if _, err := w.service.ExecuteGamble(ctx, g.ID); err != nil {
			log.Error(LogMsgFailedToExecuteGamble, "gambleID", g.ID, "error", err)
		}
	})
}
