package user

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/concurrency"
	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Constants for search testing boundaries
const (
	SearchCooldownMinutes = 30
	TestUserID            = "test-user-123"
	TestUsername          = "testuser"
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

func (m *mockSearchRepo) UpsertUser(ctx context.Context, user *domain.User) error {
	if m.shouldFailGet {
		return domain.ErrUserNotFound
	}
	if user.ID == "" {
		user.ID = "user-" + user.Username
	}
	m.users[user.Username] = user
	return nil
}

func (m *mockSearchRepo) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	item, ok := m.items[itemName]
	if !ok {
		return nil, nil // Item not found
	}
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
	if m.shouldFailGet {
		return nil, domain.ErrUserNotFound
	}
	for _, u := range m.users {
		switch platform {
		case "twitch":
			if u.TwitchID == platformID {
				return u, nil
			}
		case "discord":
			if u.DiscordID == platformID {
				return u, nil
			}
		}
	}
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

// Test fixtures
func createSearchTestService() (*service, *mockSearchRepo) {
	repo := newMockSearchRepo()
	lockManager := concurrency.NewLockManager()
	svc := NewService(repo, lockManager, nil, false).(*service)

	// Add standard test items
	repo.items[domain.ItemLootbox0] = &domain.Item{
		ID:          1,
		Name:        domain.ItemLootbox0,
		Description: "Basic Lootbox",
		BaseValue:   10,
	}

	return svc, repo
}

func createTestUser(username, userID string) *domain.User {
	return &domain.User{
		ID:       userID,
		Username: username,
		TwitchID: username + "123",
	}
}

// =============================================================================
// HandleSearch Tests - Demonstrating 5-Case Testing Model
// =============================================================================

// CASE 1: BEST CASE - Happy path
func TestHandleSearch_Success(t *testing.T) {
	// ARRANGE
	svc, repo := createSearchTestService()
	user := createTestUser(TestUsername, TestUserID)
	repo.users[TestUsername] = user

	// ACT
	message, err := svc.HandleSearch(context.Background(), "twitch", "testuser123", TestUsername)

	// ASSERT
	require.NoError(t, err)
	// Should get either lootbox or nothing found
	assert.True(t,
		message == "You have found 1x "+domain.ItemLootbox0 ||
			message == domain.MsgSearchNothingFound,
		"Expected valid search result, got: %s", message)

	// Verify cooldown was set
	cooldown, err := repo.GetLastCooldown(context.Background(), user.ID, domain.ActionSearch)
	require.NoError(t, err)
	assert.NotNil(t, cooldown, "Cooldown should be set after search")
}

// CASE 2: BOUNDARY CASE - Cooldown timing boundaries
func TestHandleSearch_CooldownBoundaries(t *testing.T) {
	tests := []struct {
		name           string
		minutesAgo     int
		expectCooldown bool
	}{
		// On boundary
		{"exactly 30 minutes ago (on boundary)", SearchCooldownMinutes, false},
		{"exactly 29 minutes ago (just inside)", SearchCooldownMinutes - 1, true},

		// Just outside
		{"31 minutes ago (just expired)", SearchCooldownMinutes + 1, false},

		// Well beyond boundaries
		{"5 minutes ago (well within)", 5, true},
		{"60 minutes ago (well expired)", 60, false},

		// Edge: just happened
		{"0 minutes ago (immediate)", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			svc, repo := createSearchTestService()
			user := createTestUser(TestUsername, TestUserID)
			repo.users[TestUsername] = user

			// Set cooldown
			pastTime := time.Now().Add(-time.Duration(tt.minutesAgo) * time.Minute)
			repo.cooldowns[user.ID] = map[string]*time.Time{
				domain.ActionSearch: &pastTime,
			}

			// ACT
			message, err := svc.HandleSearch(context.Background(), "twitch", "testuser123", TestUsername)

			// ASSERT
			require.NoError(t, err)

			if tt.expectCooldown {
				assert.True(t, strings.HasPrefix(message, "You can search again in"),
					"Expected cooldown message, got: %s", message)
			} else {
				assert.False(t, strings.HasPrefix(message, "You can search again in"),
					"Expected search to execute, got cooldown: %s", message)
			}
		})
	}
}

// CASE 3: EDGE CASE - New user creation
func TestHandleSearch_NewUserCreation(t *testing.T) {
	// ARRANGE
	svc, repo := createSearchTestService()

	// ACT - Search with non-existent user
	message, err := svc.HandleSearch(context.Background(), "twitch", "", "newuser")

	// ASSERT
	require.NoError(t, err)

	// Verify user was created
	user, exists := repo.users["newuser"]
	require.True(t, exists, "New user should be created")
	assert.NotNil(t, user)
	assert.Equal(t, "newuser", user.Username)
	assert.NotEmpty(t, user.ID, "User should have ID assigned")

	// Verify search executed
	assert.True(t,
		message == "You have found 1x "+domain.ItemLootbox0 ||
			message == domain.MsgSearchNothingFound,
		"Search should execute for new user")

	// Verify cooldown set for new user
	cooldown, err := repo.GetLastCooldown(context.Background(), user.ID, domain.ActionSearch)
	require.NoError(t, err)
	assert.NotNil(t, cooldown, "Cooldown should be set for new user")
}

// CASE 4: INVALID CASE - Input validation
func TestHandleSearch_InvalidInputs(t *testing.T) {
	tests := []struct {
		name     string
		username string
		platform string
		setup    func(*mockSearchRepo)
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "empty username",
			username: "",
			platform: "twitch",
			setup:    func(r *mockSearchRepo) {},
			wantErr:  true,
			errMsg:   "username cannot be empty",
		},
		{
			name:     "empty platform defaults to twitch",
			username: TestUsername,
			platform: "",
			setup:    func(r *mockSearchRepo) {},
			wantErr:  false, // Defaults to twitch
		},
		{
			name:     "invalid platform",
			username: TestUsername,
			platform: "invalidplatform",
			setup:    func(r *mockSearchRepo) {},
			wantErr:  true,
			errMsg:   "invalid platform",
		},
		{
			name:     "missing lootbox item",
			username: TestUsername,
			platform: "twitch",
			setup: func(r *mockSearchRepo) {
				delete(r.items, domain.ItemLootbox0)
			},
			wantErr: true,
			errMsg:  "reward item not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			svc, repo := createSearchTestService()
			tt.setup(repo)

			// ACT
			_, err := svc.HandleSearch(context.Background(), tt.platform, "", tt.username)

			// ASSERT
			if tt.wantErr {
				require.Error(t, err, "Expected error for: %s", tt.name)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// CASE 5: HOSTILE CASE - Database failures
func TestHandleSearch_DatabaseErrors(t *testing.T) {
	t.Run("user lookup failure", func(t *testing.T) {
		// ARRANGE
		svc, repo := createSearchTestService()
		repo.shouldFailGet = true

		// ACT
		_, err := svc.HandleSearch(context.Background(), "twitch", "testuser123", TestUsername)

		// ASSERT
		if err != nil {
			assert.Contains(t, err.Error(), "failed to register user")
		} else {
			t.Error("Expected error for database failure, but got nil")
		}
	})
}

// =============================================================================
// Additional Tests - Real-world scenarios
// =============================================================================

func TestHandleSearch_CooldownUpdate(t *testing.T) {
	t.Run("cooldown updates after successful search", func(t *testing.T) {
		// ARRANGE
		svc, repo := createSearchTestService()
		user := createTestUser(TestUsername, TestUserID)
		repo.users[TestUsername] = user

		// Set old cooldown
		oldTime := time.Now().Add(-2 * time.Hour)
		repo.cooldowns[user.ID] = map[string]*time.Time{
			domain.ActionSearch: &oldTime,
		}

		// ACT
		_, err := svc.HandleSearch(context.Background(), "twitch", "testuser123", TestUsername)

		// ASSERT
		require.NoError(t, err)

		// Verify cooldown was updated
		newCooldown, err := repo.GetLastCooldown(context.Background(), user.ID, domain.ActionSearch)
		require.NoError(t, err)
		assert.True(t, newCooldown.After(oldTime),
			"Cooldown should be updated to more recent time")
	})

	t.Run("cooldown not updated when on cooldown", func(t *testing.T) {
		// ARRANGE
		svc, repo := createSearchTestService()
		user := createTestUser(TestUsername, TestUserID)
		repo.users[TestUsername] = user

		// Set recent cooldown
		recentTime := time.Now().Add(-5 * time.Minute)
		repo.cooldowns[user.ID] = map[string]*time.Time{
			domain.ActionSearch: &recentTime,
		}

		// ACT
		message, err := svc.HandleSearch(context.Background(), "twitch", "testuser123", TestUsername)

		// ASSERT
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(message, "You can search again in"))

		// Verify cooldown was NOT updated
		cooldown, err := repo.GetLastCooldown(context.Background(), user.ID, domain.ActionSearch)
		require.NoError(t, err)
		assert.Equal(t, recentTime.Unix(), cooldown.Unix(),
			"Cooldown should not change when user is still on cooldown")
	})
}

func TestHandleSearch_MultipleSearches(t *testing.T) {
	t.Run("user can search multiple times after cooldown expires", func(t *testing.T) {
		// ARRANGE
		svc, repo := createSearchTestService()
		user := createTestUser(TestUsername, TestUserID)
		repo.users[TestUsername] = user

		// ACT - First search
		_, err1 := svc.HandleSearch(context.Background(), "twitch", "testuser123", TestUsername)
		require.NoError(t, err1)

		// Manually expire cooldown
		expiredTime := time.Now().Add(-2 * time.Hour)
		repo.cooldowns[user.ID][domain.ActionSearch] = &expiredTime

		// Second search after expiry
		_, err2 := svc.HandleSearch(context.Background(), "twitch", "testuser123", TestUsername)

		// ASSERT
		require.NoError(t, err2, "Should be able to search again after cooldown expires")
	})
}
