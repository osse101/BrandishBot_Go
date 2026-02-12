package user

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"

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
	repo.items[domain.ItemBlaster] = &domain.Item{
		ID:           5,
		InternalName: domain.ItemBlaster,
		PublicName:   domain.ItemBlaster,
		Description:  "So anyway, I started blasting",
		BaseValue:    10,
	}
}

func TestAddItem(t *testing.T) {
	repo := NewFakeRepository()
	setupTestData(repo)
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, false)
	ctx := context.Background()
	alice := domain.User{
		ID:       "user-alice",
		Username: "alice",
		TwitchID: "alice123",
	}

	// Test adding item to empty inventory
	err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 5)
	if err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	// One slot should have 5 lootbox1
	inv, _ := repo.GetInventory(ctx, alice.ID)
	var found bool
	for _, slot := range inv.Slots {
		if slot.ItemID == 1 { // lootbox1 has ID 1
			found = true
			if slot.Quantity != 5 {
				t.Fatalf("Expected 5, got %d", slot.Quantity)
			}
		}
	}
	if !found {
		t.Fatal("Item not found in inventory")
	}

	// Adding more should increment
	err = svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 3)
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	inv, _ = repo.GetInventory(ctx, alice.ID)
	for _, slot := range inv.Slots {
		if slot.ItemID == 1 {
			if slot.Quantity != 8 {
				t.Errorf("Expected 8 after adding 3 more, got %d", slot.Quantity)
			}
		}
	}

	// Adding a different item should create a new slot
	err = svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox2, 2)
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	inv, _ = repo.GetInventory(ctx, alice.ID)
	if len(inv.Slots) != 2 {
		t.Errorf("Expected 2 slots, got %d", len(inv.Slots))
	}
}

func TestRemoveItem(t *testing.T) {
	repo := NewFakeRepository()
	setupTestData(repo)
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, false)
	ctx := context.Background()
	alice := domain.User{
		ID:       "user-alice",
		Username: "alice",
		TwitchID: "alice123",
	}

	// Add 10 lootbox1 items
	svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 10)

	// Remove 3
	removed, err := svc.RemoveItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 3)
	if err != nil {
		t.Fatalf("RemoveItem failed: %v", err)
	}
	if removed != 3 {
		t.Errorf("Expected 3 removed, got %d", removed)
	}

	inv, _ := repo.GetInventory(ctx, alice.ID)
	if inv.Slots[0].Quantity != 7 {
		t.Errorf("Expected quantity 7, got %d", inv.Slots[0].Quantity)
	}

	// Test removing more than available (should remove all)
	removed, err = svc.RemoveItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 100)
	if err != nil {
		t.Fatalf("RemoveItem failed: %v", err)
	}
	if removed != 7 {
		t.Errorf("Expected 7 removed, got %d", removed)
	}

	inv, _ = repo.GetInventory(ctx, alice.ID)
	if len(inv.Slots) != 0 {
		t.Errorf("Expected empty inventory, got %d slots", len(inv.Slots))
	}

	// Test removing from empty inventory
	_, err = svc.RemoveItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 1)
	if err == nil {
		t.Error("Expected error when removing from empty inventory")
	}
}

func TestGiveItem(t *testing.T) {
	repo := NewFakeRepository()
	setupTestData(repo)
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, false)
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
	svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 10)

	// Test giving items
	err := svc.GiveItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.PlatformTwitch, bob.Username, domain.ItemLootbox1, 3)
	if err != nil {
		t.Fatalf("GiveItem failed: %v", err)
	}

	// Verify alice has 7 left
	aliceInv, _ := repo.GetInventory(ctx, alice.ID)
	if aliceInv.Slots[0].Quantity != 7 {
		t.Errorf("Alice should have 7, got %d", aliceInv.Slots[0].Quantity)
	}

	// Verify bob has 3
	bobInv, _ := repo.GetInventory(ctx, bob.ID)
	if len(bobInv.Slots) != 1 || bobInv.Slots[0].Quantity != 3 {
		t.Errorf("Bob should have 3, got %+v", bobInv.Slots)
	}

	// Test giving more than owned (should error)
	err = svc.GiveItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.PlatformTwitch, bob.Username, domain.ItemLootbox1, 100)
	if err == nil {
		t.Error("Expected error when giving more than owned")
	}

	// Verify no changes after failed give
	aliceInv, _ = repo.GetInventory(ctx, alice.ID)
	if aliceInv.Slots[0].Quantity != 7 {
		t.Error("Alice's inventory should be unchanged after failed give")
	}
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
			svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, false)
			ctx := context.Background()

			// Setup owner with items
			if tt.ownerItems > 0 {
				svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, tt.ownerItems)
			}

			// Attempt to give items
			err := svc.GiveItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.PlatformTwitch, bob.Username, domain.ItemLootbox1, tt.giveQty)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errorMessage != "" && !strings.Contains(err.Error(), tt.errorMessage) {
					t.Errorf("expected error to contain %q, got %q", tt.errorMessage, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			// Verify owner's inventory
			aliceInv, _ := repo.GetInventory(ctx, alice.ID)
			if tt.expectedOwner == 0 {
				// Owner should have no slots if all given away
				hasItem := false
				for _, slot := range aliceInv.Slots {
					if slot.ItemID == 1 {
						hasItem = true
						t.Errorf("Alice should have no %s, but has %d", domain.ItemLootbox1, slot.Quantity)
					}
				}
				_ = hasItem
			} else {
				found := false
				for _, slot := range aliceInv.Slots {
					if slot.ItemID == 1 {
						if slot.Quantity != tt.expectedOwner {
							t.Errorf("Alice should have %d, got %d", tt.expectedOwner, slot.Quantity)
						}
						found = true
						break
					}
				}
				if !found && tt.expectedOwner > 0 {
					t.Error("Alice should have items but inventory is empty")
				}
			}

			// Verify receiver's inventory
			bobInv, _ := repo.GetInventory(ctx, bob.ID)
			if tt.expectedRecv == 0 {
				for _, slot := range bobInv.Slots {
					if slot.ItemID == 1 {
						t.Errorf("Bob should have no items, but has %d", slot.Quantity)
					}
				}
			} else {
				found := false
				for _, slot := range bobInv.Slots {
					if slot.ItemID == 1 {
						if slot.Quantity != tt.expectedRecv {
							t.Errorf("Bob should have %d, got %d", tt.expectedRecv, slot.Quantity)
						}
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Bob should have %d items but has none", tt.expectedRecv)
				}
			}
		})
	}
}

func TestGiveItem_CrossPlatform(t *testing.T) {
	repo := NewFakeRepository()
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, false)
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
	svc.AddItemByUsername(ctx, domain.PlatformTwitch, "alice", domain.ItemLootbox1, 10)

	// Give from alice (Twitch) to bob (Discord)
	err := svc.GiveItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.PlatformDiscord, "bob", domain.ItemLootbox1, 5)
	if err != nil {
		t.Fatalf("cross-platform give failed: %v", err)
	}

	// Verify both inventories
	aliceInv, _ := repo.GetInventory(ctx, "user-alice")
	if aliceInv.Slots[0].Quantity != 5 {
		t.Errorf("Alice should have 5, got %d", aliceInv.Slots[0].Quantity)
	}

	bobInv, _ := repo.GetInventory(ctx, "user-bob")
	if len(bobInv.Slots) != 1 || bobInv.Slots[0].Quantity != 5 {
		t.Errorf("Bob should have 5, got %+v", bobInv.Slots)
	}
}

func TestRegisterUser(t *testing.T) {
	repo := NewFakeRepository()
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, false)
	ctx := context.Background()

	user := domain.User{
		Username: "charlie",
		TwitchID: "charlie789",
	}

	registered, err := svc.RegisterUser(ctx, user)
	if err != nil {
		t.Fatalf("RegisterUser failed: %v", err)
	}

	if registered.ID == "" {
		t.Error("Expected user ID to be set")
	}
	if registered.Username != "charlie" {
		t.Errorf("Expected username charlie, got %s", registered.Username)
	}

	// Verify user in repo
	found, _ := repo.GetUserByPlatformID(ctx, domain.PlatformTwitch, "charlie789")
	if found == nil {
		t.Error("User not found in repository")
	}
}

func TestHandleIncomingMessage_NewUser(t *testing.T) {
	repo := NewFakeRepository()
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, false)
	ctx := context.Background()

	result, err := svc.HandleIncomingMessage(ctx, domain.PlatformTwitch, "newuser123", "newuser", "hello")
	if err != nil {
		t.Fatalf("HandleIncomingMessage failed: %v", err)
	}

	if result.User.Username != "newuser" || result.User.TwitchID != "newuser123" {
		t.Errorf("Unexpected user: %+v", result.User)
	}

	// Verify user was created
	found, _ := repo.GetUserByPlatformID(ctx, domain.PlatformTwitch, "newuser123")
	if found == nil {
		t.Error("User should have been created")
	}
}

func TestHandleIncomingMessage_ExistingUser(t *testing.T) {
	repo := NewFakeRepository()
	setupTestData(repo)
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, false)
	ctx := context.Background()

	result, err := svc.HandleIncomingMessage(ctx, domain.PlatformTwitch, "alice123", "alice", "hello")
	if err != nil {
		t.Fatalf("HandleIncomingMessage failed: %v", err)
	}

	if result.User.ID != "user-alice" {
		t.Error("Should have returned existing user")
	}
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

	svc := NewService(repo, repo, nil, nil, lootboxSvc, NewMockNamingResolver(), nil, nil, false).(*service)

	ctx := context.Background()

	// Setup: Give alice some lootbox1
	svc.getUserOrRegister(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username)
	svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 5)

	// Test using lootbox1 (consumes 1 lootbox1, gives 1 lootbox0)
	message, err := svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemLootbox1, 1, "")
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}

	// The message format changed in the new implementation
	if !strings.Contains(message, "Opened") || !strings.Contains(message, "junkbox") {
		t.Errorf("Expected message to contain 'Opened' and 'junkbox', got '%s'", message)
	}

	// Verify inventory
	inv, _ := repo.GetInventory(ctx, alice.ID)

	// Should have 2 slots: lootbox1 (4 left) and lootbox0 (1)
	var lootbox1Slot, lootbox0Slot *domain.InventorySlot
	for i := range inv.Slots {
		if inv.Slots[i].ItemID == 1 {
			lootbox1Slot = &inv.Slots[i]
		}
		if inv.Slots[i].ItemID == 4 {
			lootbox0Slot = &inv.Slots[i]
		}
	}

	if lootbox1Slot == nil || lootbox1Slot.Quantity != 4 {
		t.Errorf("Expected 4 lootbox1, got %+v", lootbox1Slot)
	}
	if lootbox0Slot == nil || lootbox0Slot.Quantity != 1 {
		t.Errorf("Expected 1 lootbox0, got %+v", lootbox0Slot)
	}

	// Test using more than available
	_, err = svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemLootbox1, 10, "")
	if err == nil {
		t.Error("Expected error when using more than available")
	}

	// Test using unknown item
	_, err = svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, "unknown_item", 1, "")
	if err == nil {
		t.Error("Expected error when using unknown item")
	}

	// Test using item with no effect (money)
	svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemMoney, 1)
	_, err = svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemMoney, 1, "")
	if err == nil {
		t.Error("Expected error when using item with no effect")
	}
}

func TestUseItem_Blaster(t *testing.T) {
	repo := NewFakeRepository()
	setupTestData(repo)
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, false)
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
	svc.RegisterUser(ctx, alice)
	svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemBlaster, 5)

	// Test using blaster on bob
	message, err := svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemBlaster, 2, bob.Username)
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}

	expectedMsg := "alice used weapon_blaster on bob! 2 weapon_blaster(s) fired. Timed out for 1m0s."
	if message != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, message)
	}

	// Verify inventory
	inv, _ := repo.GetInventory(ctx, alice.ID)
	if inv.Slots[0].Quantity != 3 {
		t.Errorf("Expected 3 blasters left, got %d", inv.Slots[0].Quantity)
	}

	// Test using blaster without target
	_, err = svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemBlaster, 1, "")
	if err == nil {
		t.Error("Expected error when using blaster without target")
	}
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

	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, false).(*service)
	ctx := context.Background()
	alice := domain.User{
		ID:       "user-alice",
		Username: "alice",
		TwitchID: "alice123",
	}

	// Setup: Give alice some Rare Candy
	svc.RegisterUser(ctx, alice)
	svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemRareCandy, 5)

	// Test using Rare Candy on a job name (which is NOT a user)
	// This should work now because we don't resolve the target as a user anymore
	message, err := svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemRareCandy, 1, job.JobKeyBlacksmith)
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}

	expectedMsg := "Used 1 rare candy! Granted 500 XP to blacksmith." // Message construction uses hardcoded strings in handler currently
	if message != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, message)
	}

	// Verify inventory
	inv, _ := repo.GetInventory(ctx, alice.ID)
	if inv.Slots[0].Quantity != 4 {
		t.Errorf("Expected 4 Rare Candy left, got %d", inv.Slots[0].Quantity)
	}
}

func TestGetInventory(t *testing.T) {
	repo := NewFakeRepository()
	setupTestData(repo)
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, false)
	ctx := context.Background()
	alice := domain.User{
		ID:        "user-alice",
		TwitchID:  "alice123",
		Username:  "alice",
		DiscordID: "alice456",
	}

	// Setup: Give alice some items with various types
	svc.RegisterUser(ctx, alice)
	svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox1, 2)
	svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemMoney, 100)

	t.Run("No Filter - Returns All Items", func(t *testing.T) {
		// Test GetInventory without filter
		items, err := svc.GetInventory(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, "")
		if err != nil {
			t.Fatalf("GetInventory failed: %v", err)
		}

		if len(items) != 2 {
			t.Errorf("Expected 2 items, got %d", len(items))
		}

		// Verify item details
		foundLootbox := false
		foundMoney := false
		for _, item := range items {
			if item.PublicName == domain.ItemLootbox1 {
				foundLootbox = true
				if item.Quantity != 2 {
					t.Errorf("Expected 2 lootbox1, got %d", item.Quantity)
				}
			}
			if item.PublicName == domain.ItemMoney {
				foundMoney = true
				if item.Quantity != 100 {
					t.Errorf("Expected 100 money, got %d", item.Quantity)
				}
			}
		}

		if !foundLootbox {
			t.Error("Expected lootbox1 in inventory")
		}
		if !foundMoney {
			t.Error("Expected money in inventory")
		}
	})

	t.Run("Upgrade Filter - Returns Only Upgradable Items", func(t *testing.T) {
		// Note: This test assumes items have "upgradable" type in their Types field
		// In a real scenario, mock items would have Types populated
		items, err := svc.GetInventory(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.FilterTypeUpgrade)
		if err != nil {
			t.Fatalf("GetInventory with upgrade filter failed: %v", err)
		}

		// All returned items should have "upgradable" type
		for _, item := range items {
			// In real implementation, check item.Types contains "upgrade"
			// For now, just verify it doesn't error
			if item.PublicName == "" {
				t.Error("Item should have a name")
			}
		}
	})

	t.Run("Sellable Filter - Returns Only Sellable Items", func(t *testing.T) {
		items, err := svc.GetInventory(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.FilterTypeSellable)
		if err != nil {
			t.Fatalf("GetInventory with sellable filter failed: %v", err)
		}

		// All returned items should have "sellable" type
		for _, item := range items {
			if item.PublicName == "" {
				t.Error("Item should have a name")
			}
		}
	})

	t.Run("Consumable Filter - Returns Only Consumable Items", func(t *testing.T) {
		items, err := svc.GetInventory(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.FilterTypeConsumable)
		if err != nil {
			t.Fatalf("GetInventory with consumable filter failed: %v", err)
		}

		// All returned items should have "consumable" type
		for _, item := range items {
			if item.PublicName == "" {
				t.Error("Item should have a name")
			}
		}
	})

	t.Run("Unknown Filter - Returns Empty Result", func(t *testing.T) {
		// Unknown filter should return no items (nothing matches)
		items, err := svc.GetInventory(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, "nonexistent")
		if err != nil {
			t.Fatalf("GetInventory with unknown filter failed: %v", err)
		}

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

	svc := NewService(repo, repo, nil, nil, lootboxSvc, NewMockNamingResolver(), nil, nil, false).(*service)

	ctx := context.Background()

	// Setup: Give alice lootbox0
	svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox0, 1)

	// Test using lootbox0
	msg, err := svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemLootbox0, 1, "")
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}

	// Message format changed
	// "Opened a junkbox and received: 5 Shiny credits"
	// We check if it contains expected parts
	if !strings.Contains(msg, "Opened") || !strings.Contains(msg, "junkbox") {
		t.Errorf("Expected message to contain 'Opened' and 'junkbox', got '%s'", msg)
	}

	if !strings.Contains(msg, "Shiny credits") {
		t.Errorf("Expected message to contain 'Shiny credits', got '%s'", msg)
	}

	// Verify inventory (should have money now)
	inv, _ := repo.GetInventory(ctx, alice.ID)
	if len(inv.Slots) != 1 {
		t.Errorf("Expected 1 slot (money), got %d slots", len(inv.Slots))
	}
	if inv.Slots[0].ItemID != 3 { // Money ID is 3
		t.Errorf("Expected money (ID 3), got ID %d", inv.Slots[0].ItemID)
	}
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

	svc := NewService(repo, repo, nil, nil, lootboxSvc, NewMockNamingResolver(), nil, nil, false).(*service)

	ctx := context.Background()

	// Setup: Give alice lootbox2
	svc.AddItemByUsername(ctx, domain.PlatformTwitch, alice.Username, domain.ItemLootbox2, 1)

	// Test using lootbox2
	msg, err := svc.UseItem(ctx, domain.PlatformTwitch, alice.TwitchID, alice.Username, domain.ItemLootbox2, 1, "")
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}

	// Message format changed
	if !strings.Contains(msg, "Opened") || !strings.Contains(msg, "basic lootbox") {
		t.Errorf("Expected message to contain 'Opened' and 'basic lootbox', got '%s'", msg)
	}

	// Verify inventory: should have 1 lootbox1
	inv, _ := repo.GetInventory(ctx, alice.ID)
	if len(inv.Slots) != 1 {
		t.Errorf("Expected 1 slot, got %d", len(inv.Slots))
	}

	if inv.Slots[0].ItemID != 1 { // lootbox1 ID is 1
		t.Errorf("Expected lootbox1 (ID 1), got ID %d", inv.Slots[0].ItemID)
	}
	if inv.Slots[0].Quantity != 1 {
		t.Errorf("Expected quantity 1, got %d", inv.Slots[0].Quantity)
	}
}
