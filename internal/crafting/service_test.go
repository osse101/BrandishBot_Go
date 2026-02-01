package crafting

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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

// MockRepository for crafting tests with thread-safety and row locking simulation
type MockRepository struct {
	sync.RWMutex
	users              map[string]*domain.User
	items              map[string]*domain.Item
	itemsByID          map[int]*domain.Item
	inventories        map[string]*domain.Inventory
	recipes            map[int]*domain.Recipe
	disassembleRecipes map[int]*domain.DisassembleRecipe
	recipeAssociations map[int]int // disassemble recipe ID -> upgrade recipe ID
	unlockedRecipes    map[string]map[int]bool

	// User locks for simulating DB row locking
	userLocks   map[string]*sync.Mutex
	userLocksMu sync.Mutex

	// Error injection for testing
	shouldFailBeginTx         bool
	shouldFailGetInventory    bool
	shouldFailUpdateInventory bool
	shouldFailCommit          bool
	beginTxError              error
	getInventoryError         error
	updateInventoryError      error
	commitError               error
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
		userLocks:          make(map[string]*sync.Mutex),
	}
}

// ResetErrorFlags resets all error injection flags
func (m *MockRepository) ResetErrorFlags() {
	m.Lock()
	defer m.Unlock()
	m.shouldFailBeginTx = false
	m.shouldFailGetInventory = false
	m.shouldFailUpdateInventory = false
	m.shouldFailCommit = false
	m.beginTxError = nil
	m.getInventoryError = nil
	m.updateInventoryError = nil
	m.commitError = nil
}

// GetUserLock returns a mutex for a specific user ID, creating it if necessary
func (m *MockRepository) GetUserLock(userID string) *sync.Mutex {
	m.userLocksMu.Lock()
	defer m.userLocksMu.Unlock()
	if _, ok := m.userLocks[userID]; !ok {
		m.userLocks[userID] = &sync.Mutex{}
	}
	return m.userLocks[userID]
}

func (m *MockRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	m.RLock()
	defer m.RUnlock()
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
	m.RLock()
	defer m.RUnlock()
	item, ok := m.items[itemName]
	if !ok {
		return nil, nil
	}
	return item, nil
}

func (m *MockRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	m.RLock()
	defer m.RUnlock()
	item, ok := m.itemsByID[id]
	if !ok {
		return nil, nil
	}
	return item, nil
}

func (m *MockRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	m.RLock()
	defer m.RUnlock()
	inv, ok := m.inventories[userID]
	if !ok {
		return &domain.Inventory{Slots: []domain.InventorySlot{}}, nil
	}
	// Return a copy to avoid race conditions if caller modifies it
	newInv := &domain.Inventory{
		Slots: make([]domain.InventorySlot, len(inv.Slots)),
	}
	copy(newInv.Slots, inv.Slots)
	return newInv, nil
}

func (m *MockRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	m.Lock()
	defer m.Unlock()
	// deep copy to store
	newSlots := make([]domain.InventorySlot, len(inventory.Slots))
	copy(newSlots, inventory.Slots)
	m.inventories[userID] = &domain.Inventory{Slots: newSlots}
	return nil
}

func (m *MockRepository) GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error) {
	m.RLock()
	defer m.RUnlock()
	for _, recipe := range m.recipes {
		if recipe.TargetItemID == itemID {
			return recipe, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) IsRecipeUnlocked(ctx context.Context, userID string, recipeID int) (bool, error) {
	m.RLock()
	defer m.RUnlock()
	if m.unlockedRecipes[userID] == nil {
		return false, nil
	}
	return m.unlockedRecipes[userID][recipeID], nil
}

func (m *MockRepository) UnlockRecipe(ctx context.Context, userID string, recipeID int) error {
	m.Lock()
	defer m.Unlock()
	if m.unlockedRecipes[userID] == nil {
		m.unlockedRecipes[userID] = make(map[int]bool)
	}
	m.unlockedRecipes[userID][recipeID] = true
	return nil
}

func (m *MockRepository) GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]repository.UnlockedRecipeInfo, error) {
	m.RLock()
	defer m.RUnlock()
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
	m.RLock()
	defer m.RUnlock()
	if m.shouldFailBeginTx {
		if m.beginTxError != nil {
			return nil, m.beginTxError
		}
		return nil, fmt.Errorf("failed to begin transaction")
	}
	return &MockTx{repo: m, lockedUsers: make(map[string]bool)}, nil
}

func (m *MockRepository) GetDisassembleRecipeBySourceItemID(ctx context.Context, itemID int) (*domain.DisassembleRecipe, error) {
	m.RLock()
	defer m.RUnlock()
	for _, recipe := range m.disassembleRecipes {
		if recipe.SourceItemID == itemID {
			return recipe, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) GetAssociatedUpgradeRecipeID(ctx context.Context, disassembleRecipeID int) (int, error) {
	m.RLock()
	defer m.RUnlock()
	upgradeRecipeID, ok := m.recipeAssociations[disassembleRecipeID]
	if !ok {
		return 0, nil
	}
	return upgradeRecipeID, nil
}

func (m *MockRepository) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	m.RLock()
	defer m.RUnlock()
	var result []domain.Item
	for _, id := range itemIDs {
		if item, ok := m.itemsByID[id]; ok {
			result = append(result, *item)
		}
	}
	return result, nil
}

func (m *MockRepository) GetAllRecipes(ctx context.Context) ([]repository.RecipeListItem, error) {
	m.RLock()
	defer m.RUnlock()
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

// Recipe loader methods
func (m *MockRepository) GetAllCraftingRecipes(ctx context.Context) ([]domain.Recipe, error) {
	m.RLock()
	defer m.RUnlock()
	result := make([]domain.Recipe, 0, len(m.recipes))
	for _, recipe := range m.recipes {
		result = append(result, *recipe)
	}
	return result, nil
}

func (m *MockRepository) GetAllDisassembleRecipes(ctx context.Context) ([]domain.DisassembleRecipe, error) {
	m.RLock()
	defer m.RUnlock()
	result := make([]domain.DisassembleRecipe, 0, len(m.disassembleRecipes))
	for _, recipe := range m.disassembleRecipes {
		result = append(result, *recipe)
	}
	return result, nil
}

func (m *MockRepository) GetCraftingRecipeByKey(ctx context.Context, recipeKey string) (*domain.Recipe, error) {
	m.RLock()
	defer m.RUnlock()
	for _, recipe := range m.recipes {
		if recipe.RecipeKey == recipeKey {
			return recipe, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) GetDisassembleRecipeByKey(ctx context.Context, recipeKey string) (*domain.DisassembleRecipe, error) {
	m.RLock()
	defer m.RUnlock()
	for _, recipe := range m.disassembleRecipes {
		if recipe.RecipeKey == recipeKey {
			return recipe, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) InsertCraftingRecipe(ctx context.Context, recipe *domain.Recipe) (int, error) {
	m.Lock()
	defer m.Unlock()
	maxID := 0
	for id := range m.recipes {
		if id > maxID {
			maxID = id
		}
	}
	newID := maxID + 1
	recipe.ID = newID
	m.recipes[newID] = recipe
	return newID, nil
}

func (m *MockRepository) InsertDisassembleRecipe(ctx context.Context, recipe *domain.DisassembleRecipe) (int, error) {
	m.Lock()
	defer m.Unlock()
	maxID := 0
	for id := range m.disassembleRecipes {
		if id > maxID {
			maxID = id
		}
	}
	newID := maxID + 1
	recipe.ID = newID
	m.disassembleRecipes[newID] = recipe
	return newID, nil
}

func (m *MockRepository) UpdateCraftingRecipe(ctx context.Context, recipeID int, recipe *domain.Recipe) error {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.recipes[recipeID]; !ok {
		return nil // Recipe not found, silently ignore for mock
	}
	recipe.ID = recipeID
	m.recipes[recipeID] = recipe
	return nil
}

func (m *MockRepository) UpdateDisassembleRecipe(ctx context.Context, recipeID int, recipe *domain.DisassembleRecipe) error {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.disassembleRecipes[recipeID]; !ok {
		return nil // Recipe not found, silently ignore for mock
	}
	recipe.ID = recipeID
	m.disassembleRecipes[recipeID] = recipe
	return nil
}

func (m *MockRepository) ClearDisassembleOutputs(ctx context.Context, recipeID int) error {
	m.Lock()
	defer m.Unlock()
	if recipe, ok := m.disassembleRecipes[recipeID]; ok {
		recipe.Outputs = []domain.RecipeOutput{}
	}
	return nil
}

func (m *MockRepository) InsertDisassembleOutput(ctx context.Context, recipeID int, output domain.RecipeOutput) error {
	m.Lock()
	defer m.Unlock()
	if recipe, ok := m.disassembleRecipes[recipeID]; ok {
		recipe.Outputs = append(recipe.Outputs, output)
	}
	return nil
}

func (m *MockRepository) UpsertRecipeAssociation(ctx context.Context, upgradeRecipeID, disassembleRecipeID int) error {
	m.Lock()
	defer m.Unlock()
	m.recipeAssociations[disassembleRecipeID] = upgradeRecipeID
	return nil
}

// MockTx for transaction support
type MockTx struct {
	repo        *MockRepository
	lockedUsers map[string]bool
}

func (t *MockTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	// Check for error injection
	t.repo.RLock()
	shouldFail := t.repo.shouldFailGetInventory
	injectedErr := t.repo.getInventoryError
	t.repo.RUnlock()

	if shouldFail {
		if injectedErr != nil {
			return nil, injectedErr
		}
		return nil, fmt.Errorf("failed to get inventory")
	}

	// Simulate SELECT FOR UPDATE by locking the user record
	if !t.lockedUsers[userID] {
		lock := t.repo.GetUserLock(userID)
		// fmt.Printf("Tx %p: Acquiring lock for %s\n", t, userID)
		lock.Lock()
		// fmt.Printf("Tx %p: Acquired lock for %s\n", t, userID)
		t.lockedUsers[userID] = true
	}
	return t.repo.GetInventory(ctx, userID)
}

func (t *MockTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	// Check for error injection
	t.repo.RLock()
	shouldFail := t.repo.shouldFailUpdateInventory
	injectedErr := t.repo.updateInventoryError
	t.repo.RUnlock()

	if shouldFail {
		if injectedErr != nil {
			return injectedErr
		}
		return fmt.Errorf("failed to update inventory")
	}

	// Should ideally check if lock is held
	return t.repo.UpdateInventory(ctx, userID, inventory)
}

func (t *MockTx) Commit(ctx context.Context) error {
	// Check for error injection
	t.repo.RLock()
	shouldFail := t.repo.shouldFailCommit
	injectedErr := t.repo.commitError
	t.repo.RUnlock()

	// Release all locks (even on error, like Rollback does)
	for userID := range t.lockedUsers {
		// fmt.Printf("Tx %p: Releasing lock for %s\n", t, userID)
		t.repo.GetUserLock(userID).Unlock()
	}
	t.lockedUsers = make(map[string]bool)

	if shouldFail {
		if injectedErr != nil {
			return injectedErr
		}
		return fmt.Errorf("failed to commit transaction")
	}
	return nil
}

func (t *MockTx) Rollback(ctx context.Context) error {
	// Release all locks
	for userID := range t.lockedUsers {
		// fmt.Printf("Tx %p: Releasing lock for %s (Rollback)\n", t, userID)
		t.repo.GetUserLock(userID).Unlock()
	}
	t.lockedUsers = make(map[string]bool)
	return nil
}
func (t *MockTx) UpsertUser(ctx context.Context, user *domain.User) error { return nil }
func (t *MockTx) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	return nil, nil
}
func (t *MockTx) UpdateUser(ctx context.Context, user domain.User) error   { return nil }
func (t *MockTx) DeleteUser(ctx context.Context, userID string) error      { return nil }
func (t *MockTx) DeleteInventory(ctx context.Context, userID string) error { return nil }
func (t *MockTx) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	return t.repo.GetItemsByIDs(ctx, itemIDs)
}
func (t *MockTx) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	return nil, nil
}
func (t *MockTx) IsItemBuyable(ctx context.Context, itemName string) (bool, error) {
	return false, nil
}
func (t *MockTx) GetLastCooldown(ctx context.Context, userID, action string) (*time.Time, error) {
	return nil, nil
}
func (t *MockTx) UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error {
	return nil
}
func (t *MockTx) MergeUsersInTransaction(ctx context.Context, primaryUserID, secondaryUserID string, mergedUser domain.User, mergedInventory domain.Inventory) error {
	return nil
}
func (t *MockTx) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return t.repo.GetUserByPlatformID(ctx, platform, platformID)
}

// Test helper to setup test data
func setupTestData(repo *MockRepository) {
	repo.Lock()
	defer repo.Unlock()
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

// ==================== Tests ====================

func TestDisassembleItem(t *testing.T) {
	t.Run("Best Case: Success", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 } // No perfect salvage
		ctx := context.Background()

		// Arrange: Give alice 3 lootbox1 and unlock recipe
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 2, Quantity: 3},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 2, result.QuantityProcessed)
		assert.Equal(t, 2, result.Outputs[domain.ItemLootbox0])

		inv, _ := repo.GetInventory(ctx, "user-alice")
		assert.Equal(t, 1, inv.Slots[0].Quantity, "Should have 1 lootbox1 remaining")
	})

	t.Run("Best Case: Perfect Salvage", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		mockStats := &MockStatsService{}
		svc := NewService(repo, nil, mockStats, nil, nil).(*service)
		svc.rnd = func() float64 { return 0.0 } // Trigger perfect salvage
		ctx := context.Background()

		// Arrange: Give alice 1 lootbox1
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 2, Quantity: 1},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

		// Assert
		assert.NoError(t, err)
		assert.True(t, result.IsPerfectSalvage)

		// Logic: 1 item disassembled -> 1 source consumed.
		// Recipe Output: 1 Lootbox0.
		// Perfect Salvage: ceil(1 * 1.5) = 2.
		// Total Output: 1 * 2 = 2.
		assert.Equal(t, 2, result.Outputs[domain.ItemLootbox0])

		inv, _ := repo.GetInventory(ctx, "user-alice")
		assert.Equal(t, 2, inv.Slots[0].Quantity, "Should have 2 lootbox0")

		// Verify event
		foundEvent := false
		for _, e := range mockStats.events {
			if e == domain.EventCraftingPerfectSalvage {
				foundEvent = true
				break
			}
		}
		assert.True(t, foundEvent, "Should log perfect salvage event")
	})

	t.Run("Boundary Case: Exact Items", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Arrange: Give alice exactly 2 lootbox1
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 2, Quantity: 2},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 2, result.QuantityProcessed)

		inv, _ := repo.GetInventory(ctx, "user-alice")
		foundLootbox1 := false
		for _, slot := range inv.Slots {
			if slot.ItemID == 2 {
				foundLootbox1 = true
			}
		}
		assert.False(t, foundLootbox1, "Lootbox1 slot should be removed")
	})

	t.Run("Error Case: Insufficient Items", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil)
		ctx := context.Background()

		// Arrange: Alice has 1 lootbox1, wants to disassemble 2
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 2, Quantity: 1},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 1, result.QuantityProcessed, "Should only process 1 available item")
	})

	t.Run("Error Case: Recipe Not Unlocked", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil)
		ctx := context.Background()

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 2, Quantity: 1},
		}})

		// Act
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRecipeLocked)
	})

	t.Run("Error Case: No Recipe Exists", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil)
		ctx := context.Background()

		// Act
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox0, 1) // No disassemble recipe for lootbox0

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRecipeNotFound)
	})

	t.Run("Nil/Empty Case: Empty User", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo) // Need to setup items so item validation passes
		svc := NewService(repo, nil, nil, nil, nil)
		ctx := context.Background()

		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "nonexistent", "", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("Concurrent Case: Parallel Disassemble", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Arrange: Give alice 100 items
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 2, Quantity: 100},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act: Run 10 goroutines disassembling 1 item each
		var wg sync.WaitGroup
		errChan := make(chan error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
				if err != nil {
					errChan <- err
				}
			}()
		}
		wg.Wait()
		close(errChan)

		// Assert
		for err := range errChan {
			assert.NoError(t, err)
		}

		inv, _ := repo.GetInventory(ctx, "user-alice")
		for _, slot := range inv.Slots {
			if slot.ItemID == 2 {
				// Debug log if fails
				if slot.Quantity != 90 {
					fmt.Printf("FAIL: Expected 90, got %d\n", slot.Quantity)
				}
				assert.Equal(t, 90, slot.Quantity, "Should have 90 items left")
			}
		}
	})
}

func TestUpgradeItem(t *testing.T) {
	t.Run("Best Case: Success", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 } // Fail masterwork
		ctx := context.Background()

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 2}, // 2 lootbox0
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 2, result.Quantity)
		assert.Equal(t, domain.ItemLootbox1, result.ItemName)

		inv, _ := repo.GetInventory(ctx, "user-alice")
		// Should have 0 lootbox0 and 2 lootbox1
		for _, slot := range inv.Slots {
			if slot.ItemID == 1 {
				assert.Equal(t, 0, slot.Quantity)
			}
			if slot.ItemID == 2 {
				assert.Equal(t, 2, slot.Quantity)
			}
		}
	})

	t.Run("Best Case: Masterwork", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		mockStats := &MockStatsService{}
		svc := NewService(repo, nil, mockStats, nil, nil).(*service)
		svc.rnd = func() float64 { return 0.0 } // Trigger masterwork
		ctx := context.Background()

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 2},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 4, result.Quantity, "Should double quantity")
		assert.True(t, result.IsMasterwork)

		foundEvent := false
		for _, e := range mockStats.events {
			if e == domain.EventCraftingCriticalSuccess {
				foundEvent = true
				break
			}
		}
		assert.True(t, foundEvent, "Should log critical success event")
	})

	t.Run("Error Case: Insufficient Materials", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Have 1, want 2
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 1},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Act
		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 2)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, 1, result.Quantity, "Should process max available")
	})

	t.Run("Error Case: Recipe Not Unlocked", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil)
		ctx := context.Background()

		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 2, Quantity: 2},
		}})

		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrRecipeLocked)
	})

	t.Run("Concurrent Case: Parallel Upgrades", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// 100 items
		repo.UpdateInventory(ctx, "user-alice", domain.Inventory{Slots: []domain.InventorySlot{
			{ItemID: 1, Quantity: 100},
		}})
		repo.UnlockRecipe(ctx, "user-alice", 1)

		var wg sync.WaitGroup
		errChan := make(chan error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
				if err != nil {
					errChan <- err
				}
			}()
		}
		wg.Wait()
		close(errChan)

		for err := range errChan {
			assert.NoError(t, err)
		}

		inv, _ := repo.GetInventory(ctx, "user-alice")
		for _, slot := range inv.Slots {
			if slot.ItemID == 1 {
				assert.Equal(t, 90, slot.Quantity)
			}
			if slot.ItemID == 2 {
				assert.Equal(t, 10, slot.Quantity)
			}
		}
	})
}

func TestGetRecipe(t *testing.T) {
	t.Run("Best Case: Unlocked", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil)
		ctx := context.Background()

		repo.UnlockRecipe(ctx, "user-alice", 1)
		recipe, err := svc.GetRecipe(ctx, domain.ItemLootbox1, domain.PlatformTwitch, "twitch-alice", "alice")
		assert.NoError(t, err)
		assert.False(t, recipe.Locked)
	})

	t.Run("Best Case: No User Context", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil)
		ctx := context.Background()

		recipe, err := svc.GetRecipe(ctx, domain.ItemLootbox1, "", "", "")
		assert.NoError(t, err)
		assert.False(t, recipe.Locked, "Should default to false if no user")
	})

	t.Run("Boundary Case: Locked", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil)
		ctx := context.Background()

		recipe, err := svc.GetRecipe(ctx, domain.ItemLootbox1, domain.PlatformTwitch, "twitch-alice", "alice")
		assert.NoError(t, err)
		assert.True(t, recipe.Locked)
	})

	t.Run("Error Case: Item Not Found", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil)
		ctx := context.Background()

		_, err := svc.GetRecipe(ctx, "invalid-item", "", "", "")
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrItemNotFound)
	})
}

func TestGetAllRecipes(t *testing.T) {
	t.Run("Best Case: Returns Recipes", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil)
		ctx := context.Background()

		recipes, err := svc.GetAllRecipes(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, recipes)
	})
}

func TestShutdown(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo, nil, nil, nil, nil)
	assert.NoError(t, svc.Shutdown(context.Background()))
}

// Additional test for GetUnlockedRecipes
func TestGetUnlockedRecipes(t *testing.T) {
	t.Run("Best Case: Returns Unlocked", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil)
		ctx := context.Background()

		repo.UnlockRecipe(ctx, "user-alice", 1)
		recipes, err := svc.GetUnlockedRecipes(ctx, domain.PlatformTwitch, "twitch-alice", "alice")
		assert.NoError(t, err)
		assert.Len(t, recipes, 1)
	})

	t.Run("Nil/Empty Case: No Unlocked", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil)
		ctx := context.Background()

		recipes, err := svc.GetUnlockedRecipes(ctx, domain.PlatformTwitch, "twitch-alice", "alice")
		assert.NoError(t, err)
		assert.Empty(t, recipes)
	})
}

// Example: Concurrent access to GetRecipe is read-only, but let's verify it doesn't race
func TestGetRecipe_Concurrent(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil, nil)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc.GetRecipe(ctx, domain.ItemLootbox1, domain.PlatformTwitch, "twitch-alice", "alice")
		}()
	}
	wg.Wait()
}

// Phase 3: Input Validation Tests

func TestUpgradeItem_InputValidation(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil, nil)
	ctx := context.Background()

	t.Run("Negative Quantity", func(t *testing.T) {
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, -1)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidQuantity)
		assert.Contains(t, err.Error(), "quantity must be positive")
	})

	t.Run("Zero Quantity", func(t *testing.T) {
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 0)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidQuantity)
		assert.Contains(t, err.Error(), "quantity must be positive")
	})

	t.Run("Empty Platform", func(t *testing.T) {
		_, err := svc.UpgradeItem(ctx, "", "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Contains(t, err.Error(), "platform and platformID cannot be empty")
	})

	t.Run("Empty PlatformID", func(t *testing.T) {
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Contains(t, err.Error(), "platform and platformID cannot be empty")
	})

	t.Run("Invalid Platform", func(t *testing.T) {
		_, err := svc.UpgradeItem(ctx, "invalid-platform", "some-id", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidPlatform)
		assert.Contains(t, err.Error(), "invalid platform")
	})

	t.Run("Empty Item Name", func(t *testing.T) {
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", "", 1)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Contains(t, err.Error(), "item name cannot be empty")
	})
}

func TestDisassembleItem_InputValidation(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil, nil)
	ctx := context.Background()

	t.Run("Negative Quantity", func(t *testing.T) {
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, -1)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidQuantity)
		assert.Contains(t, err.Error(), "quantity must be positive")
	})

	t.Run("Zero Quantity", func(t *testing.T) {
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 0)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidQuantity)
		assert.Contains(t, err.Error(), "quantity must be positive")
	})

	t.Run("Empty Platform", func(t *testing.T) {
		_, err := svc.DisassembleItem(ctx, "", "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Contains(t, err.Error(), "platform and platformID cannot be empty")
	})

	t.Run("Empty PlatformID", func(t *testing.T) {
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Contains(t, err.Error(), "platform and platformID cannot be empty")
	})

	t.Run("Invalid Platform", func(t *testing.T) {
		_, err := svc.DisassembleItem(ctx, "invalid-platform", "some-id", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidPlatform)
		assert.Contains(t, err.Error(), "invalid platform")
	})

	t.Run("Empty Item Name", func(t *testing.T) {
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", "", 1)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Contains(t, err.Error(), "item name cannot be empty")
	})
}

func TestGetRecipe_InputValidation(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil, nil)
	ctx := context.Background()

	t.Run("Empty Item Name", func(t *testing.T) {
		_, err := svc.GetRecipe(ctx, "", domain.PlatformTwitch, "twitch-alice", "alice")
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Contains(t, err.Error(), "item name cannot be empty")
	})
}

func TestGetUnlockedRecipes_InputValidation(t *testing.T) {
	repo := NewMockRepository()
	setupTestData(repo)
	svc := NewService(repo, nil, nil, nil, nil)
	ctx := context.Background()

	t.Run("Empty Platform", func(t *testing.T) {
		_, err := svc.GetUnlockedRecipes(ctx, "", "twitch-alice", "alice")
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Contains(t, err.Error(), "platform and platformID cannot be empty")
	})

	t.Run("Empty PlatformID", func(t *testing.T) {
		_, err := svc.GetUnlockedRecipes(ctx, domain.PlatformTwitch, "", "alice")
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Contains(t, err.Error(), "platform and platformID cannot be empty")
	})

	t.Run("Invalid Platform", func(t *testing.T) {
		_, err := svc.GetUnlockedRecipes(ctx, "invalid-platform", "some-id", "alice")
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidPlatform)
		assert.Contains(t, err.Error(), "invalid platform")
	})
}

// Phase 4: Transaction Failure Tests

func TestUpgradeItem_TransactionFailures(t *testing.T) {
	t.Run("BeginTx Failure", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Unlock recipe and add materials
		repo.UnlockRecipe(ctx, "user-alice", 1)
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10}, // lootbox_tier0
			},
		}

		// Get original inventory state
		originalInv, _ := repo.GetInventory(ctx, "user-alice")
		originalQuantity := originalInv.Slots[0].Quantity

		// Inject BeginTx error
		repo.Lock()
		repo.shouldFailBeginTx = true
		repo.Unlock()

		// Attempt upgrade
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")

		// Verify inventory unchanged
		repo.ResetErrorFlags()
		inv, _ := repo.GetInventory(ctx, "user-alice")
		assert.Equal(t, originalQuantity, inv.Slots[0].Quantity, "inventory should be unchanged after BeginTx failure")
	})

	t.Run("GetInventory Failure", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Unlock recipe and add materials
		repo.UnlockRecipe(ctx, "user-alice", 1)
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10},
			},
		}

		originalQuantity := repo.inventories["user-alice"].Slots[0].Quantity

		// Inject GetInventory error
		repo.Lock()
		repo.shouldFailGetInventory = true
		repo.Unlock()

		// Attempt upgrade
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get inventory")

		// Verify inventory unchanged (rollback should have occurred)
		repo.ResetErrorFlags()
		inv, _ := repo.GetInventory(ctx, "user-alice")
		assert.Equal(t, originalQuantity, inv.Slots[0].Quantity, "inventory should be unchanged after GetInventory failure")
	})

	t.Run("UpdateInventory Failure", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Unlock recipe and add materials
		repo.UnlockRecipe(ctx, "user-alice", 1)
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10},
			},
		}

		originalQuantity := repo.inventories["user-alice"].Slots[0].Quantity

		// Inject UpdateInventory error
		repo.Lock()
		repo.shouldFailUpdateInventory = true
		repo.Unlock()

		// Attempt upgrade
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update inventory")

		// Verify inventory unchanged (rollback should have occurred)
		repo.ResetErrorFlags()
		inv, _ := repo.GetInventory(ctx, "user-alice")
		assert.Equal(t, originalQuantity, inv.Slots[0].Quantity, "inventory should be unchanged after UpdateInventory failure")
	})

	t.Run("Commit Failure", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Unlock recipe and add materials
		repo.UnlockRecipe(ctx, "user-alice", 1)
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10},
			},
		}

		// Inject Commit error
		repo.Lock()
		repo.shouldFailCommit = true
		repo.Unlock()

		// Attempt upgrade
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit transaction")

		// Note: Our mock doesn't support true transaction isolation, so inventory changes
		// persist even on commit failure. In a real database, the transaction would be rolled back.
		// The important thing is that the service correctly returns the error.
		repo.ResetErrorFlags()
	})
}

func TestDisassembleItem_TransactionFailures(t *testing.T) {
	t.Run("BeginTx Failure", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Unlock recipe and add item to disassemble
		repo.UnlockRecipe(ctx, "user-alice", 1)
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 2, Quantity: 5}, // lootbox_tier1
			},
		}

		originalQuantity := repo.inventories["user-alice"].Slots[0].Quantity

		// Inject BeginTx error
		repo.Lock()
		repo.shouldFailBeginTx = true
		repo.Unlock()

		// Attempt disassemble
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")

		// Verify inventory unchanged
		repo.ResetErrorFlags()
		inv, _ := repo.GetInventory(ctx, "user-alice")
		assert.Equal(t, originalQuantity, inv.Slots[0].Quantity, "inventory should be unchanged after BeginTx failure")
	})

	t.Run("GetInventory Failure", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Unlock recipe and add item
		repo.UnlockRecipe(ctx, "user-alice", 1)
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 2, Quantity: 5},
			},
		}

		originalQuantity := repo.inventories["user-alice"].Slots[0].Quantity

		// Inject GetInventory error
		repo.Lock()
		repo.shouldFailGetInventory = true
		repo.Unlock()

		// Attempt disassemble
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get inventory")

		// Verify inventory unchanged
		repo.ResetErrorFlags()
		inv, _ := repo.GetInventory(ctx, "user-alice")
		assert.Equal(t, originalQuantity, inv.Slots[0].Quantity, "inventory should be unchanged after GetInventory failure")
	})

	t.Run("UpdateInventory Failure", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Unlock recipe and add item
		repo.UnlockRecipe(ctx, "user-alice", 1)
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 2, Quantity: 5},
			},
		}

		originalQuantity := repo.inventories["user-alice"].Slots[0].Quantity

		// Inject UpdateInventory error
		repo.Lock()
		repo.shouldFailUpdateInventory = true
		repo.Unlock()

		// Attempt disassemble
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update inventory")

		// Verify inventory unchanged
		repo.ResetErrorFlags()
		inv, _ := repo.GetInventory(ctx, "user-alice")
		assert.Equal(t, originalQuantity, inv.Slots[0].Quantity, "inventory should be unchanged after UpdateInventory failure")
	})

	t.Run("Commit Failure", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 }
		ctx := context.Background()

		// Unlock recipe and add item
		repo.UnlockRecipe(ctx, "user-alice", 1)
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 2, Quantity: 5},
			},
		}

		// Inject Commit error
		repo.Lock()
		repo.shouldFailCommit = true
		repo.Unlock()

		// Attempt disassemble
		_, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox1, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit transaction")

		// Note: Our mock doesn't support true transaction isolation, so inventory changes
		// persist even on commit failure. In a real database, the transaction would be rolled back.
		// The important thing is that the service correctly returns the error.
		repo.ResetErrorFlags()
	})
}

// Phase 6: Multi-Material Recipe Tests

func TestUpgradeItem_MultiMaterialRecipe(t *testing.T) {
	t.Run("Both Materials Available", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 } // Prevent masterwork
		ctx := context.Background()

		// Create a recipe requiring 2 materials: 2x lootbox0 + 1x lootbox1 -> lootbox2
		repo.recipes[99] = &domain.Recipe{
			ID:           99,
			TargetItemID: 3, // lootbox_tier2
			BaseCost: []domain.RecipeCost{
				{ItemID: 1, Quantity: 2}, // 2x lootbox_tier0
				{ItemID: 2, Quantity: 1}, // 1x lootbox_tier1
			},
		}

		// Unlock the recipe
		repo.UnlockRecipe(ctx, "user-alice", 99)

		// Give user both materials
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10}, // lootbox_tier0
				{ItemID: 2, Quantity: 5},  // lootbox_tier1
			},
		}

		// Craft 2 items
		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox2, 2)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Verify both materials consumed correctly
		inv, _ := repo.GetInventory(ctx, "user-alice")
		lootbox0Slot := -1
		lootbox1Slot := -1
		lootbox2Slot := -1
		for i, slot := range inv.Slots {
			if slot.ItemID == 1 {
				lootbox0Slot = i
			} else if slot.ItemID == 2 {
				lootbox1Slot = i
			} else if slot.ItemID == 3 {
				lootbox2Slot = i
			}
		}
		assert.Equal(t, 6, inv.Slots[lootbox0Slot].Quantity, "should consume 4x lootbox0 (2 per craft)")
		assert.Equal(t, 3, inv.Slots[lootbox1Slot].Quantity, "should consume 2x lootbox1 (1 per craft)")
		assert.NotEqual(t, -1, lootbox2Slot, "should have created lootbox2")
		assert.Equal(t, 2, inv.Slots[lootbox2Slot].Quantity, "should create 2x lootbox2")
	})

	t.Run("Limited By Scarcest Material", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 } // Prevent masterwork
		ctx := context.Background()

		// Create a recipe requiring 2 materials
		repo.recipes[99] = &domain.Recipe{
			ID:           99,
			TargetItemID: 3,
			BaseCost: []domain.RecipeCost{
				{ItemID: 1, Quantity: 2}, // 2x lootbox_tier0
				{ItemID: 2, Quantity: 1}, // 1x lootbox_tier1
			},
		}
		repo.UnlockRecipe(ctx, "user-alice", 99)

		// Give user materials where lootbox1 is the bottleneck
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 100}, // Plenty of lootbox0
				{ItemID: 2, Quantity: 2},   // Only 2 lootbox1 (bottleneck)
			},
		}

		// Request 10 crafts, but should only do 2 (limited by lootbox1)
		result, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox2, 10)
		assert.NoError(t, err)
		assert.Equal(t, 2, result.Quantity, "should be limited by scarcest material (lootbox1)")
	})

	t.Run("One Material Missing", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 } // Prevent masterwork
		ctx := context.Background()

		// Create a recipe requiring 2 materials
		repo.recipes[99] = &domain.Recipe{
			ID:           99,
			TargetItemID: 3,
			BaseCost: []domain.RecipeCost{
				{ItemID: 1, Quantity: 2}, // 2x lootbox_tier0
				{ItemID: 2, Quantity: 1}, // 1x lootbox_tier1
			},
		}
		repo.UnlockRecipe(ctx, "user-alice", 99)

		// Give user only one material
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 1, Quantity: 10}, // Has lootbox0
				// Missing lootbox1
			},
		}

		// Should fail due to missing material
		_, err := svc.UpgradeItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox2, 1)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInsufficientQuantity)
	})
}

// Phase 7: Multi-Output Disassemble Tests

func TestDisassembleItem_MultipleOutputs(t *testing.T) {
	t.Run("Multiple Outputs Added Correctly", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 1.0 } // No perfect salvage
		ctx := context.Background()

		// Create disassemble recipe with multiple outputs: lootbox2 -> 2x lootbox0 + 1x lootbox1
		repo.disassembleRecipes[99] = &domain.DisassembleRecipe{
			ID:               99,
			SourceItemID:     3, // lootbox_tier2
			QuantityConsumed: 1,
			Outputs: []domain.RecipeOutput{
				{ItemID: 1, Quantity: 2}, // 2x lootbox_tier0
				{ItemID: 2, Quantity: 1}, // 1x lootbox_tier1
			},
		}
		repo.recipeAssociations[99] = 1 // Associate with upgrade recipe 1
		repo.UnlockRecipe(ctx, "user-alice", 1)

		// Give user the item to disassemble
		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 3, Quantity: 2}, // 2x lootbox_tier2
			},
		}

		// Disassemble 2 items
		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox2, 2)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.QuantityProcessed)
		assert.False(t, result.IsPerfectSalvage)

		// Verify all outputs in result map
		assert.Contains(t, result.Outputs, domain.ItemLootbox0)
		assert.Contains(t, result.Outputs, domain.ItemLootbox1)
		assert.Equal(t, 4, result.Outputs[domain.ItemLootbox0], "should get 4x lootbox0 (2 per disassemble)")
		assert.Equal(t, 2, result.Outputs[domain.ItemLootbox1], "should get 2x lootbox1 (1 per disassemble)")
	})

	t.Run("Perfect Salvage Applied To All Outputs", func(t *testing.T) {
		repo := NewMockRepository()
		setupTestData(repo)
		svc := NewService(repo, nil, nil, nil, nil).(*service)
		svc.rnd = func() float64 { return 0.0 } // Always perfect salvage
		ctx := context.Background()

		// Create disassemble recipe with multiple outputs
		repo.disassembleRecipes[99] = &domain.DisassembleRecipe{
			ID:               99,
			SourceItemID:     3,
			QuantityConsumed: 1,
			Outputs: []domain.RecipeOutput{
				{ItemID: 1, Quantity: 2}, // 2x lootbox_tier0
				{ItemID: 2, Quantity: 1}, // 1x lootbox_tier1
			},
		}
		repo.recipeAssociations[99] = 1
		repo.UnlockRecipe(ctx, "user-alice", 1)

		repo.inventories["user-alice"] = &domain.Inventory{
			Slots: []domain.InventorySlot{
				{ItemID: 3, Quantity: 1},
			},
		}

		result, err := svc.DisassembleItem(ctx, domain.PlatformTwitch, "twitch-alice", "alice", domain.ItemLootbox2, 1)
		assert.NoError(t, err)
		assert.True(t, result.IsPerfectSalvage)
		assert.Equal(t, PerfectSalvageMultiplier, result.Multiplier)

		// Verify perfect salvage multiplier applied to all outputs
		// For lootbox0: base 2 * 1.5 = 3 (ceil)
		// For lootbox1: base 1 * 1.5 = 2 (ceil)
		assert.Equal(t, 3, result.Outputs[domain.ItemLootbox0], "perfect salvage should apply 1.5x multiplier to lootbox0")
		assert.Equal(t, 2, result.Outputs[domain.ItemLootbox1], "perfect salvage should apply 1.5x multiplier to lootbox1")
	})
}
