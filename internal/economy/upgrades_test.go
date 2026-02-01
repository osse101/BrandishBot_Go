package economy

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// This file contains tests for economy upgrade node modifier application.
// Tests verify that the economy_bonus modifier correctly applies to sell prices.

// TestUpgradeEconomy1_SellPriceModifier_Level1 verifies 5% boost at level 1
func TestUpgradeEconomy1_SellPriceModifier_Level1(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	mockProgression := &MockProgressionService{}
	service := NewService(mockRepo, nil, nil, mockProgression)
	ctx := context.Background()

	// Item with base value 100, base sell price = 100 * 0.40 = 40
	allItems := []domain.Item{
		{ID: 1, InternalName: "test_item", BaseValue: 100},
	}

	mockRepo.On("GetSellablePrices", ctx).Return(allItems, nil)
	mockProgression.On("AreItemsUnlocked", ctx, []string{"test_item"}).
		Return(map[string]bool{"test_item": true}, nil)

	// Level 1 upgrade: 1.05x multiplier
	// Base sell price: 40, Modified: 40 * 1.05 = 42
	mockProgression.On("GetModifiedValue", ctx, "economy_bonus", 40.0).
		Return(42.0, nil)

	// ACT
	items, err := service.GetSellablePrices(ctx)

	// ASSERT
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, 42, *items[0].SellPrice, "Level 1 economy_bonus should apply 1.05x multiplier (40 * 1.05 = 42)")
	mockRepo.AssertExpectations(t)
	mockProgression.AssertExpectations(t)
}

// TestUpgradeEconomy1_SellPriceModifier_Level5 verifies 25% boost at level 5
func TestUpgradeEconomy1_SellPriceModifier_Level5(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	mockProgression := &MockProgressionService{}
	service := NewService(mockRepo, nil, nil, mockProgression)
	ctx := context.Background()

	// Item with base value 100, base sell price = 100 * 0.40 = 40
	allItems := []domain.Item{
		{ID: 1, InternalName: "test_item", BaseValue: 100},
	}

	mockRepo.On("GetSellablePrices", ctx).Return(allItems, nil)
	mockProgression.On("AreItemsUnlocked", ctx, []string{"test_item"}).
		Return(map[string]bool{"test_item": true}, nil)

	// Level 5 upgrade: 1.25x multiplier
	// Base sell price: 40, Modified: 40 * 1.25 = 50
	mockProgression.On("GetModifiedValue", ctx, "economy_bonus", 40.0).
		Return(50.0, nil)

	// ACT
	items, err := service.GetSellablePrices(ctx)

	// ASSERT
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, 50, *items[0].SellPrice, "Level 5 economy_bonus should apply 1.25x multiplier (40 * 1.25 = 50)")
	mockRepo.AssertExpectations(t)
	mockProgression.AssertExpectations(t)
}

// TestUpgradeEconomy1_SellPriceModifier_MultipleItems verifies modifier applies to all items
func TestUpgradeEconomy1_SellPriceModifier_MultipleItems(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	mockProgression := &MockProgressionService{}
	service := NewService(mockRepo, nil, nil, mockProgression)
	ctx := context.Background()

	allItems := []domain.Item{
		{ID: 1, InternalName: "cheap_item", BaseValue: 10},       // Base sell: 4
		{ID: 2, InternalName: "expensive_item", BaseValue: 1000}, // Base sell: 400
	}

	mockRepo.On("GetSellablePrices", ctx).Return(allItems, nil)
	mockProgression.On("AreItemsUnlocked", ctx, []string{"cheap_item", "expensive_item"}).
		Return(map[string]bool{"cheap_item": true, "expensive_item": true}, nil)

	// Level 2 upgrade: 1.10x multiplier
	mockProgression.On("GetModifiedValue", ctx, "economy_bonus", 4.0).Return(4.4, nil)
	mockProgression.On("GetModifiedValue", ctx, "economy_bonus", 400.0).Return(440.0, nil)

	// ACT
	items, err := service.GetSellablePrices(ctx)

	// ASSERT
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, 4, *items[0].SellPrice, "Cheap item: 4 * 1.10 = 4.4, rounded down to 4")
	assert.Equal(t, 440, *items[1].SellPrice, "Expensive item: 400 * 1.10 = 440")
	mockRepo.AssertExpectations(t)
	mockProgression.AssertExpectations(t)
}

// TestUpgradeEconomy1_ModifierFailureFallback verifies graceful fallback on error
func TestUpgradeEconomy1_ModifierFailureFallback(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	mockProgression := &MockProgressionService{}
	service := NewService(mockRepo, nil, nil, mockProgression)
	ctx := context.Background()

	allItems := []domain.Item{
		{ID: 1, InternalName: "test_item", BaseValue: 100},
	}

	mockRepo.On("GetSellablePrices", ctx).Return(allItems, nil)
	mockProgression.On("AreItemsUnlocked", ctx, []string{"test_item"}).
		Return(map[string]bool{"test_item": true}, nil)

	// Progression service returns error
	mockProgression.On("GetModifiedValue", ctx, "economy_bonus", 40.0).
		Return(0.0, errors.New("progression service error"))

	// ACT
	items, err := service.GetSellablePrices(ctx)

	// ASSERT
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, 40, *items[0].SellPrice, "Should fall back to base price on error")
	mockRepo.AssertExpectations(t)
	mockProgression.AssertExpectations(t)
}

// TestUpgradeEconomy1_NilProgressionService verifies graceful degradation
func TestUpgradeEconomy1_NilProgressionService(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, nil, nil, nil) // nil progression service
	ctx := context.Background()

	allItems := []domain.Item{
		{ID: 1, InternalName: "test_item", BaseValue: 100},
	}

	mockRepo.On("GetSellablePrices", ctx).Return(allItems, nil)

	// ACT
	items, err := service.GetSellablePrices(ctx)

	// ASSERT
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, 40, *items[0].SellPrice, "Should use base price when progression service is nil")
	mockRepo.AssertExpectations(t)
}

// TestUpgradeEconomy1_IntegrationWithSellItem verifies modifier applies during actual sell
func TestUpgradeEconomy1_IntegrationWithSellItem(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	mockTx := &MockTx{}
	mockProgression := &MockProgressionService{}
	service := NewService(mockRepo, nil, nil, mockProgression)
	ctx := context.Background()

	user := createTestUser()
	item := createTestItem(10, "test_item", 100) // Base value: 100, base sell: 40
	moneyItem := createMoneyItem()
	inventory := &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 10, Quantity: 5},
			{ItemID: 1, Quantity: 0}, // No money yet
		},
	}

	// Setup mocks
	mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil)
	mockRepo.On("GetItemByName", ctx, "test_item").Return(item, nil)
	mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)
	mockRepo.On("BeginTx", ctx).Return(mockTx, nil)
	mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)
	mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", ctx).Return(nil)
	mockTx.On("Rollback", ctx).Return(nil)

	// Level 3 upgrade: 1.15x multiplier
	// Base sell price: 40, Modified: 40 * 1.15 = 46
	// Selling 2 items: 2 * 46 = 92
	mockProgression.On("GetModifiedValue", ctx, "economy_bonus", 40.0).
		Return(46.0, nil)

	// ACT
	moneyGained, quantitySold, err := service.SellItem(ctx, domain.PlatformTwitch, "", "testuser", "test_item", 2)

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, 2, quantitySold, "Should sell 2 items")
	assert.Equal(t, 92, moneyGained, "Should gain 92 money (2 * 46 with 1.15x modifier)")
	mockRepo.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	mockProgression.AssertExpectations(t)
}

// TestUpgradeEconomy1_RoundingBehavior verifies integer rounding for modified prices
func TestUpgradeEconomy1_RoundingBehavior(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	mockProgression := &MockProgressionService{}
	service := NewService(mockRepo, nil, nil, mockProgression)
	ctx := context.Background()

	// Item that produces fractional result: base value 15
	// Base sell: 15 * 0.40 = 6
	// Modified: 6 * 1.05 = 6.3 -> should round down to 6
	allItems := []domain.Item{
		{ID: 1, InternalName: "test_item", BaseValue: 15},
	}

	mockRepo.On("GetSellablePrices", ctx).Return(allItems, nil)
	mockProgression.On("AreItemsUnlocked", ctx, []string{"test_item"}).
		Return(map[string]bool{"test_item": true}, nil)
	mockProgression.On("GetModifiedValue", ctx, "economy_bonus", 6.0).
		Return(6.3, nil)

	// ACT
	items, err := service.GetSellablePrices(ctx)

	// ASSERT
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, 6, *items[0].SellPrice, "Should round down fractional prices (6.3 -> 6)")
	mockRepo.AssertExpectations(t)
	mockProgression.AssertExpectations(t)
}

// TestUpgradeEconomy1_BuyPriceNotAffected verifies buy prices are unaffected
// Design decision: economy_bonus only affects SELL prices, not buy prices
// This prevents the modifier from being too powerful (better buying AND selling)
func TestUpgradeEconomy1_BuyPriceNotAffected(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	mockProgression := &MockProgressionService{}
	service := NewService(mockRepo, nil, nil, mockProgression)
	ctx := context.Background()

	allItems := []domain.Item{
		{ID: 1, InternalName: "buyable_item", BaseValue: 100},
	}

	mockRepo.On("GetBuyablePrices", ctx).Return(allItems, nil)
	mockProgression.On("AreItemsUnlocked", ctx, []string{"buyable_item"}).
		Return(map[string]bool{"buyable_item": true}, nil)

	// ACT
	items, err := service.GetBuyablePrices(ctx)

	// ASSERT
	require.NoError(t, err)
	require.Len(t, items, 1)
	// Buy prices use BaseValue directly, no modifier applied
	assert.Equal(t, 100, items[0].BaseValue, "Buy prices should not be affected by economy_bonus")
	assert.Nil(t, items[0].SellPrice, "GetBuyablePrices should not populate sell prices")
	mockRepo.AssertExpectations(t)
	mockProgression.AssertExpectations(t)
}
