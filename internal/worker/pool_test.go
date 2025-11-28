package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type testJob struct {
	executed *int32
}

func (j *testJob) Process(ctx context.Context) error {
	atomic.AddInt32(j.executed, 1)
	return nil
}

func TestPool(t *testing.T) {
	var executed int32
	pool := NewPool(2, 10)
	pool.Start()

	job := &testJob{executed: &executed}
	pool.Enqueue(job)
	pool.Enqueue(job)

	// Wait a bit for workers to process
	time.Sleep(100 * time.Millisecond)

	pool.Stop()

	if atomic.LoadInt32(&executed) != 2 {
		t.Errorf("Expected 2 jobs executed, got %d", executed)
	}
}
