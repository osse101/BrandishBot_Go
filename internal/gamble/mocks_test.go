package gamble

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// MockRepository
type MockRepository struct {
	mock.Mock
}

// GetItemByName implements [repository.Gamble].
func (m *MockRepository) GetItemByName(ctx context.Context, name string) (*domain.Item, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Item), args.Error(1)
}

func (m *MockRepository) CreateGamble(ctx context.Context, gamble *domain.Gamble) error {
	args := m.Called(ctx, gamble)
	return args.Error(0)
}

func (m *MockRepository) GetGamble(ctx context.Context, id uuid.UUID) (*domain.Gamble, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Gamble), args.Error(1)
}

func (m *MockRepository) JoinGamble(ctx context.Context, participant *domain.Participant) error {
	args := m.Called(ctx, participant)
	return args.Error(0)
}

func (m *MockRepository) UpdateGambleState(ctx context.Context, id uuid.UUID, state domain.GambleState) error {
	args := m.Called(ctx, id, state)
	return args.Error(0)
}

func (m *MockRepository) UpdateGambleStateIfMatches(ctx context.Context, id uuid.UUID, expectedState, newState domain.GambleState) (int64, error) {
	args := m.Called(ctx, id, expectedState, newState)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockRepository) SaveOpenedItems(ctx context.Context, items []domain.GambleOpenedItem) error {
	args := m.Called(ctx, items)
	return args.Error(0)
}

func (m *MockRepository) CompleteGamble(ctx context.Context, result *domain.GambleResult) error {
	args := m.Called(ctx, result)
	return args.Error(0)
}

func (m *MockRepository) GetActiveGamble(ctx context.Context) (*domain.Gamble, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Gamble), args.Error(1)
}

func (m *MockRepository) BeginGambleTx(ctx context.Context) (repository.GambleTx, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(repository.GambleTx), args.Error(1)
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

func (m *MockRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	args := m.Called(ctx, platform, platformID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Item), args.Error(1)
}

// MockLootboxService
type MockLootboxService struct {
	mock.Mock
}

func (m *MockLootboxService) OpenLootbox(ctx context.Context, lootboxName string, quantity int, boxQuality domain.QualityLevel) ([]lootbox.DroppedItem, error) {
	args := m.Called(ctx, lootboxName, quantity, boxQuality)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]lootbox.DroppedItem), args.Error(1)
}

// MockTx
type MockTx struct {
	mock.Mock
}

func (m *MockTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
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

func (m *MockTx) UpdateGambleStateIfMatches(ctx context.Context, id uuid.UUID, expectedState, newState domain.GambleState) (int64, error) {
	args := m.Called(ctx, id, expectedState, newState)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTx) SaveOpenedItems(ctx context.Context, items []domain.GambleOpenedItem) error {
	args := m.Called(ctx, items)
	return args.Error(0)
}

func (m *MockTx) CompleteGamble(ctx context.Context, result *domain.GambleResult) error {
	args := m.Called(ctx, result)
	return args.Error(0)
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

// MockJobService
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

// MockEventBus
type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) Publish(ctx context.Context, event event.Event) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventBus) Subscribe(eventType event.Type, handler event.Handler) {
	m.Called(eventType, handler)
}

// MockResilientPublisher
type MockResilientPublisher struct {
	mock.Mock
}

func (m *MockResilientPublisher) PublishWithRetry(ctx context.Context, event event.Event) {
	m.Called(ctx, event)
}

// MockNamingResolver
type MockNamingResolver struct {
	mock.Mock
}

func (m *MockNamingResolver) ResolvePublicName(publicName string) (string, bool) {
	args := m.Called(publicName)
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

// MockProgressionService
type MockProgressionService struct {
	mock.Mock
}

func (m *MockProgressionService) GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error) {
	args := m.Called(ctx, featureKey, baseValue)
	return args.Get(0).(float64), args.Error(1)
}
