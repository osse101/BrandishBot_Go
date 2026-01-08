package user

import (
	"context"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

func (m *MockRepo) GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error) {
	args := m.Called(ctx, platform, username)
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

func (m *MockRepo) GetItemsByNames(ctx context.Context, names []string) ([]domain.Item, error) {
	args := m.Called(ctx, names)
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

func (m *MockRepo) BeginTx(ctx context.Context) (repository.UserTx, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(repository.UserTx), args.Error(1)
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

func (m *MockRepo) GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]repository.UnlockedRecipeInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.UnlockedRecipeInfo), args.Error(1)
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

func (m *MockRepo) MergeUsersInTransaction(ctx context.Context, primaryUserID, secondaryUserID string, mergedUser domain.User, mergedInventory domain.Inventory) error {
	args := m.Called(ctx, primaryUserID, secondaryUserID, mergedUser, mergedInventory)
	return args.Error(0)
}

func (m *MockRepo) GetAllItems(ctx context.Context) ([]domain.Item, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Item), args.Error(1)
}

// MockLootboxService is a mock for lootbox.Service
type MockLootboxService struct {
	mock.Mock
}

func (m *MockLootboxService) OpenLootbox(ctx context.Context, lootboxName string, quantity int) ([]lootbox.DroppedItem, error) {
	args := m.Called(ctx, lootboxName, quantity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]lootbox.DroppedItem), args.Error(1)
}

// Helper to create a service with a mock repo and lootbox service
func createTestService(repo *MockRepo, lootboxSvc *MockLootboxService) *service {
	namingResolver := NewMockNamingResolver()

	return &service{
		repo:           repo,
		lootboxService: lootboxSvc,
		namingResolver: namingResolver,
	}
}

func TestProcessLootbox(t *testing.T) {
	// Setup items
	lootbox0 := &domain.Item{ID: 100, InternalName: domain.ItemLootbox0}

	money := &domain.Item{ID: 1, InternalName: domain.ItemMoney}

	t.Run("Lootbox0 drops money", func(t *testing.T) {
		repo := new(MockRepo)
		lootboxSvc := new(MockLootboxService)
		svc := createTestService(repo, lootboxSvc)

		ctx := context.Background()

		// Mock lootbox service response
		drops := []lootbox.DroppedItem{
			{ItemID: money.ID, ItemName: domain.ItemMoney, Quantity: 5, Value: 50, ShineLevel: "COMMON"},
		}
		lootboxSvc.On("OpenLootbox", ctx, domain.ItemLootbox0, 1).Return(drops, nil)

		// Setup inventory with lootbox0
		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: lootbox0.ID, Quantity: 1},
			},
		}

		// Execute
		// Pass nil user as it's not used in this test path (except for stats which is nil here)
		msg, err := svc.processLootbox(ctx, nil, inventory, lootbox0, 1)

		// Verify
		assert.NoError(t, err)
		assert.Contains(t, msg, "Opened")
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
				assert.Equal(t, 5, slot.Quantity)
			}
		}
		assert.False(t, foundLootbox, "Lootbox should be consumed")
		assert.True(t, foundMoney, "Money should be added")

		lootboxSvc.AssertExpectations(t)
	})

	t.Run("Lootbox0 insufficient quantity", func(t *testing.T) {
		repo := new(MockRepo)
		lootboxSvc := new(MockLootboxService)
		svc := createTestService(repo, lootboxSvc)
		ctx := context.Background()

		inventory := &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: lootbox0.ID, Quantity: 1},
			},
		}

		_, err := svc.processLootbox(ctx, nil, inventory, lootbox0, 2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), domain.ErrMsgNotEnoughItems)
	})
}
