package economy

import (
	"context"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/testing/leaktest"
	"github.com/stretchr/testify/assert"
)

// TestBuyItem_NoGoroutineLeak verifies no goroutines leak during buy operations
func TestBuyItem_NoGoroutineLeak(t *testing.T) {
	repo := &MockUserRepository{
		users: make(map[string]*domain.User),
		inventories: make(map[string]*domain.Inventory),
	}
	
	// Setup test user
	user := &domain.User{ID: "test-user", Username: "tester"}
	repo.users["test-user"] = user
	repo.inventories["test-user"] = &domain.Inventory{
		UserID: "test-user",
		Coins:  1000,
	}

	mockJobService := &MockJobService{}
	svc := NewService(repo, mockJobService)

	checker := leaktest.NewGoroutineChecker(t)

	// Perform buy operation
	ctx := context.Background()
	err := svc.BuyItem(ctx, "discord", "discord-id", "tester", domain.ItemLootbox1, 1)
	assert.NoError(t, err)

	// Check for leaks
	checker.Check(0)
}

// TestSellItem_NoGoroutineLeak verifies no goroutines leak during sell operations
func TestSellItem_NoGoroutineLeak(t *testing.T) {
	repo := &MockUserRepository{
		users: make(map[string]*domain.User),
		inventories: make(map[string]*domain.Inventory),
	}
	
	// Setup test user with item to sell
	user := &domain.User{ID: "test-user", Username: "tester"}
	repo.users["test-user"] = user
	repo.inventories["test-user"] = &domain.Inventory{
		UserID: "test-user",
		Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 5}, // Lootbox1
		},
	}

	mockJobService := &MockJobService{}
	svc := NewService(repo, mockJobService)

	checker := leaktest.NewGoroutineChecker(t)

	// Perform sell operation
	ctx := context.Background()
	err := svc.SellItem(ctx, "discord", "discord-id", "tester", domain.ItemLootbox1, 1)
	assert.NoError(t, err)

	// Check for leaks
	checker.Check(0)
}

// TestBuyItem_ConcurrentNoMemoryLeak verifies concurrent operations don't leak memory
func TestBuyItem_ConcurrentNoMemoryLeak(t *testing.T) {
	repo := &MockUserRepository{
		users: make(map[string]*domain.User),
		inventories: make(map[string]*domain.Inventory),
	}
	
	user := &domain.User{ID: "test-user", Username: "tester"}
	repo.users["test-user"] = user
	repo.inventories["test-user"] = &domain.Inventory{
		UserID: "test-user",
		Coins:  100000,
	}

	mockJobService := &MockJobService{}
	svc := NewService(repo, mockJobService)

	leaktest.CheckNoMemoryLeak(t, 2.0, func() {
		// Run many operations
		ctx := context.Background()
		for i := 0; i < 100; i++ {
			_ = svc.BuyItem(ctx, "discord", "discord-id", "tester", domain.ItemLootbox1, 1)
		}
	})
}

// Mock types for testing
type MockUserRepository struct {
	users       map[string]*domain.User
	inventories map[string]*domain.Inventory
}

func (m *MockUserRepository) GetUser(ctx context.Context, platform, platformID string) (*domain.User, error) {
	for _, u := range m.users {
		return u, nil
	}
	return nil, nil
}

func (m *MockUserRepository) Create(ctx context.Context, user domain.User) (domain.User, error) {
	m.users[user.ID] = &user
	return user, nil
}

func (m *MockUserRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	inv, ok := m.inventories[userID]
	if !ok {
		return &domain.Inventory{UserID: userID}, nil
	}
	return inv, nil
}

func (m *MockUserRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	m.inventories[userID] = &inventory
	return nil
}

type MockJobService struct{}

func (m *MockJobService) TryStartJob(ctx context.Context, platform, platformID, username string) error {
	return nil
}

func (m *MockJobService) CheckJobCompletion(ctx context.Context, platform, platformID, username string) (bool, error) {
	return false, nil
}
