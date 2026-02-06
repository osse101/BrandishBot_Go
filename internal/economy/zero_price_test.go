package economy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// TestBuyItem_ZeroPrice attempts to buy an item with 0 value to reproduce division by zero panic
func TestBuyItem_ZeroPrice(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, nil, nil, nil, nil)
	ctx := context.Background()

	user := createTestUser()
	// Item with 0 base value
	item := &domain.Item{
		ID:           10,
		InternalName: "free_item",
		BaseValue:    0,
	}
	moneyItem := createMoneyItem()
	inventory := createInventoryWithMoney(500)

	mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil)
	mockRepo.On("GetItemByName", ctx, "free_item").Return(item, nil)
	mockRepo.On("IsItemBuyable", ctx, "free_item").Return(true, nil)
	mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)

	mockTx := &MockTx{}
	mockRepo.On("BeginTx", ctx).Return(mockTx, nil)
	mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)

	// Expectations
	// It should update inventory with cost 0.
	// Since cost is 0, money shouldn't change, but item quantity should increase.
	// The service implementation calls UpdateInventory with the modified object.
	mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", ctx).Return(nil)
	mockTx.On("Rollback", ctx).Return(nil)

	// ACT & ASSERT
	var purchased int
	var err error
	assert.NotPanics(t, func() {
		purchased, err = service.BuyItem(ctx, domain.PlatformTwitch, "", "testuser", "free_item", 1)
	}, "Buying an item with 0 value should not panic")

	assert.NoError(t, err)
	assert.Equal(t, 1, purchased)
	mockRepo.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}
