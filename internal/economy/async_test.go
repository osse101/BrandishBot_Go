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

	// Sanity check: verify service doesn't race on its own fields.

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

// TestBuyItem_ConcurrentAccess simulates multiple goroutines buying items for the same user concurrently.
func TestBuyItem_ConcurrentAccess(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, nil, nil, nil)
	ctx := context.Background()

	user := createTestUser()
	item := createTestItem(10, domain.PublicNameLootbox, 100)
	moneyItem := createMoneyItem()

	// Setup inventory with enough money for all concurrent requests
	initialMoney := 1000000
	baseInventory := createInventoryWithMoney(initialMoney)

	// We use mock.Anything for contexts as they might differ per request if generated or wrapped.
	mockRepo.On("GetUserByPlatformID", mock.Anything, domain.PlatformTwitch, "").Return(user, nil)
	mockRepo.On("GetItemByName", mock.Anything, domain.PublicNameLootbox).Return(item, nil)
	mockRepo.On("IsItemBuyable", mock.Anything, domain.PublicNameLootbox).Return(true, nil)
	mockRepo.On("GetItemByName", mock.Anything, domain.ItemMoney).Return(moneyItem, nil)

	mockTx := &MockTx{}
	mockRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)

	// Important: Return a new copy of inventory for each call to simulate DB fetching
	// and to prevent race conditions in the test mock itself (since service modifies the returned pointer)
	mockTx.On("GetInventory", mock.Anything, user.ID).Return(func(ctx context.Context, uid string) *domain.Inventory {
		newInv := &domain.Inventory{
			Slots: make([]domain.InventorySlot, len(baseInventory.Slots)),
		}
		copy(newInv.Slots, baseInventory.Slots)
		return newInv
	}, nil)

	mockTx.On("UpdateInventory", mock.Anything, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", mock.Anything).Return(nil)
	// Usually SafeRollback checks if committed. If Commit succeeds, Rollback shouldn't be called.
	mockTx.On("Rollback", mock.Anything).Return(nil).Maybe()

	var wg sync.WaitGroup
	concurrency := 10

	// ACT
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Each goroutine attempts to buy 1 item
			_, err := service.BuyItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameLootbox, 1)
			assert.NoError(t, err)
		}()
	}

	wg.Wait()
}
