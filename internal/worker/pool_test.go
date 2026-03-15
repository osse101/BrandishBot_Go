package worker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testJob struct {
	executed *int32
	wg       *sync.WaitGroup
	err      error
}

func (j *testJob) Process(ctx context.Context) error {
	atomic.AddInt32(j.executed, 1)
	if j.wg != nil {
		j.wg.Done()
	}
	return j.err
}

type blockingJob struct {
	started chan struct{}
	block   chan struct{}
}

func (j *blockingJob) Process(ctx context.Context) error {
	close(j.started)
	<-j.block
	return nil
}

func TestPool(t *testing.T) {
	t.Parallel()

	t.Run("Basic enqueue and process", func(t *testing.T) {
		t.Parallel()
		var executed int32
		var wg sync.WaitGroup

		pool := NewPool(TestWorkerCount, TestQueueSize)
		pool.Start()

		job := &testJob{
			executed: &executed,
			wg:       &wg,
		}

		// Enqueue TestExpectedJobCount jobs
		wg.Add(TestExpectedJobCount)
		for i := 0; i < TestExpectedJobCount; i++ {
			pool.Enqueue(job)
		}

		// Wait for all jobs to complete
		wg.Wait()
		pool.Stop()

		assert.Equal(t, int32(TestExpectedJobCount), atomic.LoadInt32(&executed))
	})

	t.Run("Job returns error", func(t *testing.T) {
		t.Parallel()
		var executed int32
		var wg sync.WaitGroup

		pool := NewPool(1, 10)
		pool.Start()

		errJob := &testJob{
			executed: &executed,
			wg:       &wg,
			err:      errors.New("job failed"),
		}
		successJob := &testJob{
			executed: &executed,
			wg:       &wg,
			err:      nil,
		}

		wg.Add(2)
		pool.Enqueue(errJob)
		pool.Enqueue(successJob)

		wg.Wait()
		pool.Stop()

		// Both jobs should be executed despite the first one returning an error
		assert.Equal(t, int32(2), atomic.LoadInt32(&executed))
	})
}

func TestPool_EnqueueContext(t *testing.T) {
	t.Parallel()

	t.Run("Successful enqueue", func(t *testing.T) {
		t.Parallel()
		var executed int32
		var wg sync.WaitGroup

		pool := NewPool(1, 10)
		pool.Start()

		job := &testJob{
			executed: &executed,
			wg:       &wg,
		}

		wg.Add(1)
		err := pool.EnqueueContext(context.Background(), job)
		require.NoError(t, err)

		wg.Wait()
		pool.Stop()

		assert.Equal(t, int32(1), atomic.LoadInt32(&executed))
	})

	t.Run("Context cancellation", func(t *testing.T) {
		t.Parallel()

		// Pool with 1 worker and 0 queue size
		pool := NewPool(1, 0)
		pool.Start()

		// Enqueue a blocking job to tie up the single worker
		bJob := &blockingJob{
			started: make(chan struct{}),
			block:   make(chan struct{}),
		}
		pool.Enqueue(bJob)

		// Wait for the worker to start processing the blocking job
		<-bJob.started

		// Now the worker is busy, and queue size is 0, so the next enqueue will block
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		job := &testJob{}
		err := pool.EnqueueContext(ctx, job)

		assert.ErrorIs(t, err, context.Canceled)

		// Clean up
		close(bJob.block)
		pool.Stop()
	})

	t.Run("Pool stopped", func(t *testing.T) {
		t.Parallel()
		pool := NewPool(1, 0)
		pool.Start()

		// Enqueue a blocking job to tie up the single worker
		bJob := &blockingJob{
			started: make(chan struct{}),
			block:   make(chan struct{}),
		}
		pool.Enqueue(bJob)
		<-bJob.started // Wait for it to start

		// Stop the pool. The worker is blocked, so wait group will block on Stop()
		// We can't synchronously pool.Stop() here because the worker is stuck on bJob.block
		// But Stop() closes pool.quit, which EnqueueContext can see.
		// Wait... pool.Stop() blocks on p.wg.Wait(), so we do it in a goroutine

		go pool.Stop()

		// Give the pool.Stop() goroutine a tiny moment to close(pool.quit)
		time.Sleep(10 * time.Millisecond)

		// Now try to enqueue a job
		// Since queue is 0, and worker is busy, jobQueue is blocked
		// Since quit is closed, it should select quit channel and return context.Canceled
		job := &testJob{}
		err := pool.EnqueueContext(context.Background(), job)

		assert.ErrorIs(t, err, context.Canceled)

		// Clean up worker
		close(bJob.block)
	})
}
