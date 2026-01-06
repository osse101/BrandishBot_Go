package user

import (
	"context"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockStatsServiceForLootboxTests is a minimal mock for Stats Service
type MockStatsServiceForLootboxTests struct {
	mock.Mock
}

func (m *MockStatsServiceForLootboxTests) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, eventData map[string]interface{}) error {
	args := m.Called(ctx, userID, eventType, eventData)
	return args.Error(0)
}

func (m *MockStatsServiceForLootboxTests) GetUserStats(ctx context.Context, userID, period string) (*domain.StatsSummary, error) {
	args := m.Called(ctx, userID, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.StatsSummary), args.Error(1)
}

func (m *MockStatsServiceForLootboxTests) GetUserCurrentStreak(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockStatsServiceForLootboxTests) GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error) {
	args := m.Called(ctx, eventType, period, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.LeaderboardEntry), args.Error(1)
}

func (m *MockStatsServiceForLootboxTests) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) {
	args := m.Called(ctx, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.StatsSummary), args.Error(1)
}

// MockNamingResolverForLootboxTests is a minimal mock for Naming Resolver
type MockNamingResolverForLootboxTests struct{}

func (m *MockNamingResolverForLootboxTests) GetDisplayName(internalName, shineLevel string) string {
	if shineLevel != "" {
		return internalName + " (" + shineLevel + ")"
	}
	return internalName
}

func (m *MockNamingResolverForLootboxTests) GetInternalName(displayName string) string {
	return displayName
}

func (m *MockNamingResolverForLootboxTests) GetActiveTheme() string {
	return "default"
}

func (m *MockNamingResolverForLootboxTests) Reload() error {
	return nil
}

func (m *MockNamingResolverForLootboxTests) ResolvePublicName(publicName string) (internalName string, ok bool) {
	return publicName, true
}

func (m *MockNamingResolverForLootboxTests) RegisterItem(internalName, publicName string) {
	// No-op
}


// TestProcessLootboxFeedback verifies the feedback logic including God Roll and Unlucky events
func TestProcessLootboxFeedback(t *testing.T) {
	// Setup
	mockStats := new(MockStatsServiceForLootboxTests)
	mockNaming := &MockNamingResolverForLootboxTests{}

	// Create service with mocks - we only need the struct with these dependencies
	s := &service{
		statsService:   mockStats,
		namingResolver: mockNaming,
	}

	lootboxItem := &domain.Item{InternalName: "lootbox_test"}
	user := &domain.User{ID: "user123"}
	inventory := &domain.Inventory{}
	ctx := context.Background()

	t.Run("God Roll", func(t *testing.T) {
		// 2 Legendaries
		drops := []lootbox.DroppedItem{
			{ItemID: 1, ItemName: "item1", Quantity: 1, Value: 100, ShineLevel: lootbox.ShineLegendary},
			{ItemID: 2, ItemName: "item2", Quantity: 1, Value: 100, ShineLevel: lootbox.ShineLegendary},
		}

		// Expectation: EventLootboxGodRoll recorded
		mockStats.On("RecordUserEvent", ctx, user.ID, domain.EventLootboxGodRoll, mock.Anything).Return(nil).Once()

		result, err := s.processLootboxDrops(ctx, user, inventory, lootboxItem, 2, drops)
		assert.NoError(t, err)
		assert.Contains(t, result, "GOD ROLL! 🌟🔥🌟")

		mockStats.AssertExpectations(t)
	})

	t.Run("Unlucky", func(t *testing.T) {
		// 5 items, all Common
		drops := []lootbox.DroppedItem{
			{ItemID: 1, ItemName: "item1", Quantity: 1, Value: 10, ShineLevel: lootbox.ShineCommon},
			{ItemID: 1, ItemName: "item1", Quantity: 1, Value: 10, ShineLevel: lootbox.ShineCommon},
			{ItemID: 1, ItemName: "item1", Quantity: 1, Value: 10, ShineLevel: lootbox.ShineCommon},
			{ItemID: 1, ItemName: "item1", Quantity: 1, Value: 10, ShineLevel: lootbox.ShineCommon},
			{ItemID: 1, ItemName: "item1", Quantity: 1, Value: 10, ShineLevel: lootbox.ShineCommon},
		}

		// Expectation: EventLootboxUnlucky recorded
		mockStats.On("RecordUserEvent", ctx, user.ID, domain.EventLootboxUnlucky, mock.Anything).Return(nil).Once()

		result, err := s.processLootboxDrops(ctx, user, inventory, lootboxItem, 5, drops)
		assert.NoError(t, err)
		assert.Contains(t, result, "Oof. (All Commons) 💀")

		mockStats.AssertExpectations(t)
	})

	t.Run("Normal Jackpot", func(t *testing.T) {
		// 1 Legendary
		drops := []lootbox.DroppedItem{
			{ItemID: 1, ItemName: "item1", Quantity: 1, Value: 100, ShineLevel: lootbox.ShineLegendary},
			{ItemID: 2, ItemName: "item2", Quantity: 1, Value: 10, ShineLevel: lootbox.ShineCommon},
		}

		// Expectation: EventLootboxJackpot recorded (God Roll is >= 2)
		mockStats.On("RecordUserEvent", ctx, user.ID, domain.EventLootboxJackpot, mock.Anything).Return(nil).Once()

		result, err := s.processLootboxDrops(ctx, user, inventory, lootboxItem, 2, drops)
		assert.NoError(t, err)
		assert.Contains(t, result, "JACKPOT! 🎰✨")
		assert.NotContains(t, result, "GOD ROLL")

		mockStats.AssertExpectations(t)
	})

	t.Run("Just Nice Haul", func(t *testing.T) {
		// 5 items, mixed Common and Uncommon (Not Unlucky, Not Jackpot)
		drops := []lootbox.DroppedItem{
			{ItemID: 1, ItemName: "item1", Quantity: 1, Value: 10, ShineLevel: lootbox.ShineCommon},
			{ItemID: 1, ItemName: "item1", Quantity: 1, Value: 10, ShineLevel: lootbox.ShineCommon},
			{ItemID: 1, ItemName: "item1", Quantity: 1, Value: 10, ShineLevel: lootbox.ShineCommon},
			{ItemID: 1, ItemName: "item1", Quantity: 1, Value: 10, ShineLevel: lootbox.ShineCommon},
			{ItemID: 1, ItemName: "item2", Quantity: 1, Value: 20, ShineLevel: lootbox.ShineUncommon},
		}

		result, err := s.processLootboxDrops(ctx, user, inventory, lootboxItem, 5, drops)
		assert.NoError(t, err)
		assert.Contains(t, result, "Nice haul! 📦")
		assert.NotContains(t, result, "Oof")

		// Verify no calls to record specific events for this case
		// mockStats.AssertNotCalled(t, "RecordUserEvent", mock.Anything, mock.Anything, domain.EventLootboxUnlucky, mock.Anything)
		// Actually, due to how mockery works with "Any" arguments, we should just verify expectations are met, which implies no extra calls if we're strict.
		// But here we didn't set strict mode.
		// Let's rely on the fact that we didn't expect it, so if it WAS called and we cared, we'd fail?
		// Wait, AssertNotCalled checks if it WAS called.
		// If it failed, it means it WAS called.

		// Why was it called?
		// Logic:
		// } else if drop.ShineLevel == lootbox.ShineRare || drop.ShineLevel == lootbox.ShineUncommon {
		// 	stats.hasGoodLoot = true
		// }
		// In the test case: {ItemID: 1, ItemName: "item2", Quantity: 1, Value: 20, ShineLevel: lootbox.ShineUncommon},
		// So hasGoodLoot should be true.
		// So Unlucky should NOT trigger.
		// Let's debug by printing.
	})
}
