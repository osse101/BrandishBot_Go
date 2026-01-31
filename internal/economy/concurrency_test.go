package economy

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// TestService_Shutdown_WaitsForAsyncTasks verifies that Shutdown blocks until async tasks complete
func TestService_Shutdown_WaitsForAsyncTasks(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	mockJobService := &MockJobService{}
	service := NewService(mockRepo, mockJobService, nil, nil)
	ctx := context.Background()

	// Setup valid transaction data
	user := createTestUser()
	item := createTestItem(10, domain.PublicNameLootbox, 100)
	moneyItem := createMoneyItem()
	inventory := createInventoryWithItem(10, 5)

	mockRepo.On("GetUserByPlatformID", mock.Anything, domain.PlatformTwitch, "").Return(user, nil)
	mockRepo.On("GetItemByName", mock.Anything, domain.PublicNameLootbox).Return(item, nil)
	mockRepo.On("GetItemByName", mock.Anything, domain.ItemMoney).Return(moneyItem, nil)

	mockTx := &MockTx{}
	mockRepo.On("BeginTx", mock.Anything).Return(mockTx, nil)
	mockTx.On("GetInventory", mock.Anything, user.ID).Return(inventory, nil)
	mockTx.On("UpdateInventory", mock.Anything, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", mock.Anything).Return(nil)
	mockTx.On("Rollback", mock.Anything).Return(nil)

	// Setup async task with delay
	taskStarted := make(chan struct{})
	taskCompleted := make(chan struct{})

	mockJobService.On("AwardXP", mock.Anything, user.ID, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			close(taskStarted)
			// Simulate long running task
			time.Sleep(100 * time.Millisecond)
			close(taskCompleted)
		}).
		Return(&domain.XPAwardResult{}, nil)

	// ACT
	// 1. Trigger the async operation (SellItem spawns awardMerchantXP)
	_, _, err := service.SellItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameLootbox, 1)
	require.NoError(t, err)

	// 2. Wait for task to start to ensure goroutine is running
	select {
	case <-taskStarted:
		// Task started
	case <-time.After(1 * time.Second):
		t.Fatal("Async task did not start in time")
	}

	// 3. Initiate Shutdown immediately
	shutdownStart := time.Now()
	err = service.Shutdown(ctx)
	require.NoError(t, err)
	shutdownDuration := time.Since(shutdownStart)

	// ASSERT
	// Shutdown should have waited at least 100ms (the task duration)
	// We use 50ms as a conservative lower bound to account for scheduler jitter
	assert.GreaterOrEqual(t, shutdownDuration.Milliseconds(), int64(50), "Shutdown should wait for async task")

	// Verify task actually completed
	select {
	case <-taskCompleted:
		// Task completed
	default:
		t.Fatal("Shutdown returned before task completed")
	}

	mockRepo.AssertExpectations(t)
	mockJobService.AssertExpectations(t)
}

// TestService_ConcurrentAccess verifies that the service is thread-safe
func TestService_ConcurrentAccess(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	// Use mock job service that does nothing but return success to avoid slowing down test
	mockJobService := &MockJobService{}
	service := NewService(mockRepo, mockJobService, nil, nil)
	ctx := context.Background()

	concurrency := 50
	var wg sync.WaitGroup

	// Setup expectations for concurrent calls
	// Since mocks are also shared, they need to be thread-safe or we need to expect N calls.
	// testify/mock objects are thread-safe.

	user := createTestUser()
	item := createTestItem(10, domain.PublicNameLootbox, 100)
	moneyItem := createMoneyItem()

	// We need to return NEW inventory instances to avoid race conditions on the inventory object itself
	// because multiple goroutines might try to modify the same returned pointer.
	// In a real DB, each tx gets its own snapshot or locks. Here we simulate that by returning copies.

	mockRepo.On("GetUserByPlatformID", mock.Anything, domain.PlatformTwitch, "").Return(user, nil)
	mockRepo.On("GetItemByName", mock.Anything, domain.PublicNameLootbox).Return(item, nil)
	mockRepo.On("GetItemByName", mock.Anything, domain.ItemMoney).Return(moneyItem, nil)

	// Allow multiple transactions
	// We need to setup unique transactions for each call because we can't reuse the same mock object
	// if we want to simulate proper isolation (or at least avoid test setup issues).
	// However, since we are mocking, we can just return a NEW mock object for each call.
	// Testify allows chaining .Return(...).Once()

	for i := 0; i < concurrency; i++ {
		tx := &MockTx{}
		inv := createInventoryWithItem(10, 1000)
		tx.On("GetInventory", mock.Anything, user.ID).Return(inv, nil)
		tx.On("UpdateInventory", mock.Anything, user.ID, mock.Anything).Return(nil)
		tx.On("Commit", mock.Anything).Return(nil)
		tx.On("Rollback", mock.Anything).Return(nil)

		mockRepo.On("BeginTx", mock.Anything).Return(tx, nil).Once()
	}

	mockJobService.On("AwardXP", mock.Anything, user.ID, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.XPAwardResult{}, nil)

	// ACT
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			_, _, err := service.SellItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameLootbox, 1)
			assert.NoError(t, err)
		}()
	}

	wg.Wait()

	// Verify clean shutdown after load
	err := service.Shutdown(ctx)
	require.NoError(t, err)
}

// TestResolveItemName_Fallback verifies fallback logic when naming resolver fails
func TestResolveItemName_Fallback(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	mockResolver := &MockNamingResolver{}
	service := NewService(mockRepo, nil, mockResolver, nil)
	ctx := context.Background()

	user := createTestUser()
	rawName := "lootbox_tier1"
	item := createTestItem(10, rawName, 100)
	moneyItem := createMoneyItem()
	inventory := createInventoryWithItem(10, 5)

	mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil)

	// Resolver returns false (not found as public name)
	mockResolver.On("ResolvePublicName", rawName).Return("", false)

	// Should fall back to GetItemByName with original name
	mockRepo.On("GetItemByName", ctx, rawName).Return(item, nil)
	mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)

	mockTx := &MockTx{}
	mockRepo.On("BeginTx", ctx).Return(mockTx, nil)
	mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)
	mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", ctx).Return(nil)
	mockTx.On("Rollback", ctx).Return(nil)

	// ACT
	// Pass the raw name, which the resolver doesn't know, but the repo does
	moneyGained, _, err := service.SellItem(ctx, domain.PlatformTwitch, "", "testuser", rawName, 1)

	// ASSERT
	require.NoError(t, err)
	assert.Greater(t, moneyGained, 0)

	mockResolver.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}
