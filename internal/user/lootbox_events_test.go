package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
)

// MockStatsServiceForLootboxTests - distinct name to avoid conflicts if any
type MockStatsServiceForLootboxTests struct {
	mock.Mock
}

func (m *MockStatsServiceForLootboxTests) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, data map[string]interface{}) error {
	args := m.Called(ctx, userID, eventType, data)
	return args.Error(0)
}

func (m *MockStatsServiceForLootboxTests) GetUserStats(ctx context.Context, userID string, period string) (*domain.StatsSummary, error) {
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

func (m *MockStatsServiceForLootboxTests) GetTotalMetric(ctx context.Context, userID string, metric string) (float64, error) {
	args := m.Called(ctx, userID, metric)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockStatsServiceForLootboxTests) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) {
	args := m.Called(ctx, period)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.StatsSummary), args.Error(1)
}

// MockLootboxServiceForLootboxTests
type MockLootboxServiceForLootboxTests struct {
	mock.Mock
}

func (m *MockLootboxServiceForLootboxTests) OpenLootbox(ctx context.Context, lootboxName string, quantity int, boxShine domain.ShineLevel) ([]lootbox.DroppedItem, error) {
	args := m.Called(ctx, lootboxName, quantity, boxShine)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]lootbox.DroppedItem), args.Error(1)
}

// MockNamingResolverForLootboxTests - using testify/mock
type MockNamingResolverForLootboxTests struct {
	mock.Mock
}

func (m *MockNamingResolverForLootboxTests) ResolvePublicName(publicName string) (string, bool) {
	args := m.Called(publicName)
	return args.String(0), args.Bool(1)
}

func (m *MockNamingResolverForLootboxTests) GetDisplayName(internalName string, shineLevel domain.ShineLevel) string {
	args := m.Called(internalName, shineLevel)
	return args.String(0)
}

func (m *MockNamingResolverForLootboxTests) GetActiveTheme() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockNamingResolverForLootboxTests) Reload() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockNamingResolverForLootboxTests) RegisterItem(internalName, publicName string) {
	m.Called(internalName, publicName)
}

func TestProcessLootboxDrops_JackpotEvents(t *testing.T) {
	// Test data
	user := &domain.User{
		ID:       "user-123",
		Username: "testuser",
	}
	lootboxItem := &domain.Item{
		ID:           1,
		InternalName: "lootbox_tier1",
		BaseValue:    10,
	}

	ctx := context.Background()

	// Test Case 1: Legendary Drop (Jackpot)
	t.Run("Records Jackpot Event on Legendary Drop", func(t *testing.T) {
		// Create fresh mocks for each test to ensure isolation
		mockStats := new(MockStatsServiceForLootboxTests)
		mockLootbox := new(MockLootboxServiceForLootboxTests)
		mockNaming := new(MockNamingResolverForLootboxTests)
		mockRepo := NewFakeRepository()
		svc := NewService(mockRepo, mockRepo, mockStats, nil, mockLootbox, mockNaming, nil, nil, nil, false).(*service)
		inventory := &domain.Inventory{Slots: []domain.InventorySlot{}}

		// Prepare drops
		drops := []lootbox.DroppedItem{
			{
				ItemID:     101,
				ItemName:   "legendary_sword",
				Quantity:   1,
				Value:      1000,
				ShineLevel: domain.ShineLegendary,
			},
		}

		// Expectations
		mockNaming.On("GetDisplayName", "legendary_sword", domain.ShineLegendary).Return("Legendary Sword")
		mockNaming.On("GetDisplayName", "lootbox_tier1", domain.ShineLevel("")).Return("Lootbox Tier 1")

		// Expect stats service to be called with EventLootboxJackpot
		mockStats.On("RecordUserEvent",
			mock.Anything,
			user.ID,
			domain.EventLootboxJackpot,
			mock.MatchedBy(func(data map[string]interface{}) bool {
				return data["source"] == "lootbox" && data["item"] == "lootbox_tier1"
			}),
		).Return(nil).Once()

		// Execute
		msg, err := svc.processLootboxDrops(ctx, user, inventory, lootboxItem, 1, drops)

		// Verify
		assert.NoError(t, err)
		assert.Contains(t, msg, "JACKPOT!")
		mockStats.AssertExpectations(t)
	})

	// Test Case 2: Epic Drop (Big Win)
	t.Run("Records Big Win Event on Epic Drop", func(t *testing.T) {
		// Create fresh mocks
		mockStats := new(MockStatsServiceForLootboxTests)
		mockLootbox := new(MockLootboxServiceForLootboxTests)
		mockNaming := new(MockNamingResolverForLootboxTests)
		mockRepo := NewFakeRepository()
		svc := NewService(mockRepo, mockRepo, mockStats, nil, mockLootbox, mockNaming, nil, nil, nil, false).(*service)
		inventory := &domain.Inventory{Slots: []domain.InventorySlot{}}

		// Prepare drops
		drops := []lootbox.DroppedItem{
			{
				ItemID:     102,
				ItemName:   "epic_shield",
				Quantity:   1,
				Value:      500,
				ShineLevel: domain.ShineEpic,
			},
		}

		// Expectations
		mockNaming.On("GetDisplayName", "epic_shield", domain.ShineEpic).Return("Epic Shield")
		mockNaming.On("GetDisplayName", "lootbox_tier1", domain.ShineLevel("")).Return("Lootbox Tier 1")

		// Expect stats service to be called with EventLootboxBigWin
		mockStats.On("RecordUserEvent",
			mock.Anything,
			user.ID,
			domain.EventLootboxBigWin,
			mock.MatchedBy(func(data map[string]interface{}) bool {
				return data["source"] == "lootbox" && data["item"] == "lootbox_tier1"
			}),
		).Return(nil).Once()

		// Execute
		msg, err := svc.processLootboxDrops(ctx, user, inventory, lootboxItem, 1, drops)

		// Verify
		assert.NoError(t, err)
		assert.Contains(t, msg, "BIG WIN!")
		mockStats.AssertExpectations(t)
	})

	// Test Case 3: Common Drop (No Special Event)
	t.Run("Does Not Record Event for Common Drops", func(t *testing.T) {
		// Create fresh mocks
		mockStats := new(MockStatsServiceForLootboxTests)
		mockLootbox := new(MockLootboxServiceForLootboxTests)
		mockNaming := new(MockNamingResolverForLootboxTests)
		mockRepo := NewFakeRepository()
		svc := NewService(mockRepo, mockRepo, mockStats, nil, mockLootbox, mockNaming, nil, nil, nil, false).(*service)
		inventory := &domain.Inventory{Slots: []domain.InventorySlot{}}

		// Prepare drops
		drops := []lootbox.DroppedItem{
			{
				ItemID:     103,
				ItemName:   "common_rock",
				Quantity:   1,
				Value:      5,
				ShineLevel: domain.ShineCommon,
			},
		}

		// Expectations
		mockNaming.On("GetDisplayName", "common_rock", domain.ShineCommon).Return("Rock")
		mockNaming.On("GetDisplayName", "lootbox_tier1", domain.ShineLevel("")).Return("Lootbox Tier 1")

		// Execute
		msg, err := svc.processLootboxDrops(ctx, user, inventory, lootboxItem, 1, drops)

		// Verify
		assert.NoError(t, err)
		assert.NotContains(t, msg, "JACKPOT!")
		assert.NotContains(t, msg, "BIG WIN!")
		mockStats.AssertNotCalled(t, "RecordUserEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

// TestProcessLootboxDrops_BulkFeedbackThreshold tests the boundary conditions
// for the "Nice haul!" bulk feedback message per TEST_GUIDANCE.md 5-case model
func TestProcessLootboxDrops_BulkFeedbackThreshold(t *testing.T) {
	// BulkFeedbackThreshold = 5 (defined in item_handlers.go)
	// "Nice haul!" appears when: quantity >= BulkFeedbackThreshold AND no legendary/epic drops

	user := &domain.User{
		ID:       "user-123",
		Username: "testuser",
	}
	lootboxItem := &domain.Item{
		ID:           1,
		InternalName: "lootbox_tier1",
		BaseValue:    10,
	}

	ctx := context.Background()

	// Common drops (no epic/legendary) for bulk feedback testing
	createCommonDrops := func() []lootbox.DroppedItem {
		return []lootbox.DroppedItem{
			{
				ItemID:     103,
				ItemName:   "common_rock",
				Quantity:   1,
				Value:      5,
				ShineLevel: domain.ShineCommon,
			},
		}
	}

	// Test Case 1: Just Below Threshold (4)
	t.Run("No Bulk Feedback Below Threshold (quantity=4)", func(t *testing.T) {
		mockStats := new(MockStatsServiceForLootboxTests)
		mockNaming := new(MockNamingResolverForLootboxTests)
		mockRepo := NewFakeRepository()
		svc := NewService(mockRepo, mockRepo, mockStats, nil, nil, mockNaming, nil, nil, nil, false).(*service)
		inventory := &domain.Inventory{Slots: []domain.InventorySlot{}}

		mockNaming.On("GetDisplayName", "common_rock", domain.ShineCommon).Return("Rock")
		mockNaming.On("GetDisplayName", "lootbox_tier1", domain.ShineLevel("")).Return("Lootbox Tier 1")

		msg, err := svc.processLootboxDrops(ctx, user, inventory, lootboxItem, 4, createCommonDrops())

		assert.NoError(t, err)
		assert.NotContains(t, msg, "Nice haul!", "Should NOT show bulk feedback below threshold")
	})

	// Test Case 2: Exactly At Threshold (5)
	t.Run("Bulk Feedback At Threshold (quantity=5)", func(t *testing.T) {
		mockStats := new(MockStatsServiceForLootboxTests)
		mockNaming := new(MockNamingResolverForLootboxTests)
		mockRepo := NewFakeRepository()
		svc := NewService(mockRepo, mockRepo, mockStats, nil, nil, mockNaming, nil, nil, nil, false).(*service)
		inventory := &domain.Inventory{Slots: []domain.InventorySlot{}}

		mockNaming.On("GetDisplayName", "common_rock", domain.ShineCommon).Return("Rock")
		mockNaming.On("GetDisplayName", "lootbox_tier1", domain.ShineLevel("")).Return("Lootbox Tier 1")

		msg, err := svc.processLootboxDrops(ctx, user, inventory, lootboxItem, 5, createCommonDrops())

		assert.NoError(t, err)
		assert.Contains(t, msg, "Nice haul!", "Should show bulk feedback at threshold")
	})

	// Test Case 3: Just Above Threshold (6)
	t.Run("Bulk Feedback Above Threshold (quantity=6)", func(t *testing.T) {
		mockStats := new(MockStatsServiceForLootboxTests)
		mockNaming := new(MockNamingResolverForLootboxTests)
		mockRepo := NewFakeRepository()
		svc := NewService(mockRepo, mockRepo, mockStats, nil, nil, mockNaming, nil, nil, nil, false).(*service)
		inventory := &domain.Inventory{Slots: []domain.InventorySlot{}}

		mockNaming.On("GetDisplayName", "common_rock", domain.ShineCommon).Return("Rock")
		mockNaming.On("GetDisplayName", "lootbox_tier1", domain.ShineLevel("")).Return("Lootbox Tier 1")

		msg, err := svc.processLootboxDrops(ctx, user, inventory, lootboxItem, 6, createCommonDrops())

		assert.NoError(t, err)
		assert.Contains(t, msg, "Nice haul!", "Should show bulk feedback above threshold")
	})

	// Test Case 4: No Bulk Feedback When Jackpot (Legendary takes precedence)
	t.Run("Jackpot Takes Precedence Over Bulk Feedback", func(t *testing.T) {
		mockStats := new(MockStatsServiceForLootboxTests)
		mockNaming := new(MockNamingResolverForLootboxTests)
		mockRepo := NewFakeRepository()
		svc := NewService(mockRepo, mockRepo, mockStats, nil, nil, mockNaming, nil, nil, nil, false).(*service)
		inventory := &domain.Inventory{Slots: []domain.InventorySlot{}}

		legendaryDrops := []lootbox.DroppedItem{
			{
				ItemID:     101,
				ItemName:   "legendary_sword",
				Quantity:   1,
				Value:      1000,
				ShineLevel: domain.ShineLegendary,
			},
		}

		mockNaming.On("GetDisplayName", "legendary_sword", domain.ShineLegendary).Return("Legendary Sword")
		mockNaming.On("GetDisplayName", "lootbox_tier1", domain.ShineLevel("")).Return("Lootbox Tier 1")
		mockStats.On("RecordUserEvent", mock.Anything, user.ID, domain.EventLootboxJackpot, mock.Anything).Return(nil)

		msg, err := svc.processLootboxDrops(ctx, user, inventory, lootboxItem, 10, legendaryDrops)

		assert.NoError(t, err)
		assert.Contains(t, msg, "JACKPOT!", "Jackpot should appear")
		assert.NotContains(t, msg, "Nice haul!", "Bulk feedback should NOT appear when jackpot")
	})

	// Test Case 5: No Bulk Feedback When Big Win (Epic takes precedence)
	t.Run("Big Win Takes Precedence Over Bulk Feedback", func(t *testing.T) {
		mockStats := new(MockStatsServiceForLootboxTests)
		mockNaming := new(MockNamingResolverForLootboxTests)
		mockRepo := NewFakeRepository()
		svc := NewService(mockRepo, mockRepo, mockStats, nil, nil, mockNaming, nil, nil, nil, false).(*service)
		inventory := &domain.Inventory{Slots: []domain.InventorySlot{}}

		epicDrops := []lootbox.DroppedItem{
			{
				ItemID:     102,
				ItemName:   "epic_shield",
				Quantity:   1,
				Value:      500,
				ShineLevel: domain.ShineEpic,
			},
		}

		mockNaming.On("GetDisplayName", "epic_shield", domain.ShineEpic).Return("Epic Shield")
		mockNaming.On("GetDisplayName", "lootbox_tier1", domain.ShineLevel("")).Return("Lootbox Tier 1")
		mockStats.On("RecordUserEvent", mock.Anything, user.ID, domain.EventLootboxBigWin, mock.Anything).Return(nil)

		msg, err := svc.processLootboxDrops(ctx, user, inventory, lootboxItem, 10, epicDrops)

		assert.NoError(t, err)
		assert.Contains(t, msg, "BIG WIN!", "Big win should appear")
		assert.NotContains(t, msg, "Nice haul!", "Bulk feedback should NOT appear when big win")
	})
}
