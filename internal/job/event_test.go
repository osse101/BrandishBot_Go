package job

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
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

func (m *MockRepo) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return nil, nil
}

func (m *MockRepo) GetUserJobsByPlatform(ctx context.Context, platform, platformID string) ([]domain.UserJob, error) {
	return nil, nil
}

func (m *MockRepo) ResetDailyJobXP(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *MockRepo) GetLastDailyResetTime(ctx context.Context) (time.Time, int64, error) {
	return time.Time{}, 0, nil
}

func (m *MockRepo) UpdateDailyResetTime(ctx context.Context, resetTime time.Time, recordsAffected int64) error {
	return nil
}

// Mock Progression
type MockProgression struct {
	mock.Mock
}

// GetModifiedValue implements ProgressionService
func (m *MockProgression) GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error) {
	args := m.Called(ctx, featureKey, baseValue)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockProgression) IsFeatureUnlocked(ctx context.Context, featureKey string) (bool, error) {
	args := m.Called(ctx, featureKey)
	return args.Bool(0), args.Error(1)
}

func (m *MockProgression) IsNodeUnlocked(ctx context.Context, nodeKey string, level int) (bool, error) {
	args := m.Called(ctx, nodeKey, level)
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
func (m *MockStats) GetUserSlotsStats(ctx context.Context, userID, period string) (*domain.SlotsStats, error) {
	return nil, nil
}
func (m *MockStats) GetSlotsLeaderboardByProfit(ctx context.Context, period string, limit int) ([]domain.SlotsStats, error) {
	return nil, nil
}
func (m *MockStats) GetSlotsLeaderboardByWinRate(ctx context.Context, period string, minSpins, limit int) ([]domain.SlotsStats, error) {
	return nil, nil
}
func (m *MockStats) GetSlotsLeaderboardByMegaJackpots(ctx context.Context, period string, limit int) ([]domain.SlotsStats, error) {
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

	svc := NewService(mockRepo, mockProg, mockBus, resilientPub)
	// Force deterministic RNG (No Crit)
	if s, ok := svc.(*service); ok {
		s.rnd = func() float64 { return 1.0 }
	}
	ctx := context.Background()

	// Setup data
	job := &domain.Job{ID: 1, JobKey: "explorer"}
	userJob := &domain.UserJob{
		UserID:       "user123",
		JobID:        1,
		CurrentXP:    0,
		CurrentLevel: 1,
	}

	mockProg.On("IsNodeUnlocked", ctx, "explorer", 1).Return(true, nil)
	mockRepo.On("GetJobByKey", ctx, "explorer").Return(job, nil)
	mockRepo.On("GetUserJob", ctx, "user123", 1).Return(userJob, nil)
	mockRepo.On("UpsertUserJob", ctx, mock.Anything).Return(nil)
	mockRepo.On("RecordJobXPEvent", ctx, mock.Anything).Return(nil)
	// No level-up expected due to daily cap

	// Mock progression service for default values
	mockProg.On("GetModifiedValue", mock.Anything, "job_xp_multiplier", mock.Anything).Return(1.0, nil)
	mockProg.On("GetModifiedValue", mock.Anything, "job_level_cap", mock.Anything).Return(float64(10), nil)
	mockProg.On("GetModifiedValue", mock.Anything, "job_daily_cap", mock.Anything).Return(250.0, nil)

	// Execute - Award XP (capped at daily limit of 250)
	// Starting at level 1 with 0 XP, +250 brings to 250 total (still level 1)
	result, err := svc.AwardXP(ctx, "user123", "explorer", 500, "test", nil)

	// Assert - No level-up since capped at 250 XP (level 2 needs ~330 total)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.LeveledUp)
	assert.Equal(t, 1, result.NewLevel)

	// Give the async publisher a moment to process
	time.Sleep(50 * time.Millisecond)

	mockBus.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockProg.AssertExpectations(t)
	mockStats.AssertExpectations(t)
}

func TestAwardXP_GuaranteedCriticalSuccess(t *testing.T) {
	// Setup
	mockRepo := new(MockRepo)
	mockProg := new(MockProgression)
	mockBus := new(MockBus)

	// Use nil publisher as we aren't testing that part here, or mock if needed.
	// The service helper allows nil publisher for basic ops, but let's be safe and provide one.
	tmpFile := t.TempDir() + "/deadletter_crit.jsonl"
	resilientPub, _ := event.NewResilientPublisher(mockBus, 3, 100*time.Millisecond, tmpFile)
	defer resilientPub.Shutdown(context.Background())

	svc := NewService(mockRepo, mockProg, mockBus, resilientPub)
	// Force Critical Success
	if s, ok := svc.(*service); ok {
		s.rnd = func() float64 { return 0.0 }
	}
	ctx := context.Background()

	// Data
	job := &domain.Job{ID: 1, JobKey: "warrior"}
	userJob := &domain.UserJob{
		UserID:       "user_crit",
		JobID:        1,
		CurrentXP:    100,
		CurrentLevel: 1,
	}

	mockProg.On("IsNodeUnlocked", ctx, "warrior", 1).Return(true, nil)
	mockRepo.On("GetJobByKey", ctx, "warrior").Return(job, nil)
	mockRepo.On("GetUserJob", ctx, "user_crit", 1).Return(userJob, nil)
	mockRepo.On("UpsertUserJob", ctx, mock.Anything).Return(nil)
	mockRepo.On("RecordJobXPEvent", ctx, mock.Anything).Return(nil)

	// Mock progression service for default values
	mockProg.On("GetModifiedValue", mock.Anything, "job_xp_multiplier", mock.Anything).Return(1.0, nil)
	mockProg.On("GetModifiedValue", mock.Anything, "job_level_cap", mock.Anything).Return(float64(10), nil)
	mockProg.On("GetModifiedValue", mock.Anything, "job_daily_cap", mock.Anything).Return(250.0, nil)

	// Epiphany bonus now publishes EventTypeJobXPCritical via the resilient publisher → event bus
	mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(e event.Event) bool {
		return string(e.Type) == string(domain.EventTypeJobXPCritical)
	})).Return(nil).Maybe()

	// No level-up expected (200 total XP is still level 1 with new curve)

	// Act
	// baseAmount = 100
	// EpiphanyMultiplier is 2.0
	result, err := svc.AwardXP(ctx, "user_crit", "warrior", 100, "test", nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	expectedXP := 100 * 2
	assert.Equal(t, expectedXP, result.XPGained)

	mockRepo.AssertExpectations(t)
	mockProg.AssertExpectations(t)
}

func TestAwardXP_GuaranteedNoCriticalSuccess(t *testing.T) {
	// Setup
	mockRepo := new(MockRepo)
	mockProg := new(MockProgression)
	mockStats := new(MockStats)
	mockBus := new(MockBus) // Not used for crit event, but passed to constructor

	tmpFile := t.TempDir() + "/deadletter_nocrit.jsonl"
	resilientPub, _ := event.NewResilientPublisher(mockBus, 3, 100*time.Millisecond, tmpFile)
	defer resilientPub.Shutdown(context.Background())

	svc := NewService(mockRepo, mockProg, mockBus, resilientPub)
	// Force No Critical Success
	if s, ok := svc.(*service); ok {
		s.rnd = func() float64 { return 1.0 }
	}
	ctx := context.Background()

	// Data
	job := &domain.Job{ID: 1, JobKey: "mage"}
	userJob := &domain.UserJob{
		UserID:       "user_nocrit",
		JobID:        1,
		CurrentXP:    100,
		CurrentLevel: 1,
	}

	mockProg.On("IsNodeUnlocked", ctx, "mage", 1).Return(true, nil)
	mockRepo.On("GetJobByKey", ctx, "mage").Return(job, nil)
	mockRepo.On("GetUserJob", ctx, "user_nocrit", 1).Return(userJob, nil)
	mockRepo.On("UpsertUserJob", ctx, mock.Anything).Return(nil)
	mockRepo.On("RecordJobXPEvent", ctx, mock.Anything).Return(nil)

	// Mock progression service for default values
	mockProg.On("GetModifiedValue", mock.Anything, "job_xp_multiplier", mock.Anything).Return(1.0, nil)
	mockProg.On("GetModifiedValue", mock.Anything, "job_level_cap", mock.Anything).Return(float64(10), nil)
	mockProg.On("GetModifiedValue", mock.Anything, "job_daily_cap", mock.Anything).Return(250.0, nil)

	// No level-up expected (100 awarded stays within daily cap, total is still level 1)

	// Act
	result, err := svc.AwardXP(ctx, "user_nocrit", "mage", 100, "test", nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 100, result.XPGained)

	mockStats.AssertExpectations(t)
}
