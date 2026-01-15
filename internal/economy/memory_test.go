package economy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/testing/leaktest"
)

// TestBuyItem_NoGoroutineLeak verifies no goroutines leak during buy operations
func TestBuyItem_NoGoroutineLeak(t *testing.T) {
	// Use existing MockRepository from service_test.go
	repo := new(MockRepository)
	mockJob := new(mockJobService)

	user := createTestUser()
	item := createTestItem(2, "Lootbox1", 10)
	money := createMoneyItem()

	// Mock all required calls for BuyItem
	repo.On("GetUserByPlatformID", mock.Anything, "discord", "discord-id").Return(user, nil)
	repo.On("GetItemByName", mock.Anything, "Lootbox1").Return(item, nil)
	repo.On("GetItemByName", mock.Anything, domain.ItemMoney).Return(money, nil)
	repo.On("IsItemBuyable", mock.Anything, "Lootbox1").Return(true, nil)

	// Mock transaction
	mockTx := new(MockTx)
	repo.On("BeginTx", mock.Anything).Return(mockTx, nil)

	inv := createInventoryWithMoney(1000)
	mockTx.On("GetInventory", mock.Anything, user.ID).Return(inv, nil)
	mockTx.On("UpdateInventory", mock.Anything, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", mock.Anything).Return(nil)
	mockTx.On("Rollback", mock.Anything).Return(nil)

	// Mock job service
	mockJob.On("AwardXP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.XPAwardResult{}, nil).Maybe()

	svc := NewService(repo, mockJob, nil)
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
	mockJob := new(mockJobService)

	svc := NewService(repo, mockJob, nil)
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

// mockJobService implements JobService for testing
type mockJobService struct {
	mock.Mock
}

func (m *mockJobService) AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error) {
	args := m.Called(ctx, userID, jobKey, baseAmount, source, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.XPAwardResult), args.Error(1)
}

// MockTx implements repository.Tx for testing
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

// Ensure MockTx implements repository.Tx
var _ repository.EconomyTx = (*MockTx)(nil)
