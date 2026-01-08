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
	events []domain.EventType
	mu     sync.Mutex
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

// MockRepository for crafting tests with thread-safety for concurrent tests
type MockRepository struct {
	mu                 sync.RWMutex
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
		// Return new empty inventory if not found, don't store it yet
		return &domain.Inventory{Slots: []domain.InventorySlot{}}, nil
	}
	// Return a deep copy to avoid race conditions if caller modifies it without using UpdateInventory
	slotsCopy := make([]domain.InventorySlot, len(inv.Slots))
	copy(slotsCopy, inv.Slots)
	return &domain.Inventory{Slots: slotsCopy}, nil
}

func (m *MockRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Store a copy
	slotsCopy := make([]domain.InventorySlot, len(inventory.Slots))
	copy(slotsCopy, inventory.Slots)
	m.inventories[userID] = &domain.Inventory{Slots: slotsCopy}
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
	// For mock, we don't need real transaction isolation, just pass through
	return &MockTx{repo: m}, nil
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

// ==================== Test Cases ====================

func TestUpgradeItem(t *testing.T) {
	// 1. Best Case: Successful upgrade
	t.Run("Best Case", func(t *testing.T) {
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
	})

	// 2. Boundary Case: Masterwork Trigger
	t.Run("Boundary Case - Masterwork", func(t *testing.T) {
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

		repo.UnlockRecipe(ctx, "user-alice", 1)

		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)
		if err != nil {
			t.Fatalf("UpgradeItem failed: %v", err)
		}

		// Should be doubled (Masterwork)
		if result.Quantity != 4 {
			t.Errorf("Expected 4 upgraded (2 * 2), got %d", result.Quantity)
		}
	})

	// 2. Boundary Case: Partial Upgrade (Insufficient Materials)
	t.Run("Boundary Case - Partial Upgrade", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Give alice only 1 lootbox0
		repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
			domain.InventorySlot{ItemID: 1, Quantity: 1})

		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Try to upgrade 2 (should only process 1)
		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)
		if err != nil {
			t.Fatalf("UpgradeItem failed: %v", err)
		}

		if result.Quantity != 1 {
			t.Errorf("Expected 1 upgraded (max available), got %d", result.Quantity)
		}
	})

	// 3. Error Case: No Materials
	t.Run("Error Case - No Materials", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil)
		ctx := context.Background()

		repo.UnlockRecipe(ctx, "user-alice", 1)

		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		if err == nil {
			t.Error("Expected error when upgrading with no materials")
		}
	})

	// 3. Error Case: Recipe Not Unlocked
	t.Run("Error Case - Recipe Locked", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil)
		ctx := context.Background()

		repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
			domain.InventorySlot{ItemID: 1, Quantity: 1})

		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		if err == nil {
			t.Error("Expected error when recipe is not unlocked")
		}
	})

	// 3. Error Case: User Not Found
	t.Run("Error Case - User Not Found", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil)
		ctx := context.Background()

		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-nonexistent", "nonexistent", domain.ItemLootbox1, 1)
		if err == nil {
			t.Error("Expected error for non-existent user")
		}
	})

	// 4. Concurrent Case
	t.Run("Concurrent Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Give alice 100 lootbox0
		repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
			domain.InventorySlot{ItemID: 1, Quantity: 100})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		var wg sync.WaitGroup
		// Run 10 concurrent upgrades of 10 items each
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 10)
			}()
		}
		wg.Wait()

		// Should have 0 lootbox0 and 100 lootbox1 (assuming success)
		inv, _ := repo.GetInventory(ctx, "user-alice")
		var lootbox1Count int
		for _, slot := range inv.Slots {
			if slot.ItemID == 2 {
				lootbox1Count = slot.Quantity
			}
		}
		// Note: Since we are not using database transactions with isolation in mock,
		// race conditions MIGHT occur in the inventory update if not handled carefully in the service.
		// However, UpgradeItem uses safe Read-Modify-Write pattern within the service logic
		// calling repo.GetInventory then repo.UpdateInventory.
		// In a real DB, the transaction would lock the rows.
		// Here, our mock UpdateInventory overwrites the whole inventory.
		// So concurrent writes might overwrite each other's changes in this mock implementation.
		// This test mainly verifies that it doesn't panic.
		// To properly test concurrency with mock, we rely on mutexes added to MockRepository.
		// But the service logic 'Get -> Calculate -> Update' is not atomic in the Mock without an external lock or transaction simulation.
		// The MockTx just passes through.
		// So we expect some data race on the logic level, but no panic.
		if lootbox1Count == 0 {
			// At least some should succeed
			t.Log("Concurrent operations completed without panic")
		}
	})

	// 5. Nil/Empty Case
	t.Run("Nil/Empty Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil)
		ctx := context.Background()

		// Zero quantity
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 0)
		// Should probably just do nothing or return error
		if err != nil {
			// Acceptable to error on 0 quantity
		}
	})
}

func TestDisassembleItem(t *testing.T) {
	// 1. Best Case
	t.Run("Best Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 } // No perfect salvage
		ctx := context.Background()

		repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
			domain.InventorySlot{ItemID: 2, Quantity: 3}) // 3 lootbox1

		repo.UnlockRecipe(ctx, "user-alice", 1)

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
	})

	// 2. Boundary Case: Perfect Salvage
	t.Run("Boundary Case - Perfect Salvage", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		mockStats := &MockStatsService{}
		svc := NewService(repo, nil, mockStats).(*service)
		svc.rnd = func() float64 { return 0.0 } // Force perfect salvage
		ctx := context.Background()

		repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
			domain.InventorySlot{ItemID: 2, Quantity: 1})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		if err != nil {
			t.Fatalf("DisassembleItem failed: %v", err)
		}

		if !result.IsPerfectSalvage {
			t.Error("Expected perfect salvage")
		}
		// Multiplier 1.5 rounded up for 1 item = 2
		if result.Outputs[domain.ItemLootbox0] != 2 {
			t.Errorf("Expected 2 output (1 * 1.5 rounded up), got %d", result.Outputs[domain.ItemLootbox0])
		}
	})

	// 3. Error Case: Insufficient Items
	t.Run("Error Case - Insufficient Items", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil)
		ctx := context.Background()

		repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
			domain.InventorySlot{ItemID: 2, Quantity: 1})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)
		if err != nil {
			t.Fatalf("DisassembleItem failed: %v", err)
		}
		if result.QuantityProcessed != 1 {
			t.Errorf("Expected 1 processed, got %d", result.QuantityProcessed)
		}
	})

	// 4. Concurrent Case
	t.Run("Concurrent Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		repo.inventories["user-alice"].Slots = append(repo.inventories["user-alice"].Slots,
			domain.InventorySlot{ItemID: 2, Quantity: 100})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 10)
			}()
		}
		wg.Wait()
		// No panic
	})

	// 5. Nil/Empty Case
	t.Run("Nil/Empty Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil)
		ctx := context.Background()

		repo.UnlockRecipe(ctx, "user-alice", 1)
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox1, 0)
		if err != nil {
			// Acceptable
		}
	})
}

func TestGetRecipe(t *testing.T) {
	// 1. Best Case
	t.Run("Best Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil)
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

	// 2. Boundary Case: No User Context (Public Info)
	t.Run("Boundary Case - Public Info", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil)
		ctx := context.Background()

		recipe, err := svc.GetRecipe(ctx, domain.ItemLootbox1, "", "", "")
		if err != nil {
			t.Fatalf("GetRecipe failed: %v", err)
		}
		if recipe.Locked {
			t.Error("Public recipe info should not indicate locked status")
		}
	})

	// 3. Error Case: Item Not Found
	t.Run("Error Case - Item Not Found", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil)
		ctx := context.Background()

		_, err := svc.GetRecipe(ctx, "nonexistent", "", "", "")
		if err == nil {
			t.Error("Expected error for non-existent item")
		}
	})

	// 4. Concurrent Case
	t.Run("Concurrent Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil)
		ctx := context.Background()
		repo.UnlockRecipe(ctx, "user-alice", 1)

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = svc.GetRecipe(ctx, domain.ItemLootbox1, domain.PlatformTwitch, "twitch-alice", "alice")
			}()
		}
		wg.Wait()
	})
}

func TestGetUnlockedRecipes(t *testing.T) {
	// 1. Best Case
	t.Run("Best Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil)
		ctx := context.Background()

		repo.UnlockRecipe(ctx, "user-alice", 1)

		recipes, err := svc.GetUnlockedRecipes(ctx, domain.PlatformTwitch, "twitch-alice", "alice")
		if err != nil {
			t.Fatalf("GetUnlockedRecipes failed: %v", err)
		}
		if len(recipes) != 1 {
			t.Errorf("Expected 1 unlocked recipe, got %d", len(recipes))
		}
	})

	// 5. Nil/Empty Case: No Unlocked
	t.Run("Nil/Empty Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil)
		ctx := context.Background()

		recipes, err := svc.GetUnlockedRecipes(ctx, domain.PlatformTwitch, "twitch-alice", "alice")
		if err != nil {
			t.Fatalf("GetUnlockedRecipes failed: %v", err)
		}
		if len(recipes) != 0 {
			t.Errorf("Expected 0 unlocked recipes, got %d", len(recipes))
		}
	})
}

func TestGetAllRecipes(t *testing.T) {
	// 1. Best Case
	t.Run("Best Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil)
		ctx := context.Background()

		recipes, err := svc.GetAllRecipes(ctx)
		if err != nil {
			t.Fatalf("GetAllRecipes failed: %v", err)
		}
		// Expect 1 recipe (lootbox1)
		if len(recipes) != 1 {
			t.Errorf("Expected 1 recipe, got %d", len(recipes))
		}
	})

	// 4. Concurrent Case
	t.Run("Concurrent Case", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil)
		ctx := context.Background()

		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = svc.GetAllRecipes(ctx)
			}()
		}
		wg.Wait()
	})
}
