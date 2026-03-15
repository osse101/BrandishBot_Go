package search

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/cooldown"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// Constants for search testing boundaries
const (
	SearchCooldownMinutes = 30
	TestUserID            = "test-user-123"
	TestUsername          = "testuser"
)

// MockBus for testing event publication
type MockBus struct {
	PublishedEvents []event.Event
}

func (m *MockBus) Publish(ctx context.Context, e event.Event) error {
	m.PublishedEvents = append(m.PublishedEvents, e)
	return nil
}

func (m *MockBus) Subscribe(t event.Type, h event.Handler) {}

// mockJobService for testing job bonuses
type mockJobService struct {
	jobLevels map[string]int
}

func newMockJobService() *mockJobService {
	return &mockJobService{
		jobLevels: make(map[string]int),
	}
}

// GetJobLevel implements job.Service
func (m *mockJobService) GetJobLevel(ctx context.Context, userID, jobKey string) (int, error) {
	if level, ok := m.jobLevels[jobKey]; ok {
		return level, nil
	}
	return 0, nil
}

// Stub other methods
func (m *mockJobService) GetUserJobs(ctx context.Context, userID string) ([]domain.UserJobInfo, error) {
	return nil, nil
}
func (m *mockJobService) GetUserJobsByPlatform(ctx context.Context, platform, platformID string) ([]domain.UserJobInfo, error) {
	return nil, nil
}
func (m *mockJobService) GetPrimaryJob(ctx context.Context, platform, platformID string) (*domain.UserJobInfo, error) {
	return nil, nil
}
func (m *mockJobService) IsJobFeatureUnlocked(ctx context.Context, userID, featureKey string) (bool, error) {
	return false, nil
}
func (m *mockJobService) AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata domain.JobXPMetadata) (*domain.XPAwardResult, error) {
	return nil, nil
}
func (m *mockJobService) AwardXPByPlatform(ctx context.Context, platform, platformID, jobKey string, baseAmount int, source string, metadata domain.JobXPMetadata) (*domain.XPAwardResult, error) {
	return nil, nil
}
func (m *mockJobService) ResetDailyJobXP(ctx context.Context) (int64, error) { return 0, nil }
func (m *mockJobService) GetDailyResetStatus(ctx context.Context) (*domain.DailyResetStatus, error) {
	return nil, nil
}
func (m *mockJobService) GetAllJobs(ctx context.Context) ([]domain.Job, error) { return nil, nil }
func (m *mockJobService) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return nil, nil
}
func (m *mockJobService) CalculateLevel(totalXP int64) int { return 0 }
func (m *mockJobService) GetXPForLevel(level int) int64    { return 0 }
func (m *mockJobService) GetXPProgress(currentXP int64) (currentLevel int, xpToNext int64) {
	return 0, 0
}
func (m *mockJobService) Shutdown(ctx context.Context) error { return nil }

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

func (m *mockSearchRepo) BuildPublicNameIndex() map[string]string {
	index := make(map[string]string)
	for k, v := range m.items {
		if v.PublicName != "" {
			index[strings.ToLower(v.PublicName)] = k
		}
	}
	return index
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

func (m *mockSearchRepo) GetItemsByNames(ctx context.Context, names []string) ([]domain.Item, error) {
	var items []domain.Item
	for _, name := range names {
		if item, ok := m.items[name]; ok {
			items = append(items, *item)
		}
	}
	return items, nil
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
		case domain.PlatformTwitch:
			if u.TwitchID == platformID {
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

func (m *mockSearchRepo) GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error) {
	if m.shouldFailGet {
		return nil, domain.ErrUserNotFound
	}
	// Case-insensitive username lookup
	for _, u := range m.users {
		// Check if user has the platform
		var hasPlatform bool
		switch platform {
		case domain.PlatformTwitch:
			hasPlatform = u.TwitchID != ""
		case domain.PlatformDiscord:
			hasPlatform = u.DiscordID != ""
		}
		// Case-insensitive match
		if hasPlatform && strings.EqualFold(u.Username, username) {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockSearchRepo) GetUserOrRegister(ctx context.Context, platform, platformID, username string) (*domain.User, error) {
	if m.shouldFailGet {
		return nil, domain.ErrFailedToRegisterUser
	}
	user, _ := m.GetUserByPlatformUsername(ctx, platform, username)
	if user != nil {
		return user, nil
	}
	user = &domain.User{
		ID:       "user-" + username,
		Username: username,
	}
	switch platform {
	case domain.PlatformTwitch:
		user.TwitchID = platformID
	case domain.PlatformDiscord:
		user.DiscordID = platformID
	}
	m.users[username] = user
	return user, nil
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
func (m *mockSearchRepo) Commit(ctx context.Context) error                       { return nil }
func (m *mockSearchRepo) Rollback(ctx context.Context) error                     { return nil }
func (m *mockSearchRepo) BeginTx(ctx context.Context) (repository.UserTx, error) { return m, nil }
func (m *mockSearchRepo) GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error) {
	return nil, nil
}
func (m *mockSearchRepo) IsRecipeUnlocked(ctx context.Context, userID string, recipeID int) (bool, error) {
	return false, nil
}
func (m *mockSearchRepo) UnlockRecipe(ctx context.Context, userID string, recipeID int) error {
	return nil
}
func (m *mockSearchRepo) GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]repository.UnlockedRecipeInfo, error) {
	return nil, nil
}

func (m *mockSearchRepo) MergeUsersInTransaction(ctx context.Context, primaryUserID, secondaryUserID string, mergedUser domain.User, mergedInventory domain.Inventory) error {
	return nil // No-op
}

func (m *mockSearchRepo) GetAllItems(ctx context.Context) ([]domain.Item, error) {
	items := make([]domain.Item, 0, len(m.items))
	for _, item := range m.items {
		items = append(items, *item)
	}
	return items, nil
}

func (m *mockSearchRepo) GetRecentlyActiveUsers(ctx context.Context, limit int) ([]domain.User, error) {
	return nil, nil
}

func (m *mockSearchRepo) CreateTrap(ctx context.Context, trap *domain.Trap) error {
	return nil
}

func (m *mockSearchRepo) GetActiveTrap(ctx context.Context, targetID uuid.UUID) (*domain.Trap, error) {
	return nil, nil
}

func (m *mockSearchRepo) GetActiveTrapForUpdate(ctx context.Context, targetID uuid.UUID) (*domain.Trap, error) {
	return nil, nil
}

func (m *mockSearchRepo) TriggerTrap(ctx context.Context, trapID uuid.UUID) error {
	return nil
}

func (m *mockSearchRepo) GetTrapsByUser(ctx context.Context, setterID uuid.UUID, limit int) ([]*domain.Trap, error) {
	return nil, nil
}

func (m *mockSearchRepo) GetTriggeredTrapsForTarget(ctx context.Context, targetID uuid.UUID, limit int) ([]*domain.Trap, error) {
	return nil, nil
}

func (m *mockSearchRepo) CleanupStaleTraps(ctx context.Context, daysOld int) (int, error) {
	return 0, nil
}

func (m *mockSearchRepo) GrantSearchReward(ctx context.Context, user *domain.User, quantity int, qualityLevel domain.QualityLevel) error {
	inv, _ := m.GetInventory(ctx, user.ID)
	inv.Slots = append(inv.Slots, domain.InventorySlot{ItemID: 1, Quantity: quantity, QualityLevel: qualityLevel})
	m.inventories[user.ID] = inv
	return nil
}

func (m *mockSearchRepo) GrantItemReward(ctx context.Context, user *domain.User, item *domain.Item, quantity int, qualityLevel domain.QualityLevel) error {
	inv, _ := m.GetInventory(ctx, user.ID)
	inv.Slots = append(inv.Slots, domain.InventorySlot{ItemID: item.ID, Quantity: quantity, QualityLevel: qualityLevel})
	m.inventories[user.ID] = inv
	return nil
}

// Mock cooldown service
type mockCooldownService struct {
	repo *mockSearchRepo
}

func (m *mockCooldownService) CheckCooldown(ctx context.Context, userID, action string) (bool, time.Duration, error) {
	last, _ := m.repo.GetLastCooldown(ctx, userID, action)
	if last == nil {
		return false, 0, nil
	}
	elapsed := time.Since(*last)
	if elapsed < 30*time.Minute {
		return true, 30*time.Minute - elapsed, nil
	}
	return false, 0, nil
}

func (m *mockCooldownService) EnforceCooldown(ctx context.Context, userID, action string, fn func() error) error {
	onCooldown, remaining, _ := m.CheckCooldown(ctx, userID, action)
	if onCooldown {
		return cooldown.ErrOnCooldown{Action: action, Remaining: remaining}
	}
	err := fn()
	if err == nil {
		now := time.Now()
		m.repo.UpdateCooldown(ctx, userID, action, now)
	}
	return err
}

func (m *mockCooldownService) ResetCooldown(ctx context.Context, userID, action string) error {
	if _, ok := m.repo.cooldowns[userID]; ok {
		delete(m.repo.cooldowns[userID], action)
	}
	return nil
}

func (m *mockCooldownService) GetLastUsed(ctx context.Context, userID, action string) (*time.Time, error) {
	return m.repo.GetLastCooldown(ctx, userID, action)
}

type searchTestServiceOpts struct {
	jobService job.Service
	publisher  *event.ResilientPublisher
}

// Test fixtures
func createSearchTestService(opts ...func(*searchTestServiceOpts)) (*service, *mockSearchRepo) {
	repo := newMockSearchRepo()
	statsSvc := &mockStatsService{mockCounts: make(map[domain.EventType]int)}

	config := &searchTestServiceOpts{
		jobService: newMockJobService(),
	}
	for _, opt := range opts {
		opt(config)
	}

	svc := New(Deps{
		UserResolver:  repo,
		ItemLookup:    repo,
		RewardGranter: repo,
		CooldownSvc:   &mockCooldownService{repo: repo},
		StatsSvc:      statsSvc,
		JobSvc:        config.jobService,
		Publisher:     config.publisher,
		Rnd:           utils.RandomFloat,
	})

	// Add standard test items
	repo.items[domain.ItemLootbox0] = &domain.Item{
		ID:           1,
		InternalName: domain.ItemLootbox0,

		Description: "Basic Lootbox",
		BaseValue:   10,
	}

	return svc.(*service), repo
}

func createTestUser() *domain.User {
	return &domain.User{
		ID:       TestUserID,
		Username: TestUsername,
		TwitchID: TestUsername + "123",
	}
}

// =============================================================================
// HandleSearch Tests - Demonstrating 5-Case Testing Model
// =============================================================================

// CASE 1: BEST CASE - Happy path
func TestHandleSearch_Success(t *testing.T) {
	t.Parallel()
	// ARRANGE
	svc, repo := createSearchTestService()
	user := createTestUser()
	repo.users[TestUsername] = user

	// ACT
	svc.deps.Rnd = func() float64 { return 0.5 } // Force success
	message, err := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")

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
	t.Parallel()
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// ARRANGE
			svc, repo := createSearchTestService()
			user := createTestUser()
			repo.users[TestUsername] = user

			// Set cooldown
			pastTime := time.Now().Add(-time.Duration(tt.minutesAgo) * time.Minute)
			repo.cooldowns[user.ID] = map[string]*time.Time{
				domain.ActionSearch: &pastTime,
			}

			// ACT
			svc.deps.Rnd = func() float64 { return 0.5 } // Force success if search executes
			message, err := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")

			// ASSERT
			if tt.expectCooldown {
				require.Error(t, err)
				var cooldownErr cooldown.ErrOnCooldown
				assert.True(t, errors.As(err, &cooldownErr))
				assert.Equal(t, domain.ActionSearch, cooldownErr.Action)
			} else {
				require.NoError(t, err)
				assert.False(t, strings.HasPrefix(message, "You can search again in"),
					"Expected search to execute, got cooldown: %s", message)
			}
		})
	}
}

// CASE 3: EDGE CASE - New user creation
func TestHandleSearch_NewUserCreation(t *testing.T) {
	t.Parallel()
	// ARRANGE
	svc, repo := createSearchTestService()

	// ACT - Search with non-existent user
	svc.deps.Rnd = func() float64 { return 0.5 } // Force success
	message, err := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "", "newuser", "")

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
	t.Parallel()
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
			platform: domain.PlatformTwitch,
			setup:    func(r *mockSearchRepo) {},
			wantErr:  true,
			errMsg:   domain.ErrInvalidInput.Error(),
		},
		{
			name:     "empty platform",
			username: TestUsername,
			platform: "",
			setup:    func(r *mockSearchRepo) {},
			wantErr:  true,
			errMsg:   domain.ErrInvalidInput.Error(),
		},
		{
			name:     "invalid platform",
			username: TestUsername,
			platform: "invalidplatform",
			setup:    func(r *mockSearchRepo) {},
			wantErr:  true,
			errMsg:   domain.ErrInvalidInput.Error(),
		},
		{
			name:     "missing lootbox item",
			username: TestUsername,
			platform: domain.PlatformTwitch,
			setup: func(r *mockSearchRepo) {
				delete(r.items, domain.ItemLootbox0)
			},
			wantErr: true,
			errMsg:  domain.ErrItemNotFound.Error(),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// ARRANGE
			svc, repo := createSearchTestService()
			tt.setup(repo)

			// Fix: Set rnd to ensure we reach the item check if needed
			// Most invalid input tests fail before calling rnd, but "missing lootbox item" needs success roll
			svc.deps.Rnd = func() float64 { return 0.5 }

			// ACT
			_, err := svc.HandleSearch(context.Background(), tt.platform, "", tt.username, "")

			// ASSERT
			if tt.wantErr {
				require.Error(t, err, "Expected error for: %s", tt.name)
				if tt.name == "missing lootbox item" {
					assert.ErrorIs(t, err, domain.ErrItemNotFound)
				} else {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// CASE 5: HOSTILE CASE - Database failures
func TestHandleSearch_DatabaseErrors(t *testing.T) {
	t.Parallel()
	t.Run("user lookup failure", func(t *testing.T) {
		t.Parallel()
		// ARRANGE
		svc, repo := createSearchTestService()
		repo.shouldFailGet = true

		// ACT
		_, err := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")

		// ASSERT
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrFailedToRegisterUser)
	})
}

// CASE 6: NAMING RESOLUTION (UPDATED: Now uses Public Name directly)
func TestHandleSearch_PublicNameUsage(t *testing.T) {
	t.Parallel()
	// ARRANGE
	svc, repo := createSearchTestService()
	user := createTestUser()
	repo.users[TestUsername] = user

	// Set Public Name on item
	repo.items[domain.ItemLootbox0].PublicName = "junkbox"

	// Configure mock resolver with something different to ensure we are NOT using it
	// (Skipped since search package no longer takes namingResolver as dependency)

	// Force success
	svc.deps.Rnd = func() float64 { return 0.5 }

	// Call with devMode false
	msg, err := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")
	require.NoError(t, err)

	// ASSERT
	assert.Contains(t, msg, "Junkbox", "Should use Title-cased Public Name 'Junkbox' in search result")
	assert.NotContains(t, msg, "Mysterious Chest", "Should NOT use naming resolver display name for search result")
}

// =============================================================================
// Additional Tests - Real-world scenarios
// =============================================================================

func TestHandleSearch_CooldownUpdate(t *testing.T) {
	t.Parallel()
	t.Run("cooldown updates after successful search", func(t *testing.T) {
		t.Parallel()
		// ARRANGE
		svc, repo := createSearchTestService()
		user := createTestUser()
		repo.users[TestUsername] = user

		// Set old cooldown
		oldTime := time.Now().Add(-2 * time.Hour)
		repo.cooldowns[user.ID] = map[string]*time.Time{
			domain.ActionSearch: &oldTime,
		}

		// ACT
		svc.deps.Rnd = func() float64 { return 0.5 } // Force success
		_, err := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")

		// ASSERT
		require.NoError(t, err)

		// Verify cooldown was updated
		newCooldown, err := repo.GetLastCooldown(context.Background(), user.ID, domain.ActionSearch)
		require.NoError(t, err)
		assert.True(t, newCooldown.After(oldTime),
			"Cooldown should be updated to more recent time")
	})

	t.Run("cooldown not updated when on cooldown", func(t *testing.T) {
		t.Parallel()
		// ARRANGE
		svc, repo := createSearchTestService()
		user := createTestUser()
		repo.users[TestUsername] = user

		// Set recent cooldown
		recentTime := time.Now().Add(-5 * time.Minute)
		repo.cooldowns[user.ID] = map[string]*time.Time{
			domain.ActionSearch: &recentTime,
		}

		// ACT
		_, err := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")

		// ASSERT
		require.Error(t, err)
		var cooldownErr cooldown.ErrOnCooldown
		assert.True(t, errors.As(err, &cooldownErr))

		// Verify cooldown was NOT updated
		cooldown, err := repo.GetLastCooldown(context.Background(), user.ID, domain.ActionSearch)
		require.NoError(t, err)
		assert.Equal(t, recentTime.Unix(), cooldown.Unix(),
			"Cooldown should not change when user is still on cooldown")
	})
}

func TestHandleSearch_MultipleSearches(t *testing.T) {
	t.Parallel()
	t.Run("user can search multiple times after cooldown expires", func(t *testing.T) {
		t.Parallel()
		// ARRANGE
		svc, repo := createSearchTestService()
		user := createTestUser()
		repo.users[TestUsername] = user

		// ACT - First search
		svc.deps.Rnd = func() float64 { return 0.5 } // Force success
		_, err1 := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")
		require.NoError(t, err1)

		// Manually expire cooldown
		expiredTime := time.Now().Add(-2 * time.Hour)
		repo.cooldowns[user.ID][domain.ActionSearch] = &expiredTime

		// Second search after expiry
		_, err2 := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")

		// ASSERT
		require.NoError(t, err2, "Should be able to search again after cooldown expires")
	})
}

// MockStatsService for testing
type mockStatsService struct {
	recordedEvents []domain.StatsEvent
	mockCounts     map[domain.EventType]int
	mockStreak     int
}

func (m *mockStatsService) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, metadata interface{}) error {
	m.recordedEvents = append(m.recordedEvents, domain.StatsEvent{
		UserID:    userID,
		EventType: eventType,
		EventData: metadata,
	})
	return nil
}

func (m *mockStatsService) GetUserStats(ctx context.Context, userID string, period string) (*domain.StatsSummary, error) {
	summary := &domain.StatsSummary{
		Period:      period,
		EventCounts: make(map[domain.EventType]int),
	}
	if m.mockCounts != nil {
		summary.EventCounts = m.mockCounts
	}
	return summary, nil
}
func (m *mockStatsService) GetUserCurrentStreak(ctx context.Context, userID string) (int, error) {
	if m.mockStreak > 0 {
		return m.mockStreak, nil
	}
	return 1, nil
}
func (m *mockStatsService) GetSystemStats(ctx context.Context, period string) (*domain.StatsSummary, error) {
	return nil, nil
}
func (m *mockStatsService) GetLeaderboard(ctx context.Context, eventType domain.EventType, period string, limit int) ([]domain.LeaderboardEntry, error) {
	return nil, nil
}
func (m *mockStatsService) GetUserSlotsStats(ctx context.Context, userID, period string) (*domain.SlotsStats, error) {
	return nil, nil
}
func (m *mockStatsService) GetSlotsLeaderboardByProfit(ctx context.Context, period string, limit int) ([]domain.SlotsStats, error) {
	return nil, nil
}
func (m *mockStatsService) GetSlotsLeaderboardByWinRate(ctx context.Context, period string, minSpins, limit int) ([]domain.SlotsStats, error) {
	return nil, nil
}
func (m *mockStatsService) GetSlotsLeaderboardByMegaJackpots(ctx context.Context, period string, limit int) ([]domain.SlotsStats, error) {
	return nil, nil
}

func TestHandleSearch_CriticalSuccess(t *testing.T) {
	t.Parallel()
	// ARRANGE
	svc, repo := createSearchTestService()
	user := createTestUser()
	repo.users[TestUsername] = user

	// Force critical success: roll <= domain.SearchCriticalRate (0.05)
	svc.deps.Rnd = func() float64 { return 0.01 }

	// ACT
	msg, err := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")

	// ASSERT
	require.NoError(t, err)
	assert.Contains(t, msg, domain.MsgSearchCriticalSuccess)
	assert.Contains(t, msg, "2x") // Critical gives double reward

	// Verify inventory received 2x item
	inv, _ := repo.GetInventory(context.Background(), user.ID)
	found := false
	for _, slot := range inv.Slots {
		if slot.Quantity == 2 {
			found = true
			assert.Equal(t, domain.QualityEpic, slot.QualityLevel, "Critical success should yield Epic quality (Base Uncommon + 2)")
		}
	}
	assert.True(t, found, "Should receive 2x lootbox on critical success")
}

func TestHandleSearch_NormalSuccess(t *testing.T) {
	t.Parallel()
	// ARRANGE
	svc, repo := createSearchTestService()
	user := createTestUser()
	repo.users[TestUsername] = user

	// Force normal success: domain.SearchCriticalRate < roll <= domain.SearchSuccessRate
	svc.deps.Rnd = func() float64 { return 0.5 }

	// ACT
	msg, err := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")

	// ASSERT
	require.NoError(t, err)
	assert.Contains(t, msg, "You have found")
	assert.NotContains(t, msg, domain.MsgSearchCriticalSuccess)

	// Verify inventory received 1x item
	inv, _ := repo.GetInventory(context.Background(), user.ID)
	found := false
	for _, slot := range inv.Slots {
		if slot.Quantity == 1 {
			found = true
		}
	}
	assert.True(t, found, "Should receive 1x lootbox on normal success")
}

func TestHandleSearch_CriticalSuccess_Event(t *testing.T) {
	t.Parallel()
	// ARRANGE
	svc, repo := createSearchTestService()
	user := createTestUser()
	repo.users[TestUsername] = user

	// Force critical success: roll <= domain.SearchCriticalRate (0.05)
	svc.deps.Rnd = func() float64 { return 0.05 }

	ctx := context.Background()

	// ACT
	msg, err := svc.HandleSearch(ctx, domain.PlatformTwitch, "testuser123", TestUsername, "")
	require.NoError(t, err)

	// ASSERT
	assert.Contains(t, msg, domain.MsgSearchCriticalSuccess, "Should be a critical success")

	// Verify inventory received 2x item (stats recording now via events, not direct calls)
	inv, _ := repo.GetInventory(ctx, user.ID)
	found := false
	for _, slot := range inv.Slots {
		if slot.Quantity == 2 {
			found = true
		}
	}
	assert.True(t, found, "Should receive 2x lootbox on critical success")
}

func TestHandleSearch_NearMiss(t *testing.T) {
	t.Parallel()
	// ARRANGE
	svc, repo := createSearchTestService()
	user := createTestUser()
	repo.users[TestUsername] = user

	// Force near miss: successThreshold < roll <= successThreshold + domain.SearchNearMissRate
	svc.deps.Rnd = func() float64 { return 0.81 }

	// ACT
	msg, err := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, domain.MsgSearchNearMiss, msg)
	// Near miss event is now published via ResilientPublisher and handled by stats event handler
	_ = repo // repo still valid for other assertions
}

func TestHandleSearch_DiminishingReturns(t *testing.T) {
	t.Parallel()
	// ARRANGE
	svc, repo := createSearchTestService()
	user := createTestUser()
	repo.users[TestUsername] = user
	statsSvc := svc.deps.StatsSvc.(*mockStatsService)
	ctx := context.Background()

	// 1. Normal Search (Count 1)
	statsSvc.mockCounts[domain.StatsEventSearch] = 1
	svc.deps.Rnd = func() float64 { return 0.5 } // Guaranteed success

	msg, err := svc.HandleSearch(ctx, domain.PlatformTwitch, "testuser123", TestUsername, "")
	require.NoError(t, err)

	assert.NotContains(t, msg, domain.MsgFirstSearchBonus)
	assert.NotContains(t, msg, "(Exhausted)")

	// 2. Diminished Search (Count 6) - threshold is 6
	statsSvc.mockCounts[domain.StatsEventSearch] = 6
	// Force success (success rate remains 0.8, but we verify diminished flag effects like message)
	svc.deps.Rnd = func() float64 { return 0.05 }
	// Reset cooldown manually
	delete(repo.cooldowns[user.ID], domain.ActionSearch)

	msg, err = svc.HandleSearch(ctx, domain.PlatformTwitch, "testuser123", TestUsername, "")
	require.NoError(t, err)

	assert.Contains(t, msg, "(Exhausted)")
	// Stats events are now published via ResilientPublisher and handled by stats event handler
}

func TestHandleSearch_CriticalFail(t *testing.T) {
	t.Parallel()
	// ARRANGE
	svc, repo := createSearchTestService()
	user := createTestUser()
	repo.users[TestUsername] = user

	// Force critical fail: roll > 1.0 - domain.SearchCriticalFailRate
	svc.deps.Rnd = func() float64 { return 0.96 }

	// ACT
	msg, err := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")

	// ASSERT
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(msg, domain.MsgSearchCriticalFail))
	// Critical fail event is now published via ResilientPublisher and handled by stats event handler
	_ = repo
}

func TestHandleSearch_NormalFailure(t *testing.T) {
	t.Parallel()
	// ARRANGE
	svc, repo := createSearchTestService()
	user := createTestUser()
	repo.users[TestUsername] = user

	// Force normal failure: between near miss and critical fail
	svc.deps.Rnd = func() float64 { return 0.9 }

	// ACT
	msg, err := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")

	// ASSERT
	require.NoError(t, err)
	// Success messages for lootboxes contain the "x" quantifier (e.g., "You have found 1x ...")
	assert.NotContains(t, msg, "x ", "Should not be a success message")
	assert.NotContains(t, msg, domain.MsgSearchNearMiss)
	assert.NotContains(t, msg, domain.MsgSearchCriticalFail)

	// Should be one of the humorous failure messages
	isValid := false
	for _, failMsg := range domain.SearchFailureMessages {
		if msg == failMsg {
			isValid = true
			break
		}
	}
	assert.True(t, isValid, "Expected valid failure message, got: %s", msg)
}

func TestHandleSearch_BoundaryConditions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		roll       float64
		expectType string
	}{
		{"Exactly on critical threshold", domain.SearchCriticalRate, "crit_success"},
		{"Just above critical threshold", domain.SearchCriticalRate + 0.001, "normal_success"},
		{"Exactly on success threshold", domain.SearchSuccessRate, "normal_success"},
		{"Just above success threshold", domain.SearchSuccessRate + 0.001, "near_miss"},
		{"Edge of near miss range", domain.SearchSuccessRate + domain.SearchNearMissRate, "near_miss"},
		{"Just beyond near miss range", domain.SearchSuccessRate + domain.SearchNearMissRate + 0.001, "normal_fail"},
		{"Edge of crit fail range", 1.0 - domain.SearchCriticalFailRate, "normal_fail"},
		{"Just inside crit fail range", 1.0 - domain.SearchCriticalFailRate + 0.001, "crit_fail"},
		{"Minimum possible roll", 0.0, "crit_success"},
		{"Maximum possible roll", 1.0, "crit_fail"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc, repo := createSearchTestService()
			user := createTestUser()
			repo.users[TestUsername] = user

			svc.deps.Rnd = func() float64 { return tt.roll }

			msg, err := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")
			require.NoError(t, err)

			switch tt.expectType {
			case "crit_success":
				assert.Contains(t, msg, domain.MsgSearchCriticalSuccess)
			case "normal_success":
				assert.Contains(t, msg, "You have found")
				assert.NotContains(t, msg, domain.MsgSearchCriticalSuccess)
			case "near_miss":
				assert.Equal(t, domain.MsgSearchNearMiss, msg)
			case "crit_fail":
				assert.True(t, strings.HasPrefix(msg, domain.MsgSearchCriticalFail))
			case "normal_fail":
				assert.NotContains(t, msg, "x ")
				assert.NotEqual(t, domain.MsgSearchNearMiss, msg)
				assert.False(t, strings.HasPrefix(msg, domain.MsgSearchCriticalFail))
			}
		})
	}
}

// Test Job Bonus
func TestHandleSearch_JobBonus(t *testing.T) {
	t.Parallel()
	// ARRANGE
	svc, repo := createSearchTestService(func(opts *searchTestServiceOpts) {
		mockJob := opts.jobService.(*mockJobService)
		mockJob.jobLevels[domain.JobKeyExplorer] = 50 // High level should give bonus
	})
	user := createTestUser()
	repo.users[TestUsername] = user

	// ACT
	svc.deps.Rnd = func() float64 { return 0.5 } // Normal success
	_, err := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")
	require.NoError(t, err)

	inv, _ := repo.GetInventory(context.Background(), user.ID)
	require.NotEmpty(t, inv.Slots)
	assert.Equal(t, domain.QualityLegendary, inv.Slots[0].QualityLevel, "High job level should yield Legendary quality")
}

// Test XP Event
func TestHandleSearch_XPEvent(t *testing.T) {
	t.Parallel()
	// ARRANGE
	mockBus := &MockBus{}
	dlPath := filepath.Join(t.TempDir(), "dead_letter.json")
	rp, err := event.NewResilientPublisher(mockBus, 3, time.Millisecond, dlPath)
	require.NoError(t, err)

	svc, repo := createSearchTestService(func(opts *searchTestServiceOpts) {
		opts.publisher = rp
	})
	user := createTestUser()
	repo.users[TestUsername] = user

	ctx := context.Background()

	// 1. Normal Search
	svc.deps.Rnd = func() float64 { return 0.5 }
	_, err = svc.HandleSearch(ctx, domain.PlatformTwitch, "testuser123", TestUsername, "")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return len(mockBus.PublishedEvents) >= 1
	}, time.Second, 10*time.Millisecond)

	// Verify Event
	lastEvent := mockBus.PublishedEvents[len(mockBus.PublishedEvents)-1]
	require.Equal(t, event.Type(domain.EventTypeSearchPerformed), lastEvent.Type)
	payload := lastEvent.Payload.(domain.SearchPerformedPayload)
	assert.Equal(t, 45, payload.XPAmount, "Normal search should give 45 XP")

	// 2. Diminished Search
	statsSvc := svc.deps.StatsSvc.(*mockStatsService)
	statsSvc.mockCounts[domain.StatsEventSearch] = 6
	// Reset cooldown
	delete(repo.cooldowns[user.ID], domain.ActionSearch)

	_, err = svc.HandleSearch(ctx, domain.PlatformTwitch, "testuser123", TestUsername, "")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return len(mockBus.PublishedEvents) >= 2
	}, time.Second, 10*time.Millisecond)

	lastEvent = mockBus.PublishedEvents[len(mockBus.PublishedEvents)-1]
	payload = lastEvent.Payload.(domain.SearchPerformedPayload)
	// 45 * 0.1 = 4.5 -> 4
	assert.Equal(t, 4, payload.XPAmount, "Diminished search should give 4 XP")
}

// Test Streak Bonus
func TestHandleSearch_StreakBonus(t *testing.T) {
	t.Parallel()
	// ARRANGE
	svc, repo := createSearchTestService()
	user := createTestUser()
	repo.users[TestUsername] = user
	statsSvc := svc.deps.StatsSvc.(*mockStatsService)
	statsSvc.mockStreak = 5 // Streak 5 gives +1 bonus point

	// ACT
	svc.deps.Rnd = func() float64 { return 0.5 }
	msg, err := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")
	require.NoError(t, err)

	// ASSERT
	assert.Contains(t, msg, "Streak", "Message should mention streak bonus")

	inv, _ := repo.GetInventory(context.Background(), user.ID)
	require.NotEmpty(t, inv.Slots)
	assert.Equal(t, domain.QualityRare, inv.Slots[0].QualityLevel, "Streak 5 should bump quality to Rare")
}

// Test Exhausted Message Logic
func TestHandleSearch_ExhaustedMessage_Logic(t *testing.T) {
	t.Parallel()
	// ARRANGE
	svc, repo := createSearchTestService()
	user := createTestUser()
	repo.users[TestUsername] = user
	statsSvc := svc.deps.StatsSvc.(*mockStatsService)

	// 1. Count 5 (Threshold-1) -> No message
	statsSvc.mockCounts[domain.StatsEventSearch] = 5
	svc.deps.Rnd = func() float64 { return 0.5 }
	msg, err := svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")
	require.NoError(t, err)
	assert.NotContains(t, msg, "(Exhausted)")

	// 2. Count 6 (Threshold) -> Message
	delete(repo.cooldowns[user.ID], domain.ActionSearch)
	statsSvc.mockCounts[domain.StatsEventSearch] = 6
	msg, err = svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")
	require.NoError(t, err)
	assert.Contains(t, msg, "(Exhausted)")

	// 3. Count 7 (Threshold+1) -> No message
	delete(repo.cooldowns[user.ID], domain.ActionSearch)
	statsSvc.mockCounts[domain.StatsEventSearch] = 7
	msg, err = svc.HandleSearch(context.Background(), domain.PlatformTwitch, "testuser123", TestUsername, "")
	require.NoError(t, err)
	assert.NotContains(t, msg, "(Exhausted)", "Should only show exhausted message once")
}
