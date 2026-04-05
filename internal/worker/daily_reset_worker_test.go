package worker

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/mocks"
)

// TestTimeUntilNextReset tests reset time calculation
func TestTimeUntilNextReset(t *testing.T) {
	t.Parallel()

	location := time.FixedZone("UTC+7", 7*60*60)
	tests := []struct {
		name string
		now  time.Time
		want time.Duration
	}{
		{
			name: "01:00 UTC+7 should be 23 hours until next reset",
			now:  time.Date(2026, 2, 2, 1, 0, 0, 0, location),
			want: 23 * time.Hour,
		},
		{
			name: "23:59:59 UTC+7 should be 1 second until next reset",
			now:  time.Date(2026, 2, 2, 23, 59, 59, 0, location),
			want: 1 * time.Second,
		},
		{
			name: "Exactly 00:00:00 UTC+7 (on boundary) should be 24 hours",
			now:  time.Date(2026, 2, 2, 0, 0, 0, 0, location),
			want: 24 * time.Hour,
		},
		{
			name: "00:00:01 UTC+7 (just after boundary) should be almost 24 hours",
			now:  time.Date(2026, 2, 2, 0, 0, 1, 0, location),
			want: 24*time.Hour - 1*time.Second,
		},
		{
			name: "Different timezone input (UTC) converting to UTC+7",
			// 17:00 UTC is 00:00 UTC+7 next day. So if it's 16:00 UTC, it's 23:00 UTC+7.
			// Next reset is in 1 hour.
			now:  time.Date(2026, 2, 1, 16, 0, 0, 0, time.UTC),
			want: 1 * time.Hour,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := timeUntilNextResetFrom(tt.now)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestDailyResetWorkerStart tests that worker schedules a reset
func TestDailyResetWorkerStart(t *testing.T) {
	t.Parallel()

	jobSvc := mocks.NewMockJobService(t)
	mockBus := mocks.NewMockEventBus(t)

	deadFile := filepath.Join(t.TempDir(), "test_dead.jsonl")
	publisher, err := event.NewResilientPublisher(mockBus, 1, 10*time.Millisecond, deadFile)
	assert.NoError(t, err)
	defer publisher.Shutdown(context.Background())

	worker := NewDailyResetWorker(jobSvc, publisher)

	// Expect GetDailyResetStatus to be called during Start
	jobSvc.On("GetDailyResetStatus", mock.Anything).Return(&domain.DailyResetStatus{
		LastResetTime: time.Now().UTC(),
	}, nil)

	// Start should not panic
	worker.Start()

	// Shutdown should complete without error
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = worker.Shutdown(ctx)
	assert.NoError(t, err)
}

// TestDailyResetWorkerShutdown tests graceful shutdown
func TestDailyResetWorkerShutdown(t *testing.T) {
	t.Parallel()

	jobSvc := mocks.NewMockJobService(t)
	mockBus := mocks.NewMockEventBus(t)

	deadFile := filepath.Join(t.TempDir(), "test_dead.jsonl")
	publisher, err := event.NewResilientPublisher(mockBus, 1, 10*time.Millisecond, deadFile)
	assert.NoError(t, err)
	defer publisher.Shutdown(context.Background())

	worker := NewDailyResetWorker(jobSvc, publisher)

	// Expect GetDailyResetStatus to be called during Start
	jobSvc.On("GetDailyResetStatus", mock.Anything).Return(&domain.DailyResetStatus{
		LastResetTime: time.Now().UTC(),
	}, nil)

	worker.Start()

	// Allow time for any scheduled timers
	time.Sleep(100 * time.Millisecond)

	// Shutdown should complete without hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = worker.Shutdown(ctx)
	assert.NoError(t, err)
}

// TestDailyResetWorkerShutdownTimeout tests timeout during shutdown
func TestDailyResetWorkerShutdownTimeout(t *testing.T) {
	t.Parallel()

	jobSvc := mocks.NewMockJobService(t)
	mockBus := mocks.NewMockEventBus(t)

	deadFile := filepath.Join(t.TempDir(), "test_dead.jsonl")
	publisher, err := event.NewResilientPublisher(mockBus, 1, 10*time.Millisecond, deadFile)
	assert.NoError(t, err)
	defer publisher.Shutdown(context.Background())

	worker := NewDailyResetWorker(jobSvc, publisher)

	// Expect GetDailyResetStatus to be called during Start
	jobSvc.On("GetDailyResetStatus", mock.Anything).Return(&domain.DailyResetStatus{
		LastResetTime: time.Now().UTC(),
	}, nil)

	worker.Start()

	// Shutdown with very short timeout should timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// This might timeout (expected) or succeed quickly (also ok)
	_ = worker.Shutdown(ctx)

	// Verify worker still shuts down eventually
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	err = worker.Shutdown(ctx2)
	assert.NoError(t, err)
}

func TestDailyResetWorker_ExecuteReset(t *testing.T) {
	t.Parallel()

	jobSvc := mocks.NewMockJobService(t)
	mockBus := mocks.NewMockEventBus(t)

	deadFile := filepath.Join(t.TempDir(), "test_dead.jsonl")
	publisher, err := event.NewResilientPublisher(mockBus, 1, 10*time.Millisecond, deadFile)
	assert.NoError(t, err)
	defer publisher.Shutdown(context.Background())

	worker := NewDailyResetWorker(jobSvc, publisher)

	// Setup expectations
	jobSvc.On("ResetDailyJobXP", mock.Anything).Return(int64(42), nil)

	// ResilientPublisher uses Publish in a background goroutine, but it sends to the bus.
	// We expect the bus to receive an event of type EventTypeDailyResetComplete.
	mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
		return evt.Type == event.Type(domain.EventTypeDailyResetComplete)
	})).Return(nil)

	// Execute
	worker.executeReset()

	// Wait for the waitgroup to ensure the goroutine completes
	worker.wg.Wait()

	// Give the publisher time to process and publish
	time.Sleep(50 * time.Millisecond)

	// Assert expectations
	jobSvc.AssertExpectations(t)
	mockBus.AssertExpectations(t)
}

func TestDailyResetWorker_CheckMissedReset(t *testing.T) {
	t.Parallel()

	jobSvc := mocks.NewMockJobService(t)
	mockBus := mocks.NewMockEventBus(t)

	deadFile := filepath.Join(t.TempDir(), "test_dead.jsonl")
	publisher, err := event.NewResilientPublisher(mockBus, 1, 10*time.Millisecond, deadFile)
	assert.NoError(t, err)
	defer publisher.Shutdown(context.Background())

	worker := NewDailyResetWorker(jobSvc, publisher)

	// Mocking a missed reset (LastResetTime was 2 days ago)
	location := time.FixedZone("UTC+7", 7*60*60)
	lastReset := time.Now().In(location).AddDate(0, 0, -2)

	jobSvc.On("GetDailyResetStatus", mock.Anything).Return(&domain.DailyResetStatus{
		LastResetTime:   lastReset,
		RecordsAffected: 10,
	}, nil)

	// Expect executeReset to be triggered
	jobSvc.On("ResetDailyJobXP", mock.Anything).Return(int64(42), nil)
	mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
		return evt.Type == event.Type(domain.EventTypeDailyResetComplete)
	})).Return(nil)

	// Execute
	worker.checkMissedReset(context.Background())

	// Wait for the waitgroup to ensure the goroutine completes
	worker.wg.Wait()

	// Assert expectations
	jobSvc.AssertExpectations(t)
}

func TestDailyResetWorker_CheckMissedReset_NoMiss(t *testing.T) {
	t.Parallel()

	jobSvc := mocks.NewMockJobService(t)
	worker := NewDailyResetWorker(jobSvc, nil)

	// Mocking a recent reset (LastResetTime was just now)
	lastReset := time.Now().UTC()

	jobSvc.On("GetDailyResetStatus", mock.Anything).Return(&domain.DailyResetStatus{
		LastResetTime:   lastReset,
		RecordsAffected: 42,
	}, nil)

	// ResetDailyJobXP should NOT be called

	// Execute
	worker.checkMissedReset(context.Background())

	// Assert expectations
	jobSvc.AssertExpectations(t)
}
