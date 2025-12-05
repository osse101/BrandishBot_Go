package job

import (
	"context"
	"math"
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

	// Helper to calculate XP for a specific level to ensure we test accurate boundaries
	xpForLevel := func(lvl int) int64 {
		cumulative := int64(0)
		for i := 1; i <= lvl; i++ {
			cumulative += int64(BaseXP * math.Pow(float64(i), LevelExponent))
		}
		return cumulative
	}

	tests := []struct {
		xp       int64
		expected int
	}{
		{0, 0},
		{xpForLevel(1) / 2, 0},            // Halfway to level 1
		{xpForLevel(1), 1},                // Exact level 1
		{xpForLevel(1) + 100, 1},          // Level 1 + some over
		{xpForLevel(2), 2},                // Exact level 2
		{xpForLevel(4), 4},                // Exact level 4
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, svc.CalculateLevel(tt.xp), "XP: %d", tt.xp)
	}
}

func TestGetXPForLevel(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog)

	// Same formula logic as service to verify consistency
	expectedLevel2XP := int64(BaseXP*math.Pow(1, LevelExponent)) + int64(BaseXP*math.Pow(2, LevelExponent))

	tests := []struct {
		level    int
		expected int64
	}{
		{0, 0},
		{1, int64(BaseXP)},
		{2, expectedLevel2XP},
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
	jobKey := JobKeyBlacksmith
	jobID := 1
	baseXP := BlacksmithXPPerItem

	job := &domain.Job{ID: jobID, JobKey: jobKey}
	
	prog.On("IsFeatureUnlocked", ctx, "feature_jobs_xp").Return(true, nil)
	repo.On("GetJobByKey", ctx, jobKey).Return(job, nil)
	repo.On("GetUserJob", ctx, userID, jobID).Return(nil, nil) // New user job
	repo.On("UpsertUserJob", ctx, mock.MatchedBy(func(uj *domain.UserJob) bool {
		return uj.UserID == userID && uj.CurrentXP == int64(BlacksmithXPPerItem) && uj.CurrentLevel == 0
	})).Return(nil)
	repo.On("RecordJobXPEvent", ctx, mock.MatchedBy(func(e *domain.JobXPEvent) bool {
		return e.XPAmount == BlacksmithXPPerItem
	})).Return(nil)

	result, err := svc.AwardXP(ctx, userID, jobKey, baseXP, "test", nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(BlacksmithXPPerItem), result.NewXP)
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
	jobKey := JobKeyBlacksmith
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

func TestAwardXP_DailyCap(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog)
	ctx := context.Background()

	userID := "u1"
	jobKey := JobKeyBlacksmith
	jobID := 1
	// Attempt to award more than the default daily cap
	amount := DefaultDailyCap + 100

	job := &domain.Job{ID: jobID, JobKey: jobKey}

	prog.On("IsFeatureUnlocked", ctx, "feature_jobs_xp").Return(true, nil)
	repo.On("GetJobByKey", ctx, jobKey).Return(job, nil)
	// User has 0 XP gained today
	repo.On("GetUserJob", ctx, userID, jobID).Return(&domain.UserJob{
		UserID: userID, JobID: jobID, XPGainedToday: 0,
	}, nil)

	// Should clamp to DefaultDailyCap
	repo.On("UpsertUserJob", ctx, mock.MatchedBy(func(uj *domain.UserJob) bool {
		return uj.XPGainedToday == int64(DefaultDailyCap) && uj.CurrentXP == int64(DefaultDailyCap)
	})).Return(nil)
	repo.On("RecordJobXPEvent", ctx, mock.MatchedBy(func(e *domain.JobXPEvent) bool {
		return e.XPAmount == DefaultDailyCap
	})).Return(nil)

	result, err := svc.AwardXP(ctx, userID, jobKey, amount, "test", nil)

	assert.NoError(t, err)
	// mock matcher verifies XPGainedToday
	assert.Equal(t, int(DefaultDailyCap), result.XPGained)
}

func TestAwardXP_DailyCap_Reached(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog)
	ctx := context.Background()

	userID := "u1"
	jobKey := JobKeyBlacksmith
	jobID := 1

	job := &domain.Job{ID: jobID, JobKey: jobKey}

	prog.On("IsFeatureUnlocked", ctx, "feature_jobs_xp").Return(true, nil)
	repo.On("GetJobByKey", ctx, jobKey).Return(job, nil)
	// User has already reached the cap
	repo.On("GetUserJob", ctx, userID, jobID).Return(&domain.UserJob{
		UserID: userID, JobID: jobID, XPGainedToday: int64(DefaultDailyCap),
	}, nil)

	result, err := svc.AwardXP(ctx, userID, jobKey, 10, "test", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "daily XP cap reached")
}

func TestAwardXP_MaxLevel(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog)
	ctx := context.Background()

	userID := "u1"
	jobKey := JobKeyBlacksmith
	jobID := 1
	
	// DefaultMaxLevel is 10.
	// XP for Level 11 is roughly: 100 * sum(i^1.5 for i=1..11).
	// Let's just set CurrentXP to a very high number that definitely exceeds Level 10 requirement.
	// We verify that despite having enough XP for level >10, the Level field is clamped.
	startXP := int64(50000) 
	awardAmount := 10 // Small amount to avoid daily cap

	job := &domain.Job{ID: jobID, JobKey: jobKey}

	prog.On("IsFeatureUnlocked", ctx, "feature_jobs_xp").Return(true, nil)
	repo.On("GetJobByKey", ctx, jobKey).Return(job, nil)
	repo.On("GetUserJob", ctx, userID, jobID).Return(&domain.UserJob{
		UserID: userID, JobID: jobID, CurrentXP: startXP, CurrentLevel: 10, XPGainedToday: 0,
	}, nil)

	// Resulting level should be clamped to DefaultMaxLevel (10)
	// Even though 50010 XP is way higher than needed for Level 10
	repo.On("UpsertUserJob", ctx, mock.MatchedBy(func(uj *domain.UserJob) bool {
		return uj.CurrentLevel == DefaultMaxLevel
	})).Return(nil)
	repo.On("RecordJobXPEvent", ctx, mock.Anything).Return(nil)

	result, err := svc.AwardXP(ctx, userID, jobKey, awardAmount, "test", nil)

	assert.NoError(t, err)
	assert.Equal(t, DefaultMaxLevel, result.NewLevel)
}

func TestGetJobBonus(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog)
	ctx := context.Background()

	jobID := 1
	jobKey := JobKeyExplorer
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
