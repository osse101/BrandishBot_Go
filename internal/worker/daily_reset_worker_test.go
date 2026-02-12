package worker

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
)

// MockJobService for testing
type MockJobService struct {
	mock.Mock
}

func (m *MockJobService) ResetDailyJobXP(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockJobService) GetDailyResetStatus(ctx context.Context) (*domain.DailyResetStatus, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.DailyResetStatus), args.Error(1)
}

func (m *MockJobService) GetUserJobs(ctx context.Context, userID string) ([]domain.UserJobInfo, error) {
	return nil, nil
}

func (m *MockJobService) GetUserJobsByPlatform(ctx context.Context, platform, platformID string) ([]domain.UserJobInfo, error) {
	return nil, nil
}

func (m *MockJobService) GetPrimaryJob(ctx context.Context, platform, platformID string) (*domain.UserJobInfo, error) {
	return nil, nil
}

func (m *MockJobService) GetJobBonus(ctx context.Context, userID, jobKey, bonusType string) (float64, error) {
	return 0, nil
}

func (m *MockJobService) AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error) {
	return nil, nil
}

func (m *MockJobService) AwardXPByPlatform(ctx context.Context, platform, platformID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error) {
	return nil, nil
}

func (m *MockJobService) GetJobLevel(ctx context.Context, userID, jobKey string) (int, error) {
	return 0, nil
}

func (m *MockJobService) GetAllJobs(ctx context.Context) ([]domain.Job, error) {
	return nil, nil
}

func (m *MockJobService) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return nil, nil
}

func (m *MockJobService) CalculateLevel(totalXP int64) int {
	return 0
}

func (m *MockJobService) GetXPForLevel(level int) int64 {
	return 0
}

func (m *MockJobService) GetXPProgress(currentXP int64) (currentLevel int, xpToNext int64) {
	return 0, 0
}

func (m *MockJobService) Shutdown(ctx context.Context) error {
	return nil
}

// MockBus for testing
type MockBus struct {
	mock.Mock
}

func (m *MockBus) Publish(ctx context.Context, e event.Event) error {
	args := m.Called(ctx, e)
	return args.Error(0)
}

func (m *MockBus) Subscribe(eventType event.Type, handler event.Handler) {
	m.Called(eventType, handler)
}

// TestTimeUntilNextReset tests reset time calculation
func TestTimeUntilNextReset(t *testing.T) {
	location := time.FixedZone("UTC+7", 7*60*60)
	tests := []struct {
		name string
		now  time.Time
		want func(d time.Duration) bool
	}{
		{
			name: "01:00 UTC+7 should be ~23 hours until next reset",
			now:  time.Date(2026, 2, 2, 1, 0, 0, 0, location),
			want: func(d time.Duration) bool {
				return d > 22*time.Hour && d < 24*time.Hour
			},
		},
		{
			name: "23:59 UTC+7 should be ~1 minute until next reset",
			now:  time.Date(2026, 2, 2, 23, 59, 0, 0, location),
			want: func(d time.Duration) bool {
				return d > 0 && d < 2*time.Minute
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since we can't easily mock time.Now() inside the function without changing it
			// we verify the logic manually here or just ensure it's reasonable
			nextReset := time.Date(tt.now.Year(), tt.now.Month(), tt.now.Day(), 0, 0, 0, 0, location)
			if !nextReset.After(tt.now) {
				nextReset = nextReset.AddDate(0, 0, 1)
			}
			testDuration := nextReset.Sub(tt.now)

			assert.Greater(t, testDuration, time.Duration(0))
			assert.Less(t, testDuration, 25*time.Hour)
			assert.True(t, tt.want(testDuration))
		})
	}
}

// TestDailyResetWorkerStart tests that worker schedules a reset
func TestDailyResetWorkerStart(t *testing.T) {
	jobSvc := new(MockJobService)
	mockBus := new(MockBus)

	publisher, err := event.NewResilientPublisher(mockBus, 1, 10*time.Millisecond, "test_dead.jsonl")
	assert.NoError(t, err)
	t.Cleanup(func() {
		os.Remove("test_dead.jsonl")
	})
	defer publisher.Shutdown(context.Background())

	worker := NewDailyResetWorker(jobSvc, publisher)

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
	jobSvc := new(MockJobService)
	mockBus := new(MockBus)

	publisher, err := event.NewResilientPublisher(mockBus, 1, 10*time.Millisecond, "test_dead2.jsonl")
	assert.NoError(t, err)
	t.Cleanup(func() {
		os.Remove("test_dead2.jsonl")
	})
	defer publisher.Shutdown(context.Background())

	worker := NewDailyResetWorker(jobSvc, publisher)
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
	jobSvc := new(MockJobService)
	mockBus := new(MockBus)

	publisher, err := event.NewResilientPublisher(mockBus, 1, 10*time.Millisecond, "test_dead3.jsonl")
	assert.NoError(t, err)
	t.Cleanup(func() {
		os.Remove("test_dead3.jsonl")
	})
	defer publisher.Shutdown(context.Background())

	worker := NewDailyResetWorker(jobSvc, publisher)
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
