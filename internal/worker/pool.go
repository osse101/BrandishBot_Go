package worker

import (
	"context"
	"sync"

	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// Job represents a task to be executed by a worker
type Job interface {
	Process(ctx context.Context) error
}

// Pool represents a worker pool
type Pool struct {
	workers  int
	jobQueue chan Job
	wg       sync.WaitGroup
	quit     chan struct{}
}

// NewPool creates a new worker pool
func NewPool(workers int, queueSize int) *Pool {
	return &Pool{
		workers:  workers,
		jobQueue: make(chan Job, queueSize),
		quit:     make(chan struct{}),
	}
}

// Start starts the workers
func (p *Pool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

// worker is the worker loop
func (p *Pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case job := <-p.jobQueue:
			// Create a background context for the job
			// In a real app, we might want to pass a context with timeout
			ctx := context.Background()
			if err := job.Process(ctx); err != nil {
				// Log error but don't crash worker
				// We need a way to log here. For now, we'll assume a global logger or just print
				// Ideally, inject logger into Pool
				logger.FromContext(ctx).Error(LogMsgWorkerJobFailed, "error", err)
			}
		case <-p.quit:
			return
		}
	}
}

// Enqueue adds a job to the queue
// It returns true if the job was enqueued, false if the queue is full (non-blocking)
// or we could make it blocking. For now, let's make it blocking but with a select to avoid deadlocks on stop?
// Actually, for simplicity, let's just send to channel.
func (p *Pool) Enqueue(job Job) {
	p.jobQueue <- job
}

// Stop stops the workers and waits for them to finish
func (p *Pool) Stop() {
	close(p.quit)
	p.wg.Wait()
}
