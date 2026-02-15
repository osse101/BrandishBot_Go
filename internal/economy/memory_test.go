package economy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/testing/leaktest"
)

// TestBuyItem_NoGoroutineLeak verifies no goroutines leak during buy operations
func TestBuyItem_NoGoroutineLeak(t *testing.T) {
	// Use existing MockRepository from service_test.go
	repo := new(MockRepository)
	mockJob := new(MockJobService)

	user := createTestUser()
	item := createTestItem(2, "Lootbox1", 10)
	money := createMoneyItem()

	// Mock all required calls for BuyItem
	repo.On("GetUserByPlatformID", mock.Anything, "discord", "discord-id").Return(user, nil, nil)
	repo.On("GetItemByName", mock.Anything, "Lootbox1").Return(item, nil, nil)
	repo.On("GetItemByName", mock.Anything, domain.ItemMoney).Return(money, nil, nil)
	repo.On("IsItemBuyable", mock.Anything, "Lootbox1").Return(true, nil, nil)

	// Mock transaction
	mockTx := new(MockTx)
	repo.On("BeginTx", mock.Anything).Return(mockTx, nil, nil)

	inv := createInventoryWithMoney(1000)
	mockTx.On("GetInventory", mock.Anything, user.ID).Return(inv, nil, nil)
	mockTx.On("UpdateInventory", mock.Anything, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", mock.Anything).Return(nil)
	mockTx.On("Rollback", mock.Anything).Return(nil)

	// Mock job service
	mockJob.On("AwardXP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.XPAwardResult{}, nil).Maybe()

	svc := NewService(repo, nil, nil, nil)
	checker := leaktest.NewGoroutineChecker(t)

	// Perform buy operation
	ctx := context.Background()
	_, err := svc.BuyItem(ctx, "discord", "discord-id", "tester", "Lootbox1", 1)

	// Should succeed or have a documented error
	if err != nil {
		t.Logf("BuyItem error (expected in some cases): %v", err)
	}

	// Wait for async XP award to complete
	_ = svc.Shutdown(context.Background())

	// Check for leaks (allow 1 for background workers)
	checker.Check(1)
}

// TestService_Shutdown_NoGoroutineLeak verifies shutdown properly waits for goroutines
func TestService_Shutdown_NoGoroutineLeak(t *testing.T) {
	repo := new(MockRepository)

	svc := NewService(repo, nil, nil, nil)
	checker := leaktest.NewGoroutineChecker(t)

	// Call shutdown (no-op if nothing running)
	ctx := context.Background()
	err := svc.Shutdown(ctx)

	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Should have no leaks after shutdown
	checker.Check(0)
}
