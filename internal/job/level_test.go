package job

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestGetXPProgress(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, false)

	// Helper to calculate XP for a specific level to ensure we test accurate boundaries
	xpForLevel := func(lvl int) int64 {
		cumulative := int64(0)
		for i := 1; i <= lvl; i++ {
			cumulative += int64(BaseXP * math.Pow(float64(i), LevelExponent))
		}
		return cumulative
	}

	tests := []struct {
		name             string
		currentXP        int64
		expectedLevel    int
		expectedLevelXP  int64
		expectedLevelReq int64
		expectedXPToNext int64
	}{
		{
			name:             "Zero XP",
			currentXP:        0,
			expectedLevel:    0,
			expectedLevelXP:  0,
			expectedLevelReq: int64(BaseXP),
			expectedXPToNext: int64(BaseXP),
		},
		{
			name:             "Halfway to Level 1",
			currentXP:        xpForLevel(1) / 2,
			expectedLevel:    0,
			expectedLevelXP:  xpForLevel(1) / 2,
			expectedLevelReq: int64(BaseXP),
			expectedXPToNext: int64(BaseXP) - (xpForLevel(1) / 2),
		},
		{
			name:             "Exact Level 1",
			currentXP:        xpForLevel(1),
			expectedLevel:    1,
			expectedLevelXP:  0,
			expectedLevelReq: xpForLevel(2) - xpForLevel(1),
			expectedXPToNext: xpForLevel(2) - xpForLevel(1),
		},
		{
			name:             "Level 2 halfway",
			currentXP:        xpForLevel(2) + (xpForLevel(3)-xpForLevel(2))/2,
			expectedLevel:    2,
			expectedLevelXP:  (xpForLevel(3) - xpForLevel(2)) / 2,
			expectedLevelReq: xpForLevel(3) - xpForLevel(2),
			expectedXPToNext: (xpForLevel(3) - xpForLevel(2)) - ((xpForLevel(3) - xpForLevel(2)) / 2),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentLevel, levelXP, levelRequirement, xpToNext := svc.GetXPProgress(tt.currentXP)
			assert.Equal(t, tt.expectedLevel, currentLevel, "Current Level mismatch")
			assert.Equal(t, tt.expectedLevelXP, levelXP, "Level XP mismatch")
			assert.Equal(t, tt.expectedLevelReq, levelRequirement, "Level Requirement mismatch")
			assert.Equal(t, tt.expectedXPToNext, xpToNext, "XP to Next mismatch")
		})
	}
}

func TestGetXPProgress_NegativeXP(t *testing.T) {
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, false)

	currentLevel, levelXP, levelRequirement, xpToNext := svc.GetXPProgress(-100)
	assert.Equal(t, 0, currentLevel)
	assert.Equal(t, int64(-100), levelXP) // Let's see what the actual value should be
	assert.Equal(t, int64(BaseXP), levelRequirement)
	assert.Equal(t, int64(350), xpToNext) // BaseXP + 100
}

func TestIsJobFeatureUnlocked(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepository)
	prog := new(MockProgressionService)
	svc := NewService(repo, prog, nil, nil, false)
	userID := "test-user"
	featureKey := "test-feature"

	t.Run("Config not found returns false", func(t *testing.T) {
		prog.On("GetJobUnlockConfig", ctx, featureKey).Return((*domain.JobUnlockConfig)(nil), nil).Once()

		unlocked, err := svc.IsJobFeatureUnlocked(ctx, userID, featureKey)

		assert.NoError(t, err)
		assert.False(t, unlocked)
		prog.AssertExpectations(t)
	})

	t.Run("Config fetch error returns false and error", func(t *testing.T) {
		expectedErr := errors.New("db error")
		prog.On("GetJobUnlockConfig", ctx, featureKey).Return((*domain.JobUnlockConfig)(nil), expectedErr).Once()

		unlocked, err := svc.IsJobFeatureUnlocked(ctx, userID, featureKey)

		assert.Error(t, err)
		assert.False(t, unlocked)
	})

	t.Run("Level sufficient returns true", func(t *testing.T) {
		config := &domain.JobUnlockConfig{
			JobKey:        "warrior",
			RequiredLevel: 10,
		}
		userJob := &domain.UserJob{
			CurrentXP:    100000, // Enough for level 10
			CurrentLevel: 10,     // Explicitly mock level 10 for get job level
		}
		job := &domain.Job{ID: 1, JobKey: "warrior"}

		prog.On("GetJobUnlockConfig", ctx, featureKey).Return(config, nil).Once()
		repo.On("GetJobByKey", ctx, "warrior").Return(job, nil).Once()
		repo.On("GetUserJob", ctx, userID, 1).Return(userJob, nil).Once()

		unlocked, err := svc.IsJobFeatureUnlocked(ctx, userID, featureKey)

		assert.NoError(t, err)
		assert.True(t, unlocked)
		prog.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("Level insufficient returns false", func(t *testing.T) {
		config := &domain.JobUnlockConfig{
			JobKey:        "warrior",
			RequiredLevel: 10,
		}
		userJob := &domain.UserJob{
			CurrentXP:    0, // Level 0
			CurrentLevel: 0,
		}
		job := &domain.Job{ID: 1, JobKey: "warrior"}

		prog.On("GetJobUnlockConfig", ctx, featureKey).Return(config, nil).Once()
		repo.On("GetJobByKey", ctx, "warrior").Return(job, nil).Once()
		repo.On("GetUserJob", ctx, userID, 1).Return(userJob, nil).Once()

		unlocked, err := svc.IsJobFeatureUnlocked(ctx, userID, featureKey)

		assert.NoError(t, err)
		assert.False(t, unlocked)
		prog.AssertExpectations(t)
		repo.AssertExpectations(t)
	})

	t.Run("GetJobLevel error returns false and error", func(t *testing.T) {
		config := &domain.JobUnlockConfig{
			JobKey:        "warrior",
			RequiredLevel: 10,
		}

		prog.On("GetJobUnlockConfig", ctx, featureKey).Return(config, nil).Once()
		repo.On("GetJobByKey", ctx, "warrior").Return((*domain.Job)(nil), errors.New("db error")).Once()

		unlocked, err := svc.IsJobFeatureUnlocked(ctx, userID, featureKey)

		assert.Error(t, err)
		assert.False(t, unlocked)
	})
}
