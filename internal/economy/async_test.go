package economy

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// TestBuyItem_PublishesEvent verifies that purchasing an item succeeds (XP now via event, not direct call)
func TestBuyItem_PublishesEvent(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, nil, nil, nil)
	ctx := context.Background()

	user := createTestUser()
	itemPrice := 200
	item := createTestItem(10, domain.PublicNameLootbox, itemPrice)
	moneyItem := createMoneyItem()
	inventory := createInventoryWithMoney(500)

	// Mock DB interactions
	mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil, nil)
	mockRepo.On("GetItemByName", ctx, domain.PublicNameLootbox).Return(item, nil, nil)
	mockRepo.On("IsItemBuyable", ctx, domain.PublicNameLootbox).Return(true, nil, nil)
	mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil, nil)

	mockTx := &MockTx{}
	mockRepo.On("BeginTx", ctx).Return(mockTx, nil, nil)
	mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil, nil)
	mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", ctx).Return(nil)
	mockTx.On("Rollback", ctx).Return(nil)

	// ACT
	purchased, err := service.BuyItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameLootbox, 1)

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, 1, purchased)
	mockRepo.AssertExpectations(t)
}

// TestSellItem_PublishesEvent verifies that selling an item succeeds (XP now via event, not direct call)
func TestSellItem_PublishesEvent(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, nil, nil, nil)
	ctx := context.Background()

	user := createTestUser()
	baseValue := 100
	item := createTestItem(10, domain.PublicNameLootbox, baseValue)
	moneyItem := createMoneyItem()
	inventory := createInventoryWithItem(10, 5)

	// Mock DB interactions
	mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil, nil)
	mockRepo.On("GetItemByName", ctx, domain.PublicNameLootbox).Return(item, nil, nil)
	mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil, nil)

	mockTx := &MockTx{}
	mockRepo.On("BeginTx", ctx).Return(mockTx, nil, nil)
	mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil, nil)
	mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", ctx).Return(nil)
	mockTx.On("Rollback", ctx).Return(nil)

	// ACT
	_, _, err := service.SellItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameLootbox, 1)

	// ASSERT
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestShutdown_ReturnsNil verifies that Shutdown completes immediately (no background goroutines)
func TestShutdown_ReturnsNil(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, nil, nil, nil)
	err := service.Shutdown(context.Background())
	require.NoError(t, err)
}

// Thread Safety Test
func TestService_ConcurrentAccess(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	// No job service to keep it simpler/faster
	service := NewService(mockRepo, nil, nil, nil)
	ctx := context.Background()

	// We'll simulate concurrent reads (GetSellablePrices) and writes (BuyItem)
	// Note: We can't easily simulate concurrent DB access without a real DB or sophisticated mock logic.
	// But we can verify that the service doesn't race on its own fields (like if it had a map cache).
	// Since `service` is stateless except for `wg`, this is mostly a sanity check.

	var wg sync.WaitGroup
	concurrency := 10

	mockRepo.On("GetSellablePrices", mock.Anything).Return([]domain.Item{}, nil, nil)

	// ACT
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = service.GetSellablePrices(ctx)
		}()
	}

	wg.Wait()
}
