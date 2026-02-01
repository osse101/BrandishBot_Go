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
	"github.com/osse101/BrandishBot_Go/internal/job"
)

// TestBuyItem_AwardsXP verifies that purchasing an item correctly triggers an async XP award
func TestBuyItem_AwardsXP(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	mockJobService := &MockJobService{}
	service := NewService(mockRepo, mockJobService, nil, nil)
	ctx := context.Background()

	user := createTestUser()
	itemPrice := 200
	item := createTestItem(10, domain.PublicNameLootbox, itemPrice)
	moneyItem := createMoneyItem()
	inventory := createInventoryWithMoney(500)

	// Mock DB interactions
	mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil)
	mockRepo.On("GetItemByName", ctx, domain.PublicNameLootbox).Return(item, nil)
	mockRepo.On("IsItemBuyable", ctx, domain.PublicNameLootbox).Return(true, nil)
	mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)

	mockTx := &MockTx{}
	mockRepo.On("BeginTx", ctx).Return(mockTx, nil)
	mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)
	mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", ctx).Return(nil)
	mockTx.On("Rollback", ctx).Return(nil)

	// Mock Async Job Service
	// XP = ceil(200 / 10) = 20
	expectedXP := 20
	mockJobService.On("AwardXP", mock.Anything, user.ID, job.JobKeyMerchant, expectedXP, ActionTypeBuy, mock.MatchedBy(func(m map[string]interface{}) bool {
		return m[MetadataKeyAction] == ActionTypeBuy &&
			m[MetadataKeyItemName] == domain.PublicNameLootbox &&
			m[MetadataKeyValue] == itemPrice
	})).Return(&domain.XPAwardResult{LeveledUp: false}, nil)

	// ACT
	purchased, err := service.BuyItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameLootbox, 1)

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, 1, purchased)

	// Wait for async tasks
	err = service.Shutdown(ctx)
	require.NoError(t, err, "Shutdown should complete without error")

	mockRepo.AssertExpectations(t)
	mockJobService.AssertExpectations(t)
}

// TestSellItem_AwardsXP verifies that selling an item correctly triggers an async XP award
func TestSellItem_AwardsXP(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	mockJobService := &MockJobService{}
	service := NewService(mockRepo, mockJobService, nil, nil)
	ctx := context.Background()

	user := createTestUser()
	baseValue := 100
	item := createTestItem(10, domain.PublicNameLootbox, baseValue)
	moneyItem := createMoneyItem()
	inventory := createInventoryWithItem(10, 5)

	// Mock DB interactions
	mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil)
	mockRepo.On("GetItemByName", ctx, domain.PublicNameLootbox).Return(item, nil)
	mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)

	mockTx := &MockTx{}
	mockRepo.On("BeginTx", ctx).Return(mockTx, nil)
	mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)
	mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", ctx).Return(nil)
	mockTx.On("Rollback", ctx).Return(nil)

	// Calculate expected XP
	// Sell Price = 100 * 0.4 = 40
	// XP = ceil(40 / 10) = 4
	expectedXP := 4
	moneyGained := 40

	mockJobService.On("AwardXP", mock.Anything, user.ID, job.JobKeyMerchant, expectedXP, ActionTypeSell, mock.MatchedBy(func(m map[string]interface{}) bool {
		return m[MetadataKeyAction] == ActionTypeSell &&
			m[MetadataKeyItemName] == domain.PublicNameLootbox &&
			m[MetadataKeyValue] == moneyGained
	})).Return(&domain.XPAwardResult{LeveledUp: true, NewLevel: 5}, nil)

	// ACT
	_, _, err := service.SellItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameLootbox, 1)

	// ASSERT
	require.NoError(t, err)

	// Wait for async tasks
	err = service.Shutdown(ctx)
	require.NoError(t, err, "Shutdown should complete without error")

	mockRepo.AssertExpectations(t)
	mockJobService.AssertExpectations(t)
}

// TestShutdown_WaitsForTasks verifies that Shutdown blocks until async tasks are complete
func TestShutdown_WaitsForTasks(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	mockJobService := &MockJobService{}
	service := NewService(mockRepo, mockJobService, nil, nil)
	ctx := context.Background()

	user := createTestUser()
	itemPrice := 100
	item := createTestItem(10, domain.PublicNameLootbox, itemPrice)
	moneyItem := createMoneyItem()
	inventory := createInventoryWithMoney(500)

	// Mock DB interactions
	mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil)
	mockRepo.On("GetItemByName", ctx, domain.PublicNameLootbox).Return(item, nil)
	mockRepo.On("IsItemBuyable", ctx, domain.PublicNameLootbox).Return(true, nil)
	mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)

	mockTx := &MockTx{}
	mockRepo.On("BeginTx", ctx).Return(mockTx, nil)
	mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)
	mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", ctx).Return(nil)
	mockTx.On("Rollback", ctx).Return(nil)

	// Coordinate using channels
	jobStarted := make(chan struct{})
	jobBlock := make(chan struct{})

	// Setup slow job
	mockJobService.On("AwardXP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			close(jobStarted) // Signal job started
			<-jobBlock        // Wait until allowed to finish
		}).
		Return(&domain.XPAwardResult{LeveledUp: false}, nil)

	// ACT
	// 1. Trigger the async job
	_, err := service.BuyItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameLootbox, 1)
	require.NoError(t, err)

	// 2. Wait for job to start
	select {
	case <-jobStarted:
		// Job is running now
	case <-time.After(1 * time.Second):
		t.Fatal("Job did not start in time")
	}

	// 3. Start shutdown in a goroutine
	shutdownDone := make(chan struct{})
	go func() {
		err := service.Shutdown(context.Background())
		assert.NoError(t, err)
		close(shutdownDone)
	}()

	// 4. Ensure shutdown blocks (give it a small window)
	select {
	case <-shutdownDone:
		t.Fatal("Shutdown should be blocked by the running job")
	case <-time.After(100 * time.Millisecond):
		// Expected behavior: Shutdown is blocked
	}

	// 5. Unblock the job
	close(jobBlock)

	// 6. Ensure shutdown completes now
	select {
	case <-shutdownDone:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Shutdown did not complete after job finished")
	}

	mockRepo.AssertExpectations(t)
	mockJobService.AssertExpectations(t)
}

// TestShutdown_Timeout verifies that Shutdown returns an error if context expires
func TestShutdown_Timeout(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	// We don't need JobService for this, we just need to increment the waitgroup manually
	// But `service` struct fields are private, so we can't access `wg`.
	// However, we can use a "stuck" job to simulate this.
	mockJobService := &MockJobService{}

	service := NewService(mockRepo, mockJobService, nil, nil)
	ctx := context.Background()

	user := createTestUser()
	itemPrice := 100
	item := createTestItem(10, domain.PublicNameLootbox, itemPrice)
	moneyItem := createMoneyItem()
	inventory := createInventoryWithMoney(500)

	// Mock DB interactions for BuyItem
	mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil)
	mockRepo.On("GetItemByName", ctx, domain.PublicNameLootbox).Return(item, nil)
	mockRepo.On("IsItemBuyable", ctx, domain.PublicNameLootbox).Return(true, nil)
	mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)

	mockTx := &MockTx{}
	mockRepo.On("BeginTx", ctx).Return(mockTx, nil)
	mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)
	mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", ctx).Return(nil)
	mockTx.On("Rollback", ctx).Return(nil)

	// Make the job hang forever (until test ends)
	done := make(chan struct{})
	defer close(done)

	mockJobService.On("AwardXP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			<-done // Block until test finishes
		}).
		Return(nil, nil)

	// Trigger the stuck job
	_, err := service.BuyItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameLootbox, 1)
	require.NoError(t, err)

	// Create a short timeout context
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// ACT
	err = service.Shutdown(shutdownCtx)

	// ASSERT
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shutdown timed out")
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

	mockRepo.On("GetSellablePrices", mock.Anything).Return([]domain.Item{}, nil)

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
