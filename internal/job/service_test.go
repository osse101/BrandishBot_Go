package job

import (
	"context"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetAllJobs(ctx context.Context) ([]domain.Job, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Job), args.Error(1)
}

func (m *MockRepository) GetJobByKey(ctx context.Context, jobKey string) (*domain.Job, error) {
	args := m.Called(ctx, jobKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Job), args.Error(1)
}

func (m *MockRepository) GetUserJobs(ctx context.Context, userID string) ([]domain.UserJob, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.UserJob), args.Error(1)
}

func (m *MockRepository) GetUserJob(ctx context.Context, userID string, jobID int) (*domain.UserJob, error) {
	args := m.Called(ctx, userID, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserJob), args.Error(1)
}

func (m *MockRepository) UpsertUserJob(ctx context.Context, userJob *domain.UserJob) error {
	args := m.Called(ctx, userJob)
	return args.Error(0)
}

func (m *MockRepository) RecordJobXPEvent(ctx context.Context, event *domain.JobXPEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockRepository) GetJobLevelBonuses(ctx context.Context, jobID int, level int) ([]domain.JobLevelBonus, error) {
	args := m.Called(ctx, jobID, level)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.JobLevelBonus), args.Error(1)
}

// MockProgressionService
type MockProgressionService struct {
	mock.Mock
}

func (m *MockProgressionService) IsFeatureUnlocked(ctx context.Context, featureKey string) (bool, error) {
	args := m.Called(ctx, featureKey)
	return args.Bool(0), args.Error(1)
}

func (m *MockProgressionService) GetProgressionStatus(ctx context.Context) (*domain.ProgressionStatus, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ProgressionStatus), args.Error(1)
}

// Tests

func TestCalculateLevel(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog)

	tests := []struct {
		xp       int64
		expected int
	}{
		{0, 0},
		{50, 0},
		{100, 1},
		{200, 1},
		{383, 2}, // 100 + 283
		{1000, 4},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, svc.CalculateLevel(tt.xp), "XP: %d", tt.xp)
	}
}

func TestGetXPForLevel(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog)

	tests := []struct {
		level    int
		expected int64
	}{
		{0, 0},
		{1, 100},
		{2, 383}, // 100 + 283
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, svc.GetXPForLevel(tt.level))
	}
}

func TestAwardXP_Success(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog)
	ctx := context.Background()

	userID := "user1"
	jobKey := "blacksmith"
	jobID := 1
	baseXP := 50

	job := &domain.Job{ID: jobID, JobKey: jobKey}
	
	prog.On("IsFeatureUnlocked", ctx, "feature_jobs_xp").Return(true, nil)
	repo.On("GetJobByKey", ctx, jobKey).Return(job, nil)
	repo.On("GetUserJob", ctx, userID, jobID).Return(nil, nil) // New user job
	repo.On("UpsertUserJob", ctx, mock.MatchedBy(func(uj *domain.UserJob) bool {
		return uj.UserID == userID && uj.CurrentXP == 50 && uj.CurrentLevel == 0
	})).Return(nil)
	repo.On("RecordJobXPEvent", ctx, mock.Anything).Return(nil)

	result, err := svc.AwardXP(ctx, userID, jobKey, baseXP, "test", nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(50), result.NewXP)
	assert.Equal(t, 0, result.NewLevel)
	
	repo.AssertExpectations(t)
	prog.AssertExpectations(t)
}

func TestAwardXP_LevelUp(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog)
	ctx := context.Background()

	userID := "user1"
	jobKey := "blacksmith"
	jobID := 1
	baseXP := 150 

	job := &domain.Job{ID: jobID, JobKey: jobKey}
	
	prog.On("IsFeatureUnlocked", ctx, "feature_jobs_xp").Return(true, nil)
	repo.On("GetJobByKey", ctx, jobKey).Return(job, nil)
	// Current XP 0
	repo.On("GetUserJob", ctx, userID, jobID).Return(&domain.UserJob{
		UserID: userID, JobID: jobID, CurrentXP: 0, CurrentLevel: 0,
	}, nil)
	
	// 150 XP should be Level 1 (requires 100)
	repo.On("UpsertUserJob", ctx, mock.MatchedBy(func(uj *domain.UserJob) bool {
		return uj.CurrentXP == 150 && uj.CurrentLevel == 1
	})).Return(nil)
	repo.On("RecordJobXPEvent", ctx, mock.Anything).Return(nil)

	result, err := svc.AwardXP(ctx, userID, jobKey, baseXP, "test", nil)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.NewLevel)
	assert.True(t, result.LeveledUp)
}

func TestAwardXP_Locked(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog)
	ctx := context.Background()

	prog.On("IsFeatureUnlocked", ctx, "feature_jobs_xp").Return(false, nil)

	_, err := svc.AwardXP(ctx, "u1", "j1", 10, "t", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not unlocked")
}

func TestGetJobBonus(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog)
	ctx := context.Background()

	jobID := 1
	jobKey := "explorer"
	userID := "u1"

	job := &domain.Job{ID: jobID, JobKey: jobKey}
	userJob := &domain.UserJob{UserID: userID, JobID: jobID, CurrentLevel: 5}
	bonuses := []domain.JobLevelBonus{
		{BonusType: "chance", BonusValue: 0.1},
		{BonusType: "chance", BonusValue: 0.25}, // Higher value should be picked
	}

	repo.On("GetJobByKey", ctx, jobKey).Return(job, nil).Twice() // Once for GetJobLevel, once in GetJobBonus
	repo.On("GetUserJob", ctx, userID, jobID).Return(userJob, nil)
	repo.On("GetJobLevelBonuses", ctx, jobID, 5).Return(bonuses, nil)

	val, err := svc.GetJobBonus(ctx, userID, jobKey, "chance")
	assert.NoError(t, err)
	assert.Equal(t, 0.25, val)
}

func TestGetPrimaryJob(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog)
	ctx := context.Background()

	jobs := []domain.Job{
		{ID: 1, JobKey: "j1", DisplayName: "Job1"},
		{ID: 2, JobKey: "j2", DisplayName: "Job2"},
	}
	userJobs := []domain.UserJob{
		{JobID: 2, CurrentLevel: 10, CurrentXP: 5000}, // Highest level first
		{JobID: 1, CurrentLevel: 5, CurrentXP: 1000},
	}

	repo.On("GetAllJobs", ctx).Return(jobs, nil)
	repo.On("GetUserJobs", ctx, "u1").Return(userJobs, nil)

	primary, err := svc.GetPrimaryJob(ctx, "u1")
	assert.NoError(t, err)
	assert.NotNil(t, primary)
	assert.Equal(t, "j2", primary.JobKey)
	assert.Equal(t, 10, primary.Level)
}
