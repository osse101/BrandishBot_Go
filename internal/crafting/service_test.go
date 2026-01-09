package crafting

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// MockStatsService for crafting tests
type MockStatsService struct {
	mu     sync.Mutex
	events []domain.EventType
}

func (m *MockStatsService) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, eventData map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, eventType)
	return nil
}

// Stubs for other interface methods not used in these tests
func (m *MockStatsService) GetUserStats(ctx context.Context, userID string, period string) (*domain.StatsSummary, error) {
	return nil, nil
}
func (m *MockStatsService) GetUserCurrentStreak(ctx context.Context, userID string) (int, error) {
	return 0, nil
}
func (m *MockStatsService) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) {
	return nil, nil
}
func (m *MockStatsService) GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error) {
	return nil, nil
}

// MockRepository for crafting tests - Thread Safe
type MockRepository struct {
	mu                 sync.RWMutex
	txLock             sync.Mutex // Serializes transactions to simulate DB locking
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
	m.mu.RLock()
	defer m.mu.RUnlock()
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
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.items[itemName]
	if !ok {
		return nil, nil
	}
	return item, nil
}

func (m *MockRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.itemsByID[id]
	if !ok {
		return nil, nil
	}
	return item, nil
}

func (m *MockRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inv, ok := m.inventories[userID]
	if !ok {
		return &domain.Inventory{Slots: []domain.InventorySlot{}}, nil
	}
	// Return a copy to simulate DB separation and avoid race conditions on the slice
	invCopy := &domain.Inventory{
		Slots: make([]domain.InventorySlot, len(inv.Slots)),
	}
	copy(invCopy.Slots, inv.Slots)
	return invCopy, nil
}

func (m *MockRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Store a copy
	invCopy := domain.Inventory{
		Slots: make([]domain.InventorySlot, len(inventory.Slots)),
	}
	copy(invCopy.Slots, inventory.Slots)
	m.inventories[userID] = &invCopy
	return nil
}

func (m *MockRepository) GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, recipe := range m.recipes {
		if recipe.TargetItemID == itemID {
			return recipe, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) IsRecipeUnlocked(ctx context.Context, userID string, recipeID int) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.unlockedRecipes[userID] == nil {
		return false, nil
	}
	return m.unlockedRecipes[userID][recipeID], nil
}

func (m *MockRepository) UnlockRecipe(ctx context.Context, userID string, recipeID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.unlockedRecipes[userID] == nil {
		m.unlockedRecipes[userID] = make(map[int]bool)
	}
	m.unlockedRecipes[userID][recipeID] = true
	return nil
}

func (m *MockRepository) GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]repository.UnlockedRecipeInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []repository.UnlockedRecipeInfo
	if m.unlockedRecipes[userID] == nil {
		return result, nil
	}

	for recipeID := range m.unlockedRecipes[userID] {
		if recipe, ok := m.recipes[recipeID]; ok {
			if item, ok := m.itemsByID[recipe.TargetItemID]; ok {
				result = append(result, repository.UnlockedRecipeInfo{
					ItemName: item.InternalName,
					ItemID:   item.ID,
				})
			}
		}
	}
	return result, nil
}

func (m *MockRepository) BeginTx(ctx context.Context) (repository.CraftingTx, error) {
	// Simulate "SELECT FOR UPDATE" or DB transaction serialization
	// This prevents concurrent tests from having race conditions on read-modify-write cycles
	m.txLock.Lock()
	return &MockTx{repo: m, active: true}, nil
}

func (m *MockRepository) GetDisassembleRecipeBySourceItemID(ctx context.Context, itemID int) (*domain.DisassembleRecipe, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, recipe := range m.disassembleRecipes {
		if recipe.SourceItemID == itemID {
			return recipe, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) GetAssociatedUpgradeRecipeID(ctx context.Context, disassembleRecipeID int) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	upgradeRecipeID, ok := m.recipeAssociations[disassembleRecipeID]
	if !ok {
		return 0, nil
	}
	return upgradeRecipeID, nil
}

func (m *MockRepository) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []domain.Item
	for _, id := range itemIDs {
		if item, ok := m.itemsByID[id]; ok {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (m *MockRepository) GetAllRecipes(ctx context.Context) ([]repository.RecipeListItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []repository.RecipeListItem
	for _, recipe := range m.recipes {
		if item, ok := m.itemsByID[recipe.TargetItemID]; ok {
			result = append(result, repository.RecipeListItem{
				ItemName: item.InternalName,
				ItemID:   item.ID,
			})
		}
	}
	return result, nil
}

// MockTx for transaction support
type MockTx struct {
	repo   *MockRepository
	active bool
	mu     sync.Mutex
}

func (t *MockTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return t.repo.GetInventory(ctx, userID)
}

func (t *MockTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return t.repo.UpdateInventory(ctx, userID, inventory)
}

func (t *MockTx) Commit(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.active {
		return nil
	}
	t.active = false
	t.repo.txLock.Unlock()
	return nil
}

func (t *MockTx) Rollback(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.active {
		return nil
	}
	t.active = false
	t.repo.txLock.Unlock()
	return nil
}
func (tx *MockTx) UpsertUser(ctx context.Context, user *domain.User) error { return nil }
func (tx *MockTx) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	return nil, nil
}
func (tx *MockTx) UpdateUser(ctx context.Context, user domain.User) error   { return nil }
func (tx *MockTx) DeleteUser(ctx context.Context, userID string) error      { return nil }
func (tx *MockTx) DeleteInventory(ctx context.Context, userID string) error { return nil }
func (tx *MockTx) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	return tx.repo.GetItemsByIDs(ctx, itemIDs)
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

// ==================== UpgradeItem Tests ====================

func TestUpgradeItem(t *testing.T) {
	// 5-Case Testing Model:
	// 1. Best Case: Successful upgrade
	// 2. Boundary Case: Exact materials, max possible
	// 3. Error Case: Insufficient materials, no recipe, invalid user
	// 4. Concurrent Case: Parallel upgrades
	// 5. Nil/Empty Case: Empty inputs

	t.Run("Best Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		// Force RNG to fail masterwork
		svc := NewService(repo, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Give alice 2 lootbox0
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 2},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)
		if err != nil {
			t.Fatalf("UpgradeItem failed: %v", err)
		}

		if result.Quantity != 2 {
			t.Errorf("Expected 2 upgraded, got %d", result.Quantity)
		}
	})

	t.Run("Boundary Case - Exact Materials", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Give alice exactly 1 lootbox0
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 1},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		if err != nil {
			t.Fatalf("UpgradeItem failed: %v", err)
		}
		if result.Quantity != 1 {
			t.Errorf("Expected 1 upgraded, got %d", result.Quantity)
		}

		// Verify inventory empty
		inv, _ := repo.GetInventory(ctx, "user-alice")
		for _, slot := range inv.Slots {
			if slot.ItemID == 1 && slot.Quantity > 0 {
				t.Error("Expected 0 lootbox0 left")
			}
		}
	})

	t.Run("Error Case - Insufficient Materials", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil)
		ctx := context.Background()

		// Give alice 1 lootbox0, try to upgrade 2
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 1},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// The service logic currently caps it at max possible (1), so it shouldn't error, but return 1.
		// Wait, looking at implementation:
		// maxPossible := calculateMaxPossibleCrafts...
		// if maxPossible == 0 { return error }
		// actualQuantity := min(maxPossible, quantity)
		// So it should succeed with 1.

		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)
		if err != nil {
			t.Fatalf("UpgradeItem failed: %v", err)
		}
		if result.Quantity != 1 {
			t.Errorf("Expected 1 upgraded, got %d", result.Quantity)
		}
	})

	t.Run("Error Case - No Recipe", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil)
		ctx := context.Background()

		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox0, 1)
		if err == nil {
			t.Error("Expected error when item has no recipe")
		}
	})

	t.Run("Feature Case - Masterwork", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		mockStats := &MockStatsService{}
		svc := NewService(repo, nil, mockStats, nil).(*service)
		svc.rnd = func() float64 { return 0.0 } // Force masterwork
		ctx := context.Background()

		// Give alice 2 lootbox0
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 2},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)
		if err != nil {
			t.Fatalf("UpgradeItem failed: %v", err)
		}

		// Should produce 4 (2 * 2)
		if result.Quantity != 4 {
			t.Errorf("Expected 4 items (masterwork), got %d", result.Quantity)
		}
		if !result.IsMasterwork {
			t.Error("Expected IsMasterwork to be true")
		}

		// Verify event
		mockStats.mu.Lock()
		foundEvent := false
		for _, e := range mockStats.events {
			if e == domain.EventCraftingCriticalSuccess {
				foundEvent = true
				break
			}
		}
		mockStats.mu.Unlock()
		if !foundEvent {
			t.Error("Expected EventCraftingCriticalSuccess")
		}
	})

	t.Run("Concurrent Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Give alice 100 lootbox0
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 100},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		var wg sync.WaitGroup
		// Run 10 concurrent upgrades of 1 item each
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
			}()
		}
		wg.Wait()

		// Should have used 10 items, created 10 items
		inv, _ := repo.GetInventory(ctx, "user-alice")
		lootbox0 := 0
		lootbox1 := 0
		for _, slot := range inv.Slots {
			if slot.ItemID == 1 {
				lootbox0 = slot.Quantity
			}
			if slot.ItemID == 2 {
				lootbox1 = slot.Quantity
			}
		}

		if lootbox0 != 90 {
			t.Errorf("Expected 90 lootbox0 left, got %d", lootbox0)
		}
		if lootbox1 != 10 {
			t.Errorf("Expected 10 lootbox1 created, got %d", lootbox1)
		}
	})

	t.Run("Nil/Empty Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil)
		ctx := context.Background()

		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "", "", domain.ItemLootbox1, 1)
		if err == nil {
			t.Error("Expected error for empty user")
		}
	})
}

// ==================== DisassembleItem Tests ====================

func TestDisassembleItem(t *testing.T) {
	// 5-Case Testing Model:
	// 1. Best Case: Successful disassemble
	// 2. Boundary Case: Disassemble all items
	// 3. Error Case: No items, recipe not unlocked
	// 4. Concurrent Case: Parallel disassemble
	// 5. Nil/Empty Case: Empty item name

	t.Run("Best Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 } // No perfect salvage
		ctx := context.Background()

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 2, Quantity: 3}, // 3 lootbox1
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1) // Unlock upgrade (which unlocks disassemble)

		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)
		if err != nil {
			t.Fatalf("DisassembleItem failed: %v", err)
		}

		if result.QuantityProcessed != 2 {
			t.Errorf("Expected 2 processed, got %d", result.QuantityProcessed)
		}
	})

	t.Run("Boundary Case - Disassemble Max", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 2, Quantity: 5},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Request more than available
		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 10)
		if err != nil {
			t.Fatalf("DisassembleItem failed: %v", err)
		}
		if result.QuantityProcessed != 5 {
			t.Errorf("Expected 5 processed, got %d", result.QuantityProcessed)
		}
	})

	t.Run("Error Case - Recipe Not Unlocked", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil)
		ctx := context.Background()

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 2, Quantity: 1},
		}})

		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		if err == nil {
			t.Error("Expected error when recipe not unlocked")
		}
	})

	t.Run("Concurrent Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 2, Quantity: 20},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
			}()
		}
		wg.Wait()

		inv, _ := repo.GetInventory(ctx, "user-alice")
		lootbox1 := 0
		lootbox0 := 0
		for _, slot := range inv.Slots {
			if slot.ItemID == 2 {
				lootbox1 = slot.Quantity
			}
			if slot.ItemID == 1 {
				lootbox0 = slot.Quantity
			}
		}

		if lootbox1 != 10 {
			t.Errorf("Expected 10 lootbox1 left, got %d", lootbox1)
		}
		if lootbox0 != 10 {
			t.Errorf("Expected 10 lootbox0 created, got %d", lootbox0)
		}
	})

	t.Run("Nil/Empty Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil)
		ctx := context.Background()

		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", "", 1)
		if err == nil {
			t.Error("Expected error for empty item name")
		}
	})
}

// ==================== GetRecipe Tests ====================

func TestGetRecipe(t *testing.T) {
	// 5-Case Testing Model
	// 1. Best Case: Get recipe info
	// 2. Boundary Case: (Not applicable for getter)
	// 3. Error Case: Item not found
	// 4. Concurrent Case: Parallel reads
	// 5. Nil/Empty Case: No username provided

	t.Run("Best Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil)
		ctx := context.Background()
		repo.UnlockRecipe(ctx, "user-alice", 1)

		recipe, err := svc.GetRecipe(ctx, domain.ItemLootbox1, domain.PlatformTwitch, "twitch-alice", "alice")
		if err != nil {
			t.Fatalf("GetRecipe failed: %v", err)
		}
		if recipe.Locked {
			t.Error("Recipe should be unlocked")
		}
	})

	t.Run("Boundary Case - Locked Recipe", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil)
		ctx := context.Background()
		// Don't unlock recipe

		recipe, err := svc.GetRecipe(ctx, domain.ItemLootbox1, domain.PlatformTwitch, "twitch-alice", "alice")
		if err != nil {
			t.Fatalf("GetRecipe failed: %v", err)
		}
		if !recipe.Locked {
			t.Error("Recipe should be locked")
		}
	})

	t.Run("Error Case - Item Not Found", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil)
		ctx := context.Background()

		_, err := svc.GetRecipe(ctx, "nonexistent", "", "", "")
		if err == nil {
			t.Error("Expected error for nonexistent item")
		}
	})

	t.Run("Concurrent Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil)
		ctx := context.Background()

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = svc.GetRecipe(ctx, domain.ItemLootbox1, "", "", "")
			}()
		}
		wg.Wait()
	})

	t.Run("Nil/Empty Case - No User", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil)
		ctx := context.Background()

		recipe, err := svc.GetRecipe(ctx, domain.ItemLootbox1, "", "", "")
		if err != nil {
			t.Fatalf("GetRecipe failed: %v", err)
		}
		if recipe.Locked {
			t.Error("Recipe should not be locked when no user check performed")
		}
	})
}

// ==================== GetUnlockedRecipes Tests ====================

func TestGetUnlockedRecipes(t *testing.T) {
	// 5-Case Testing Model
	// 1. Best Case: Get list
	// 2. Boundary Case: No recipes
	// 3. Error Case: User not found
	// 4. Concurrent Case: Parallel reads
	// 5. Nil/Empty Case: (Covered by Boundary)

	t.Run("Best Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil)
		ctx := context.Background()
		repo.UnlockRecipe(ctx, "user-alice", 1)

		list, err := svc.GetUnlockedRecipes(ctx, domain.PlatformTwitch, "twitch-alice", "alice")
		if err != nil {
			t.Fatalf("GetUnlockedRecipes failed: %v", err)
		}
		if len(list) != 1 {
			t.Errorf("Expected 1 recipe, got %d", len(list))
		}
	})

	t.Run("Boundary Case - No Recipes", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil)
		ctx := context.Background()

		list, err := svc.GetUnlockedRecipes(ctx, domain.PlatformTwitch, "twitch-alice", "alice")
		if err != nil {
			t.Fatalf("GetUnlockedRecipes failed: %v", err)
		}
		if len(list) != 0 {
			t.Errorf("Expected 0 recipes, got %d", len(list))
		}
	})

	t.Run("Error Case - User Not Found", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil)
		ctx := context.Background()

		_, err := svc.GetUnlockedRecipes(ctx, domain.PlatformTwitch, "twitch-unknown", "unknown")
		if err == nil {
			t.Error("Expected error for unknown user")
		}
	})

	t.Run("Concurrent Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil)
		ctx := context.Background()

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = svc.GetUnlockedRecipes(ctx, domain.PlatformTwitch, "twitch-alice", "alice")
			}()
		}
		wg.Wait()
	})
}
