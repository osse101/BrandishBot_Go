package user

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// MockProgressionService for testing
type MockProgressionService struct {
	mock.Mock
}

func (m *MockProgressionService) GetModifiedValue(ctx context.Context, userID string, featureKey string, baseValue float64) (float64, error) {
	args := m.Called(ctx, userID, featureKey, baseValue)
	return args.Get(0).(float64), args.Error(1)
}

func TestUpgradeSearchQuality_ModifierApplied(t *testing.T) {
	// ARRANGE
	mockProg := new(MockProgressionService)
	svc, repo := createSearchTestService(func(opts *searchTestServiceOpts) {})

	// Manually inject progression service for test (field is not exported)
	s := svc
	s.progressionSvc = mockProg

	user := createTestUser()
	repo.users[TestUsername] = user
	ctx := context.Background()

	params := searchParams{
		dailyCount: 0, // Base index will be 4
	}

	// 4 base points * 1.5 multiplier = 6
	mockProg.On("GetModifiedValue", ctx, TestUserID, "search_quality", float64(4)).Return(6.0, nil)

	// ACT
	quality := s.calculateSearchQuality(ctx, TestUserID, false, params)

	// ASSERT
	assert.Equal(t, domain.QualityEpic, quality, "Index 6 should result in Epic quality")
	mockProg.AssertExpectations(t)
}

func TestUpgradeSearchQuality_FallbackOnError(t *testing.T) {
	// ARRANGE
	mockProg := new(MockProgressionService)
	svc, repo := createSearchTestService()
	s := svc
	s.progressionSvc = mockProg

	user := createTestUser()
	repo.users[TestUsername] = user
	ctx := context.Background()

	params := searchParams{
		dailyCount: 0, // Base index will be 4
	}

	mockProg.On("GetModifiedValue", ctx, TestUserID, "search_quality", float64(4)).Return(0.0, errors.New("error"))

	// ACT
	quality := s.calculateSearchQuality(ctx, TestUserID, false, params)

	// ASSERT
	assert.Equal(t, domain.QualityUncommon, quality, "Index 4 should result in Uncommon quality when modifier fails")
	mockProg.AssertExpectations(t)
}

func TestUpgradeSearchQuality_NilService(t *testing.T) {
	// ARRANGE
	svc, repo := createSearchTestService()
	s := svc
	s.progressionSvc = nil

	user := createTestUser()
	repo.users[TestUsername] = user
	ctx := context.Background()

	params := searchParams{
		dailyCount: 0, // Base index will be 4
	}

	// ACT
	quality := s.calculateSearchQuality(ctx, TestUserID, false, params)

	// ASSERT
	assert.Equal(t, domain.QualityUncommon, quality, "Index 4 should result in Uncommon quality without modifier")
}
