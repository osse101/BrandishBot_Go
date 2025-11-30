package economy

import (
	"context"
	"errors"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/concurrency"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository implements Repository interface for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	args := m.Called(ctx, platform, platformID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockRepository) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	args := m.Called(ctx, itemName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Item), args.Error(1)
}

func (m *MockRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Inventory), args.Error(1)
}

func (m *MockRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	args := m.Called(ctx, userID, inventory)
	return args.Error(0)
}

func (m *MockRepository) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Item), args.Error(1)
}

func (m *MockRepository) IsItemBuyable(ctx context.Context, itemName string) (bool, error) {
	args := m.Called(ctx, itemName)
	return args.Bool(0), args.Error(1)
}

// Test fixtures
func createTestUser() *domain.User {
	return &domain.User{
		ID:       "user-123",
		Username: "testuser",
	}
}

func createTestItem(id int, name string, value int) *domain.Item {
	return &domain.Item{
		ID:        id,
		Name:      name,
		BaseValue: value,
	}
}

func createMoneyItem() *domain.Item {
	return createTestItem(1, domain.ItemMoney, 1)
}

func createInventoryWithItem(itemID, quantity int) *domain.Inventory {
	return &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: itemID, Quantity: quantity},
		},
	}
}

func createInventoryWithMoney(amount int) *domain.Inventory {
	return createInventoryWithItem(1, amount) // ItemID 1 is money
}

// Test boundary constants - from business requirements spec
const (
	MinQuantity   = 1
	BaseItemPrice = 100
)

// =============================================================================
// SellItem Tests - Demonstrating 5-Case Testing Model
// =============================================================================

// CASE 1: BEST CASE - Happy path
func TestSellItem_Success(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, concurrency.NewLockManager())
	ctx := context.Background()

	user := createTestUser()
	item := createTestItem(10, "Sword", 100)
	moneyItem := createMoneyItem()
	inventory := &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 10, Quantity: 5}, // 5 swords
			{ItemID: 1, Quantity: 50}, // 50 money
		},
	}

	mockRepo.On("GetUserByPlatformID", ctx, "twitch", "").Return(user, nil)
	mockRepo.On("GetItemByName", ctx, "Sword").Return(item, nil)
	mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)
	mockRepo.On("GetInventory", ctx, user.ID).Return(inventory, nil)
	mockRepo.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)

	// ACT
	moneyGained, quantitySold, err := service.SellItem(ctx, "twitch", "", "testuser", "Sword", 3)

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, 300, moneyGained, "Should receive correct money (3 * 100)")
	assert.Equal(t, 3, quantitySold, "Should sell requested quantity")
	mockRepo.AssertExpectations(t)
}

// CASE 2: WORST CASE - Boundary conditions
func TestSellItem_SellAllItems(t *testing.T) {
	// ARRANGE - User sells every last item they have
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, concurrency.NewLockManager())
	ctx := context.Background()

	user := createTestUser()
	item := createTestItem(10, "Material", 5)
	moneyItem := createMoneyItem()
	inventory := &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 10, Quantity: 100}, // Max stack
		},
	}

	mockRepo.On("GetUserByPlatformID", ctx, "twitch", "").Return(user, nil)
	mockRepo.On("GetItemByName", ctx, "Material").Return(item, nil)
	mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)
	mockRepo.On("GetInventory", ctx, user.ID).Return(inventory, nil)
	mockRepo.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)

	// ACT
	moneyGained, quantitySold, err := service.SellItem(ctx, "twitch", "", "testuser", "Material", 100)

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, 500, moneyGained, "Should receive correct money (100 * 5)")
	assert.Equal(t, 100, quantitySold, "Should sell all items")
	mockRepo.AssertExpectations(t)
}

// CASE 3: EDGE CASE - Partial quantity available
func TestSellItem_PartialQuantity(t *testing.T) {
	// ARRANGE - User requests 100 but only has 30
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, concurrency.NewLockManager())
	ctx := context.Background()

	user := createTestUser()
	item := createTestItem(10, "Potion", 20)
	moneyItem := createMoneyItem()
	inventory := &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 10, Quantity: 30}, // Only 30 available
		},
	}

	mockRepo.On("GetUserByPlatformID", ctx, "twitch", "").Return(user, nil)
	mockRepo.On("GetItemByName", ctx, "Potion").Return(item, nil)
	mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)
	mockRepo.On("GetInventory", ctx, user.ID).Return(inventory, nil)
	mockRepo.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)

	// ACT - Request 100 but only have 30
	moneyGained, quantitySold, err := service.SellItem(ctx, "twitch", "", "testuser", "Potion", 100)

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, 600, moneyGained, "Should sell what's available (30 * 20)")
	assert.Equal(t, 30, quantitySold, "Should return actual quantity sold")
}

// CASE 4: INVALID CASE - Bad inputs
func TestSellItem_InvalidInputs(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*MockRepository)
		username    string
		itemName    string
		expectErr   bool
		errorMsg    string
		description string
	}{
		{
			name: "user not found",
			setup: func(m *MockRepository) {
				m.On("GetUserByPlatformID", mock.Anything, "twitch", "").
					Return(nil, nil)
			},
			username:    "nonexistent",
			itemName:    "Sword",
			expectErr:   true,
			errorMsg:    "user not found",
			description: "Should fail when user does not exist",
		},
		{
			name: "item not found",
			setup: func(m *MockRepository) {
				user := createTestUser()
				m.On("GetUserByPlatformID", mock.Anything, "twitch", "").Return(user, nil)
				m.On("GetItemByName", mock.Anything, "InvalidItem").Return(nil, nil)
			},
			username:    "testuser",
			itemName:    "InvalidItem",
			expectErr:   true,
			errorMsg:    "item not found",
			description: "Should fail when item does not exist",
		},
		{
			name: "item not in inventory",
			setup: func(m *MockRepository) {
				user := createTestUser()
				item := createTestItem(10, "Sword", 100)
				moneyItem := createMoneyItem()
				emptyInventory := &domain.Inventory{Slots: []domain.InventorySlot{}}

				m.On("GetUserByPlatformID", mock.Anything, "twitch", "").Return(user, nil)
				m.On("GetItemByName", mock.Anything, "Sword").Return(item, nil)
				m.On("GetItemByName", mock.Anything, domain.ItemMoney).Return(moneyItem, nil)
				m.On("GetInventory", mock.Anything, user.ID).Return(emptyInventory, nil)
			},
			username:    "testuser",
			itemName:    "Sword",
			expectErr:   true,
			errorMsg:    "not in inventory",
			description: "Should fail when user does not own the item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			mockRepo := &MockRepository{}
			service := NewService(mockRepo, concurrency.NewLockManager())
			ctx := context.Background()
			tt.setup(mockRepo)

			// ACT
			_, _, err := service.SellItem(ctx, "twitch", "", tt.username, tt.itemName, 1)

			// ASSERT
			if tt.expectErr {
				require.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err, tt.description)
			}
		})
	}
}

// Quantity Boundary Tests - Critical for validating input constraints
func TestSellItem_QuantityBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		quantity    int
		expectErr   bool
		description string
	}{
		{"negative quantity", -1, true, "Negative quantities must be rejected"},
		{"zero quantity", 0, true, "Zero quantity is invalid"},
		{"min boundary", MinQuantity, false, "Minimum valid quantity should succeed"},
		{"low range", 5, false, "Small valid quantity should succeed"},
		{"mid range", 500, false, "Mid-range valid quantity should succeed"},
		{"high range", 5000, false, "Large valid quantity should succeed"},
		{"near max", domain.MaxTransactionQuantity - 100, false, "Quantity near maximum should succeed"},
		{"max boundary", domain.MaxTransactionQuantity, false, "Maximum valid quantity should succeed"},
		{"over max boundary", domain.MaxTransactionQuantity + 1, true, "Quantities over maximum must be rejected"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			mockRepo := &MockRepository{}
			service := NewService(mockRepo, concurrency.NewLockManager())
			ctx := context.Background()

			user := createTestUser()
			item := createTestItem(10, "Sword", 100)
			moneyItem := createMoneyItem()
			inventory := createInventoryWithItem(10, 100)

			mockRepo.On("GetUserByPlatformID", ctx, "twitch", "").Return(user, nil)
			mockRepo.On("GetItemByName", ctx, "Sword").Return(item, nil)
			mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)
			mockRepo.On("GetInventory", ctx, user.ID).Return(inventory, nil)
			mockRepo.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)

			// ACT
			_, _, err := service.SellItem(ctx, "twitch", "", "testuser", "Sword", tt.quantity)

			// ASSERT
			if tt.expectErr {
				require.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), "quantity")
			} else {
				require.NoError(t, err, tt.description)
			}
		})
	}
}

// CASE 5: HOSTILE CASE - Database errors and malicious scenarios
func TestSellItem_DatabaseErrors(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*MockRepository, context.Context)
		expectErr   bool
		errorMsg    string
		description string
	}{
		{
			name: "database error on GetUser",
			setup: func(m *MockRepository, ctx context.Context) {
				dbError := errors.New("database connection lost")
				m.On("GetUserByPlatformID", ctx, "twitch", "").Return(nil, dbError)
			},
			expectErr:   true,
			errorMsg:    "failed to get user",
			description: "Should fail when database connection is lost during user fetch",
		},
		{
			name: "database error on UpdateInventory",
			setup: func(m *MockRepository, ctx context.Context) {
				user := createTestUser()
				item := createTestItem(10, "Sword", 100)
				moneyItem := createMoneyItem()
				inventory := createInventoryWithItem(10, 5)

				m.On("GetUserByPlatformID", ctx, "twitch", "").Return(user, nil)
				m.On("GetItemByName", ctx, "Sword").Return(item, nil)
				m.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)
				m.On("GetInventory", ctx, user.ID).Return(inventory, nil)
				m.On("UpdateInventory", ctx, user.ID, mock.Anything).
					Return(errors.New("deadlock detected"))
			},
			expectErr:   true,
			errorMsg:    "failed to update inventory",
			description: "Should fail when database deadlock occurs during inventory update",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			mockRepo := &MockRepository{}
			service := NewService(mockRepo, concurrency.NewLockManager())
			ctx := context.Background()
			tt.setup(mockRepo, ctx)

			// ACT
			_, _, err := service.SellItem(ctx, "twitch", "", "testuser", "Sword", 1)

			// ASSERT
			if tt.expectErr {
				require.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err, tt.description)
			}
		})
	}
}

// =============================================================================
// GetSellablePrices Tests
// =============================================================================

func TestGetSellablePrices_Success(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, concurrency.NewLockManager())
	ctx := context.Background()

	expectedItems := []domain.Item{
		{ID: 10, Name: "Sword", BaseValue: 100},
		{ID: 20, Name: "Potion", BaseValue: 50},
	}

	mockRepo.On("GetSellablePrices", ctx).Return(expectedItems, nil)

	// ACT
	items, err := service.GetSellablePrices(ctx)

	// ASSERT
	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, "Sword", items[0].Name)
	assert.Equal(t, 100, items[0].BaseValue)
	mockRepo.AssertExpectations(t)
}

func TestGetSellablePrices_DatabaseError(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, concurrency.NewLockManager())
	ctx := context.Background()

	mockRepo.On("GetSellablePrices", ctx).
		Return(nil, errors.New("connection timeout"))

	// ACT
	items, err := service.GetSellablePrices(ctx)

	// ASSERT
	require.Error(t, err)
	assert.Nil(t, items)
	assert.Contains(t, err.Error(), "connection timeout")
}

// =============================================================================
// BuyItem Tests - 5-Case Testing Model
// =============================================================================

// CASE 1: BEST CASE
func TestBuyItem_Success(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, concurrency.NewLockManager())
	ctx := context.Background()

	user := createTestUser()
	item := createTestItem(10, "Sword", 100)
	moneyItem := createMoneyItem()
	inventory := &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: moneyItem.ID, Quantity: 500}, // Enough for 5 items
		},
	}

	mockRepo.On("GetUserByPlatformID", ctx, "twitch", "").Return(user, nil)
	mockRepo.On("GetItemByName", ctx, "Sword").Return(item, nil)
	mockRepo.On("IsItemBuyable", ctx, "Sword").Return(true, nil)
	mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)
	mockRepo.On("GetInventory", ctx, user.ID).Return(inventory, nil)
	mockRepo.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)

	// ACT
	purchased, err := service.BuyItem(ctx, "twitch", "", "testuser", "Sword", 3)

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, 3, purchased)
	mockRepo.AssertExpectations(t)
}

// CASE 2: BOUNDARY CASE - Money boundaries
func TestBuyItem_MoneyBoundaries(t *testing.T) {
	tests := []struct {
		name           string
		moneyBalance   int
		itemPrice      int
		quantityWanted int
		expectedQty    int
		expectErr      bool
		description    string
	}{
		// Lower boundaries
		{
			name:           "no money",
			moneyBalance:   0,
			itemPrice:      BaseItemPrice,
			quantityWanted: 1,
			expectedQty:    0,
			expectErr:      true,
			description:    "Buying with no money should fail",
		},
		{
			name:           "not enough for one",
			moneyBalance:   50,
			itemPrice:      BaseItemPrice,
			quantityWanted: 1,
			expectedQty:    0,
			expectErr:      true,
			description:    "Buying when cannot afford one should fail",
		},
		{
			name:           "exactly enough for one",
			moneyBalance:   BaseItemPrice,
			itemPrice:      BaseItemPrice,
			quantityWanted: 1,
			expectedQty:    1,
			expectErr:      false,
			description:    "Buying exactly one with exact funds should succeed",
		},
		{
			name:           "can afford one but wants more",
			moneyBalance:   BaseItemPrice,
			itemPrice:      BaseItemPrice,
			quantityWanted: 5,
			expectedQty:    1,
			expectErr:      false,
			description:    "Should buy only what can be afforded",
		},

		// Upper boundaries
		{
			name:           "exact money for all",
			moneyBalance:   500,
			itemPrice:      BaseItemPrice,
			quantityWanted: 5,
			expectedQty:    5,
			expectErr:      false,
			description:    "Buying all requested with exact funds should succeed",
		},
		{
			name:           "extra money leftover",
			moneyBalance:   550,
			itemPrice:      BaseItemPrice,
			quantityWanted: 5,
			expectedQty:    5,
			expectErr:      false,
			description:    "Buying all requested with extra funds should succeed",
		},

		// Partial purchase
		{
			name:           "can afford half",
			moneyBalance:   250,
			itemPrice:      BaseItemPrice,
			quantityWanted: 5,
			expectedQty:    2,
			expectErr:      false,
			description:    "Should buy partial quantity based on funds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			mockRepo := &MockRepository{}
			service := NewService(mockRepo, concurrency.NewLockManager())
			ctx := context.Background()

			user := createTestUser()
			item := createTestItem(10, "Sword", tt.itemPrice)
			moneyItem := createMoneyItem()
			inventory := createInventoryWithMoney(tt.moneyBalance)

			mockRepo.On("GetUserByPlatformID", ctx, "twitch", "").Return(user, nil)
			mockRepo.On("GetItemByName", ctx, "Sword").Return(item, nil)
			mockRepo.On("IsItemBuyable", ctx, "Sword").Return(true, nil)
			mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)
			mockRepo.On("GetInventory", ctx, user.ID).Return(inventory, nil)
			mockRepo.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil).Maybe()

			// ACT
			purchased, err := service.BuyItem(ctx, "twitch", "", "testuser", "Sword", tt.quantityWanted)

			// ASSERT
			if tt.expectErr {
				require.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), "insufficient funds")
			} else {
				require.NoError(t, err, tt.description)
				assert.Equal(t, tt.expectedQty, purchased)
			}
		})
	}
}

// CASE 2: BOUNDARY CASE - Quantity boundaries
func TestBuyItem_QuantityBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		quantity    int
		expectErr   bool
		description string
	}{
		{"negative quantity", -1, true, "Negative quantities must be rejected"},
		{"zero quantity", 0, true, "Zero quantity is invalid"},
		{"min boundary", MinQuantity, false, "Minimum valid quantity should succeed"},
		{"low range", 3, false, "Small valid quantity should succeed"},
		{"mid range", 250, false, "Mid-range valid quantity should succeed"},
		{"high range", 7500, false, "Large valid quantity should succeed"},
		{"near max", domain.MaxTransactionQuantity - 50, false, "Quantity near maximum should succeed"},
		{"max boundary", domain.MaxTransactionQuantity, false, "Maximum valid quantity should succeed"},
		{"over max boundary", domain.MaxTransactionQuantity + 1, true, "Quantities over maximum must be rejected"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			mockRepo := &MockRepository{}
			service := NewService(mockRepo, concurrency.NewLockManager())
			ctx := context.Background()

			user := createTestUser()
			item := createTestItem(10, "Sword", 100)
			moneyItem := createMoneyItem()
			inventory := createInventoryWithMoney(10000000)

			mockRepo.On("GetUserByPlatformID", ctx, "twitch", "").Return(user, nil)
			mockRepo.On("GetItemByName", ctx, "Sword").Return(item, nil)
			mockRepo.On("IsItemBuyable", ctx, "Sword").Return(true, nil)
			mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)
			mockRepo.On("GetInventory", ctx, user.ID).Return(inventory, nil)
			mockRepo.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)

			// ACT
			purchased, err := service.BuyItem(ctx, "twitch", "", "testuser", "Sword", tt.quantity)

			// ASSERT
			if tt.expectErr {
				require.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), "quantity")
			} else {
				require.NoError(t, err, tt.description)
				assert.Equal(t, tt.quantity, purchased)
			}
		})
	}
}

// CASE 4: INVALID CASE
func TestBuyItem_InvalidInputs(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*MockRepository, context.Context)
		expectErr   bool
		errorMsg    string
		description string
	}{
		{
			name: "user not found",
			setup: func(m *MockRepository, ctx context.Context) {
				m.On("GetUserByPlatformID", ctx, "twitch", "").Return(nil, nil)
			},
			expectErr:   true,
			errorMsg:    "user not found",
			description: "Should fail when user does not exist",
		},
		{
			name: "item not found",
			setup: func(m *MockRepository, ctx context.Context) {
				m.On("GetUserByPlatformID", ctx, "twitch", "").Return(createTestUser(), nil)
				m.On("GetItemByName", ctx, "Sword").Return(nil, nil)
			},
			expectErr:   true,
			errorMsg:    "item not found",
			description: "Should fail when item does not exist",
		},
		{
			name: "item not buyable",
			setup: func(m *MockRepository, ctx context.Context) {
				user := createTestUser()
				item := createTestItem(10, "Sword", 100)
				m.On("GetUserByPlatformID", ctx, "twitch", "").Return(user, nil)
				m.On("GetItemByName", ctx, "Sword").Return(item, nil)
				m.On("IsItemBuyable", ctx, "Sword").Return(false, nil)
			},
			expectErr:   true,
			errorMsg:    "is not buyable",
			description: "Should fail when item is not buyable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			mockRepo := &MockRepository{}
			service := NewService(mockRepo, concurrency.NewLockManager())
			ctx := context.Background()
			tt.setup(mockRepo, ctx)

			// ACT
			_, err := service.BuyItem(ctx, "twitch", "", "testuser", "Sword", 1)

			// ASSERT
			if tt.expectErr {
				require.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err, tt.description)
			}
		})
	}
}

// CASE 5: HOSTILE CASE - Database errors
func TestBuyItem_DatabaseErrors(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*MockRepository, context.Context)
		expectErr   bool
		errorMsg    string
		description string
	}{
		{
			name: "database error on UpdateInventory",
			setup: func(m *MockRepository, ctx context.Context) {
				user := createTestUser()
				item := createTestItem(10, "Sword", 100)
				moneyItem := createMoneyItem()
				inventory := createInventoryWithMoney(500)

				m.On("GetUserByPlatformID", ctx, "twitch", "").Return(user, nil)
				m.On("GetItemByName", ctx, "Sword").Return(item, nil)
				m.On("IsItemBuyable", ctx, "Sword").Return(true, nil)
				m.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)
				m.On("GetInventory", ctx, user.ID).Return(inventory, nil)
				m.On("UpdateInventory", ctx, user.ID, mock.Anything).
					Return(errors.New("deadlock detected"))
			},
			expectErr:   true,
			errorMsg:    "failed to update inventory",
			description: "Should fail when database deadlock occurs during inventory update",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			mockRepo := &MockRepository{}
			service := NewService(mockRepo, concurrency.NewLockManager())
			ctx := context.Background()
			tt.setup(mockRepo, ctx)

			// ACT
			_, err := service.BuyItem(ctx, "twitch", "", "testuser", "Sword", 1)

			// ASSERT
			if tt.expectErr {
				require.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err, tt.description)
			}
		})
	}
}
