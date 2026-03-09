package worker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestTimeUntilNextWeeklyReset(t *testing.T) {
	t.Parallel()

	// Since timeUntilNextWeeklyReset uses time.Now(), we can't fully mock it without changing the signature
	// But we can verify it returns a positive duration less than or equal to 7 days
	got := timeUntilNextWeeklyReset()

	assert.Greater(t, got, time.Duration(0), "Time until reset should be positive")
	assert.LessOrEqual(t, got, 7*24*time.Hour, "Time until reset should be <= 7 days")
}

func TestWeeklyResetWorker_StartAndShutdown(t *testing.T) {
	t.Parallel()

	questSvc := mocks.NewMockQuestService(t)
	worker := NewWeeklyResetWorker(questSvc)

	worker.Start()

	// Wait a tiny bit to let the goroutine schedule the timer
	time.Sleep(10 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestWeeklyResetWorker_ExecuteReset(t *testing.T) {
	t.Parallel()

	questSvc := mocks.NewMockQuestService(t)
	worker := NewWeeklyResetWorker(questSvc)

	// Set up expectation
	questSvc.On("ResetWeeklyQuests", mock.Anything).Return(nil)

	// Since executeReset adds to wg and starts a goroutine for the next schedule,
	// we will manually call it. Note that executeReset calls w.wg.Done(), so we need to add to wg first.
	worker.wg.Add(1)
	worker.executeReset()

	// Allow goroutines to finish
	time.Sleep(100 * time.Millisecond)

	questSvc.AssertExpectations(t)

	// Shutdown to clean up the newly scheduled timer
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestWeeklyResetWorker_ExecuteReset_Error(t *testing.T) {
	t.Parallel()

	questSvc := mocks.NewMockQuestService(t)
	worker := NewWeeklyResetWorker(questSvc)

	// Set up expectation
	questSvc.On("ResetWeeklyQuests", mock.Anything).Return(assert.AnError)

	worker.wg.Add(1)
	worker.executeReset()

	// Allow goroutines to finish
	time.Sleep(100 * time.Millisecond)

	questSvc.AssertExpectations(t)

	// Shutdown to clean up the newly scheduled timer
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := worker.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestWeeklyResetWorker_ShutdownTimeout(t *testing.T) {
	t.Parallel()

	questSvc := mocks.NewMockQuestService(t)
	worker := NewWeeklyResetWorker(questSvc)

	// Add to wg to block shutdown
	worker.wg.Add(1)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := worker.Shutdown(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// Cleanup wg so test can exit cleanly
	worker.wg.Done()
}
