package user

import (
	"context"
	"strings"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
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

func (m *MockRepository) GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error) {
	// Case-insensitive username lookup
	for _, u := range m.users {
		// Check if user has the platform
		var hasPlatform bool
		switch platform {
		case domain.PlatformTwitch:
			hasPlatform = u.TwitchID != ""
		case domain.PlatformYoutube:
			hasPlatform = u.YoutubeID != ""
		case domain.PlatformDiscord:
			hasPlatform = u.DiscordID != ""
		}
		
		// Case-insensitive username match
		if hasPlatform && strings.EqualFold(u.Username, username) {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
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
