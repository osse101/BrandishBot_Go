package user

import (
	"context"
	"strings"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// MockRepository is a stateful "fake" implementation of Repository for testing.
// It stores state in memory (maps) to enable integration-style unit tests.
//
// IMPORTANT: This mock must remain in the user package to avoid import cycles.
// The generated mock in mocks/mock_repository.go is for cross-package testing only.
// See docs/development/FEATURE_DEVELOPMENT_GUIDE.md for mock usage patterns.
type FakeRepository struct {
	users           map[string]*domain.User // keyed by user ID
	inventories     map[string]*domain.Inventory
	items           map[string]*domain.Item
	recipes         map[int]*domain.Recipe // keyed by recipe ID
	unlockedRecipes map[string]map[int]bool
	cooldowns       map[string]map[string]*time.Time // userID -> action -> timestamp
}

func NewFakeRepository() *FakeRepository {
	return &FakeRepository{
		users:           make(map[string]*domain.User),
		items:           make(map[string]*domain.Item),
		inventories:     make(map[string]*domain.Inventory),
		recipes:         make(map[int]*domain.Recipe),
		unlockedRecipes: make(map[string]map[int]bool),
		cooldowns:       make(map[string]map[string]*time.Time),
	}
}

func (f *FakeRepository) UpsertUser(ctx context.Context, user *domain.User) error {
	if user.ID == "" {
		user.ID = "user-" + user.Username
	}
	f.users[user.Username] = user
	return nil
}

func (f *FakeRepository) UpdateUser(ctx context.Context, user domain.User) error {
	f.users[user.Username] = &user
	return nil
}

func (f *FakeRepository) DeleteUser(ctx context.Context, userID string) error {
	for k, v := range f.users {
		if v.ID == userID {
			delete(f.users, k)
		}
	}
	return nil
}

func (f *FakeRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	for _, u := range f.users {
		if u.ID == userID {
			return u, nil
		}
	}
	return nil, nil
}

func (f *FakeRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	for _, u := range f.users {
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

func (f *FakeRepository) GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error) {
	// Case-insensitive username lookup
	for _, u := range f.users {
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

func (f *FakeRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	if inv, ok := f.inventories[userID]; ok {
		return inv, nil
	}
	// Return empty inventory if not exists
	return &domain.Inventory{Slots: []domain.InventorySlot{}}, nil
}

func (f *FakeRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	f.inventories[userID] = &inventory
	return nil
}

func (f *FakeRepository) DeleteInventory(ctx context.Context, userID string) error {
	delete(f.inventories, userID)
	return nil
}

func (f *FakeRepository) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	if item, ok := f.items[itemName]; ok {
		return item, nil
	}
	return nil, nil
}

func (f *FakeRepository) GetItemsByNames(ctx context.Context, names []string) ([]domain.Item, error) {
	items := make([]domain.Item, 0, len(names))
	for _, name := range names {
		if item, ok := f.items[name]; ok {
			items = append(items, *item)
		}
	}
	return items, nil
}

func (f *FakeRepository) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	items := make([]domain.Item, 0, len(itemIDs))
	for _, id := range itemIDs {
		for _, item := range f.items {
			if item.ID == id {
				items = append(items, *item)
				break
			}
		}
	}
	return items, nil
}

func (f *FakeRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	for _, item := range f.items {
		if item.ID == id {
			return item, nil
		}
	}
	return nil, nil
}

func (f *FakeRepository) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	items := make([]domain.Item, 0, len(f.items))
	for _, item := range f.items {
		items = append(items, *item)
	}
	return items, nil
}

func (f *FakeRepository) IsItemBuyable(ctx context.Context, itemName string) (bool, error) {
	// For testing, assume lootbox0 and lootbox1 are buyable
	if itemName == domain.ItemLootbox0 || itemName == domain.ItemLootbox1 {
		return true, nil
	}
	return false, nil
}

// MockTx wraps MockRepository for transaction testing
type MockTx struct {
	repo *FakeRepository
}

func (f *FakeRepository) BeginTx(ctx context.Context) (repository.UserTx, error) {
	return &MockTx{repo: f}, nil
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

func (f *FakeRepository) GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error) {
	if recipe, ok := f.recipes[itemID]; ok {
		return recipe, nil
	}
	return nil, nil
}

func (f *FakeRepository) IsRecipeUnlocked(ctx context.Context, userID string, recipeID int) (bool, error) {
	if f.unlockedRecipes[userID] == nil {
		return false, nil
	}
	return f.unlockedRecipes[userID][recipeID], nil
}

func (f *FakeRepository) UnlockRecipe(ctx context.Context, userID string, recipeID int) error {
	if f.unlockedRecipes[userID] == nil {
		f.unlockedRecipes[userID] = make(map[int]bool)
	}
	f.unlockedRecipes[userID][recipeID] = true
	return nil
}

func (f *FakeRepository) GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]repository.UnlockedRecipeInfo, error) {
	userUnlocks, ok := f.unlockedRecipes[userID]
	if !ok {
		return []repository.UnlockedRecipeInfo{}, nil
	}

	recipes := make([]repository.UnlockedRecipeInfo, 0, len(userUnlocks))
	for recipeID := range userUnlocks {
		if recipe, exists := f.recipes[recipeID]; exists {
			for _, item := range f.items {
				if item.ID == recipe.TargetItemID {
					recipes = append(recipes, repository.UnlockedRecipeInfo{
						ItemName: item.InternalName,
						ItemID:   item.ID,
					})
					break
				}
			}
		}
	}

	return recipes, nil
}

func (f *FakeRepository) GetLastCooldown(ctx context.Context, userID, action string) (*time.Time, error) {
	if userCooldowns, ok := f.cooldowns[userID]; ok {
		return userCooldowns[action], nil
	}
	return nil, nil
}

func (f *FakeRepository) UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error {
	if _, ok := f.cooldowns[userID]; !ok {
		f.cooldowns[userID] = make(map[string]*time.Time)
	}
	f.cooldowns[userID][action] = &timestamp
	return nil
}

func (f *FakeRepository) MergeUsersInTransaction(ctx context.Context, primaryUserID, secondaryUserID string, mergedUser domain.User, mergedInventory domain.Inventory) error {
	return nil // No-op for mock
}
func (f *FakeRepository) GetAllItems(ctx context.Context) ([]domain.Item, error) {
	items := make([]domain.Item, 0, len(f.items))
	for _, item := range f.items {
		items = append(items, *item)
	}
	return items, nil
}
