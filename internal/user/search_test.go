package user

import (
	"context"
	"strings"
	"testing"
	"time"

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

func (m *mockSearchRepo) UpdateUser(ctx context.Context, user domain.User) error {
	m.users[user.Username] = &user
	return nil
}

func (m *mockSearchRepo) DeleteUser(ctx context.Context, userID string) error {
	for k, v := range m.users {
		if v.ID == userID {
			delete(m.users, k)
		}
	}
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

func (m *mockSearchRepo) DeleteInventory(ctx context.Context, userID string) error {
	delete(m.inventories, userID)
	return nil
}

func (m *mockSearchRepo) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	return nil, nil
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
func (m *mockSearchRepo) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	if user, ok := m.users[userID]; ok {
		return user, nil
	}
	// Also search by value ID if key is username
	for _, u := range m.users {
		if u.ID == userID {
			return u, nil
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
func (m *mockSearchRepo) Commit(ctx context.Context) error                   { return nil }
func (m *mockSearchRepo) Rollback(ctx context.Context) error                 { return nil }
func (m *mockSearchRepo) BeginTx(ctx context.Context) (repository.Tx, error) { return m, nil }
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
	svc := NewService(repo, nil, nil, NewMockNamingResolver(), false).(*service)

	// Add standard test items
	repo.items[domain.ItemLootbox0] = &domain.Item{
		ID:           1,
		InternalName: domain.ItemLootbox0,

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
	// Should get either lootbox or nothing found (or funny failure)
	isValid := false
	if strings.HasPrefix(message, "You have found") {
		isValid = true
	} else if strings.HasPrefix(message, domain.MsgSearchCriticalSuccess) {
		isValid = true
	} else {
		for _, msg := range domain.SearchFailureMessages {
			if message == msg {
				isValid = true
				break
			}
		}
	}
	assert.True(t, isValid, "Expected valid search result, got: %s", message)

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
	isValid := false
	if strings.HasPrefix(message, "You have found") {
		isValid = true
	} else if strings.HasPrefix(message, domain.MsgSearchCriticalSuccess) {
		isValid = true
	} else {
		for _, msg := range domain.SearchFailureMessages {
			if message == msg {
				isValid = true
				break
			}
		}
	}
	assert.True(t, isValid, "Search should execute for new user, got: %s", message)

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

// CASE 6: NAMING RESOLUTION
func TestHandleSearch_NamingResolution(t *testing.T) {
	// ARRANGE
	svc, repo := createSearchTestService()
	user := createTestUser(TestUsername, TestUserID)
	repo.users[TestUsername] = user

	// Configure mock resolver
	mockResolver := svc.namingResolver.(*MockNamingResolver)
	mockResolver.DisplayNames[domain.ItemLootbox0] = "Mysterious Chest"

	// Mock RNG is not available, so we loop until success
	found := false
	maxAttempts := 100 // Should be plenty given 80% success rate

	for i := 0; i < maxAttempts; i++ {
		// Reset cooldown manually
		if repo.cooldowns[user.ID] == nil {
			repo.cooldowns[user.ID] = make(map[string]*time.Time)
		}
		// Clear cooldown
		delete(repo.cooldowns[user.ID], domain.ActionSearch)

		// Call with devMode false (default in createSearchTestService)
		msg, err := svc.HandleSearch(context.Background(), "twitch", "testuser123", TestUsername)
		require.NoError(t, err)

		if strings.Contains(msg, "Mysterious Chest") {
			found = true
			break
		}
	}

	assert.True(t, found, "Should use display name 'Mysterious Chest' in search result at least once")
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

// MockStatsService for testing
type mockStatsService struct {
	recordedEvents []domain.StatsEvent
}

func (m *mockStatsService) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata map[string]interface{}) error {
	m.recordedEvents = append(m.recordedEvents, domain.StatsEvent{
		UserID:    userID,
		EventType: eventType,
		EventData: metadata,
	})
	return nil
}

func (m *mockStatsService) GetUserStats(ctx context.Context, userID string, period string) (*domain.StatsSummary, error) {
	return nil, nil
}
func (m *mockStatsService) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) {
	return nil, nil
}
func (m *mockStatsService) GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error) {
	return nil, nil
}

func TestHandleSearch_NearMiss_Statistical(t *testing.T) {
	// ARRANGE
	repo := newMockSearchRepo()
	// Add required lootbox item
	repo.items[domain.ItemLootbox0] = &domain.Item{
		ID:           1,
		InternalName: domain.ItemLootbox0,

		BaseValue: 10,
	}

	statsSvc := &mockStatsService{}
	// Enable devMode to bypass cooldowns for loop
	svc := NewService(repo, statsSvc, nil, NewMockNamingResolver(), true).(*service)

	// Create user
	user := createTestUser(TestUsername, TestUserID)
	repo.users[TestUsername] = user

	nearMissCount := 0
	iterations := 1000

	for i := 0; i < iterations; i++ {
		msg, err := svc.HandleSearch(context.Background(), "twitch", "testuser123", TestUsername)
		require.NoError(t, err)

		if msg == domain.MsgSearchNearMiss {
			nearMissCount++
		}
	}

	t.Logf("Near misses in %d iterations: %d", iterations, nearMissCount)

	// We expect roughly 5% = 50. Let's assert > 0 to ensure the path is reachable.
	// Probability of 0 near misses in 1000 trials with p=0.05 is 0.95^1000 ~= 5e-23 (impossible)
	assert.Greater(t, nearMissCount, 0, "Should have encountered at least one near miss")

	// Verify events were recorded
	assert.Equal(t, nearMissCount, len(statsSvc.recordedEvents), "Should record event for each near miss")
	if len(statsSvc.recordedEvents) > 0 {
		assert.Equal(t, domain.EventSearchNearMiss, statsSvc.recordedEvents[0].EventType)
	}
}

func TestHandleSearch_FirstDaily(t *testing.T) {
	// ARRANGE
	svc, repo := createSearchTestService()
	user := createTestUser(TestUsername, TestUserID)
	repo.users[TestUsername] = user
	ctx := context.Background()

	// 1. First search ever (lastUsed is nil)
	msg, err := svc.HandleSearch(ctx, "twitch", "testuser123", TestUsername)
	require.NoError(t, err)

	assert.Contains(t, msg, domain.MsgFirstSearchBonus, "Expected bonus message for first ever search")
	assert.Contains(t, msg, domain.MsgSearchCriticalSuccess, "Expected critical success for first search")

	// 2. Second search same day (lastUsed is now)
	// Modify cooldown to be 31 mins ago (expired) but SAME DAY.
	now := time.Now()
	past31m := now.Add(-31 * time.Minute)

	if past31m.Day() == now.Day() {
		repo.cooldowns[TestUserID] = map[string]*time.Time{
			domain.ActionSearch: &past31m,
		}

		msg, err = svc.HandleSearch(ctx, "twitch", "testuser123", TestUsername)
		require.NoError(t, err)

		assert.NotContains(t, msg, domain.MsgFirstSearchBonus, "Did not expect bonus message for second search same day")
	}

	// 3. Search next day
	yesterday := now.Add(-25 * time.Hour)
	repo.cooldowns[TestUserID] = map[string]*time.Time{
		domain.ActionSearch: &yesterday,
	}

	msg, err = svc.HandleSearch(ctx, "twitch", "testuser123", TestUsername)
	require.NoError(t, err)

	assert.Contains(t, msg, domain.MsgFirstSearchBonus, "Expected bonus message for next day search")
}
