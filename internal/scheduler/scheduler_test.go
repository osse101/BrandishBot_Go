package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/worker"
)

// MockJob is a simple job for testing
type MockJob struct {
	RunCount int
	Done     chan struct{}
	mu       sync.Mutex
}

func (m *MockJob) Process(ctx context.Context) error {
	m.mu.Lock()
	m.RunCount++
	m.mu.Unlock()

	// Signal that job ran
	select {
	case m.Done <- struct{}{}:
	default:
	}
	return nil
}

// BlockingJob blocks until released
type BlockingJob struct {
	release chan struct{}
}

func (b *BlockingJob) Process(ctx context.Context) error {
	select {
	case <-b.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestScheduler(t *testing.T) {
	// Create worker pool
	pool := worker.NewPool(1, 10)
	pool.Start()
	defer pool.Stop()

	// Create scheduler
	sched := New(pool)
	defer sched.Stop()

	// Create mock job
	job := &MockJob{
		Done: make(chan struct{}, 10),
	}

	// Schedule job every 10ms
	sched.Schedule(10*time.Millisecond, job)

	// Wait for at least 2 runs
	timeout := time.After(100 * time.Millisecond)
	runCount := 0

	for runCount < 2 {
		select {
		case <-job.Done:
			runCount++
		case <-timeout:
			t.Fatal("Timeout waiting for job execution")
		}
	}

	assert.GreaterOrEqual(t, runCount, 2)
}

func TestScheduler_StopWhileBlocked(t *testing.T) {
	// 1 worker, 0 queue size -> Enqueue blocks if worker busy
	pool := worker.NewPool(1, 0)
	pool.Start()
	defer pool.Stop()

	// Occupy the worker
	release := make(chan struct{})
	defer close(release) // Ensure worker is unblocked on exit
	blockJob := &BlockingJob{release: release}
	go func() {
		pool.Enqueue(blockJob)
	}()

	// Wait for worker to pick up job
	time.Sleep(50 * time.Millisecond)

	sched := New(pool)
	// Do not defer sched.Stop() immediately as we test it manually

	// Schedule a job that will try to enqueue and block
	job := &MockJob{Done: make(chan struct{}, 1)}
	sched.Schedule(1*time.Millisecond, job)

	// Wait for scheduler to be blocked
	time.Sleep(50 * time.Millisecond)

	// Try to stop - should return immediately due to context cancellation
	done := make(chan struct{})
	go func() {
		sched.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Scheduler.Stop() hung while blocked on Enqueue")
	}

	// Wait for stop to complete
	<-done
}
