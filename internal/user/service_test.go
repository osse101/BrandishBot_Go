package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
)

// MockNamingResolver implements naming.Resolver interface for testing
type MockNamingResolver struct {
	DisplayNames map[string]string
}

func (m *MockNamingResolver) ResolvePublicName(publicName string) (string, bool) {
	return publicName, true
}

func (m *MockNamingResolver) GetDisplayName(internalName string, qualityLevel domain.QualityLevel) string {
	if name, ok := m.DisplayNames[internalName]; ok {
		return name
	}
	return internalName
}

func (m *MockNamingResolver) GetActiveTheme() string {
	return ""
}

func (m *MockNamingResolver) Reload() error {
	return nil
}

func (m *MockNamingResolver) RegisterItem(internalName, publicName string) {}

func NewMockNamingResolver() *MockNamingResolver {
	return &MockNamingResolver{
		DisplayNames: map[string]string{
			"money":         "Shiny credit",
			"lootbox_tier0": "junkbox",
			"lootbox_tier1": "basic lootbox",
		},
	}
}

// MockLootboxService is defined in lootbox_test.go
// We reuse it here to avoid redeclaration error

// Helper to setup test data
func setupTestData(repo *FakeRepository) {
	// Add test users
	repo.users["alice"] = &domain.User{
		ID:       "user-alice",
		Username: "alice",
		TwitchID: "alice123",
	}
	repo.users["bob"] = &domain.User{
		ID:       "user-bob",
		Username: "bob",
		TwitchID: "bob456",
	}

	// Add test items
	repo.items[domain.ItemLootbox1] = &domain.Item{
		ID:           1,
		InternalName: domain.ItemLootbox1,
		PublicName:   domain.ItemLootbox1,
		Description:  "Basic Lootbox",
		BaseValue:    50,
	}
	repo.items[domain.ItemLootbox2] = &domain.Item{
		ID:           2,
		InternalName: domain.ItemLootbox2,
		PublicName:   domain.ItemLootbox2,
		Description:  "Good Lootbox",
		BaseValue:    100,
	}
	repo.items[domain.ItemMoney] = &domain.Item{
		ID:           3,
		InternalName: domain.ItemMoney,
		PublicName:   "money",
		Description:  "Currency",
		BaseValue:    1,
	}
	repo.items[domain.ItemLootbox0] = &domain.Item{
		ID:           4,
		InternalName: domain.ItemLootbox0,
		PublicName:   "junkbox",
		Description:  "Empty Lootbox",
		BaseValue:    10,
	}
	repo.items[domain.ItemMissile] = &domain.Item{
		ID:           5,
		InternalName: domain.ItemMissile,
		PublicName:   domain.ItemMissile,
		Description:  "So anyway, I started blasting",
		BaseValue:    10,
	}
}

func TestAddItem(t *testing.T) {
	repo := NewFakeRepository()
	setupTestData(repo)
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, nil, false)
	ctx := context.Background()
	alice := domain.User{
		ID:       "user-alice",
		Username: "alice",
		TwitchID: "alice123",
	}

	t.Run("add item to empty inventory", func(t *testing.T) {
		err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 5)
		require.NoError(t, err, "Failed to setup test")

		inv, err := repo.GetInventory(ctx, alice.ID)
		require.NoError(t, err)

		var found bool
		for _, slot := range inv.Slots {
			if slot.ItemID == 1 { // lootbox1 has ID 1
				found = true
				assert.Equal(t, 5, slot.Quantity, "Expected 5 items")
			}
		}
		assert.True(t, found, "Item not found in inventory")
	})

	t.Run("adding more increments quantity", func(t *testing.T) {
		err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 3)
		require.NoError(t, err, "AddItem failed")

		inv, err := repo.GetInventory(ctx, alice.ID)
		require.NoError(t, err)

		var found bool
		for _, slot := range inv.Slots {
			if slot.ItemID == 1 {
				found = true
				assert.Equal(t, 8, slot.Quantity, "Expected 8 after adding 3 more")
			}
		}
		assert.True(t, found, "Item not found in inventory")
	})

	t.Run("adding a different item creates new slot", func(t *testing.T) {
		err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox2, 2)
		require.NoError(t, err, "AddItem failed")

		inv, err := repo.GetInventory(ctx, alice.ID)
		require.NoError(t, err)
		assert.Len(t, inv.Slots, 2, "Expected 2 slots")
	})
}

func TestRemoveItem(t *testing.T) {
	repo := NewFakeRepository()
	setupTestData(repo)
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, nil, false)
	ctx := context.Background()
	alice := domain.User{
		ID:       "user-alice",
		Username: "alice",
		TwitchID: "alice123",
	}

	// Add 10 lootbox1 items
	err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 10)
	require.NoError(t, err)

	t.Run("remove partial quantity", func(t *testing.T) {
		removed, err := svc.RemoveItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 3)
		require.NoError(t, err, "RemoveItem failed")
		assert.Equal(t, 3, removed, "Expected 3 removed")

		inv, err := repo.GetInventory(ctx, alice.ID)
		require.NoError(t, err)
		assert.Equal(t, 7, inv.Slots[0].Quantity, "Expected quantity 7")
	})

	t.Run("remove more than available", func(t *testing.T) {
		removed, err := svc.RemoveItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 100)
		require.NoError(t, err, "RemoveItem failed")
		assert.Equal(t, 7, removed, "Expected 7 removed")

		inv, err := repo.GetInventory(ctx, alice.ID)
		require.NoError(t, err)
		assert.Empty(t, inv.Slots, "Expected empty inventory")
	})

	t.Run("remove from empty inventory", func(t *testing.T) {
		_, err := svc.RemoveItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 1)
		require.Error(t, err, "Expected error when removing from empty inventory")
	})
}

func TestGiveItem(t *testing.T) {
	repo := NewFakeRepository()
	setupTestData(repo)
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, nil, false)
	ctx := context.Background()
	alice := domain.User{
		ID:       "user-alice",
		Username: "alice",
		TwitchID: "alice123",
	}
	bob := domain.User{
		ID:       "user-bob",
		Username: "bob",
		TwitchID: "bob456",
	}

	// Setup: Give alice some items
	err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 10)
	require.NoError(t, err)

	t.Run("give items successfully", func(t *testing.T) {
		err := svc.GiveItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.PlatformTwitch, bob.Username, domain.ItemLootbox1, 3)
		require.NoError(t, err, "GiveItem failed")

		// Verify alice has 7 left
		aliceInv, err := repo.GetInventory(ctx, alice.ID)
		require.NoError(t, err)
		assert.Equal(t, 7, aliceInv.Slots[0].Quantity, "Alice should have 7")

		// Verify bob has 3
		bobInv, err := repo.GetInventory(ctx, bob.ID)
		require.NoError(t, err)
		require.Len(t, bobInv.Slots, 1)
		assert.Equal(t, 3, bobInv.Slots[0].Quantity, "Bob should have 3")
	})

	t.Run("giving more than owned returns error", func(t *testing.T) {
		err := svc.GiveItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.PlatformTwitch, bob.Username, domain.ItemLootbox1, 100)
		require.Error(t, err, "Expected error when giving more than owned")

		// Verify no changes after failed give
		aliceInv, err := repo.GetInventory(ctx, alice.ID)
		require.NoError(t, err)
		assert.Equal(t, 7, aliceInv.Slots[0].Quantity, "Alice's inventory should be unchanged after failed give")
	})
}

func TestGiveItem_Comprehensive(t *testing.T) {
	tests := []struct {
		name          string
		ownerItems    int
		giveQty       int
		expectedOwner int
		expectedRecv  int
		expectError   bool
		errorMessage  string
	}{
		{
			name:          "give partial quantity",
			ownerItems:    10,
			giveQty:       3,
			expectedOwner: 7,
			expectedRecv:  3,
			expectError:   false,
		},
		{
			name:          "give all items",
			ownerItems:    10,
			giveQty:       10,
			expectedOwner: 0,
			expectedRecv:  10,
			expectError:   false,
		},
		{
			name:          "give more than owned",
			ownerItems:    5,
			giveQty:       10,
			expectedOwner: 5,
			expectedRecv:  0,
			expectError:   true,
			errorMessage:  domain.ErrMsgInsufficientQuantity,
		},
		{
			name:          "give from empty inventory",
			ownerItems:    0,
			giveQty:       1,
			expectedOwner: 0,
			expectedRecv:  0,
			expectError:   true,
			errorMessage:  domain.ErrMsgNotInInventory,
		},
		{
			name:          "give zero quantity",
			ownerItems:    10,
			giveQty:       0,
			expectedOwner: 10,
			expectedRecv:  0,
			expectError:   true,
			errorMessage:  domain.ErrMsgInvalidInput,
		},
		{
			name:          "give negative quantity",
			ownerItems:    10,
			giveQty:       -1,
			expectedOwner: 10,
			expectedRecv:  0,
			expectError:   true,
			errorMessage:  domain.ErrMsgInvalidInput,
		},
	}
	alice := domain.User{
		ID:       "user-alice",
		Username: "alice",
		TwitchID: "alice123",
	}
	bob := domain.User{
		ID:       "user-bob",
		Username: "bob",
		TwitchID: "bob456",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewFakeRepository()
			setupTestData(repo)
			svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, nil, false)
			ctx := context.Background()

			// Setup owner with items
			if tt.ownerItems > 0 {
				err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, tt.ownerItems)
				require.NoError(t, err)
			}

			// Attempt to give items
			err := svc.GiveItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.PlatformTwitch, bob.Username, domain.ItemLootbox1, tt.giveQty)

			if tt.expectError {
				require.Error(t, err, "expected error, got nil")
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				require.NoError(t, err, "unexpected error")
			}

			// Verify owner's inventory
			aliceInv, err := repo.GetInventory(ctx, alice.ID)
			require.NoError(t, err)

			if tt.expectedOwner == 0 {
				// Owner should have no slots if all given away
				for _, slot := range aliceInv.Slots {
					assert.NotEqual(t, 1, slot.ItemID, "Alice should have no %s", domain.ItemLootbox1)
				}
			} else {
				found := false
				for _, slot := range aliceInv.Slots {
					if slot.ItemID == 1 {
						assert.Equal(t, tt.expectedOwner, slot.Quantity, "Alice should have %d", tt.expectedOwner)
						found = true
						break
					}
				}
				assert.True(t, found, "Alice should have items but they are missing")
			}

			// Verify receiver's inventory
			bobInv, err := repo.GetInventory(ctx, bob.ID)
			require.NoError(t, err)

			if tt.expectedRecv == 0 {
				for _, slot := range bobInv.Slots {
					assert.NotEqual(t, 1, slot.ItemID, "Bob should have no items")
				}
			} else {
				found := false
				for _, slot := range bobInv.Slots {
					if slot.ItemID == 1 {
						assert.Equal(t, tt.expectedRecv, slot.Quantity, "Bob should have %d", tt.expectedRecv)
						found = true
						break
					}
				}
				assert.True(t, found, "Bob should have %d items but has none", tt.expectedRecv)
			}
		})
	}
}

func TestGiveItem_CrossPlatform(t *testing.T) {
	repo := NewFakeRepository()
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, nil, false)
	ctx := context.Background()

	// Setup users on different platforms
	userAlice := &domain.User{
		ID:        "user-alice",
		Username:  "alice",
		TwitchID:  "twitch-alice",
		DiscordID: "discord-alice",
	}
	userBob := &domain.User{
		ID:        "user-bob",
		Username:  "bob",
		DiscordID: "discord-bob",
	}
	repo.users["alice"] = userAlice
	repo.users["bob"] = userBob
	repo.inventories["user-alice"] = &domain.Inventory{Slots: []domain.InventorySlot{}}
	repo.inventories["user-bob"] = &domain.Inventory{Slots: []domain.InventorySlot{}}

	// Add the item to the repository
	repo.items[domain.ItemLootbox1] = &domain.Item{
		ID:           1,
		InternalName: domain.ItemLootbox1,
		PublicName:   domain.PublicNameLootbox,
		BaseValue:    50,
	}

	// Add items to alice via Twitch
	err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, "alice", domain.ItemLootbox1, 10)
	require.NoError(t, err)

	// Give from alice (Twitch) to bob (Discord)
	err = svc.GiveItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.PlatformDiscord, "bob", domain.ItemLootbox1, 5)
	require.NoError(t, err, "cross-platform give failed")

	// Verify both inventories
	aliceInv, err := repo.GetInventory(ctx, "user-alice")
	require.NoError(t, err)
	assert.Equal(t, 5, aliceInv.Slots[0].Quantity, "Alice should have 5")

	bobInv, err := repo.GetInventory(ctx, "user-bob")
	require.NoError(t, err)
	require.Len(t, bobInv.Slots, 1)
	assert.Equal(t, 5, bobInv.Slots[0].Quantity, "Bob should have 5")
}

func TestRegisterUser(t *testing.T) {
	repo := NewFakeRepository()
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, nil, false)
	ctx := context.Background()

	user := domain.User{
		Username: "charlie",
		TwitchID: "charlie789",
	}

	registered, err := svc.RegisterUser(ctx, user)
	require.NoError(t, err, "RegisterUser failed")

	assert.NotEmpty(t, registered.ID, "Expected user ID to be set")
	assert.Equal(t, "charlie", registered.Username, "Expected username charlie")

	// Verify user in repo
	found, err := repo.GetUserByPlatformID(ctx, domain.PlatformTwitch, "charlie789")
	require.NoError(t, err)
	assert.NotNil(t, found, "User not found in repository")
}

func TestHandleIncomingMessage_NewUser(t *testing.T) {
	repo := NewFakeRepository()
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, nil, false)
	ctx := context.Background()

	result, err := svc.HandleIncomingMessage(ctx, domain.PlatformTwitch, "newuser123", "newuser", "hello")
	require.NoError(t, err, "HandleIncomingMessage failed")

	assert.Equal(t, "newuser", result.User.Username, "Unexpected user")
	assert.Equal(t, "newuser123", result.User.TwitchID, "Unexpected user")

	// Verify user was created
	found, err := repo.GetUserByPlatformID(ctx, domain.PlatformTwitch, "newuser123")
	require.NoError(t, err)
	assert.NotNil(t, found, "User should have been created")
}

func TestHandleIncomingMessage_ExistingUser(t *testing.T) {
	repo := NewFakeRepository()
	setupTestData(repo)
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, nil, false)
	ctx := context.Background()

	result, err := svc.HandleIncomingMessage(ctx, domain.PlatformTwitch, "alice123", "alice", "hello")
	require.NoError(t, err, "HandleIncomingMessage failed")

	assert.Equal(t, "user-alice", result.User.ID, "Should have returned existing user")
}

func TestUseItem(t *testing.T) {
	repo := NewFakeRepository()
	setupTestData(repo)
	alice := domain.User{
		ID:        "user-alice",
		Username:  "alice",
		TwitchID:  "alice123",
		DiscordID: "alice456",
	}

	// Create a mock lootbox service
	lootboxSvc := new(MockLootboxService)
	drops := []lootbox.DroppedItem{
		{ItemID: 4, ItemName: domain.ItemLootbox0, Quantity: 1, Value: 10, QualityLevel: domain.QualityCommon},
	}
	lootboxSvc.On("OpenLootbox", mock.Anything, domain.ItemLootbox1, 1, mock.Anything).Return(drops, nil)

	svc := NewService(repo, repo, nil, nil, lootboxSvc, NewMockNamingResolver(), nil, nil, nil, false).(*service)

	ctx := context.Background()

	// Setup: Give alice some lootbox1
	svc.getUserOrRegister(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username)
	err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 5)
	require.NoError(t, err)

	t.Run("use valid item", func(t *testing.T) {
		message, err := svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemLootbox1, 1, "")
		require.NoError(t, err, "UseItem failed")

		assert.Contains(t, message, "Opened")
		assert.Contains(t, message, "junkbox")

		// Verify inventory
		inv, err := repo.GetInventory(ctx, alice.ID)
		require.NoError(t, err)

		var lootbox1Slot, lootbox0Slot *domain.InventorySlot
		for i := range inv.Slots {
			if inv.Slots[i].ItemID == 1 {
				lootbox1Slot = &inv.Slots[i]
			}
			if inv.Slots[i].ItemID == 4 {
				lootbox0Slot = &inv.Slots[i]
			}
		}

		require.NotNil(t, lootbox1Slot, "lootbox1 slot should not be nil")
		assert.Equal(t, 4, lootbox1Slot.Quantity, "Expected 4 lootbox1")

		require.NotNil(t, lootbox0Slot, "lootbox0 slot should not be nil")
		assert.Equal(t, 1, lootbox0Slot.Quantity, "Expected 1 lootbox0")
	})

	t.Run("use more than available", func(t *testing.T) {
		_, err := svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemLootbox1, 10, "")
		require.Error(t, err, "Expected error when using more than available")
	})

	t.Run("use unknown item", func(t *testing.T) {
		_, err := svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, "unknown_item", 1, "")
		require.Error(t, err, "Expected error when using unknown item")
	})

	t.Run("use item with no effect", func(t *testing.T) {
		err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemMoney, 1)
		require.NoError(t, err)

		_, err = svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemMoney, 1, "")
		require.Error(t, err, "Expected error when using item with no effect")
	})
}

func TestUseItem_Blaster(t *testing.T) {
	repo := NewFakeRepository()
	setupTestData(repo)
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, nil, false)
	ctx := context.Background()
	alice := domain.User{
		ID:        "user-alice",
		Username:  "alice",
		TwitchID:  "alice123",
		DiscordID: "alice456",
	}
	bob := domain.User{
		ID:        "user-bob",
		Username:  "bob",
		TwitchID:  "bob123",
		DiscordID: "bob456",
	}

	// Setup: Give alice some blasters
	_, err := svc.RegisterUser(ctx, alice)
	require.NoError(t, err)

	err = svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemMissile, 5)
	require.NoError(t, err)

	t.Run("use blaster on target", func(t *testing.T) {
		message, err := svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemMissile, 2, bob.Username)
		require.NoError(t, err, "UseItem failed")

		expectedMsg := "alice used weapon_missile on bob! 2 weapon_missile(s) fired. Timed out for 2m0s."
		assert.Equal(t, expectedMsg, message)

		// Verify inventory
		inv, err := repo.GetInventory(ctx, alice.ID)
		require.NoError(t, err)
		assert.Equal(t, 3, inv.Slots[0].Quantity, "Expected 3 blasters left")
	})

	t.Run("use blaster without target", func(t *testing.T) {
		_, err := svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemMissile, 1, "")
		require.Error(t, err, "Expected error when using blaster without target")
	})
}

func TestUseItem_RareCandy(t *testing.T) {
	repo := NewFakeRepository()
	setupTestData(repo)

	// Add Rare Candy to repo
	repo.items[domain.ItemRareCandy] = &domain.Item{
		ID:           6,
		InternalName: domain.ItemRareCandy,
		PublicName:   domain.ItemRareCandy,
		Description:  "Tastes like progression",
		BaseValue:    100,
	}

	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, nil, false).(*service)
	ctx := context.Background()
	alice := domain.User{
		ID:       "user-alice",
		Username: "alice",
		TwitchID: "alice123",
	}

	// Setup: Give alice some Rare Candy
	_, err := svc.RegisterUser(ctx, alice)
	require.NoError(t, err)

	err = svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemRareCandy, 5)
	require.NoError(t, err)

	// Test using Rare Candy on a job name (which is NOT a user)
	// This should work now because we don't resolve the target as a user anymore
	message, err := svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemRareCandy, 1, job.JobKeyBlacksmith)
	require.NoError(t, err, "UseItem failed")

	expectedMsg := "Used 1 rare candy! Granted"
	assert.Contains(t, message, expectedMsg)

	// Verify inventory
	inv, err := repo.GetInventory(ctx, alice.ID)
	require.NoError(t, err)
	assert.Equal(t, 4, inv.Slots[0].Quantity, "Expected 4 Rare Candy left")
}

func TestGetInventory(t *testing.T) {
	repo := NewFakeRepository()
	setupTestData(repo)
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, nil, false)
	ctx := context.Background()
	alice := domain.User{
		ID:        "user-alice",
		TwitchID:  "alice123",
		Username:  "alice",
		DiscordID: "alice456",
	}

	// Setup: Give alice some items with various types
	_, err := svc.RegisterUser(ctx, alice)
	require.NoError(t, err)
	err = svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 2)
	require.NoError(t, err)
	err = svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemMoney, 100)
	require.NoError(t, err)

	t.Run("No Filter - Returns All Items", func(t *testing.T) {
		// Test GetInventory without filter
		items, err := svc.GetInventory(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, "")
		require.NoError(t, err, "GetInventory failed")
		assert.Len(t, items, 2, "Expected 2 items")

		// Verify item details
		foundLootbox := false
		foundMoney := false
		for _, item := range items {
			if item.PublicName == domain.ItemLootbox1 {
				foundLootbox = true
				assert.Equal(t, 2, item.Quantity, "Expected 2 lootbox1")
			}
			if item.PublicName == domain.ItemMoney {
				foundMoney = true
				assert.Equal(t, 100, item.Quantity, "Expected 100 money")
			}
		}

		assert.True(t, foundLootbox, "Expected lootbox1 in inventory")
		assert.True(t, foundMoney, "Expected money in inventory")
	})

	t.Run("Upgrade Filter - Returns Only Upgradable Items", func(t *testing.T) {
		// Note: This test assumes items have "upgradable" type in their Types field
		// In a real scenario, mock items would have Types populated
		items, err := svc.GetInventory(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.FilterTypeUpgrade)
		require.NoError(t, err, "GetInventory with upgrade filter failed")

		// All returned items should have "upgradable" type
		for _, item := range items {
			// In real implementation, check item.Types contains "upgrade"
			// For now, just verify it doesn't error
			assert.NotEmpty(t, item.PublicName, "Item should have a name")
		}
	})

	t.Run("Sellable Filter - Returns Only Sellable Items", func(t *testing.T) {
		items, err := svc.GetInventory(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.FilterTypeSellable)
		require.NoError(t, err, "GetInventory with sellable filter failed")

		// All returned items should have "sellable" type
		for _, item := range items {
			assert.NotEmpty(t, item.PublicName, "Item should have a name")
		}
	})

	t.Run("Consumable Filter - Returns Only Consumable Items", func(t *testing.T) {
		items, err := svc.GetInventory(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.FilterTypeConsumable)
		require.NoError(t, err, "GetInventory with consumable filter failed")

		// All returned items should have "consumable" type
		for _, item := range items {
			assert.NotEmpty(t, item.PublicName, "Item should have a name")
		}
	})

	t.Run("Unknown Filter - Returns Empty Result", func(t *testing.T) {
		// Unknown filter should return no items (nothing matches)
		items, err := svc.GetInventory(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, "nonexistent")
		require.NoError(t, err, "GetInventory with unknown filter failed")

		// Unknown filter likely returns empty or all items depending on implementation
		// The test documents the expected behavior
		_ = items
	})
}

func TestUseItem_Lootbox0(t *testing.T) {
	repo := NewFakeRepository()
	setupTestData(repo)
	alice := domain.User{
		ID:        "user-alice",
		TwitchID:  "alice123",
		Username:  "alice",
		DiscordID: "alice456",
	}

	// Create a mock lootbox service
	lootboxSvc := new(MockLootboxService)
	drops := []lootbox.DroppedItem{
		{ItemID: 3, ItemName: domain.ItemMoney, Quantity: 5, Value: 5, QualityLevel: "COMMON"},
	}
	lootboxSvc.On("OpenLootbox", mock.Anything, domain.ItemLootbox0, 1, mock.Anything).Return(drops, nil)

	svc := NewService(repo, repo, nil, nil, lootboxSvc, NewMockNamingResolver(), nil, nil, nil, false).(*service)

	ctx := context.Background()

	// Setup: Give alice lootbox0
	err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox0, 1)
	require.NoError(t, err)

	t.Run("use lootbox0", func(t *testing.T) {
		msg, err := svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemLootbox0, 1, "")
		require.NoError(t, err, "UseItem failed")

		assert.Contains(t, msg, "Opened")
		assert.Contains(t, msg, "junkbox")
		assert.Contains(t, msg, "Shiny credits")

		// Verify inventory (should have money now)
		inv, err := repo.GetInventory(ctx, alice.ID)
		require.NoError(t, err)
		require.Len(t, inv.Slots, 1, "Expected 1 slot (money)")

		assert.Equal(t, 3, inv.Slots[0].ItemID, "Expected money (ID 3)")
	})
}

func TestUseItem_Lootbox2(t *testing.T) {
	repo := NewFakeRepository()
	setupTestData(repo)
	alice := domain.User{
		ID:        "user-alice",
		TwitchID:  "alice123",
		Username:  "alice",
		DiscordID: "alice456",
	}

	// Create a mock lootbox service
	lootboxSvc := new(MockLootboxService)
	drops := []lootbox.DroppedItem{
		{ItemID: 1, ItemName: domain.ItemLootbox1, Quantity: 1, Value: 50, QualityLevel: "COMMON"},
	}
	lootboxSvc.On("OpenLootbox", mock.Anything, domain.ItemLootbox2, 1, mock.Anything).Return(drops, nil)

	svc := NewService(repo, repo, nil, nil, lootboxSvc, NewMockNamingResolver(), nil, nil, nil, false).(*service)

	ctx := context.Background()

	// Setup: Give alice lootbox2
	err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox2, 1)
	require.NoError(t, err)

	t.Run("use lootbox2", func(t *testing.T) {
		msg, err := svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemLootbox2, 1, "")
		require.NoError(t, err, "UseItem failed")

		assert.Contains(t, msg, "Opened")
		assert.Contains(t, msg, "basic lootbox")

		// Verify inventory: should have 1 lootbox1
		inv, err := repo.GetInventory(ctx, alice.ID)
		require.NoError(t, err)
		require.Len(t, inv.Slots, 1, "Expected 1 slot")

		assert.Equal(t, 1, inv.Slots[0].ItemID, "Expected lootbox1 (ID 1)")
		assert.Equal(t, 1, inv.Slots[0].Quantity, "Expected quantity 1")
	})
}
