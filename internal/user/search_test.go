package user

import (
	"context"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/concurrency"
	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Mock repository for search tests
type mockSearchRepo struct {
	users         map[string]*domain.User
	items         map[string]*domain.Item
	inventories   map[string]*domain.Inventory
	cooldowns     map[string]map[string]*time.Time
	shouldFailGet bool
}

func newMockSearchRepo() *mockSearchRepo {
	return &mockSearchRepo{
		users:       make(map[string]*domain.User),
		items:       make(map[string]*domain.Item),
		inventories: make(map[string]*domain.Inventory),
		cooldowns:   make(map[string]map[string]*time.Time),
	}
}

func (m *mockSearchRepo) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	if m.shouldFailGet {
		return nil, domain.ErrUserNotFound
	}
	return m.users[username], nil
}

func (m *mockSearchRepo) UpsertUser(ctx context.Context, user *domain.User) error {
	if user.ID == "" {
		user.ID = "user-" + user.Username
	}
	m.users[user.Username] = user
	return nil
}

func (m *mockSearchRepo) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	item := m.items[itemName]
	return item, nil
}

func (m *mockSearchRepo) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	inv, ok := m.inventories[userID]
	if !ok {
		return &domain.Inventory{Slots: []domain.InventorySlot{}}, nil
	}
	return inv, nil
}

func (m *mockSearchRepo) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	m.inventories[userID] = &inventory
	return nil
}

func (m *mockSearchRepo) GetLastCooldown(ctx context.Context, userID, action string) (*time.Time, error) {
	if userCooldowns, ok := m.cooldowns[userID]; ok {
		return userCooldowns[action], nil
	}
	return nil, nil
}

func (m *mockSearchRepo) UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error {
	if _, ok := m.cooldowns[userID]; !ok {
		m.cooldowns[userID] = make(map[string]*time.Time)
	}
	m.cooldowns[userID][action] = &timestamp
	return nil
}

// Implement remaining interface methods as no-ops
func (m *mockSearchRepo) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return nil, nil
}
func (m *mockSearchRepo) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	return nil, nil
}
func (m *mockSearchRepo) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	return nil, nil
}
func (m *mockSearchRepo) IsItemBuyable(ctx context.Context, itemName string) (bool, error) {
	return false, nil
}
func (m *mockSearchRepo) BeginTx(ctx context.Context) (repository.Tx, error) { return nil, nil }
func (m *mockSearchRepo) GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error) {
	return nil, nil
}
func (m *mockSearchRepo) IsRecipeUnlocked(ctx context.Context, userID string, recipeID int) (bool, error) {
	return false, nil
}
func (m *mockSearchRepo) UnlockRecipe(ctx context.Context, userID string, recipeID int) error {
	return nil
}
func (m *mockSearchRepo) GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]crafting.UnlockedRecipeInfo, error) {
	return nil, nil
}

func TestHandleSearch_NewUser(t *testing.T) {
	repo := newMockSearchRepo()
	lockManager := concurrency.NewLockManager()
	svc := NewService(repo, lockManager).(*service)

	// Add lootbox0 item
	repo.items[domain.ItemLootbox0] = &domain.Item{
		ID:          1,
		Name:        domain.ItemLootbox0,
		Description: "Basic Lootbox",
		BaseValue:   10,
	}

	ctx := context.Background()
	message, err := svc.HandleSearch(ctx, "newuser", "twitch")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify user was created
	user := repo.users["newuser"]
	if user == nil {
		t.Fatal("Expected user to be created")
	}

	// Verify result message
	if message != "You have found 1x "+domain.ItemLootbox0 && message != domain.MsgSearchNothingFound {
		t.Errorf("Unexpected message: %s", message)
	}

	// Verify cooldown was set
	cooldown := repo.cooldowns[user.ID][domain.ActionSearch]
	if cooldown == nil {
		t.Error("Expected cooldown to be set")
	}
}

func TestHandleSearch_ExistingUser(t *testing.T) {
	repo := newMockSearchRepo()
	lockManager := concurrency.NewLockManager()
	svc := NewService(repo, lockManager).(*service)

	// Add existing user
	repo.users["existinguser"] = &domain.User{
		ID:       "user-existing",
		Username: "existinguser",
		TwitchID: "existing123",
	}

	// Add lootbox0 item
	repo.items[domain.ItemLootbox0] = &domain.Item{
		ID:          1,
		Name:        domain.ItemLootbox0,
		Description: "Basic Lootbox",
		BaseValue:   10,
	}

	ctx := context.Background()
	message, err := svc.HandleSearch(ctx, "existinguser", "twitch")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify result message
	if message != "You have found 1x "+domain.ItemLootbox0 && message != domain.MsgSearchNothingFound {
		t.Errorf("Unexpected message: %s", message)
	}
}

func TestHandleSearch_Cooldown(t *testing.T) {
	repo := newMockSearchRepo()
	lockManager := concurrency.NewLockManager()
	svc := NewService(repo, lockManager).(*service)

	// Add existing user
	userID := "user-cooldowntest"
	repo.users["cooldownuser"] = &domain.User{
		ID:       userID,
		Username: "cooldownuser",
		TwitchID: "cooldown123",
	}

	// Add lootbox0 item
	repo.items[domain.ItemLootbox0] = &domain.Item{
		ID:          1,
		Name:        domain.ItemLootbox0,
		Description: "Basic Lootbox",
		BaseValue:   10,
	}

	// Set recent cooldown (5 minutes ago)
	recentTime := time.Now().Add(-5 * time.Minute)
	repo.cooldowns[userID] = map[string]*time.Time{
		domain.ActionSearch: &recentTime,
	}

	ctx := context.Background()
	message, err := svc.HandleSearch(ctx, "cooldownuser", "twitch")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should be on cooldown
	if message[:23] != "You can search again in" {
		t.Errorf("Expected cooldown message, got: %s", message)
	}
}

func TestHandleSearch_CooldownExpired(t *testing.T) {
	repo := newMockSearchRepo()
	lockManager := concurrency.NewLockManager()
	svc := NewService(repo, lockManager).(*service)

	// Add existing user
	userID := "user-expiredtest"
	repo.users["expireduser"] = &domain.User{
		ID:       userID,
		Username: "expireduser",
		TwitchID: "expired123",
	}

	// Add lootbox0 item
	repo.items[domain.ItemLootbox0] = &domain.Item{
		ID:          1,
		Name:        domain.ItemLootbox0,
		Description: "Basic Lootbox",
		BaseValue:   10,
	}

	// Set old cooldown (31 minutes ago - expired)
	oldTime := time.Now().Add(-31 * time.Minute)
	repo.cooldowns[userID] = map[string]*time.Time{
		domain.ActionSearch: &oldTime,
	}

	ctx := context.Background()
	message, err := svc.HandleSearch(ctx, "expireduser", "twitch")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should be able to search again
	if message != "You have found 1x "+domain.ItemLootbox0 && message != domain.MsgSearchNothingFound {
		t.Errorf("Expected search result, got: %s", message)
	}

	// Verify cooldown was updated
	newCooldown := repo.cooldowns[userID][domain.ActionSearch]
	if newCooldown.Before(oldTime) {
		t.Error("Expected cooldown to be updated")
	}
}
