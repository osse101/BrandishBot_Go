package job

import (
	"context"
	"testing"

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
	mockRepo := new(MockRepo)
	mockProg := new(MockProgression)
	mockStats := new(MockStats)
	mockBus := new(MockBus)

	svc := NewService(mockRepo, mockProg, mockStats, mockBus)
	ctx := context.Background()

	// Setup data
	jobKey := "explorer"
	userID := "user123"
	jobID := 1

	testJob := &domain.Job{ID: jobID, JobKey: jobKey}

	// Initial state: Level 0, 0 XP
	initialProgress := &domain.UserJob{
		UserID: userID,
		JobID: jobID,
		CurrentXP: 0,
		CurrentLevel: 0,
		XPGainedToday: 0,
	}

	// Expectations
	mockProg.On("IsFeatureUnlocked", ctx, "feature_jobs_xp").Return(true, nil)
	mockRepo.On("GetJobByKey", ctx, jobKey).Return(testJob, nil)
	mockRepo.On("GetUserJob", ctx, userID, jobID).Return(initialProgress, nil)

	// Expect upsert with new XP (1000 XP should be enough for level 1)
	mockRepo.On("UpsertUserJob", ctx, mock.MatchedBy(func(uj *domain.UserJob) bool {
		return uj.CurrentLevel > 0 // Verify level increased
	})).Return(nil)

	mockRepo.On("RecordJobXPEvent", ctx, mock.Anything).Return(nil)

	mockStats.On("RecordUserEvent", ctx, userID, domain.EventJobLevelUp, mock.Anything).Return(nil)

	// CRITICAL: Expect Publish to be called
	mockBus.On("Publish", ctx, mock.MatchedBy(func(e event.Event) bool {
		// Cast domain.EventType to event.Type
		return e.Type == event.Type(domain.EventJobLevelUp) &&
			e.Payload.(map[string]interface{})["new_level"].(int) > 0
	})).Return(nil)

	// Execute
	// Award 1000 XP
	result, err := svc.AwardXP(ctx, userID, jobKey, 1000, "test", nil)

	assert.NoError(t, err)
	assert.True(t, result.LeveledUp)

	mockBus.AssertExpectations(t)
	mockStats.AssertExpectations(t)
}
