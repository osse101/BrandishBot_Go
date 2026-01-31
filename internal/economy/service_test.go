package economy

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// MockRepository implements Repository interface for testing
type MockRepository struct {
	mock.Mock
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

func (m *MockRepository) BeginTx(ctx context.Context) (repository.EconomyTx, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(repository.EconomyTx), args.Error(1)
}

func (m *MockRepository) GetBuyablePrices(ctx context.Context) ([]domain.Item, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Item), args.Error(1)
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
		ID:           id,
		InternalName: name,
		BaseValue:    value,
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
	mockTx := &MockTx{}
	service := NewService(mockRepo, nil, nil, nil)
	ctx := context.Background()

	user := createTestUser()
	item := createTestItem(10, domain.PublicNameLootbox, 100)
	moneyItem := createMoneyItem()
	inventory := &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 10, Quantity: 5}, // 5 lootboxes
			{ItemID: 1, Quantity: 50}, // 50 money
		},
	}

	mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil)
	mockRepo.On("GetItemByName", ctx, domain.PublicNameLootbox).Return(item, nil)
	mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)
	mockRepo.On("BeginTx", ctx).Return(mockTx, nil)
	mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)
	mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", ctx).Return(nil)
	mockTx.On("Rollback", ctx).Return(nil)

	// ACT
	moneyGained, quantitySold, err := service.SellItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameLootbox, 3)

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, 120, moneyGained, "Should receive correct money (3 * 40) - 40% of base value")
	assert.Equal(t, 3, quantitySold, "Should sell requested quantity")
	mockRepo.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

// CASE 2: WORST CASE - Boundary conditions
func TestSellItem_SellAllItems(t *testing.T) {
	// ARRANGE - User sells every last item they have
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, nil, nil, nil)
	ctx := context.Background()

	user := createTestUser()
	item := createTestItem(10, domain.PublicNameJunkbox, 5)
	moneyItem := createMoneyItem()
	inventory := &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 10, Quantity: 100}, // Max stack
		},
	}

	mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil)
	mockRepo.On("GetItemByName", ctx, domain.PublicNameJunkbox).Return(item, nil)
	mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)

	// Add mock transaction expectations
	mockTx := &MockTx{}
	mockRepo.On("BeginTx", ctx).Return(mockTx, nil)
	mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)
	mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", ctx).Return(nil)
	mockTx.On("Rollback", ctx).Return(nil)

	// ACT
	moneyGained, quantitySold, err := service.SellItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameJunkbox, 100)

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, 200, moneyGained, "Should receive correct money (100 * 2) - 40% of base value (5)")
	assert.Equal(t, 100, quantitySold, "Should sell all items")
	mockRepo.AssertExpectations(t)
}

// CASE 3: EDGE CASE - Partial quantity available
func TestSellItem_PartialQuantity(t *testing.T) {
	// ARRANGE - User requests 100 but only has 30
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, nil, nil, nil)
	ctx := context.Background()

	user := createTestUser()
	item := createTestItem(10, domain.PublicNameMissile, 20)
	moneyItem := createMoneyItem()
	inventory := &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: 10, Quantity: 30}, // Only 30 available
		},
	}

	mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil)
	mockRepo.On("GetItemByName", ctx, domain.PublicNameMissile).Return(item, nil)
	mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)

	mockTx := &MockTx{}
	mockRepo.On("BeginTx", ctx).Return(mockTx, nil)
	mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)
	mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", ctx).Return(nil)
	mockTx.On("Rollback", ctx).Return(nil)

	// ACT - Request 100 but only have 30
	moneyGained, quantitySold, err := service.SellItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameMissile, 100)

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, 240, moneyGained, "Should sell what's available (30 * 8) - 40% of base value (20)")
	assert.Equal(t, 30, quantitySold, "Should return actual quantity sold")
}

// CASE 4: INVALID CASE - Bad inputs
func TestSellItem_InvalidInputs(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*MockRepository)
		username      string
		itemName      string
		expectErr     bool
		expectedError error
		description   string
	}{
		{
			name: domain.ErrMsgUserNotFound,
			setup: func(m *MockRepository) {
				m.On("GetUserByPlatformID", mock.Anything, domain.PlatformTwitch, "").
					Return(nil, nil)
			},
			username:      "nonexistent",
			itemName:      domain.PublicNameLootbox,
			expectErr:     true,
			expectedError: domain.ErrUserNotFound,
			description:   "Should fail when user does not exist",
		},
		{
			name: domain.ErrMsgItemNotFound,
			setup: func(m *MockRepository) {
				user := createTestUser()
				m.On("GetUserByPlatformID", mock.Anything, domain.PlatformTwitch, "").Return(user, nil)
				m.On("GetItemByName", mock.Anything, "InvalidItem").Return(nil, nil)
			},
			username:      "testuser",
			itemName:      "InvalidItem",
			expectErr:     true,
			expectedError: domain.ErrItemNotFound,
			description:   "Should fail when item does not exist",
		},
		{
			name: "item not in inventory",
			setup: func(m *MockRepository) {
				user := createTestUser()
				item := createTestItem(10, domain.PublicNameLootbox, 100)
				moneyItem := createMoneyItem()
				emptyInventory := &domain.Inventory{Slots: []domain.InventorySlot{}}

				m.On("GetUserByPlatformID", mock.Anything, domain.PlatformTwitch, "").Return(user, nil)
				m.On("GetItemByName", mock.Anything, domain.PublicNameLootbox).Return(item, nil)
				m.On("GetItemByName", mock.Anything, domain.ItemMoney).Return(moneyItem, nil)

				mockTx := &MockTx{}
				m.On("BeginTx", mock.Anything).Return(mockTx, nil)
				mockTx.On("GetInventory", mock.Anything, user.ID).Return(emptyInventory, nil)
				mockTx.On("Rollback", mock.Anything).Return(nil)
			},
			username:      "testuser",
			itemName:      domain.PublicNameLootbox,
			expectErr:     true,
			expectedError: domain.ErrNotInInventory,
			description:   "Should fail when user does not own the item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			mockRepo := &MockRepository{}
			service := NewService(mockRepo, nil, nil, nil)
			ctx := context.Background()
			tt.setup(mockRepo)

			// ACT
			_, _, err := service.SellItem(ctx, domain.PlatformTwitch, "", tt.username, tt.itemName, 1)

			// ASSERT
			if tt.expectErr {
				require.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
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
			service := NewService(mockRepo, nil, nil, nil)
			ctx := context.Background()

			user := createTestUser()
			item := createTestItem(10, domain.PublicNameLootbox, 100)
			moneyItem := createMoneyItem()
			inventory := createInventoryWithItem(10, 100)

			mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil)
			mockRepo.On("GetItemByName", ctx, domain.PublicNameLootbox).Return(item, nil)
			mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)

			mockTx := &MockTx{}
			// Only expect Tx if validation passes
			if !tt.expectErr || (tt.name != "negative quantity" && tt.name != "zero quantity" && tt.name != "over max boundary") {
				// Actually validation happens BEFORE everything.
				// Wait, validateBuyRequest is called first. If fails, NO calls.
				// But mock setup above sets expectations.

				// Logic:
				// If validation fails (quantity <= 0 or > max), we return error. Tx NOT called.
				// The test setups generic expectations.
				// But we need conditional expectations for Tx.
				if !tt.expectErr {
					mockRepo.On("BeginTx", ctx).Return(mockTx, nil)
					mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)
					mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)
					mockTx.On("Commit", ctx).Return(nil)
					mockTx.On("Rollback", ctx).Return(nil)
				}
			}

			// ACT
			_, _, err := service.SellItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameLootbox, tt.quantity)

			// ASSERT
			if tt.expectErr {
				require.Error(t, err, tt.description)
				if tt.name == "over max boundary" {
					assert.Contains(t, err.Error(), "exceeds maximum allowed")
				} else {
					assert.ErrorIs(t, err, domain.ErrInvalidInput)
				}
			} else {
				require.NoError(t, err, tt.description)
			}
		})
	}
}

// CASE 5: HOSTILE CASE - Database errors and malicious scenarios
func TestSellItem_DatabaseErrors(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*MockRepository, context.Context)
		expectErr     bool
		expectedError error
		description   string
	}{
		{
			name: "database error on GetUser",
			setup: func(m *MockRepository, ctx context.Context) {
				dbError := errors.New("database connection lost")
				m.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(nil, dbError)
			},
			expectErr:     true,
			expectedError: domain.ErrFailedToGetUser,
			description:   "Should fail when database connection is lost during user fetch",
		},
		{
			name: "database error on UpdateInventory",
			setup: func(m *MockRepository, ctx context.Context) {
				user := createTestUser()
				item := createTestItem(10, domain.PublicNameLootbox, 100)
				moneyItem := createMoneyItem()
				inventory := createInventoryWithItem(10, 5)

				m.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil)
				m.On("GetItemByName", ctx, domain.PublicNameLootbox).Return(item, nil)
				m.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)
				mockTx := &MockTx{}
				m.On("BeginTx", ctx).Return(mockTx, nil)
				mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)
				mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).
					Return(errors.New(domain.ErrMsgDeadlockDetected))
				mockTx.On("Rollback", ctx).Return(nil).Maybe()
			},
			expectErr:     true,
			expectedError: domain.ErrFailedToUpdateInventory,
			description:   "Should fail when database deadlock occurs during inventory update",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			mockRepo := &MockRepository{}
			service := NewService(mockRepo, nil, nil, nil)
			ctx := context.Background()
			tt.setup(mockRepo, ctx)

			// ACT
			_, _, err := service.SellItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameLootbox, 1)

			// ASSERT
			if tt.expectErr {
				require.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
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
	service := NewService(mockRepo, nil, nil, nil)
	ctx := context.Background()

	expectedItems := []domain.Item{
		{ID: 10, InternalName: domain.PublicNameLootbox, BaseValue: 100},
		{ID: 20, InternalName: domain.PublicNameMissile, BaseValue: 50},
	}

	mockRepo.On("GetSellablePrices", ctx).Return(expectedItems, nil)

	// ACT
	items, err := service.GetSellablePrices(ctx)

	// ASSERT
	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, domain.PublicNameLootbox, items[0].InternalName)
	assert.Equal(t, 100, items[0].BaseValue, "Base value should remain unchanged (buy price)")
	require.NotNil(t, items[0].SellPrice, "Sell price should be populated")
	assert.Equal(t, 40, *items[0].SellPrice, "Sell price should be 40% of base value")
	require.NotNil(t, items[1].SellPrice, "Sell price should be populated")
	assert.Equal(t, 20, *items[1].SellPrice, "Sell price should be 40% of base value")
	mockRepo.AssertExpectations(t)
}

func TestGetSellablePrices_DatabaseError(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, nil, nil, nil)
	ctx := context.Background()

	mockRepo.On("GetSellablePrices", ctx).
		Return(nil, errors.New(domain.ErrMsgConnectionTimeout))

	// ACT
	items, err := service.GetSellablePrices(ctx)

	// ASSERT
	require.Error(t, err)
	assert.Nil(t, items)
	assert.Contains(t, err.Error(), domain.ErrMsgConnectionTimeout)
}

// =============================================================================
// BuyItem Tests - 5-Case Testing Model
// =============================================================================

// CASE 1: BEST CASE
func TestBuyItem_Success(t *testing.T) {
	// ARRANGE
	mockRepo := &MockRepository{}
	service := NewService(mockRepo, nil, nil, nil)
	ctx := context.Background()

	user := createTestUser()
	item := createTestItem(10, domain.PublicNameLootbox, 100)
	moneyItem := createMoneyItem()
	inventory := &domain.Inventory{
		Slots: []domain.InventorySlot{
			{ItemID: moneyItem.ID, Quantity: 500}, // Enough for 5 items
		},
	}

	mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil)
	mockRepo.On("GetItemByName", ctx, domain.PublicNameLootbox).Return(item, nil)
	mockRepo.On("IsItemBuyable", ctx, domain.PublicNameLootbox).Return(true, nil)
	mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)

	// Transaction expectations
	mockTx := &MockTx{}
	mockRepo.On("BeginTx", ctx).Return(mockTx, nil)
	mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)
	mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)
	mockTx.On("Commit", ctx).Return(nil)
	mockTx.On("Rollback", ctx).Return(nil)

	// ACT
	purchased, err := service.BuyItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameLootbox, 3)

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, 3, purchased)
	mockRepo.AssertExpectations(t)
	mockTx.AssertExpectations(t)
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
			service := NewService(mockRepo, nil, nil, nil)
			ctx := context.Background()

			user := createTestUser()
			item := createTestItem(10, domain.PublicNameLootbox, tt.itemPrice)
			moneyItem := createMoneyItem()
			inventory := createInventoryWithMoney(tt.moneyBalance)

			mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil)
			mockRepo.On("GetItemByName", ctx, domain.PublicNameLootbox).Return(item, nil)
			mockRepo.On("IsItemBuyable", ctx, domain.PublicNameLootbox).Return(true, nil)
			mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)

			mockTx := &MockTx{}

			// Determine if Tx is reached
			// Validation happens first. Boundaries check validation.
			// "no money" fails domain.ErrMsgInsufficientFunds INSIDE GetInventory logic -> so reached Tx.
			// "not enough for one" -> reached Tx.

			mockRepo.On("BeginTx", ctx).Return(mockTx, nil)
			mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)

			if !tt.expectErr {
				mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil).Maybe()
				mockTx.On("Commit", ctx).Return(nil)
			}
			mockTx.On("Rollback", ctx).Return(nil)

			// ACT
			purchased, err := service.BuyItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameLootbox, tt.quantityWanted)

			// ASSERT
			if tt.expectErr {
				require.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), domain.ErrMsgInsufficientFunds)
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
			service := NewService(mockRepo, nil, nil, nil)
			ctx := context.Background()

			user := createTestUser()
			item := createTestItem(10, domain.PublicNameLootbox, 100)
			moneyItem := createMoneyItem()
			inventory := createInventoryWithMoney(10000000)

			mockRepo.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil)
			mockRepo.On("GetItemByName", ctx, domain.PublicNameLootbox).Return(item, nil)
			mockRepo.On("IsItemBuyable", ctx, domain.PublicNameLootbox).Return(true, nil)
			mockRepo.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)

			// Setup Tx if quantity is valid
			if !tt.expectErr {
				mockTx := &MockTx{}
				mockRepo.On("BeginTx", ctx).Return(mockTx, nil)
				mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)
				mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).Return(nil)
				mockTx.On("Commit", ctx).Return(nil)
				mockTx.On("Rollback", ctx).Return(nil).Maybe()
			}

			// ACT
			purchased, err := service.BuyItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameLootbox, tt.quantity)

			// ASSERT
			if tt.expectErr {
				require.Error(t, err, tt.description)
				if tt.name == "over max boundary" {
					assert.Contains(t, err.Error(), "exceeds maximum allowed")
				} else {
					assert.ErrorIs(t, err, domain.ErrInvalidInput)
				}
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
		name          string
		setup         func(*MockRepository, context.Context)
		expectErr     bool
		expectedError error
		description   string
	}{
		{
			name: "user not found",
			setup: func(m *MockRepository, ctx context.Context) {
				m.On("GetUserByPlatformID", mock.Anything, domain.PlatformTwitch, "").Return(nil, nil)
			},
			expectErr:     true,
			expectedError: domain.ErrUserNotFound,
			description:   "Should fail when user does not exist",
		},
		{
			name: "item not found",
			setup: func(m *MockRepository, ctx context.Context) {
				user := createTestUser()
				m.On("GetUserByPlatformID", mock.Anything, domain.PlatformTwitch, "").Return(user, nil)
				m.On("GetItemByName", mock.Anything, "InvalidItem").Return(nil, nil)
			},
			expectErr:     true,
			expectedError: domain.ErrItemNotFound,
			description:   "Should fail when item does not exist",
		},
		{
			name: "item not buyable",
			setup: func(m *MockRepository, ctx context.Context) {
				user := createTestUser()
				item := createTestItem(10, "InvalidItem", 100)
				m.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil)
				m.On("GetItemByName", ctx, "InvalidItem").Return(item, nil)

				mockTx := &MockTx{}
				m.On("BeginTx", ctx).Return(mockTx, nil)
				m.On("IsItemBuyable", ctx, "InvalidItem").Return(false, nil)
				mockTx.On("Rollback", ctx).Return(nil).Maybe()
			},
			expectErr:     true,
			expectedError: domain.ErrNotBuyable,
			description:   "Should fail when item is not buyable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			mockRepo := &MockRepository{}
			service := NewService(mockRepo, nil, nil, nil)
			ctx := context.Background()
			tt.setup(mockRepo, ctx)

			// ACT
			_, err := service.BuyItem(ctx, domain.PlatformTwitch, "", "testuser", "InvalidItem", 1)

			// ASSERT
			if tt.expectErr {
				require.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				require.NoError(t, err, tt.description)
			}
		})
	}
}

// CASE 5: HOSTILE CASE - Database errors
func TestBuyItem_DatabaseErrors(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*MockRepository, context.Context)
		expectErr     bool
		expectedError error
		description   string
	}{
		{
			name: "database error on UpdateInventory",
			setup: func(m *MockRepository, ctx context.Context) {
				user := createTestUser()
				item := createTestItem(10, domain.PublicNameLootbox, 100)
				moneyItem := createMoneyItem()
				inventory := createInventoryWithMoney(500)

				m.On("GetUserByPlatformID", ctx, domain.PlatformTwitch, "").Return(user, nil)
				m.On("GetItemByName", ctx, domain.PublicNameLootbox).Return(item, nil)
				m.On("IsItemBuyable", ctx, domain.PublicNameLootbox).Return(true, nil)
				m.On("GetItemByName", ctx, domain.ItemMoney).Return(moneyItem, nil)
				mockTx := &MockTx{}
				m.On("BeginTx", ctx).Return(mockTx, nil)
				mockTx.On("GetInventory", ctx, user.ID).Return(inventory, nil)
				mockTx.On("UpdateInventory", ctx, user.ID, mock.Anything).
					Return(errors.New(domain.ErrMsgDeadlockDetected))
				mockTx.On("Rollback", ctx).Return(nil).Maybe()
			},
			expectErr:     true,
			expectedError: domain.ErrFailedToUpdateInventory,
			description:   "Should fail when database deadlock occurs during inventory update",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			mockRepo := &MockRepository{}
			service := NewService(mockRepo, nil, nil, nil)
			ctx := context.Background()
			tt.setup(mockRepo, ctx)

			// ACT
			_, err := service.BuyItem(ctx, domain.PlatformTwitch, "", "testuser", domain.PublicNameLootbox, 1)

			// ASSERT
			if tt.expectErr {
				require.Error(t, err, tt.description)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				require.NoError(t, err, tt.description)
			}
		})
	}
}
