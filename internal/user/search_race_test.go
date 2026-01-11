package user

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/stretchr/testify/assert"
)

// ThreadSafeMockRepo mimics the database with thread-safe map access
type ThreadSafeMockRepo struct {
	mu          sync.RWMutex
	users       map[string]*domain.User
	items       map[string]*domain.Item
	inventories map[string]*domain.Inventory
	cooldowns   map[string]map[string]*time.Time

	// Simulating row locks
	rowLocks    sync.Map // map[string]*sync.Mutex
}

type ThreadSafeMockTx struct {
	repo    *ThreadSafeMockRepo
	heldLocks []*sync.Mutex
}

func newThreadSafeMockRepo() *ThreadSafeMockRepo {
	return &ThreadSafeMockRepo{
		users:       make(map[string]*domain.User),
		items:       make(map[string]*domain.Item),
		inventories: make(map[string]*domain.Inventory),
		cooldowns:   make(map[string]map[string]*time.Time),
	}
}

// Implement Repository interface methods safely

func (m *ThreadSafeMockRepo) UpsertUser(ctx context.Context, user *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.Username] = user
	return nil
}

func (m *ThreadSafeMockRepo) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.users {
		switch platform {
		case domain.PlatformTwitch:
			if u.TwitchID == platformID {
				return u, nil
			}
		}
	}
	return nil, nil // Not found
}

func (m *ThreadSafeMockRepo) GetLastCooldown(ctx context.Context, userID, action string) (*time.Time, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if userCooldowns, ok := m.cooldowns[userID]; ok {
		return userCooldowns[action], nil
	}
	return nil, nil
}

func (m *ThreadSafeMockRepo) UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.cooldowns[userID]; !ok {
		m.cooldowns[userID] = make(map[string]*time.Time)
	}
	m.cooldowns[userID][action] = &timestamp
	return nil
}

// Transaction support
func (m *ThreadSafeMockRepo) BeginTx(ctx context.Context) (repository.Tx, error) {
	return &ThreadSafeMockTx{
		repo: m,
		heldLocks: make([]*sync.Mutex, 0),
	}, nil
}

// Tx implementation
func (tx *ThreadSafeMockTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return tx.repo.GetInventory(ctx, userID)
}
func (tx *ThreadSafeMockTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return tx.repo.UpdateInventory(ctx, userID, inventory)
}

func (tx *ThreadSafeMockTx) GetLastCooldownForUpdate(ctx context.Context, userID, action string) (*time.Time, error) {
	// Simulate SELECT FOR UPDATE by acquiring a lock
	key := userID + ":" + action
	lockInterface, _ := tx.repo.rowLocks.LoadOrStore(key, &sync.Mutex{})
	lock := lockInterface.(*sync.Mutex)

	lock.Lock()
	tx.heldLocks = append(tx.heldLocks, lock)

	// Now proceed with normal read
	return tx.repo.GetLastCooldown(ctx, userID, action)
}

func (tx *ThreadSafeMockTx) UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error {
	return tx.repo.UpdateCooldown(ctx, userID, action, timestamp)
}

func (tx *ThreadSafeMockTx) Commit(ctx context.Context) error {
	// Simulate latency
	time.Sleep(50 * time.Millisecond)

	// Release locks
	for _, lock := range tx.heldLocks {
		lock.Unlock()
	}
	tx.heldLocks = nil
	return nil
}

func (tx *ThreadSafeMockTx) Rollback(ctx context.Context) error {
	// Release locks
	for _, lock := range tx.heldLocks {
		lock.Unlock()
	}
	tx.heldLocks = nil
	return nil
}

// Boilerplate stubs
func (m *ThreadSafeMockRepo) GetUserByID(ctx context.Context, userID string) (*domain.User, error) { return nil, nil }
func (m *ThreadSafeMockRepo) UpdateUser(ctx context.Context, user domain.User) error { return nil }
func (m *ThreadSafeMockRepo) DeleteUser(ctx context.Context, userID string) error    { return nil }
func (m *ThreadSafeMockRepo) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if inv, ok := m.inventories[userID]; ok {
		return inv, nil
	}
	return &domain.Inventory{Slots: []domain.InventorySlot{}}, nil
}
func (m *ThreadSafeMockRepo) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inventories[userID] = &inventory
	return nil
}
func (m *ThreadSafeMockRepo) DeleteInventory(ctx context.Context, userID string) error { return nil }
func (m *ThreadSafeMockRepo) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if item, ok := m.items[itemName]; ok {
		return item, nil
	}
	return nil, nil
}
func (m *ThreadSafeMockRepo) GetItemByID(ctx context.Context, id int) (*domain.Item, error)           { return nil, nil }
func (m *ThreadSafeMockRepo) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) { return nil, nil }
func (m *ThreadSafeMockRepo) GetSellablePrices(ctx context.Context) ([]domain.Item, error)            { return nil, nil }
func (m *ThreadSafeMockRepo) IsItemBuyable(ctx context.Context, itemName string) (bool, error)        { return false, nil }
func (m *ThreadSafeMockRepo) GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error) {
	return nil, nil
}
func (m *ThreadSafeMockRepo) IsRecipeUnlocked(ctx context.Context, userID string, recipeID int) (bool, error) {
	return false, nil
}
func (m *ThreadSafeMockRepo) UnlockRecipe(ctx context.Context, userID string, recipeID int) error {
	return nil
}
func (m *ThreadSafeMockRepo) GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]crafting.UnlockedRecipeInfo, error) {
	return nil, nil
}

// Test Race Condition in HandleSearch
func TestHandleSearch_RaceCondition(t *testing.T) {
	repo := newThreadSafeMockRepo()

	// Setup user
	user := &domain.User{
		ID:       "test-race-id",
		Username: "racer",
		TwitchID: "race123",
	}
	repo.users["racer"] = user

	// Setup item (required for search success path)
	repo.items[domain.ItemLootbox0] = &domain.Item{ID: 1, InternalName: domain.ItemLootbox0, BaseValue: 10}

	// Create service
	// Using a simple mock naming resolver since it's used in success path
	svc := NewService(repo, nil, nil, NewMockNamingResolver(), false)

	// Concurrency parameters
	concurrentCalls := 10
	var successfulSearches int32
	var wg sync.WaitGroup

	// Launch concurrent searches
	for i := 0; i < concurrentCalls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Use the same user
			msg, err := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "race123", "racer")
			if err == nil {
				// Check if the message indicates a success (found item, nothing found, etc.)
				// rather than a cooldown warning.
				// Cooldown message starts with "You can search again in"
				if msg != "" && len(msg) > 23 && msg[:23] == "You can search again in" {
					return // Cooldown hit
				}
				// Consider any other message a "successful search execution"
				atomic.AddInt32(&successfulSearches, 1)
				fmt.Println("Search succeeded: " + msg)
			}
		}()
	}

	wg.Wait()

	t.Logf("Successful searches: %d out of %d", successfulSearches, concurrentCalls)

	// Now that we have fixed it, we expect exactly 1 success.
	assert.Equal(t, 1, int(successfulSearches), "Race condition fixed: Only one search should succeed")
}
