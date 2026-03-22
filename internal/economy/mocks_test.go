package economy

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// MockRepository implements repository.Economy for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	args := m.Called(ctx, platform, platformID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockRepository) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	args := m.Called(ctx, itemName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Item), args.Error(1)
}

func (m *MockRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Inventory), args.Error(1)
}

func (m *MockRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	args := m.Called(ctx, userID, inventory)
	return args.Error(0)
}

func (m *MockRepository) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Item), args.Error(1)
}

func (m *MockRepository) IsItemBuyable(ctx context.Context, itemName string) (bool, error) {
	args := m.Called(ctx, itemName)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) BeginTx(ctx context.Context) (repository.EconomyTx, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(repository.EconomyTx), args.Error(1)
}

func (m *MockRepository) GetBuyablePrices(ctx context.Context) ([]domain.Item, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Item), args.Error(1)
}

// MockTx implements repository.EconomyTx for testing
type MockTx struct {
	mock.Mock
}

func (m *MockTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	// Allow passing a generator function for concurrent tests to avoid race conditions
	if fn, ok := args.Get(0).(func(context.Context, string) *domain.Inventory); ok {
		return fn(ctx, userID), args.Error(1)
	}

	return args.Get(0).(*domain.Inventory), args.Error(1)
}

func (m *MockTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	args := m.Called(ctx, userID, inventory)
	return args.Error(0)
}

func (m *MockTx) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTx) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Ensure MockTx implements repository.EconomyTx
var _ repository.EconomyTx = (*MockTx)(nil)

// MockJobService implements JobService for testing
type MockJobService struct {
	mock.Mock
}

func (m *MockJobService) AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata domain.JobXPMetadata) (*domain.XPAwardResult, error) {
	args := m.Called(ctx, userID, jobKey, baseAmount, source, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.XPAwardResult), args.Error(1)
}

func (m *MockJobService) AwardXPByPlatform(ctx context.Context, platform, platformID, jobKey string, baseAmount int, source string, metadata domain.JobXPMetadata) (*domain.XPAwardResult, error) {
	args := m.Called(ctx, platform, platformID, jobKey, baseAmount, source, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.XPAwardResult), args.Error(1)
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
func (m *MockJobService) IsJobFeatureUnlocked(ctx context.Context, userID, featureKey string) (bool, error) {
	return true, nil
}
func (m *MockJobService) GetJobLevel(ctx context.Context, userID, jobKey string) (int, error) {
	return 0, nil
}
func (m *MockJobService) ResetDailyJobXP(ctx context.Context) (int64, error) { return 0, nil }
func (m *MockJobService) GetDailyResetStatus(ctx context.Context) (*domain.DailyResetStatus, error) {
	return nil, nil
}
func (m *MockJobService) GetAllJobs(ctx context.Context) ([]domain.Job, error) { return nil, nil }
func (m *MockJobService) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return nil, nil
}
func (m *MockJobService) CalculateLevel(totalXP int64) int { return 0 }
func (m *MockJobService) GetXPForLevel(level int) int64    { return 0 }
func (m *MockJobService) GetXPProgress(currentXP int64) (int, int64, int64, int64) {
	return 0, 0, 0, 0
}
func (m *MockJobService) Shutdown(ctx context.Context) error { return nil }

// MockProgressionService implements ProgressionService for testing
type MockProgressionService struct {
	mock.Mock
}

// IsFeatureUnlocked implements [ProgressionService].
func (m *MockProgressionService) IsFeatureUnlocked(ctx context.Context, featureKey string) (bool, error) {
	args := m.Called(ctx, featureKey)
	return args.Bool(0), args.Error(1)
}

func (m *MockProgressionService) IsItemUnlocked(ctx context.Context, itemName string) (bool, error) {
	args := m.Called(ctx, itemName)
	return args.Bool(0), args.Error(1)
}

func (m *MockProgressionService) AreItemsUnlocked(ctx context.Context, itemNames []string) (map[string]bool, error) {
	args := m.Called(ctx, itemNames)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]bool), args.Error(1)
}

func (m *MockProgressionService) GetModifiedValue(ctx context.Context, userID string, featureKey string, baseValue float64) (float64, error) {
	args := m.Called(ctx, featureKey, baseValue)
	return args.Get(0).(float64), args.Error(1)
}

// MockNamingResolver implements naming.Resolver for testing
type MockNamingResolver struct {
	mock.Mock
}

func (m *MockNamingResolver) ResolvePublicName(publicName string) (string, bool) {
	args := m.Called(publicName)
	return args.String(0), args.Bool(1)
}

func (m *MockNamingResolver) ResolveInternalName(internalName string) (string, bool) {
	args := m.Called(internalName)
	return args.String(0), args.Bool(1)
}

func (m *MockNamingResolver) GetDisplayName(internalName string, qualityLevel domain.QualityLevel) string {
	args := m.Called(internalName, qualityLevel)
	return args.String(0)
}

func (m *MockNamingResolver) GetActiveTheme() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockNamingResolver) Reload() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockNamingResolver) RegisterItem(internalName, publicName string) {
	m.Called(internalName, publicName)
}
