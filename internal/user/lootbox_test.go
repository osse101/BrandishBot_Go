package user

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepo is a minimal mock for the repository needed for lootbox tests
// MockRepo is a minimal mock for the repository needed for lootbox tests
type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) UpsertUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockRepo) UpdateUser(ctx context.Context, user domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockRepo) DeleteUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockRepo) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	args := m.Called(ctx, platform, platformID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockRepo) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockRepo) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Inventory), args.Error(1)
}

func (m *MockRepo) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	args := m.Called(ctx, userID, inventory)
	return args.Error(0)
}

func (m *MockRepo) DeleteInventory(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockRepo) GetItemByName(ctx context.Context, name string) (*domain.Item, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Item), args.Error(1)
}

func (m *MockRepo) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	args := m.Called(ctx, itemIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Item), args.Error(1)
}

func (m *MockRepo) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Item), args.Error(1)
}

func (m *MockRepo) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockRepo) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Item), args.Error(1)
}

func (m *MockRepo) IsItemBuyable(ctx context.Context, itemName string) (bool, error) {
	args := m.Called(ctx, itemName)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepo) BeginTx(ctx context.Context) (repository.Tx, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(repository.Tx), args.Error(1)
}

func (m *MockRepo) GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error) {
	args := m.Called(ctx, itemID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Recipe), args.Error(1)
}

func (m *MockRepo) IsRecipeUnlocked(ctx context.Context, userID string, recipeID int) (bool, error) {
	args := m.Called(ctx, userID, recipeID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepo) UnlockRecipe(ctx context.Context, userID string, recipeID int) error {
	args := m.Called(ctx, userID, recipeID)
	return args.Error(0)
}

func (m *MockRepo) GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]crafting.UnlockedRecipeInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]crafting.UnlockedRecipeInfo), args.Error(1)
}

func (m *MockRepo) GetLastCooldown(ctx context.Context, userID, action string) (*time.Time, error) {
	args := m.Called(ctx, userID, action)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*time.Time), args.Error(1)
}

func (m *MockRepo) UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error {
	args := m.Called(ctx, userID, action, timestamp)
	return args.Error(0)
}

// Helper to create a service with a mock repo
func createTestService(repo *MockRepo) *service {
	return &service{
		repo:       repo,
		lootTables: make(map[string][]LootItem),
	}
}

func TestProcessLootbox(t *testing.T) {
	// Setup items
	lootbox0 := &domain.Item{ID: 100, Name: domain.ItemLootbox0}
	money := &domain.Item{ID: 1, Name: domain.ItemMoney}

	t.Run("Lootbox0 drops money", func(t *testing.T) {
		repo := new(MockRepo)
		svc := createTestService(repo)

		// Setup loot table manually for test
		svc.lootTables[domain.ItemLootbox0] = []LootItem{
			{ItemName: domain.ItemMoney, Min: 1, Max: 10, Chance: 1.0},
		}

		ctx := context.Background()

		// Mock repo responses
		repo.On("GetItemByName", ctx, domain.ItemMoney).Return(money, nil)

		// Setup inventory with lootbox0
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: lootbox0.ID, Quantity: 1},
			},
		}

		// Execute
		msg, err := svc.processLootbox(ctx, inventory, lootbox0, 1)

		// Verify
		assert.NoError(t, err)
		assert.Contains(t, msg, "Opened 1 lootbox0")
		assert.Contains(t, msg, "money")

		// Verify inventory changes
		// Should have consumed lootbox0 and gained money
		foundLootbox := false
		foundMoney := false
		for _, slot := range inventory.Slots {
			if slot.ItemID == lootbox0.ID {
				foundLootbox = true
			}
			if slot.ItemID == money.ID {
				foundMoney = true
				assert.GreaterOrEqual(t, slot.Quantity, 1)
				assert.LessOrEqual(t, slot.Quantity, 10)
			}
		}
		assert.False(t, foundLootbox, "Lootbox should be consumed")
		assert.True(t, foundMoney, "Money should be added")
	})

	t.Run("Lootbox0 insufficient quantity", func(t *testing.T) {
		repo := new(MockRepo)
		svc := createTestService(repo)
		ctx := context.Background()

		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: lootbox0.ID, Quantity: 1},
			},
		}

		_, err := svc.processLootbox(ctx, inventory, lootbox0, 2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not enough items")
	})
}

func TestLoadLootTables(t *testing.T) {
	// Create a temporary file
	content := []byte(`{
		"test_box": [
			{"item_name": "money", "min": 1, "max": 10, "chance": 1.0}
		]
	}`)
	tmpfile, err := os.CreateTemp("", "loot_tables.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	repo := new(MockRepo)
	svc := createTestService(repo)

	err = svc.LoadLootTables(tmpfile.Name())
	assert.NoError(t, err)

	assert.Contains(t, svc.lootTables, "test_box")
	assert.Equal(t, 1, len(svc.lootTables["test_box"]))
	assert.Equal(t, "money", svc.lootTables["test_box"][0].ItemName)
}
