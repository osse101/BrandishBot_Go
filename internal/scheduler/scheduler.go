package scheduler

import (
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/worker"
)

// Scheduler manages scheduled jobs
type Scheduler struct {
	workerPool *worker.Pool
	quit       chan struct{}
	wg         sync.WaitGroup
}

// New creates a new scheduler
func New(pool *worker.Pool) *Scheduler {
	return &Scheduler{
		workerPool: pool,
		quit:       make(chan struct{}),
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
				// Enqueue job to worker pool
				// Note: Enqueue is blocking in current implementation if queue is full?
				// Let's check worker.Pool.Enqueue implementation.
				// It sends to channel: p.jobQueue <- job
				// If queue is full, this will block the scheduler goroutine.
				// This is acceptable for now, or we could use a non-blocking send.
				s.workerPool.Enqueue(job)
				// We can log here if needed, but better to keep scheduler simple.
			case <-s.quit:
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
	close(s.quit)
	s.wg.Wait()
}
