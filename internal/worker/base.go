package worker

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// BaseWorker provides common functionality for background workers that manage timers
type BaseWorker struct {
	mu       sync.Mutex
	timers   map[uuid.UUID]*time.Timer
	shutdown chan struct{}
	wg       sync.WaitGroup
}

func (w *BaseWorker) init() {
	if w.timers == nil {
		w.timers = make(map[uuid.UUID]*time.Timer)
	}
	if w.shutdown == nil {
		w.shutdown = make(chan struct{})
	}
}

func (w *BaseWorker) stopTimer(id uuid.UUID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if timer, ok := w.timers[id]; ok {
		timer.Stop()
		delete(w.timers, id)
	}
}

func (w *BaseWorker) registerTimer(id uuid.UUID, timer *time.Timer) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.timers[id] = timer
}

func (w *BaseWorker) removeTimer(id uuid.UUID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.timers, id)
}

func (w *BaseWorker) shutdownInternal(ctx context.Context, workerName string) error {
	log := logger.FromContext(ctx)
	log.Info("Shutting down " + workerName)

	close(w.shutdown)

	// Cancel all pending timers
	w.mu.Lock()
	for id, timer := range w.timers {
		timer.Stop()
		log.Info("Cancelled pending "+workerName+" execution", workerName+"ID", id)
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
		log.Info(workerName + " shutdown complete")
		return nil
	case <-ctx.Done():
		log.Warn(workerName + " shutdown timeout")
		return ctx.Err()
	}
}
