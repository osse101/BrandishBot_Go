package crafting

import (
	"context"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/concurrency"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// MockRepository for crafting tests
type MockRepository struct {
	users               map[string]*domain.User
	items               map[string]*domain.Item
	itemsByID           map[int]*domain.Item
	inventories         map[string]*domain.Inventory
	recipes             map[int]*domain.Recipe
	disassembleRecipes  map[int]*domain.DisassembleRecipe
	recipeAssociations  map[int]int // disassemble recipe ID -> upgrade recipe ID
	unlockedRecipes     map[string]map[int]bool
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

func (m *MockRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
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
	return []UnlockedRecipeInfo{}, nil
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

// Test helper to setup test data
func setupTestData(repo *MockRepository) {
	// Setup users
	repo.users["alice"] = &domain.User{ID: "user-alice", Username: "alice"}
	repo.users["bob"] = &domain.User{ID: "user-bob", Username: "bob"}

	// Setup items
	repo.items["lootbox0"] = &domain.Item{ID: 1, Name: "lootbox0", Description: "Basic lootbox"}
	repo.items["lootbox1"] = &domain.Item{ID: 2, Name: "lootbox1", Description: "Advanced lootbox"}
	repo.items["lootbox2"] = &domain.Item{ID: 3, Name: "lootbox2", Description: "Premium lootbox"}

	repo.itemsByID[1] = repo.items["lootbox0"]
	repo.itemsByID[2] = repo.items["lootbox1"]
	repo.itemsByID[3] = repo.items["lootbox2"]

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

func TestDisassembleItem_Success(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	lockManager := concurrency.NewLockManager()
	svc := NewService(repo, lockManager)
	ctx := context.Background()

	// Give alice some lootbox1
	repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
		domain.InventorySlot{ItemID: 2, Quantity: 3}) // 3 lootbox1

	// Unlock the upgrade recipe (which unlocks the disassemble recipe)
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Disassemble 2 lootbox1
	outputs, processed, err := svc.DisassembleItem(ctx, "alice", "twitch", "lootbox1", 2)
	if err != nil {
		t.Fatalf("DisassembleItem failed: %v", err)
	}

	if processed != 2 {
		t.Errorf("Expected 2 processed, got %d", processed)
	}

	if outputs["lootbox0"] != 2 {
		t.Errorf("Expected 2 lootbox0 output, got %d", outputs["lootbox0"])
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
	lockManager := concurrency.NewLockManager()
	svc := NewService(repo, lockManager)
	ctx := context.Background()

	// Give alice only 1 lootbox1
	repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
		domain.InventorySlot{ItemID: 2, Quantity: 1})

	// Unlock the recipe
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Try to disassemble 2 (should only process 1)
	outputs, processed, err := svc.DisassembleItem(ctx, "alice", "twitch", "lootbox1", 2)
	if err != nil {
		t.Fatalf("DisassembleItem failed: %v", err)
	}

	if processed != 1 {
		t.Errorf("Expected 1 processed (max available), got %d", processed)
	}

	if outputs["lootbox0"] != 1 {
		t.Errorf("Expected 1 lootbox0 output, got %d", outputs["lootbox0"])
	}
}

func TestDisassembleItem_NoItems(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	lockManager := concurrency.NewLockManager()
	svc := NewService(repo, lockManager)
	ctx := context.Background()

	// Alice has no lootbox1
	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Try to disassemble
	_, _, err := svc.DisassembleItem(ctx, "alice", "twitch", "lootbox1", 1)
	if err == nil {
		t.Error("Expected error when disassembling with no items")
	}
}

func TestDisassembleItem_RecipeNotUnlocked(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	lockManager := concurrency.NewLockManager()
	svc := NewService(repo, lockManager)
	ctx := context.Background()

	// Give alice lootbox1 but don't unlock the recipe
	repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
		domain.InventorySlot{ItemID: 2, Quantity: 1})

	// Try to disassemble without unlocked recipe
	_, _, err := svc.DisassembleItem(ctx, "alice", "twitch", "lootbox1", 1)
	if err == nil {
		t.Error("Expected error when recipe is not unlocked")
	}
}

func TestDisassembleItem_NoRecipe(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	lockManager := concurrency.NewLockManager()
	svc := NewService(repo, lockManager)
	ctx := context.Background()

	// Try to disassemble lootbox0 which has no disassemble recipe
	_, _, err := svc.DisassembleItem(ctx, "alice", "twitch", "lootbox0", 1)
	if err == nil {
		t.Error("Expected error when item has no disassemble recipe")
	}
}

func TestDisassembleItem_RemovesSlotWhenEmpty(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	lockManager := concurrency.NewLockManager()
	svc := NewService(repo, lockManager)
	ctx := context.Background()

	// Give alice exactly 1 lootbox1
	repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
		domain.InventorySlot{ItemID: 2, Quantity: 1})

	repo.UnlockRecipe(ctx, "user-alice", 1)

	// Disassemble all lootbox1
	_, _, err := svc.DisassembleItem(ctx, "alice", "twitch", "lootbox1", 1)
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
