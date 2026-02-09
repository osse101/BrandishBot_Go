package job

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
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

func (m *MockRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	args := m.Called(ctx, platform, platformID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockRepository) GetUserJobsByPlatform(ctx context.Context, platform, platformID string) ([]domain.UserJob, error) {
	args := m.Called(ctx, platform, platformID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.UserJob), args.Error(1)
}

func (m *MockRepository) ResetDailyJobXP(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockRepository) GetLastDailyResetTime(ctx context.Context) (time.Time, int64, error) {
	args := m.Called(ctx)
	return time.Time{}, int64(args.Int(1)), args.Error(2)
}

func (m *MockRepository) UpdateDailyResetTime(ctx context.Context, resetTime time.Time, recordsAffected int64) error {
	args := m.Called(ctx, resetTime, recordsAffected)
	return args.Error(0)
}

// MockProgressionService
type MockProgressionService struct {
	mock.Mock
}

func (m *MockProgressionService) IsFeatureUnlocked(ctx context.Context, featureKey string) (bool, error) {
	args := m.Called(ctx, featureKey)
	return args.Bool(0), args.Error(1)
}

func (m *MockProgressionService) IsNodeUnlocked(ctx context.Context, nodeKey string, level int) (bool, error) {
	args := m.Called(ctx, nodeKey, level)
	return args.Bool(0), args.Error(1)
}

func (m *MockProgressionService) GetProgressionStatus(ctx context.Context) (*domain.ProgressionStatus, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ProgressionStatus), args.Error(1)
}

func (m *MockProgressionService) GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error) {
	args := m.Called(ctx, featureKey, baseValue)
	return args.Get(0).(float64), args.Error(1)
}

// MockStatsService
type MockStatsService struct {
	mock.Mock
}

func (m *MockStatsService) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata map[string]interface{}) error {
	args := m.Called(ctx, userID, eventType, metadata)
	return args.Error(0)
}

func (m *MockStatsService) GetUserStats(ctx context.Context, userID string, period string) (*domain.StatsSummary, error) {
	args := m.Called(ctx, userID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.StatsSummary), args.Error(1)
}

func (m *MockStatsService) GetUserCurrentStreak(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockStatsService) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) {
	args := m.Called(ctx, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.StatsSummary), args.Error(1)
}

func (m *MockStatsService) GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error) {
	args := m.Called(ctx, eventType, period, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.LeaderboardEntry), args.Error(1)
}

func (m *MockStatsService) GetUserSlotsStats(ctx context.Context, userID, period string) (*domain.SlotsStats, error) {
	args := m.Called(ctx, userID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SlotsStats), args.Error(1)
}

func (m *MockStatsService) GetSlotsLeaderboardByProfit(ctx context.Context, period string, limit int) ([]domain.SlotsStats, error) {
	args := m.Called(ctx, period, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.SlotsStats), args.Error(1)
}

func (m *MockStatsService) GetSlotsLeaderboardByWinRate(ctx context.Context, period string, minSpins, limit int) ([]domain.SlotsStats, error) {
	args := m.Called(ctx, period, minSpins, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.SlotsStats), args.Error(1)
}

func (m *MockStatsService) GetSlotsLeaderboardByMegaJackpots(ctx context.Context, period string, limit int) ([]domain.SlotsStats, error) {
	args := m.Called(ctx, period, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.SlotsStats), args.Error(1)
}

// Tests

func TestCalculateLevel(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)

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
		{xpForLevel(1) / 2, 0},   // Halfway to level 1
		{xpForLevel(1), 1},       // Exact level 1
		{xpForLevel(1) + 100, 1}, // Level 1 + some over
		{xpForLevel(2), 2},       // Exact level 2
		{xpForLevel(4), 4},       // Exact level 4
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, svc.CalculateLevel(tt.xp), "XP: %d", tt.xp)
	}
}

func TestGetXPForLevel(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)

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
	svc := NewService(repo, prog, nil, nil, nil).(*service)
	// Force RNG to fail Epiphany
	svc.rnd = func() float64 { return 1.0 }

	ctx := context.Background()

	userID := "user1"
	jobKey := JobKeyBlacksmith
	jobID := 1
	baseXP := BlacksmithXPPerItem

	job := &domain.Job{ID: jobID, JobKey: jobKey}

	prog.On("IsNodeUnlocked", ctx, jobKey, 1).Return(true, nil)
	repo.On("GetJobByKey", ctx, jobKey).Return(job, nil)
	repo.On("GetUserJob", ctx, userID, jobID).Return(nil, nil) // New user job
	repo.On("UpsertUserJob", ctx, mock.MatchedBy(func(uj *domain.UserJob) bool {
		return uj.UserID == userID && uj.CurrentXP == int64(BlacksmithXPPerItem) && uj.CurrentLevel == 0
	})).Return(nil)
	repo.On("RecordJobXPEvent", ctx, mock.MatchedBy(func(e *domain.JobXPEvent) bool {
		return e.XPAmount == BlacksmithXPPerItem
	})).Return(nil)

	prog.On("GetModifiedValue", ctx, "job_xp_multiplier", 1.0).Return(1.0, nil)
	prog.On("GetModifiedValue", ctx, "job_level_cap", mock.Anything).Return(float64(DefaultMaxLevel), nil)
	prog.On("GetModifiedValue", ctx, "job_daily_cap", float64(DefaultDailyCap)).Return(float64(DefaultDailyCap), nil)

	result, err := svc.AwardXP(ctx, userID, jobKey, baseXP, "test", nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(BlacksmithXPPerItem), result.NewXP)
	assert.Equal(t, 0, result.NewLevel)

	repo.AssertExpectations(t)
	prog.AssertExpectations(t)
}

func TestAwardXP_Epiphany(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	statsSvc := new(MockStatsService)
	svc := NewService(repo, prog, statsSvc, nil, nil).(*service)

	// Force RNG to trigger Epiphany (value < 0.05)
	svc.rnd = func() float64 { return 0.01 }

	ctx := context.Background()

	userID := "user1"
	jobKey := JobKeyBlacksmith
	jobID := 1
	baseXP := 100
	expectedXP := 200 // 100 * 2.0

	job := &domain.Job{ID: jobID, JobKey: jobKey}

	prog.On("IsNodeUnlocked", ctx, jobKey, 1).Return(true, nil)
	repo.On("GetJobByKey", ctx, jobKey).Return(job, nil)
	repo.On("GetUserJob", ctx, userID, jobID).Return(nil, nil)

	// Expect doubled XP
	repo.On("UpsertUserJob", ctx, mock.MatchedBy(func(uj *domain.UserJob) bool {
		return uj.CurrentXP == int64(expectedXP)
	})).Return(nil)
	repo.On("RecordJobXPEvent", ctx, mock.MatchedBy(func(e *domain.JobXPEvent) bool {
		return e.XPAmount == expectedXP
	})).Return(nil)

	prog.On("GetModifiedValue", ctx, "job_xp_multiplier", 1.0).Return(1.0, nil)
	prog.On("GetModifiedValue", ctx, "job_level_cap", mock.Anything).Return(float64(DefaultMaxLevel), nil)
	prog.On("GetModifiedValue", ctx, "job_daily_cap", float64(DefaultDailyCap)).Return(float64(DefaultDailyCap), nil)

	// Expect stats event for Epiphany
	statsSvc.On("RecordUserEvent", ctx, userID, domain.EventJobXPCritical, mock.MatchedBy(func(m map[string]interface{}) bool {
		return m["job"] == jobKey && m["bonus_xp"] == (expectedXP-baseXP)
	})).Return(nil)

	// Level up from 0 to 1 is expected (200 XP > 100 needed for level 1)
	statsSvc.On("RecordUserEvent", ctx, userID, domain.EventJobLevelUp, mock.MatchedBy(func(m map[string]interface{}) bool {
		return m["job"] == jobKey && m["level"] == 1 && m["old_level"] == 0
	})).Return(nil)

	result, err := svc.AwardXP(ctx, userID, jobKey, baseXP, "test", nil)

	assert.NoError(t, err)
	assert.Equal(t, expectedXP, result.XPGained)

	repo.AssertExpectations(t)
	prog.AssertExpectations(t)
	statsSvc.AssertExpectations(t)
}

func TestAwardXP_LevelUp(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	statsSvc := new(MockStatsService)
	svc := NewService(repo, prog, statsSvc, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 }

	ctx := context.Background()

	userID := "user1"
	jobKey := JobKeyBlacksmith
	jobID := 1
	baseXP := 150

	job := &domain.Job{ID: jobID, JobKey: jobKey}

	prog.On("IsNodeUnlocked", ctx, jobKey, 1).Return(true, nil)
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

	prog.On("GetModifiedValue", ctx, "job_xp_multiplier", 1.0).Return(1.0, nil)
	prog.On("GetModifiedValue", ctx, "job_level_cap", mock.Anything).Return(float64(DefaultMaxLevel), nil)
	prog.On("GetModifiedValue", ctx, "job_daily_cap", float64(DefaultDailyCap)).Return(float64(DefaultDailyCap), nil)

	// Expect stats event for Level Up
	statsSvc.On("RecordUserEvent", ctx, userID, domain.EventJobLevelUp, mock.MatchedBy(func(m map[string]interface{}) bool {
		return m["level"] == 1 && m["job"] == jobKey
	})).Return(nil)

	result, err := svc.AwardXP(ctx, userID, jobKey, baseXP, "test", nil)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.NewLevel)
	assert.True(t, result.LeveledUp)
	statsSvc.AssertExpectations(t)
}

func TestAwardXP_Locked_Job(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	prog.On("IsNodeUnlocked", ctx, "j1", 1).Return(false, nil)

	_, err := svc.AwardXP(ctx, "u1", "j1", 10, "t", nil)
	assert.ErrorIs(t, err, domain.ErrFeatureLocked)
}

func TestAwardXP_DailyCap(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 }

	ctx := context.Background()

	userID := "u1"
	jobKey := JobKeyBlacksmith
	jobID := 1
	// Attempt to award more than the default daily cap
	amount := DefaultDailyCap + 100

	job := &domain.Job{ID: jobID, JobKey: jobKey}

	prog.On("IsNodeUnlocked", ctx, jobKey, 1).Return(true, nil)
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

	prog.On("GetModifiedValue", ctx, "job_xp_multiplier", 1.0).Return(1.0, nil)
	prog.On("GetModifiedValue", ctx, "job_level_cap", mock.Anything).Return(float64(DefaultMaxLevel), nil)
	prog.On("GetModifiedValue", ctx, "job_daily_cap", float64(DefaultDailyCap)).Return(float64(DefaultDailyCap), nil)

	result, err := svc.AwardXP(ctx, userID, jobKey, amount, "test", nil)

	assert.NoError(t, err)
	// mock matcher verifies XPGainedToday
	assert.Equal(t, int(DefaultDailyCap), result.XPGained)
}

func TestAwardXP_DailyCap_Reached(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 }

	ctx := context.Background()

	userID := "u1"
	jobKey := JobKeyBlacksmith
	jobID := 1

	job := &domain.Job{ID: jobID, JobKey: jobKey}

	prog.On("IsNodeUnlocked", ctx, jobKey, 1).Return(true, nil)
	repo.On("GetJobByKey", ctx, jobKey).Return(job, nil)
	// User has already reached the cap
	repo.On("GetUserJob", ctx, userID, jobID).Return(&domain.UserJob{
		UserID: userID, JobID: jobID, XPGainedToday: int64(DefaultDailyCap),
	}, nil)

	prog.On("GetModifiedValue", ctx, "job_xp_multiplier", 1.0).Return(1.0, nil)
	prog.On("GetModifiedValue", ctx, "job_level_cap", mock.Anything).Return(float64(DefaultMaxLevel), nil)
	prog.On("GetModifiedValue", ctx, "job_daily_cap", float64(DefaultDailyCap)).Return(float64(DefaultDailyCap), nil)

	_, err := svc.AwardXP(ctx, userID, jobKey, 10, "test", nil)

	assert.ErrorIs(t, err, domain.ErrDailyCapReached)
}

func TestAwardXP_RareCandy_BypassesDailyCap(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 }

	ctx := context.Background()

	userID := "u1"
	jobKey := JobKeyBlacksmith
	jobID := 1
	rarecandyXP := 500

	job := &domain.Job{ID: jobID, JobKey: jobKey}

	prog.On("IsNodeUnlocked", ctx, jobKey, 1).Return(true, nil)
	repo.On("GetJobByKey", ctx, jobKey).Return(job, nil)
	// User has already reached the cap
	repo.On("GetUserJob", ctx, userID, jobID).Return(&domain.UserJob{
		UserID: userID, JobID: jobID, XPGainedToday: int64(DefaultDailyCap),
	}, nil)

	prog.On("GetModifiedValue", ctx, "job_xp_multiplier", 1.0).Return(1.0, nil)
	prog.On("GetModifiedValue", ctx, "job_level_cap", mock.Anything).Return(float64(DefaultMaxLevel), nil)
	prog.On("GetModifiedValue", ctx, "job_daily_cap", float64(DefaultDailyCap)).Return(float64(DefaultDailyCap), nil)
	prog.On("GetModifiedValue", ctx, "job_max_level", float64(DefaultMaxLevel)).Return(float64(DefaultMaxLevel), nil)

	repo.On("UpsertUserJob", ctx, mock.MatchedBy(func(uj *domain.UserJob) bool {
		// Verify that XP was awarded and xp_gained_today includes rare candy XP
		return uj.XPGainedToday == int64(DefaultDailyCap+rarecandyXP) && uj.CurrentXP == int64(rarecandyXP)
	})).Return(nil)

	repo.On("RecordJobXPEvent", ctx, mock.MatchedBy(func(e *domain.JobXPEvent) bool {
		return e.XPAmount == rarecandyXP && e.SourceType == SourceRareCandy
	})).Return(nil)

	result, err := svc.AwardXP(ctx, userID, jobKey, rarecandyXP, SourceRareCandy, nil)

	assert.NoError(t, err)
	assert.Equal(t, rarecandyXP, result.XPGained)
}

func TestAwardXP_RareCandy_ExceedsNormalCap(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 }

	ctx := context.Background()

	userID := "u1"
	jobKey := JobKeyBlacksmith
	jobID := 1
	initialXP := 400
	rarecandyXP := 1500 // 3 rare candies

	job := &domain.Job{ID: jobID, JobKey: jobKey}

	prog.On("IsNodeUnlocked", ctx, jobKey, 1).Return(true, nil)
	repo.On("GetJobByKey", ctx, jobKey).Return(job, nil)
	// User has 400 XP gained today
	repo.On("GetUserJob", ctx, userID, jobID).Return(&domain.UserJob{
		UserID: userID, JobID: jobID, XPGainedToday: int64(initialXP), CurrentXP: int64(initialXP),
	}, nil)

	prog.On("GetModifiedValue", ctx, "job_xp_multiplier", 1.0).Return(1.0, nil)
	prog.On("GetModifiedValue", ctx, "job_level_cap", mock.Anything).Return(float64(DefaultMaxLevel), nil)
	prog.On("GetModifiedValue", ctx, "job_daily_cap", float64(DefaultDailyCap)).Return(float64(DefaultDailyCap), nil)
	prog.On("GetModifiedValue", ctx, "job_max_level", float64(DefaultMaxLevel)).Return(float64(DefaultMaxLevel), nil)

	repo.On("UpsertUserJob", ctx, mock.MatchedBy(func(uj *domain.UserJob) bool {
		// Verify that XP was awarded and xp_gained_today exceeds normal cap
		return uj.XPGainedToday == int64(initialXP+rarecandyXP) && uj.CurrentXP == int64(initialXP+rarecandyXP)
	})).Return(nil)

	repo.On("RecordJobXPEvent", ctx, mock.MatchedBy(func(e *domain.JobXPEvent) bool {
		return e.XPAmount == rarecandyXP && e.SourceType == SourceRareCandy
	})).Return(nil)

	result, err := svc.AwardXP(ctx, userID, jobKey, rarecandyXP, SourceRareCandy, nil)

	assert.NoError(t, err)
	assert.Equal(t, rarecandyXP, result.XPGained)
	assert.Equal(t, int64(initialXP+rarecandyXP), result.NewXP)
}

func TestAwardXP_MaxLevel(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 }

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

	prog.On("IsNodeUnlocked", ctx, jobKey, 1).Return(true, nil)
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

	prog.On("GetModifiedValue", ctx, "job_xp_multiplier", 1.0).Return(1.0, nil)
	prog.On("GetModifiedValue", ctx, "job_level_cap", mock.Anything).Return(float64(DefaultMaxLevel), nil)
	prog.On("GetModifiedValue", ctx, "job_daily_cap", float64(DefaultDailyCap)).Return(float64(DefaultDailyCap), nil)

	result, err := svc.AwardXP(ctx, userID, jobKey, awardAmount, "test", nil)

	assert.NoError(t, err)
	assert.Equal(t, DefaultMaxLevel, result.NewLevel)
}

func TestGetJobBonus(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
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
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	jobs := []domain.Job{
		{ID: 1, JobKey: "j1", DisplayName: "Job1"},
		{ID: 2, JobKey: "j2", DisplayName: "Job2"},
	}
	userJobs := []domain.UserJob{
		{JobID: 2, CurrentLevel: 10, CurrentXP: 5000}, // Highest level first
		{JobID: 1, CurrentLevel: 5, CurrentXP: 1000},
	}

	repo.On("GetUserByPlatformID", ctx, "twitch", "u1").Return(&domain.User{ID: "u1"}, nil)
	repo.On("GetAllJobs", ctx).Return(jobs, nil)
	prog.On("IsNodeUnlocked", ctx, mock.Anything, 1).Return(true, nil)
	prog.On("GetModifiedValue", ctx, "job_level_cap", mock.Anything).Return(float64(DefaultMaxLevel), nil)
	repo.On("GetUserJobs", ctx, "u1").Return(userJobs, nil)

	primary, err := svc.GetPrimaryJob(ctx, "twitch", "u1")
	assert.NoError(t, err)
	assert.NotNil(t, primary)
	assert.Equal(t, "j2", primary.JobKey)
	assert.Equal(t, 10, primary.Level)
}

// GetAllJobs Tests

func TestGetAllJobs_Success(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	jobs := []domain.Job{
		{ID: 1, JobKey: JobKeyBlacksmith, DisplayName: "Blacksmith"},
		{ID: 2, JobKey: JobKeyExplorer, DisplayName: "Explorer"},
	}

	repo.On("GetAllJobs", ctx).Return(jobs, nil)

	result, err := svc.GetAllJobs(ctx)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, JobKeyBlacksmith, result[0].JobKey)
}

func TestGetAllJobs_Empty(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	repo.On("GetAllJobs", ctx).Return([]domain.Job{}, nil)

	result, err := svc.GetAllJobs(ctx)
	assert.NoError(t, err)
	assert.Len(t, result, 0)
}

func TestGetAllJobs_RepositoryError(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	repo.On("GetAllJobs", ctx).Return(nil, assert.AnError)

	result, err := svc.GetAllJobs(ctx)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// GetUserJobs Edge Cases

func TestGetUserJobs_NoProgress(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	jobs := []domain.Job{
		{ID: 1, JobKey: JobKeyBlacksmith, DisplayName: "Blacksmith"},
	}

	repo.On("GetAllJobs", ctx).Return(jobs, nil)
	prog.On("IsNodeUnlocked", ctx, JobKeyBlacksmith, 1).Return(true, nil) // Mock unlock status
	prog.On("GetModifiedValue", ctx, "job_level_cap", mock.Anything).Return(float64(DefaultMaxLevel), nil)
	repo.On("GetUserJobs", ctx, "u1").Return([]domain.UserJob{}, nil) // No progress

	result, err := svc.GetUserJobs(ctx, "u1")
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 0, result[0].Level)
	assert.Equal(t, int64(0), result[0].CurrentXP)
	assert.Equal(t, int64(BaseXP), result[0].XPToNextLevel) // XP to level 1
}

func TestGetUserJobs_RepositoryError(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	repo.On("GetAllJobs", ctx).Return(nil, assert.AnError)

	result, err := svc.GetUserJobs(ctx, "u1")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetUserJobs_UserJobsError(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	jobs := []domain.Job{
		{ID: 1, JobKey: JobKeyBlacksmith, DisplayName: "Blacksmith"},
	}

	repo.On("GetAllJobs", ctx).Return(jobs, nil)
	repo.On("GetUserJobs", ctx, "u1").Return(nil, assert.AnError)

	result, err := svc.GetUserJobs(ctx, "u1")
	assert.Error(t, err)
	assert.Nil(t, result)
}

// GetPrimaryJob Edge Cases

func TestGetPrimaryJob_NoJobs(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	repo.On("GetUserByPlatformID", ctx, "twitch", "u1").Return(&domain.User{ID: "u1"}, nil)
	repo.On("GetAllJobs", ctx).Return([]domain.Job{}, nil)
	prog.On("IsNodeUnlocked", ctx, mock.Anything, 1).Return(true, nil)
	prog.On("GetModifiedValue", ctx, "job_level_cap", mock.Anything).Return(float64(DefaultMaxLevel), nil)
	repo.On("GetUserJobs", ctx, "u1").Return([]domain.UserJob{}, nil)

	result, err := svc.GetPrimaryJob(ctx, "twitch", "u1")
	assert.NoError(t, err)
	assert.Nil(t, result) // No jobs means no primary
}

func TestGetPrimaryJob_TieOnLevel_HigherXP(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	jobs := []domain.Job{
		{ID: 1, JobKey: JobKeyBlacksmith, DisplayName: "Blacksmith"},
		{ID: 2, JobKey: JobKeyExplorer, DisplayName: "Explorer"},
	}
	// Same level, different XP - should pick higher XP
	userJobs := []domain.UserJob{
		{JobID: 1, CurrentLevel: 5, CurrentXP: 1000},
		{JobID: 2, CurrentLevel: 5, CurrentXP: 1500}, // Higher XP
	}

	repo.On("GetUserByPlatformID", ctx, "twitch", "u1").Return(&domain.User{ID: "u1"}, nil)
	prog.On("IsNodeUnlocked", ctx, mock.Anything, 1).Return(true, nil)
	prog.On("GetModifiedValue", ctx, "job_level_cap", mock.Anything).Return(float64(DefaultMaxLevel), nil)
	repo.On("GetAllJobs", ctx).Return(jobs, nil)
	repo.On("GetUserJobs", ctx, "u1").Return(userJobs, nil)

	result, err := svc.GetPrimaryJob(ctx, "twitch", "u1")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, JobKeyExplorer, result.JobKey)
	assert.Equal(t, int64(1500), result.CurrentXP)
}

func TestGetPrimaryJob_ErrorPropagation(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	repo.On("GetUserByPlatformID", ctx, "twitch", "u1").Return(&domain.User{ID: "u1"}, nil)
	prog.On("IsNodeUnlocked", ctx, mock.Anything, 1).Return(true, nil)
	prog.On("GetModifiedValue", ctx, "job_level_cap", mock.Anything).Return(float64(DefaultMaxLevel), nil)
	repo.On("GetAllJobs", ctx).Return(nil, assert.AnError)

	result, err := svc.GetPrimaryJob(ctx, "twitch", "u1")
	assert.Error(t, err)
	assert.Nil(t, result)
}

// GetJobLevel Error Paths

func TestGetJobLevel_JobNotFound(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	repo.On("GetJobByKey", ctx, "invalid_job").Return(nil, assert.AnError)

	level, err := svc.GetJobLevel(ctx, "u1", "invalid_job")
	assert.Error(t, err)
	assert.Equal(t, 0, level)
}

func TestGetJobLevel_RepositoryError(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	job := &domain.Job{ID: 1, JobKey: JobKeyBlacksmith}
	repo.On("GetJobByKey", ctx, JobKeyBlacksmith).Return(job, nil)
	repo.On("GetUserJob", ctx, "u1", 1).Return(nil, assert.AnError)

	level, err := svc.GetJobLevel(ctx, "u1", JobKeyBlacksmith)
	assert.Error(t, err)
	assert.Equal(t, 0, level)
}

func TestGetJobLevel_NoProgress(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	job := &domain.Job{ID: 1, JobKey: JobKeyBlacksmith}
	repo.On("GetJobByKey", ctx, JobKeyBlacksmith).Return(job, nil)
	repo.On("GetUserJob", ctx, "u1", 1).Return(nil, nil) // No progress yet

	level, err := svc.GetJobLevel(ctx, "u1", JobKeyBlacksmith)
	assert.NoError(t, err)
	assert.Equal(t, 0, level)
}

// GetJobBonus Edge Cases

func TestGetJobBonus_ZeroLevel(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	job := &domain.Job{ID: 1, JobKey: JobKeyExplorer}
	userJob := &domain.UserJob{UserID: "u1", JobID: 1, CurrentLevel: 0}

	repo.On("GetJobByKey", ctx, JobKeyExplorer).Return(job, nil)
	repo.On("GetUserJob", ctx, "u1", 1).Return(userJob, nil)
	// GetJobBonus returns early if level is 0

	val, err := svc.GetJobBonus(ctx, "u1", JobKeyExplorer, "chance")
	assert.NoError(t, err)
	assert.Equal(t, 0.0, val)
}

func TestGetJobBonus_NoBonusesConfigured(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	job := &domain.Job{ID: 1, JobKey: JobKeyExplorer}
	userJob := &domain.UserJob{UserID: "u1", JobID: 1, CurrentLevel: 5}

	repo.On("GetJobByKey", ctx, JobKeyExplorer).Return(job, nil).Twice()
	repo.On("GetUserJob", ctx, "u1", 1).Return(userJob, nil)
	repo.On("GetJobLevelBonuses", ctx, 1, 5).Return([]domain.JobLevelBonus{}, nil)

	val, err := svc.GetJobBonus(ctx, "u1", JobKeyExplorer, "chance")
	assert.NoError(t, err)
	assert.Equal(t, 0.0, val) // No bonuses configured
}

func TestGetJobBonus_MultipleBonusTypes(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	job := &domain.Job{ID: 1, JobKey: JobKeyExplorer}
	userJob := &domain.UserJob{UserID: "u1", JobID: 1, CurrentLevel: 10}
	bonuses := []domain.JobLevelBonus{
		{BonusType: "chance", BonusValue: 0.3},
		{BonusType: "multiplier", BonusValue: 1.5}, // Different type
		{BonusType: "chance", BonusValue: 0.2},     // Lower value of same type
	}

	repo.On("GetJobByKey", ctx, JobKeyExplorer).Return(job, nil).Twice()
	repo.On("GetUserJob", ctx, "u1", 1).Return(userJob, nil)
	repo.On("GetJobLevelBonuses", ctx, 1, 10).Return(bonuses, nil)

	val, err := svc.GetJobBonus(ctx, "u1", JobKeyExplorer, "chance")
	assert.NoError(t, err)
	assert.Equal(t, 0.3, val) // Should pick highest of "chance" type
}

func TestGetJobBonus_RepositoryError(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	repo.On("GetJobByKey", ctx, JobKeyExplorer).Return(nil, assert.AnError)

	val, err := svc.GetJobBonus(ctx, "u1", JobKeyExplorer, "chance")
	assert.Error(t, err)
	assert.Equal(t, 0.0, val)
}

// AwardXP Advanced Scenarios

func TestAwardXP_RepositoryFailure_GetJob(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	prog.On("IsNodeUnlocked", ctx, JobKeyBlacksmith, 1).Return(true, nil)
	repo.On("GetJobByKey", ctx, JobKeyBlacksmith).Return(nil, assert.AnError)

	result, err := svc.AwardXP(ctx, "u1", JobKeyBlacksmith, 10, "test", nil)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestAwardXP_RepositoryFailure_Upsert(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)
	ctx := context.Background()

	job := &domain.Job{ID: 1, JobKey: JobKeyBlacksmith}

	prog.On("IsNodeUnlocked", ctx, JobKeyBlacksmith, 1).Return(true, nil)
	repo.On("GetJobByKey", ctx, JobKeyBlacksmith).Return(job, nil)
	repo.On("GetUserJob", ctx, "u1", 1).Return(nil, nil)
	repo.On("UpsertUserJob", ctx, mock.Anything).Return(assert.AnError)

	prog.On("GetModifiedValue", ctx, "job_xp_multiplier", 1.0).Return(1.0, nil)
	prog.On("GetModifiedValue", ctx, "job_level_cap", mock.Anything).Return(float64(DefaultMaxLevel), nil)
	prog.On("GetModifiedValue", ctx, "job_daily_cap", float64(DefaultDailyCap)).Return(float64(DefaultDailyCap), nil)

	result, err := svc.AwardXP(ctx, "u1", JobKeyBlacksmith, 10, "test", nil)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestAwardXP_PartialDailyCapRemaining(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 }

	ctx := context.Background()

	job := &domain.Job{ID: 1, JobKey: JobKeyBlacksmith}
	// User has 200 XP gained today, cap is 250, so only 50 remaining
	userJob := &domain.UserJob{
		UserID:        "u1",
		JobID:         1,
		CurrentXP:     2000,
		CurrentLevel:  5,
		XPGainedToday: 200,
	}

	prog.On("IsNodeUnlocked", ctx, JobKeyBlacksmith, 1).Return(true, nil)
	repo.On("GetJobByKey", ctx, JobKeyBlacksmith).Return(job, nil)
	repo.On("GetUserJob", ctx, "u1", 1).Return(userJob, nil)

	prog.On("GetModifiedValue", ctx, "job_xp_multiplier", 1.0).Return(1.0, nil)
	prog.On("GetModifiedValue", ctx, "job_level_cap", mock.Anything).Return(float64(DefaultMaxLevel), nil)
	prog.On("GetModifiedValue", ctx, "job_daily_cap", float64(DefaultDailyCap)).Return(float64(DefaultDailyCap), nil)

	// Try to award 100, but should only get 50
	repo.On("UpsertUserJob", ctx, mock.MatchedBy(func(uj *domain.UserJob) bool {
		return uj.XPGainedToday == 250 && uj.CurrentXP == 2050 // 2000 + 50
	})).Return(nil)
	repo.On("RecordJobXPEvent", ctx, mock.MatchedBy(func(e *domain.JobXPEvent) bool {
		return e.XPAmount == 50
	})).Return(nil)

	result, err := svc.AwardXP(ctx, "u1", JobKeyBlacksmith, 100, "test", nil)
	assert.NoError(t, err)
	assert.Equal(t, 50, result.XPGained) // Only 50 awarded
}

// XP Calculation Edge Cases

func TestGetXPForLevel_NegativeLevel(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)

	result := svc.GetXPForLevel(-5)
	assert.Equal(t, int64(0), result)
}

func TestGetXPForLevel_VeryHighLevel(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)

	// Should not panic or overflow
	result := svc.GetXPForLevel(50)
	assert.Greater(t, result, int64(0))
	// Just verify it calculates something reasonable
}

func TestCalculateLevel_VeryHighXP(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, nil)

	// Very high XP should still work and cap at iteration limit
	result := svc.CalculateLevel(10000000)
	assert.Greater(t, result, 0)
	assert.LessOrEqual(t, result, MaxIterationLevel)
}
