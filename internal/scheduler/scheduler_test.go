package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/worker"
)

// MockJob is a simple job for testing
type MockJob struct {
	RunCount int
	Done     chan struct{}
}

func (m *MockJob) Process(ctx context.Context) error {
	m.RunCount++
	// Signal that job ran
	select {
	case m.Done <- struct{}{}:
	default:
	}
	return nil
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
