package job

import (
	"context"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock EventBus
type MockBus struct {
	mock.Mock
}

func (m *MockBus) Publish(ctx context.Context, event event.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockBus) Subscribe(eventType event.Type, handler event.Handler) {
	m.Called(eventType, handler)
}

// Mock Repository
type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) GetAllJobs(ctx context.Context) ([]domain.Job, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Job), args.Error(1)
}
func (m *MockRepo) GetJobByKey(ctx context.Context, jobKey string) (*domain.Job, error) {
	args := m.Called(ctx, jobKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Job), args.Error(1)
}
func (m *MockRepo) GetUserJob(ctx context.Context, userID string, jobID int) (*domain.UserJob, error) {
	args := m.Called(ctx, userID, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserJob), args.Error(1)
}
func (m *MockRepo) UpsertUserJob(ctx context.Context, userJob *domain.UserJob) error {
	args := m.Called(ctx, userJob)
	return args.Error(0)
}
func (m *MockRepo) RecordJobXPEvent(ctx context.Context, event *domain.JobXPEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}
func (m *MockRepo) GetJobLevelBonuses(ctx context.Context, jobID int, level int) ([]domain.JobLevelBonus, error) {
	return nil, nil
}
func (m *MockRepo) GetUserJobs(ctx context.Context, userID string) ([]domain.UserJob, error) {
	return nil, nil
}

// Mock Progression
type MockProgression struct {
	mock.Mock
}

// GetModifiedValue implements ProgressionService - returns base value (no modifiers for tests)
func (m *MockProgression) GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error) {
	return baseValue, nil
}

func (m *MockProgression) IsFeatureUnlocked(ctx context.Context, featureKey string) (bool, error) {
	args := m.Called(ctx, featureKey)
	return args.Bool(0), args.Error(1)
}
func (m *MockProgression) GetProgressionStatus(ctx context.Context) (*domain.ProgressionStatus, error) {
	return nil, nil
}

// Mock Stats
type MockStats struct {
	mock.Mock
}

func (m *MockStats) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata map[string]interface{}) error {
	args := m.Called(ctx, userID, eventType, metadata)
	return args.Error(0)
}
func (m *MockStats) GetUserStats(ctx context.Context, userID string, period string) (*domain.StatsSummary, error) {
	return nil, nil
}
func (m *MockStats) GetUserCurrentStreak(ctx context.Context, userID string) (int, error) {
	return 0, nil
}
func (m *MockStats) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) {
	return nil, nil
}
func (m *MockStats) GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error) {
	return nil, nil
}

func TestAwardXP_PublishesEventOnLevelUp(t *testing.T) {
	// Setup
	mockRepo := new(MockRepo)
	mockProg := new(MockProgression)
	mockStats := new(MockStats)
	mockBus := new(MockBus)

	// Create a ResilientPublisher with the mocked bus
	// Use a temp file for dead-letter in tests
	tmpFile := t.TempDir() + "/deadletter.jsonl"
	resilientPub, err := event.NewResilientPublisher(mockBus, 3, 100*time.Millisecond, tmpFile)
	if err != nil {
		t.Fatalf("Failed to create resilient publisher: %v", err)
	}
	defer resilientPub.Shutdown(context.Background())

	svc := NewService(mockRepo, mockProg, mockStats, mockBus, resilientPub)
	ctx := context.Background()

	// Setup data
	job := &domain.Job{ID: 1, JobKey: "explorer"}
	userJob := &domain.UserJob{
		UserID:       "user123",
		JobID:        1,
		CurrentXP:    0,
		CurrentLevel: 1,
	}

	mockProg.On("IsFeatureUnlocked", ctx, "feature_jobs_xp").Return(true, nil)
	mockRepo.On("GetJobByKey", ctx, "explorer").Return(job, nil)
	mockRepo.On("GetUserJob", ctx, "user123", 1).Return(userJob, nil)
	mockRepo.On("UpsertUserJob", ctx, mock.Anything).Return(nil)
	mockRepo.On("RecordJobXPEvent", ctx, mock.Anything).Return(nil)
	mockStats.On("RecordUserEvent", ctx, "user123", domain.EventJobLevelUp, mock.Anything).Return(nil)

	// Mock expects event to be published to the bus
	// The ResilientPublisher will call bus.Publish on the first attempt
	mockBus.On("Publish", ctx, mock.MatchedBy(func(e event.Event) bool {
		if e.Type != event.Type(domain.EventJobLevelUp) {
			return false
		}
		payload, ok := e.Payload.(map[string]interface{})
		if !ok {
			return false
		}
		return payload["user_id"] == "user123" &&
			payload["job_key"] == "explorer" &&
			payload["new_level"] == 2 &&
			payload["old_level"] == 1
	})).Return(nil).Once()

	// Execute - Award enough XP to level up from 1 -> 2
	result, err := svc.AwardXP(ctx, "user123", "explorer", 500, "test", nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.LeveledUp)
	assert.Equal(t, 2, result.NewLevel)

	// Give the async publisher a moment to process
	time.Sleep(50 * time.Millisecond)

	mockBus.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockProg.AssertExpectations(t)
	mockStats.AssertExpectations(t)
}
