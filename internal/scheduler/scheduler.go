package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/worker"
)

// Scheduler manages scheduled jobs
type Scheduler struct {
	workerPool *worker.Pool
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// New creates a new scheduler
func New(pool *worker.Pool) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		workerPool: pool,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Schedule registers a job to run at a fixed interval
func (s *Scheduler) Schedule(interval time.Duration, job worker.Job) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Attempt to enqueue job with context cancellation support
				// If the pool is full and we are stopping, this will return quickly.
				_ = s.workerPool.EnqueueContext(s.ctx, job)
			case <-s.ctx.Done():
				return
			}
		}
	}()
}

// Start starts the scheduler (noop for now as Schedule starts goroutines immediately)
// But we might want to defer starting until Start() is called.
// For simplicity, Schedule starts immediately.
func (s *Scheduler) Start() {
	// No-op in this simple implementation
}

// Stop stops all scheduled jobs
func (s *Scheduler) Stop() {
	s.cancel()
	s.wg.Wait()
}
