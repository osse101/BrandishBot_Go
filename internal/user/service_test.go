package user

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/stretchr/testify/mock"
)

// MockRepository implements Repository interface for testing
type MockRepository struct {
	users           map[string]*domain.User // keyed by user ID
	inventories     map[string]*domain.Inventory
	items           map[string]*domain.Item
	recipes         map[int]*domain.Recipe // keyed by recipe ID
	unlockedRecipes map[string]map[int]bool
	cooldowns       map[string]map[string]*time.Time // userID -> action -> timestamp
}

// MockNamingResolver implements naming.Resolver interface for testing
type MockNamingResolver struct {
	DisplayNames map[string]string
}

func (m *MockNamingResolver) ResolvePublicName(publicName string) (string, bool) {
	return publicName, true
}

func (m *MockNamingResolver) GetDisplayName(internalName, shineLevel string) string {
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
		DisplayNames: make(map[string]string),
	}
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		users:           make(map[string]*domain.User),
		items:           make(map[string]*domain.Item),
		inventories:     make(map[string]*domain.Inventory),
		recipes:         make(map[int]*domain.Recipe),
		unlockedRecipes: make(map[string]map[int]bool),
		cooldowns:       make(map[string]map[string]*time.Time),
	}
}

func (m *MockRepository) UpsertUser(ctx context.Context, user *domain.User) error {
	if user.ID == "" {
		user.ID = "user-" + user.Username
	}
	m.users[user.Username] = user
	return nil
}

func (m *MockRepository) UpdateUser(ctx context.Context, user domain.User) error {
	m.users[user.Username] = &user
	return nil
}

func (m *MockRepository) DeleteUser(ctx context.Context, userID string) error {
	for k, v := range m.users {
		if v.ID == userID {
			delete(m.users, k)
		}
	}
	return nil
}

func (m *MockRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	for _, u := range m.users {
		if u.ID == userID {
			return u, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	for _, u := range m.users {
		switch platform {
		case domain.PlatformTwitch:
			if u.TwitchID == platformID {
				return u, nil
			}
		case domain.PlatformYoutube:
			if u.YoutubeID == platformID {
				return u, nil
			}
		case domain.PlatformDiscord:
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

func (m *MockRepository) DeleteInventory(ctx context.Context, userID string) error {
	delete(m.inventories, userID)
	return nil
}

func (m *MockRepository) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	if item, ok := m.items[itemName]; ok {
		return item, nil
	}
	return nil, nil
}

func (m *MockRepository) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	var items []domain.Item
	for _, id := range itemIDs {
		for _, item := range m.items {
			if item.ID == id {
				items = append(items, *item)
				break
			}
		}
	}
	return items, nil
}

func (m *MockRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	for _, item := range m.items {
		if item.ID == id {
			return item, nil
		}
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

func (m *MockRepository) GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error) {
	if recipe, ok := m.recipes[itemID]; ok {
		return recipe, nil
	}
	return nil, nil
}

func (m *MockRepository) IsRecipeUnlocked(ctx context.Context, userID string, recipeID int) (bool, error) {
	if m.unlockedRecipes[userID] == nil {
		return false, nil
	}
	return m.unlockedRecipes[userID][recipeID], nil
}

func (m *MockRepository) UnlockRecipe(ctx context.Context, userID string, recipeID int) error {
	if m.unlockedRecipes[userID] == nil {
		m.unlockedRecipes[userID] = make(map[int]bool)
	}
	m.unlockedRecipes[userID][recipeID] = true
	return nil
}

func (r *MockRepository) GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]crafting.UnlockedRecipeInfo, error) {
	var recipes []crafting.UnlockedRecipeInfo

	// For each unlocked recipe, get the recipe and item info
	if userUnlocks, ok := r.unlockedRecipes[userID]; ok {
		for recipeID := range userUnlocks {
			if recipe, exists := r.recipes[recipeID]; exists {
				// Find the item name
				for _, item := range r.items {
					if item.ID == recipe.TargetItemID {
						recipes = append(recipes, crafting.UnlockedRecipeInfo{
							ItemName: item.InternalName,

							ItemID: item.ID,
						})
						break
					}
				}
			}
		}
	}

	return recipes, nil
}

func (m *MockRepository) GetLastCooldown(ctx context.Context, userID, action string) (*time.Time, error) {
	if userCooldowns, ok := m.cooldowns[userID]; ok {
		return userCooldowns[action], nil
	}
	return nil, nil
}

func (m *MockRepository) UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error {
	if _, ok := m.cooldowns[userID]; !ok {
		m.cooldowns[userID] = make(map[string]*time.Time)
	}
	m.cooldowns[userID][action] = &timestamp
	return nil
}

func (m *MockRepository) MergeUsersInTransaction(ctx context.Context, primaryUserID, secondaryUserID string, mergedUser domain.User, mergedInventory domain.Inventory) error {
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
		ID:           1,
		InternalName: domain.ItemLootbox1,

		Description: "Basic Lootbox",
		BaseValue:   50,
	}
	repo.items[domain.ItemLootbox2] = &domain.Item{
		ID:           2,
		InternalName: domain.ItemLootbox2,

		Description: "Good Lootbox",
		BaseValue:   100,
	}
	repo.items[domain.ItemMoney] = &domain.Item{
		ID:           3,
		InternalName: domain.ItemMoney,

		Description: "Currency",
		BaseValue:   1,
	}
	repo.items[domain.ItemLootbox0] = &domain.Item{
		ID:           4,
		InternalName: domain.ItemLootbox0,

		Description: "Empty Lootbox",
		BaseValue:   10,
	}
	repo.items[domain.ItemBlaster] = &domain.Item{
		ID:           5,
		InternalName: domain.ItemBlaster,

		Description: "So anyway, I started blasting",
		BaseValue:   10,
	}
}

func TestAddItem(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil, NewMockNamingResolver(), nil, false)
	ctx := context.Background()

	// Test adding item to empty inventory
	err := svc.AddItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox1, 5)
	if err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	// One slot should have 5 lootbox1
	inv, _ := repo.GetInventory(ctx, "user-alice")
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
	err = svc.AddItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox1, 3)
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	inv, _ = repo.GetInventory(ctx, "user-alice")
	for _, slot := range inv.Slots {
		if slot.ItemID == 1 {
			if slot.Quantity != 8 {
				t.Errorf("Expected 8 after adding 3 more, got %d", slot.Quantity)
			}
		}
	}

	// Adding a different item should create a new slot
	err = svc.AddItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox2, 2)
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
	svc := NewService(repo, nil, nil, nil, NewMockNamingResolver(), nil, false)
	ctx := context.Background()

	// Add 10 lootbox1 items
	svc.AddItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox1, 10)

	// Remove 3
	removed, err := svc.RemoveItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox1, 3)
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
	removed, err = svc.RemoveItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox1, 100)
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
	_, err = svc.RemoveItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox1, 1)
	if err == nil {
		t.Error("Expected error when removing from empty inventory")
	}
}

func TestGiveItem(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil, NewMockNamingResolver(), nil, false)
	ctx := context.Background()

	// Setup: Give alice some items
	svc.AddItem(ctx, domain.PlatformTwitch, "alice123", "alice", domain.ItemLootbox1, 10)

	// Test giving items
	err := svc.GiveItem(ctx, domain.PlatformTwitch, "alice123", "alice", domain.PlatformTwitch, "bob456", "bob", domain.ItemLootbox1, 3)
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
	err = svc.GiveItem(ctx, domain.PlatformTwitch, "alice123", "alice", domain.PlatformTwitch, "bob456", "bob", domain.ItemLootbox1, 100)
	if err == nil {
		t.Error("Expected error when giving more than owned")
	}

	// Verify no changes after failed give
	aliceInv, _ = repo.GetInventory(ctx, "user-alice")
	if aliceInv.Slots[0].Quantity != 7 {
		t.Error("Alice's inventory should be unchanged after failed give")
	}
}

func TestRegisterUser(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo, nil, nil, nil, NewMockNamingResolver(), nil, false)
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
	repo := NewMockRepository()
	svc := NewService(repo, nil, nil, nil, NewMockNamingResolver(), nil, false)
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
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil, NewMockNamingResolver(), nil, false)
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
	repo := NewMockRepository()
	setupTestData(repo)
	
	// Create a mock lootbox service
	lootboxSvc := new(MockLootboxService)
	drops := []lootbox.DroppedItem{
		{ItemID: 4, ItemName: domain.ItemLootbox0, Quantity: 1, Value: 10, ShineLevel: "COMMON"},
	}
	lootboxSvc.On("OpenLootbox", mock.Anything, domain.ItemLootbox1, 1).Return(drops, nil)
	
	svc := NewService(repo, nil, nil, lootboxSvc, NewMockNamingResolver(), nil, false).(*service)

	ctx := context.Background()

	// Setup: Give alice some lootbox1
	svc.AddItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox1, 5)

	// Test using lootbox1 (consumes 1 lootbox1, gives 1 lootbox0)
	message, err := svc.UseItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox1, 1, "")
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}

	// The message format changed in the new implementation
	if !strings.Contains(message, "Opened") || !strings.Contains(message, "lootbox_tier0") {
		t.Errorf("Expected message to contain 'Opened' and 'lootbox_tier0', got '%s'", message)
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
	_, err = svc.UseItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox1, 10, "")
	if err == nil {
		t.Error("Expected error when using more than available")
	}

	// Test using unknown item
	_, err = svc.UseItem(ctx, domain.PlatformTwitch, "", "alice", "unknown_item", 1, "")
	if err == nil {
		t.Error("Expected error when using unknown item")
	}

	// Test using item with no effect (money)
	svc.AddItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemMoney, 1)
	_, err = svc.UseItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemMoney, 1, "")
	if err == nil {
		t.Error("Expected error when using item with no effect")
	}
}

func TestUseItem_Blaster(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil, NewMockNamingResolver(), nil, false)
	ctx := context.Background()

	// Setup: Give alice some blasters
	svc.AddItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemBlaster, 5)

	// Test using blaster on bob
	message, err := svc.UseItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemBlaster, 2, "bob")
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
	_, err = svc.UseItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemBlaster, 1, "")
	if err == nil {
		t.Error("Expected error when using blaster without target")
	}
}

func TestGetInventory(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil, NewMockNamingResolver(), nil, false)
	ctx := context.Background()

	// Setup: Give alice some items
	svc.AddItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox1, 2)
	svc.AddItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemMoney, 100)

	// Test GetInventory
	items, err := svc.GetInventory(ctx, domain.PlatformTwitch, "", "alice", "")
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
	
	// Create a mock lootbox service
	lootboxSvc := new(MockLootboxService)
	drops := []lootbox.DroppedItem{
		{ItemID: 3, ItemName: domain.ItemMoney, Quantity: 5, Value: 5, ShineLevel: "COMMON"},
	}
	lootboxSvc.On("OpenLootbox", mock.Anything, domain.ItemLootbox0, 1).Return(drops, nil)
	
	svc := NewService(repo, nil, nil, lootboxSvc, NewMockNamingResolver(), nil, false).(*service)

	ctx := context.Background()

	// Setup: Give alice lootbox0
	svc.AddItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox0, 1)

	// Test using lootbox0
	msg, err := svc.UseItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox0, 1, "")
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}

	// Message format changed
	// "Opened 1 lootbox0 and received: 5x money"
	// We check if it contains expected parts
	if !strings.Contains(msg, "Opened") || !strings.Contains(msg, "lootbox_tier0") {
		t.Errorf("Expected message to contain 'Opened' and 'lootbox_tier0', got '%s'", msg)
	}

	if !strings.Contains(msg, "money") {
		t.Errorf("Expected message to contain 'money', got '%s'", msg)
	}

	// Verify inventory (should have money now)
	inv, _ := repo.GetInventory(ctx, "user-alice")
	if len(inv.Slots) != 1 {
		t.Errorf("Expected 1 slot (money), got %d slots", len(inv.Slots))
	}
	if inv.Slots[0].ItemID != 3 { // Money ID is 3
		t.Errorf("Expected money (ID 3), got ID %d", inv.Slots[0].ItemID)
	}
}

func TestUseItem_Lootbox2(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	
	// Create a mock lootbox service
	lootboxSvc := new(MockLootboxService)
	drops := []lootbox.DroppedItem{
		{ItemID: 1, ItemName: domain.ItemLootbox1, Quantity: 1, Value: 50, ShineLevel: "COMMON"},
	}
	lootboxSvc.On("OpenLootbox", mock.Anything, domain.ItemLootbox2, 1).Return(drops, nil)
	
	svc := NewService(repo, nil, nil, lootboxSvc, NewMockNamingResolver(), nil, false).(*service)

	ctx := context.Background()

	// Setup: Give alice lootbox2
	svc.AddItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox2, 1)

	// Test using lootbox2
	msg, err := svc.UseItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox2, 1, "")
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}

	// Message format changed
	if !strings.Contains(msg, "Opened") || !strings.Contains(msg, "lootbox_tier1") {
		t.Errorf("Expected message to contain 'Opened' and 'lootbox_tier1', got '%s'", msg)
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
