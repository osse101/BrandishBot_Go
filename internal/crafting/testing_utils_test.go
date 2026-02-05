package crafting

import (
	"context"
	"fmt"
	"sync"
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

// MockJobService for testing XP awards
type MockJobService struct {
	mu    sync.Mutex
	calls []struct {
		UserID   string
		JobKey   string
		Amount   int
		Source   string
		Metadata map[string]interface{}
	}
	blockChan chan struct{} // If set, AwardXP waits for this channel to close
}

func (m *MockJobService) AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error) {
	if m.blockChan != nil {
		<-m.blockChan
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, struct {
		UserID   string
		JobKey   string
		Amount   int
		Source   string
		Metadata map[string]interface{}
	}{UserID: userID, JobKey: jobKey, Amount: baseAmount, Source: source, Metadata: metadata})
	return &domain.XPAwardResult{LeveledUp: false}, nil
}

// MockProgressionService for testing modifiers
type MockProgressionService struct {
	modifiers   map[string]float64
	returnValue float64 // Generic fallback
	returnError error
	calls       []struct {
		ctx        context.Context
		featureKey string
		baseValue  float64
	}
}

func (m *MockProgressionService) GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error) {
	m.calls = append(m.calls, struct {
		ctx        context.Context
		featureKey string
		baseValue  float64
	}{ctx, featureKey, baseValue})

	if m.returnError != nil {
		return 0, m.returnError
	}

	// Check specific modifier map first
	if m.modifiers != nil {
		if val, ok := m.modifiers[featureKey]; ok {
			return val, nil
		}
	}

	// Fallback to generic return value if set
	if m.returnValue > 0 {
		return m.returnValue, nil
	}

	return baseValue, nil
}

// MockNamingResolver for testing name resolution
type MockNamingResolver struct {
	publicToInternal map[string]string
}

func (m *MockNamingResolver) ResolvePublicName(publicName string) (internalName string, ok bool) {
	internal, ok := m.publicToInternal[publicName]
	return internal, ok
}

// Stubs for other naming.Resolver methods
func (m *MockNamingResolver) GetDisplayName(internalName string, shineLevel domain.ShineLevel) string {
	return internalName
}
func (m *MockNamingResolver) GetActiveTheme() string { return "" }
func (m *MockNamingResolver) Reload() error          { return nil }
func (m *MockNamingResolver) RegisterItem(internalName, publicName string) {
	if m.publicToInternal == nil {
		m.publicToInternal = make(map[string]string)
	}
	m.publicToInternal[publicName] = internalName
}
