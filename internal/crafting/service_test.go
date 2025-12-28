package crafting

import (
	"context"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// MockStatsService for crafting tests
type MockStatsService struct {
	events []domain.EventType
}

func (m *MockStatsService) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, eventData map[string]interface{}) error {
	m.events = append(m.events, eventType)
	return nil
}
// Stubs for other interface methods not used in these tests
func (m *MockStatsService) GetUserStats(ctx context.Context, userID string, period string) (*domain.StatsSummary, error) { return nil, nil }
func (m *MockStatsService) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) { return nil, nil }
func (m *MockStatsService) GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error) { return nil, nil }

// MockRepository for crafting tests
type MockRepository struct {
	users              map[string]*domain.User
	items              map[string]*domain.Item
	itemsByID          map[int]*domain.Item
	inventories        map[string]*domain.Inventory
	recipes            map[int]*domain.Recipe
	disassembleRecipes map[int]*domain.DisassembleRecipe
	recipeAssociations map[int]int // disassemble recipe ID -> upgrade recipe ID
	unlockedRecipes    map[string]map[int]bool
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		users:              make(map[string]*domain.User),
		items:              make(map[string]*domain.Item),
		itemsByID:          make(map[int]*domain.Item),
		inventories:        make(map[string]*domain.Inventory),
		recipes:            make(map[int]*domain.Recipe),
		disassembleRecipes: make(map[int]*domain.DisassembleRecipe),
		recipeAssociations: make(map[int]int),
		unlockedRecipes:    make(map[string]map[int]bool),
	}
}

func (m *MockRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	for _, user := range m.users {
		switch platform {
		case domain.PlatformTwitch:
			if user.TwitchID == platformID {
				return user, nil
			}
		case domain.PlatformDiscord:
			if user.DiscordID == platformID {
				return user, nil
			}
		}
	}
	return nil, nil
}

func (m *MockRepository) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	item, ok := m.items[itemName]
	if !ok {
		return nil, nil
	}
	return item, nil
}

func (m *MockRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	item, ok := m.itemsByID[id]
	if !ok {
		return nil, nil
	}
	return item, nil
}

func (m *MockRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	inv, ok := m.inventories[userID]
	if !ok {
		return &domain.Inventory{Slots: []domain.InventorySlot{}}, nil
	}
	return inv, nil
}

func (m *MockRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	m.inventories[userID] = &inventory
	return nil
}

func (m *MockRepository) GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error) {
	for _, recipe := range m.recipes {
		if recipe.TargetItemID == itemID {
			return recipe, nil
		}
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

func (m *MockRepository) GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]UnlockedRecipeInfo, error) {
	var result []UnlockedRecipeInfo
	if m.unlockedRecipes[userID] == nil {
		return result, nil
	}

	for recipeID := range m.unlockedRecipes[userID] {
		if recipe, ok := m.recipes[recipeID]; ok {
			if item, ok := m.itemsByID[recipe.TargetItemID]; ok {
				result = append(result, UnlockedRecipeInfo{
					ItemName: item.InternalName,
					ItemID:   item.ID,
				})
			}
		}
	}
	return result, nil
}

func (m *MockRepository) BeginTx(ctx context.Context) (repository.Tx, error) {
	return &MockTx{repo: m}, nil
}

func (m *MockRepository) GetDisassembleRecipeBySourceItemID(ctx context.Context, itemID int) (*domain.DisassembleRecipe, error) {
	for _, recipe := range m.disassembleRecipes {
		if recipe.SourceItemID == itemID {
			return recipe, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) GetAssociatedUpgradeRecipeID(ctx context.Context, disassembleRecipeID int) (int, error) {
	upgradeRecipeID, ok := m.recipeAssociations[disassembleRecipeID]
	if !ok {
		return 0, nil
	}
	return upgradeRecipeID, nil
}

func (m *MockRepository) GetAllRecipes(ctx context.Context) ([]RecipeListItem, error) {
	var result []RecipeListItem
	for _, recipe := range m.recipes {
		if item, ok := m.itemsByID[recipe.TargetItemID]; ok {
			result = append(result, RecipeListItem{
				ItemName: item.InternalName,
				ItemID:   item.ID,
			})
		}
	}
	return result, nil
}

// MockTx for transaction support
type MockTx struct {
	repo *MockRepository
}

func (t *MockTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return t.repo.GetInventory(ctx, userID)
}

func (t *MockTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return t.repo.UpdateInventory(ctx, userID, inventory)
}

func (t *MockTx) Commit(ctx context.Context) error {
	return nil
}

func (t *MockTx) Rollback(ctx context.Context) error {
	return nil
}
func (tx *MockTx) UpsertUser(ctx context.Context, user *domain.User) error { return nil }
func (tx *MockTx) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	return nil, nil
}
func (tx *MockTx) UpdateUser(ctx context.Context, user domain.User) error { return nil }
func (tx *MockTx) DeleteUser(ctx context.Context, userID string) error    { return nil }
func (tx *MockTx) DeleteInventory(ctx context.Context, userID string) error { return nil }
func (tx *MockTx) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	return nil, nil
}
func (tx *MockTx) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	return nil, nil
}
func (tx *MockTx) IsItemBuyable(ctx context.Context, itemName string) (bool, error) {
	return false, nil
}
func (tx *MockTx) GetLastCooldown(ctx context.Context, userID, action string) (*time.Time, error) {
	return nil, nil
}
func (tx *MockTx) UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error {
	return nil
}
func (tx *MockTx) MergeUsersInTransaction(ctx context.Context, primaryUserID, secondaryUserID string, mergedUser domain.User, mergedInventory domain.Inventory) error {
	return nil
}
func (tx *MockTx) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return tx.repo.GetUserByPlatformID(ctx, platform, platformID)
}

// Test helper to setup test data
func setupTestData(repo *MockRepository) {
	// Setup users
	repo.users["alice"] = &domain.User{ID: "user-alice", Username: "alice", TwitchID: "twitch-alice"}
	repo.users["bob"] = &domain.User{ID: "user-bob", Username: "bob", TwitchID: "twitch-bob"}

	// Setup items
	repo.items[domain.ItemLootbox0] = &domain.Item{ID: 1, InternalName: domain.ItemLootbox0, Description: "Basic lootbox"}
	repo.items[domain.ItemLootbox1] = &domain.Item{ID: 2, InternalName: domain.ItemLootbox1, Description: "Advanced lootbox"}
	repo.items[domain.ItemLootbox2] = &domain.Item{ID: 3, InternalName: domain.ItemLootbox2, Description: "Premium lootbox"}

	repo.itemsByID[1] = repo.items[domain.ItemLootbox0]
	repo.itemsByID[2] = repo.items[domain.ItemLootbox1]
	repo.itemsByID[3] = repo.items[domain.ItemLootbox2]

	// Setup upgrade recipe: lootbox0 -> lootbox1
	repo.recipes[1] = &domain.Recipe{
		ID:           1,
		TargetItemID: 2, // lootbox1
		BaseCost: []domain.RecipeCost{
			{ItemID: 1, Quantity: 1}, // 1 lootbox0
		},
	}

	// Setup disassemble recipe: lootbox1 -> lootbox0
	repo.disassembleRecipes[1] = &domain.DisassembleRecipe{
		ID:               1,
		SourceItemID:     2, // lootbox1
		QuantityConsumed: 1,
		Outputs: []domain.RecipeOutput{
			{ItemID: 1, Quantity: 1}, // 1 lootbox0
		},
	}

	// Link the recipes
	repo.recipeAssociations[1] = 1 // disassemble recipe 1 linked to upgrade recipe 1

	// Setup inventories
	repo.inventories["user-alice"] = &domain.Inventory{
		Slots: []domain.InventorySlot{},
	}
}

// ==================== Disassemble Tests ====================

func TestDisassembleItem_Success(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	// Give alice some lootbox1
	repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
		domain.InventorySlot{ItemID: 2, Quantity: 3}) // 3 lootbox1

	// Unlock the upgrade recipe (which unlocks the disassemble recipe)
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Disassemble 2 lootbox1
	result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)
	if err != nil {
		t.Fatalf("DisassembleItem failed: %v", err)
	}

	if result.QuantityProcessed != 2 {
		t.Errorf("Expected 2 processed, got %d", result.QuantityProcessed)
	}

	if result.Outputs[domain.ItemLootbox0] != 2 {
		t.Errorf("Expected 2 lootbox0 output, got %d", result.Outputs[domain.ItemLootbox0])
	}

	// Verify inventory
	inv, _ := repo.GetInventory(ctx, "user-alice")

	// Should have 1 lootbox1 left and 2 lootbox0
	var lootbox1Count, lootbox0Count int
	for _, slot := range inv.Slots {
		if slot.ItemID == 2 {
			lootbox1Count = slot.Quantity
		}
		if slot.ItemID == 1 {
			lootbox0Count = slot.Quantity
		}
	}

	if lootbox1Count != 1 {
		t.Errorf("Expected 1 lootbox1 remaining, got %d", lootbox1Count)
	}
	if lootbox0Count != 2 {
		t.Errorf("Expected 2 lootbox0, got %d", lootbox0Count)
	}
}

func TestDisassembleItem_InsufficientItems(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	// Give alice only 1 loot box1
	repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
		domain.InventorySlot{ItemID: 2, Quantity: 1})

	// Unlock the recipe
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Try to disassemble 2 (should only process 1)
	result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)
	if err != nil {
		t.Fatalf("DisassembleItem failed: %v", err)
	}

	if result.QuantityProcessed != 1 {
		t.Errorf("Expected 1 processed (max available), got %d", result.QuantityProcessed)
	}

	if result.Outputs[domain.ItemLootbox0] != 1 {
		t.Errorf("Expected 1 lootbox0 output, got %d", result.Outputs[domain.ItemLootbox0])
	}
}

func TestDisassembleItem_NoItems(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	// Alice has no lootbox1
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Try to disassemble
	_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox1, 1)
	if err == nil {
		t.Error("Expected error when disassembling with no items")
	}
}

func TestDisassembleItem_RecipeNotUnlocked(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	// Give alice lootbox1 but don't unlock the recipe
	repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
		domain.InventorySlot{ItemID: 2, Quantity: 1})

	// Try to disassemble without unlocked recipe
	_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox1, 1)
	if err == nil {
		t.Error("Expected error when recipe is not unlocked")
	}
}

func TestDisassembleItem_NoRecipe(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	// Try to disassemble lootbox0 which has no disassemble recipe
	_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox0, 1)
	if err == nil {
		t.Error("Expected error when item has no disassemble recipe")
	}
}

func TestDisassembleItem_RemovesSlotWhenEmpty(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	// Give alice exactly 1 lootbox1
	repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
		domain.InventorySlot{ItemID: 2, Quantity: 1})

	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Disassemble all lootbox1
	_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
	if err != nil {
		t.Fatalf("DisassembleItem failed: %v", err)
	}

	// Verify lootbox1 slot is removed
	inv, _ := repo.GetInventory(ctx, "user-alice")
	for _, slot := range inv.Slots {
		if slot.ItemID == 2 {
			t.Error("Expected lootbox1 slot to be removed")
		}
	}
}

// ==================== UpgradeItem Tests ====================

func TestUpgradeItem_Success(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	// Force RNG to fail masterwork
	svc := NewService(repo, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 }

	ctx := context.Background()

	// Give alice 2 lootbox0
	repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
		domain.InventorySlot{ItemID: 1, Quantity: 2})

	// Unlock the upgrade recipe
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Upgrade 2 lootbox0 to 2 lootbox1
	result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)
	if err != nil {
		t.Fatalf("UpgradeItem failed: %v", err)
	}

	if result.ItemName != domain.ItemLootbox1 {
		t.Errorf("Expected itemName lootbox1, got %s", result.ItemName)
	}

	if result.Quantity != 2 {
		t.Errorf("Expected 2 upgraded, got %d", result.Quantity)
	}

	// Verify inventory
	inv, _ := repo.GetInventory(ctx, "user-alice")
	var lootbox1Count, lootbox0Count int
	for _, slot := range inv.Slots {
		if slot.ItemID == 2 {
			lootbox1Count = slot.Quantity
		}
		if slot.ItemID == 1 {
			lootbox0Count = slot.Quantity
		}
	}

	if lootbox1Count != 2 {
		t.Errorf("Expected 2 lootbox1, got %d", lootbox1Count)
	}
	if lootbox0Count != 0 {
		t.Errorf("Expected 0 lootbox0, got %d", lootbox0Count)
	}
}

func TestUpgradeItem_InsufficientMaterials(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil).(*service)
	svc.rnd = func() float64 { return 1.0 }
	ctx := context.Background()

	// Give alice only 1 lootbox0
	repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
		domain.InventorySlot{ItemID: 1, Quantity: 1})

	// Unlock the recipe
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Try to upgrade 2 (should only process 1)
	result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)
	if err != nil {
		t.Fatalf("UpgradeItem failed: %v", err)
	}

	if result.Quantity != 1 {
		t.Errorf("Expected 1 upgraded (max available), got %d", result.Quantity)
	}
}

func TestUpgradeItem_NoMaterials(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	// Unlock recipe but alice has no materials
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Try to upgrade
	_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
	if err == nil {
		t.Error("Expected error when upgrading with no materials")
	}
}

func TestUpgradeItem_RecipeNotUnlocked(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	// Give alice materials but don't unlock recipe
	repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
		domain.InventorySlot{ItemID: 1, Quantity: 1})

	// Try to upgrade without unlocked recipe
	_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
	if err == nil {
		t.Error("Expected error when recipe is not unlocked")
	}
}

func TestUpgradeItem_NoRecipe(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	// Try to upgrade lootbox0 which has no recipe
	_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox0, 1)
	if err == nil {
		t.Error("Expected error when item has no recipe")
	}
}

func TestUpgradeItem_UserNotFound(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	// Try with non-existent user
	_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-nonexistent", "nonexistent", domain.ItemLootbox1, 1)
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

// ==================== GetRecipe Tests ====================

func TestGetRecipe_WithoutUsername(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	// Get recipe without username (no lock status)
	recipe, err := svc.GetRecipe(ctx, domain.ItemLootbox1, "", "", "")
	if err != nil {
		t.Fatalf("GetRecipe failed: %v", err)
	}

	if recipe.ItemName != domain.ItemLootbox1 {
		t.Errorf("Expected itemName lootbox1, got %s", recipe.ItemName)
	}

	if recipe.Locked {
		t.Error("Locked should be false when no username provided")
	}

	if len(recipe.BaseCost) != 1 {
		t.Errorf("Expected 1 base cost, got %d", len(recipe.BaseCost))
	}
}

func TestGetRecipe_Unlocked(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	// Unlock the recipe
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Get recipe with username
	recipe, err := svc.GetRecipe(ctx, domain.ItemLootbox1, domain.PlatformTwitch, "twitch-alice", "alice")
	if err != nil {
		t.Fatalf("GetRecipe failed: %v", err)
	}

	if recipe.Locked {
		t.Error("Recipe should be unlocked")
	}
}

func TestGetRecipe_Locked(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	// Don't unlock the recipe
	recipe, err := svc.GetRecipe(ctx, domain.ItemLootbox1, domain.PlatformTwitch, "twitch-alice", "alice")
	if err != nil {
		t.Fatalf("GetRecipe failed: %v", err)
	}

	if !recipe.Locked {
		t.Error("Recipe should be locked")
	}
}

func TestGetRecipe_ItemNotFound(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	// Try to get recipe for non-existent item
	_, err := svc.GetRecipe(ctx, "nonexistent", "", "", "")
	if err == nil {
		t.Error("Expected error for non-existent item")
	}
}

// ==================== GetUnlockedRecipes Tests ====================

func TestGetUnlockedRecipes_Success(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	// Unlock recipe for alice
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Get unlocked recipes
	recipes, err := svc.GetUnlockedRecipes(ctx, domain.PlatformTwitch, "twitch-alice", "alice")
	if err != nil {
		t.Fatalf("GetUnlockedRecipes failed: %v", err)
	}

	if len(recipes) != 1 {
		t.Errorf("Expected 1 unlocked recipe, got %d", len(recipes))
	}

	if len(recipes) > 0 && recipes[0].ItemName != domain.ItemLootbox1 {
		t.Errorf("Expected recipe for lootbox1, got %s", recipes[0].ItemName)
	}
}

func TestGetUnlockedRecipes_NoUnlockedRecipes(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	// Don't unlock any recipes
	recipes, err := svc.GetUnlockedRecipes(ctx, domain.PlatformTwitch, "twitch-alice", "alice")
	if err != nil {
		t.Fatalf("GetUnlockedRecipes failed: %v", err)
	}

	if len(recipes) != 0 {
		t.Errorf("Expected 0 unlocked recipes, got %d", len(recipes))
	}
}

func TestGetUnlockedRecipes_UserNotFound(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil)
	ctx := context.Background()

	// Try with non-existent user
	_, err := svc.GetUnlockedRecipes(ctx, domain.PlatformTwitch, "twitch-nonexistent", "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestUpgradeItem_Masterwork(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	mockStats := &MockStatsService{}

	// Create service and inject mock RNG that always returns 0 (triggers masterwork)
	svc := NewService(repo, nil, mockStats).(*service)
	svc.rnd = func() float64 { return 0.0 }

	ctx := context.Background()

	// Give alice 2 lootbox0
	repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
		domain.InventorySlot{ItemID: 1, Quantity: 2})

	// Unlock the upgrade recipe
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Upgrade 2 lootbox0
	result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)
	if err != nil {
		t.Fatalf("UpgradeItem failed: %v", err)
	}

	if result.ItemName != domain.ItemLootbox1 {
		t.Errorf("Expected itemName lootbox1, got %s", result.ItemName)
	}

	// Should be doubled (Masterwork)
	if result.Quantity != 4 {
		t.Errorf("Expected 4 upgraded (2 * 2), got %d", result.Quantity)
	}

	// Verify inventory
	inv, _ := repo.GetInventory(ctx, "user-alice")
	var lootbox1Count int
	for _, slot := range inv.Slots {
		if slot.ItemID == 2 {
			lootbox1Count = slot.Quantity
		}
	}

	if lootbox1Count != 4 {
		t.Errorf("Expected 4 lootbox1 in inventory, got %d", lootbox1Count)
	}

	// Verify event was logged
	foundEvent := false
	for _, e := range mockStats.events {
		if e == domain.EventCraftingCriticalSuccess {
			foundEvent = true
			break
		}
	}
	if !foundEvent {
		t.Error("Expected EventCraftingCriticalSuccess to be recorded")
	}
}
