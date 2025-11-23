package user

import (
	"context"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// MockRepository implements Repository interface for testing
type MockRepository struct {
	users      map[string]*domain.User
	items      map[string]*domain.Item
	inventories map[string]*domain.Inventory
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		users:       make(map[string]*domain.User),
		items:       make(map[string]*domain.Item),
		inventories: make(map[string]*domain.Inventory),
	}
}

func (m *MockRepository) UpsertUser(ctx context.Context, user *domain.User) error {
	if user.ID == "" {
		user.ID = "user-" + user.Username
	}
	m.users[user.Username] = user
	return nil
}

func (m *MockRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	for _, u := range m.users {
		switch platform {
		case "twitch":
			if u.TwitchID == platformID {
				return u, nil
			}
		case "youtube":
			if u.YoutubeID == platformID {
				return u, nil
			}
		case "discord":
			if u.DiscordID == platformID {
				return u, nil
			}
		}
	}
	return nil, nil
}

func (m *MockRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	if inv, ok := m.inventories[userID]; ok {
		return inv, nil
	}
	// Return empty inventory if not exists
	return &domain.Inventory{Slots: []domain.InventorySlot{}}, nil
}

func (m *MockRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	m.inventories[userID] = &inventory
	return nil
}

func (m *MockRepository) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	if item, ok := m.items[itemName]; ok {
		return item, nil
	}
	return nil, nil
}

func (m *MockRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	for _, item := range m.items {
		if item.ID == id {
			return item, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	if user, ok := m.users[username]; ok {
		return user, nil
	}
	return nil, nil
}

func (m *MockRepository) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	var items []domain.Item
	for _, item := range m.items {
		items = append(items, *item)
	}
	return items, nil
}

func (m *MockRepository) IsItemBuyable(ctx context.Context, itemName string) (bool, error) {
	// For testing, assume lootbox0 and lootbox1 are buyable
	if itemName == domain.ItemLootbox0 || itemName == domain.ItemLootbox1 {
		return true, nil
	}
	return false, nil
}

// MockTx wraps MockRepository for transaction testing
type MockTx struct {
	repo *MockRepository
}

func (m *MockRepository) BeginTx(ctx context.Context) (repository.Tx, error) {
	return &MockTx{repo: m}, nil
}

func (mt *MockTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return mt.repo.GetInventory(ctx, userID)
}

func (mt *MockTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return mt.repo.UpdateInventory(ctx, userID, inventory)
}

func (mt *MockTx) Commit(ctx context.Context) error {
	return nil // No-op for mock
}

func (mt *MockTx) Rollback(ctx context.Context) error {
	return nil // No-op for mock
}

// Helper to setup test data
func setupTestData(repo *MockRepository) {
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
		ID:          1,
		Name:        domain.ItemLootbox1,
		Description: "Basic Lootbox",
		BaseValue:   50,
	}
	repo.items[domain.ItemLootbox2] = &domain.Item{
		ID:          2,
		Name:        domain.ItemLootbox2,
		Description: "Good Lootbox",
		BaseValue:   100,
	}
	repo.items[domain.ItemMoney] = &domain.Item{
		ID:          3,
		Name:        domain.ItemMoney,
		Description: "Currency",
		BaseValue:   1,
	}
	repo.items[domain.ItemLootbox0] = &domain.Item{
		ID:          4,
		Name:        domain.ItemLootbox0,
		Description: "Empty Lootbox",
		BaseValue:   10,
	}
	repo.items[domain.ItemBlaster] = &domain.Item{
		ID:          5,
		Name:        domain.ItemBlaster,
		Description: "So anyway, I started blasting",
		BaseValue:   10,
	}
}

func TestAddItem(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo)
	ctx := context.Background()

	// Test adding item to empty inventory
	err := svc.AddItem(ctx, "alice", "twitch", domain.ItemLootbox1, 5)
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	// Verify inventory
	inv, _ := repo.GetInventory(ctx, "user-alice")
	if len(inv.Slots) != 1 {
		t.Errorf("Expected 1 slot, got %d", len(inv.Slots))
	}
	if inv.Slots[0].ItemID != 1 || inv.Slots[0].Quantity != 5 {
		t.Errorf("Unexpected inventory state: %+v", inv.Slots[0])
	}

	// Test adding more of same item
	err = svc.AddItem(ctx, "alice", "twitch", domain.ItemLootbox1, 3)
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	inv, _ = repo.GetInventory(ctx, "user-alice")
	if inv.Slots[0].Quantity != 8 {
		t.Errorf("Expected quantity 8, got %d", inv.Slots[0].Quantity)
	}

	// Test adding different item
	err = svc.AddItem(ctx, "alice", "twitch", domain.ItemLootbox2, 2)
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	inv, _ = repo.GetInventory(ctx, "user-alice")
	if len(inv.Slots) != 2 {
		t.Errorf("Expected 2 slots, got %d", len(inv.Slots))
	}
}

func TestRemoveItem(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo)
	ctx := context.Background()

	// Setup: Add items first
	svc.AddItem(ctx, "alice", "twitch", domain.ItemLootbox1, 10)

	// Test removing partial quantity
	removed, err := svc.RemoveItem(ctx, "alice", "twitch", domain.ItemLootbox1, 3)
	if err != nil {
		t.Fatalf("RemoveItem failed: %v", err)
	}
	if removed != 3 {
		t.Errorf("Expected 3 removed, got %d", removed)
	}

	inv, _ := repo.GetInventory(ctx, "user-alice")
	if inv.Slots[0].Quantity != 7 {
		t.Errorf("Expected quantity 7, got %d", inv.Slots[0].Quantity)
	}

	// Test removing more than available (should remove all)
	removed, err = svc.RemoveItem(ctx, "alice", "twitch", domain.ItemLootbox1, 100)
	if err != nil {
		t.Fatalf("RemoveItem failed: %v", err)
	}
	if removed != 7 {
		t.Errorf("Expected 7 removed, got %d", removed)
	}

	inv, _ = repo.GetInventory(ctx, "user-alice")
	if len(inv.Slots) != 0 {
		t.Errorf("Expected empty inventory, got %d slots", len(inv.Slots))
	}

	// Test removing from empty inventory
	_, err = svc.RemoveItem(ctx, "alice", "twitch", domain.ItemLootbox1, 1)
	if err == nil {
		t.Error("Expected error when removing from empty inventory")
	}
}

func TestGiveItem(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo)
	ctx := context.Background()

	// Setup: Give alice some items
	svc.AddItem(ctx, "alice", "twitch", domain.ItemLootbox1, 10)

	// Test giving items
	err := svc.GiveItem(ctx, "alice", "bob", "twitch", domain.ItemLootbox1, 3)
	if err != nil {
		t.Fatalf("GiveItem failed: %v", err)
	}

	// Verify alice has 7 left
	aliceInv, _ := repo.GetInventory(ctx, "user-alice")
	if aliceInv.Slots[0].Quantity != 7 {
		t.Errorf("Alice should have 7, got %d", aliceInv.Slots[0].Quantity)
	}

	// Verify bob has 3
	bobInv, _ := repo.GetInventory(ctx, "user-bob")
	if len(bobInv.Slots) != 1 || bobInv.Slots[0].Quantity != 3 {
		t.Errorf("Bob should have 3, got %+v", bobInv.Slots)
	}

	// Test giving more than owned (should error)
	err = svc.GiveItem(ctx, "alice", "bob", "twitch", domain.ItemLootbox1, 100)
	if err == nil {
		t.Error("Expected error when giving more than owned")
	}

	// Verify no changes after failed give
	aliceInv, _ = repo.GetInventory(ctx, "user-alice")
	if aliceInv.Slots[0].Quantity != 7 {
		t.Error("Alice's inventory should be unchanged after failed give")
	}
}

func TestSellItem(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo)
	ctx := context.Background()

	// Setup: Give alice some lootboxes
	svc.AddItem(ctx, "alice", "twitch", domain.ItemLootbox1, 10)

	// Test selling items
	moneyGained, itemsSold, err := svc.SellItem(ctx, "alice", "twitch", domain.ItemLootbox1, 3)
	if err != nil {
		t.Fatalf("SellItem failed: %v", err)
	}

	// lootbox1 base_value is 50, so 3 * 50 = 150
	if moneyGained != 150 {
		t.Errorf("Expected 150 money, got %d", moneyGained)
	}
	if itemsSold != 3 {
		t.Errorf("Expected 3 sold, got %d", itemsSold)
	}

	// Verify inventory
	inv, _ := repo.GetInventory(ctx, "user-alice")
	
	// Should have 2 slots: lootbox1 (7 left) and money (150)
	if len(inv.Slots) != 2 {
		t.Fatalf("Expected 2 slots, got %d", len(inv.Slots))
	}

	// Find lootbox1 and money slots
	var lootboxSlot, moneySlot *domain.InventorySlot
	for i := range inv.Slots {
		if inv.Slots[i].ItemID == 1 {
			lootboxSlot = &inv.Slots[i]
		}
		if inv.Slots[i].ItemID == 3 {
			moneySlot = &inv.Slots[i]
		}
	}

	if lootboxSlot == nil || lootboxSlot.Quantity != 7 {
		t.Errorf("Expected 7 lootbox1, got %+v", lootboxSlot)
	}
	if moneySlot == nil || moneySlot.Quantity != 150 {
		t.Errorf("Expected 150 money, got %+v", moneySlot)
	}

	// Test selling more than owned (should sell all)
	moneyGained, itemsSold, err = svc.SellItem(ctx, "alice", "twitch", domain.ItemLootbox1, 100)
	if err != nil {
		t.Fatalf("SellItem failed: %v", err)
	}

	// Should sell all 7 remaining: 7 * 50 = 350
	if moneyGained != 350 {
		t.Errorf("Expected 350 money, got %d", moneyGained)
	}
	if itemsSold != 7 {
		t.Errorf("Expected 7 sold, got %d", itemsSold)
	}

	// Verify inventory - should only have money now
	inv, _ = repo.GetInventory(ctx, "user-alice")
	if len(inv.Slots) != 1 {
		t.Errorf("Expected 1 slot (money only), got %d", len(inv.Slots))
	}
	if inv.Slots[0].ItemID != 3 || inv.Slots[0].Quantity != 500 {
		t.Errorf("Expected 500 money total, got %+v", inv.Slots[0])
	}
}

func TestSellItem_NotInInventory(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo)
	ctx := context.Background()

	// Try to sell item not in inventory
	_, _, err := svc.SellItem(ctx, "alice", "twitch", domain.ItemLootbox1, 1)
	if err == nil {
		t.Error("Expected error when selling item not in inventory")
	}
}

func TestRegisterUser(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
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
	found, _ := repo.GetUserByUsername(ctx, "charlie")
	if found == nil {
		t.Error("User not found in repository")
	}
}

func TestHandleIncomingMessage_NewUser(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)
	ctx := context.Background()

	user, err := svc.HandleIncomingMessage(ctx, "twitch", "newuser123", "newuser")
	if err != nil {
		t.Fatalf("HandleIncomingMessage failed: %v", err)
	}

	if user.Username != "newuser" || user.TwitchID != "newuser123" {
		t.Errorf("Unexpected user: %+v", user)
	}

	// Verify user was created
	found, _ := repo.GetUserByPlatformID(ctx, "twitch", "newuser123")
	if found == nil {
		t.Error("User should have been created")
	}
}

func TestHandleIncomingMessage_ExistingUser(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo)
	ctx := context.Background()

	user, err := svc.HandleIncomingMessage(ctx, "twitch", "alice123", "alice")
	if err != nil {
		t.Fatalf("HandleIncomingMessage failed: %v", err)
	}

	if user.ID != "user-alice" {
		t.Error("Should have returned existing user")
	}
}

func TestBuyItem(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo)
	ctx := context.Background()

	// Setup: Give alice some money
	svc.AddItem(ctx, "alice", "twitch", domain.ItemMoney, 500)

	// Test buying items (lootbox1 cost 50)
	bought, err := svc.BuyItem(ctx, "alice", "twitch", domain.ItemLootbox1, 2)
	if err != nil {
		t.Fatalf("BuyItem failed: %v", err)
	}

	if bought != 2 {
		t.Errorf("Expected 2 bought, got %d", bought)
	}

	// Verify inventory
	inv, _ := repo.GetInventory(ctx, "user-alice")
	
	// Should have 2 slots: money (400 left) and lootbox1 (2)
	var moneySlot, lootboxSlot *domain.InventorySlot
	for i := range inv.Slots {
		if inv.Slots[i].ItemID == 3 {
			moneySlot = &inv.Slots[i]
		}
		if inv.Slots[i].ItemID == 1 {
			lootboxSlot = &inv.Slots[i]
		}
	}

	if moneySlot == nil || moneySlot.Quantity != 400 {
		t.Errorf("Expected 400 money, got %+v", moneySlot)
	}
	if lootboxSlot == nil || lootboxSlot.Quantity != 2 {
		t.Errorf("Expected 2 lootbox1, got %+v", lootboxSlot)
	}

	// Test buying more than affordable (partial fulfillment)
	// Has 400 money, lootbox1 cost 50. Can buy 8 max.
	// Try to buy 10
	bought, err = svc.BuyItem(ctx, "alice", "twitch", domain.ItemLootbox1, 10)
	if err != nil {
		t.Fatalf("BuyItem failed: %v", err)
	}

	if bought != 8 {
		t.Errorf("Expected 8 bought (max affordable), got %d", bought)
	}

	// Verify inventory - money should be 0 (removed)
	inv, _ = repo.GetInventory(ctx, "user-alice")
	moneySlot = nil
	for i := range inv.Slots {
		if inv.Slots[i].ItemID == 3 {
			moneySlot = &inv.Slots[i]
		}
	}
	if moneySlot != nil {
		t.Errorf("Expected money slot to be removed, got %+v", moneySlot)
	}

	// Test buying with no money
	_, err = svc.BuyItem(ctx, "alice", "twitch", domain.ItemLootbox1, 1)
	if err == nil {
		t.Error("Expected error when buying with no money")
	}

	// Test buying non-buyable item
	_, err = svc.BuyItem(ctx, "alice", "twitch", domain.ItemLootbox2, 1)
	if err == nil {
		t.Error("Expected error when buying non-buyable item")
	}
}

func TestUseItem(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo)
	ctx := context.Background()

	// Setup: Give alice some lootbox1
	svc.AddItem(ctx, "alice", "twitch", domain.ItemLootbox1, 5)

	// Test using lootbox1 (consumes 1 lootbox1, gives 1 lootbox0)
	message, err := svc.UseItem(ctx, "alice", "twitch", domain.ItemLootbox1, 1, "")
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}

	if message != "Used 1 lootbox1" {
		t.Errorf("Expected message 'Used 1 lootbox1', got '%s'", message)
	}

	// Verify inventory
	inv, _ := repo.GetInventory(ctx, "user-alice")
	
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
	_, err = svc.UseItem(ctx, "alice", "twitch", domain.ItemLootbox1, 10, "")
	if err == nil {
		t.Error("Expected error when using more than available")
	}

	// Test using unknown item
	_, err = svc.UseItem(ctx, "alice", "twitch", "unknown_item", 1, "")
	if err == nil {
		t.Error("Expected error when using unknown item")
	}

	// Test using item with no effect (money)
	svc.AddItem(ctx, "alice", "twitch", domain.ItemMoney, 1)
	_, err = svc.UseItem(ctx, "alice", "twitch", domain.ItemMoney, 1, "")
	if err == nil {
		t.Error("Expected error when using item with no effect")
	}
}

func TestUseItem_Blaster(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo)
	ctx := context.Background()

	// Setup: Give alice some blasters
	svc.AddItem(ctx, "alice", "twitch", domain.ItemBlaster, 5)

	// Test using blaster on bob
	message, err := svc.UseItem(ctx, "alice", "twitch", domain.ItemBlaster, 2, "bob")
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}

	expectedMsg := "alice has BLASTED bob 2 times! They are timed out for 1m0s."
	if message != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, message)
	}

	// Verify inventory
	inv, _ := repo.GetInventory(ctx, "user-alice")
	if inv.Slots[0].Quantity != 3 {
		t.Errorf("Expected 3 blasters left, got %d", inv.Slots[0].Quantity)
	}

	// Test using blaster without target
	_, err = svc.UseItem(ctx, "alice", "twitch", domain.ItemBlaster, 1, "")
	if err == nil {
		t.Error("Expected error when using blaster without target")
	}
}

func TestGetInventory(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo)
	ctx := context.Background()

	// Setup: Give alice some items
	svc.AddItem(ctx, "alice", "twitch", domain.ItemLootbox1, 2)
	svc.AddItem(ctx, "alice", "twitch", domain.ItemMoney, 100)

	// Test GetInventory
	items, err := svc.GetInventory(ctx, "alice")
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
		if item.Name == domain.ItemLootbox1 {
			foundLootbox = true
			if item.Quantity != 2 {
				t.Errorf("Expected 2 lootbox1, got %d", item.Quantity)
			}
			if item.Value != 50 {
				t.Errorf("Expected value 50 for lootbox1, got %d", item.Value)
			}
		}
		if item.Name == domain.ItemMoney {
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
}

func TestUseItem_Lootbox0(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo)
	ctx := context.Background()

	// Setup: Give alice lootbox0
	svc.AddItem(ctx, "alice", "twitch", domain.ItemLootbox0, 1)

	// Test using lootbox0
	msg, err := svc.UseItem(ctx, "alice", "twitch", domain.ItemLootbox0, 1, "")
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}

	if msg != "The lootbox was empty!" {
		t.Errorf("Expected 'The lootbox was empty!', got '%s'", msg)
	}

	// Verify inventory (should be empty)
	inv, _ := repo.GetInventory(ctx, "user-alice")
	if len(inv.Slots) != 0 {
		t.Errorf("Expected empty inventory, got %d slots", len(inv.Slots))
	}
}

func TestUseItem_Lootbox2(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo)
	ctx := context.Background()

	// Setup: Give alice lootbox2
	svc.AddItem(ctx, "alice", "twitch", domain.ItemLootbox2, 1)

	// Test using lootbox2
	msg, err := svc.UseItem(ctx, "alice", "twitch", domain.ItemLootbox2, 1, "")
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}

	expectedMsg := "Used 1 lootbox2"
	if msg != expectedMsg {
		t.Errorf("Expected '%s', got '%s'", expectedMsg, msg)
	}

	// Verify inventory: should have 1 lootbox1
	inv, _ := repo.GetInventory(ctx, "user-alice")
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
