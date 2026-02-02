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
	pool := NewPool(TestWorkerCount, TestQueueSize)
	pool.Start()

	job := &testJob{executed: &executed}
	pool.Enqueue(job)
	pool.Enqueue(job)

	// Wait a bit for workers to process
	time.Sleep(TestWorkerProcessWaitTime * time.Millisecond)

	pool.Stop()

	if atomic.LoadInt32(&executed) != TestExpectedJobCount {
		t.Errorf("Expected %d jobs executed, got %d", TestExpectedJobCount, executed)
	}
}
