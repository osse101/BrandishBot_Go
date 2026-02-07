package progression

import (
	"context"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// MockUser implements repository.User for testing
type MockUser struct {
	users map[string]*domain.User
}

// NewMockUser creates a new mock user repository with test users
func NewMockUser() *MockUser {
	return &MockUser{
		users: map[string]*domain.User{
			"test-user-1": {
				ID:        "test-user-1",
				Username:  "testuser",
				DiscordID: "user1",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			"test-user-2": {
				ID:        "test-user-2",
				Username:  "testuser2",
				DiscordID: "user2",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			"test-user-3": {
				ID:        "test-user-3",
				Username:  "testuser3",
				DiscordID: "user3",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
}

// UpsertUser inserts or updates a user
func (m *MockUser) UpsertUser(ctx context.Context, user *domain.User) error {
	if user == nil {
		return nil
	}
	m.users[user.ID] = user
	return nil
}

// GetUserByPlatformID returns a user by platform and platform ID
func (m *MockUser) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	for _, user := range m.users {
		switch platform {
		case "discord":
			if user.DiscordID == platformID {
				return user, nil
			}
		case "twitch":
			if user.TwitchID == platformID {
				return user, nil
			}
		case "youtube":
			if user.YoutubeID == platformID {
				return user, nil
			}
		}
	}
	return nil, nil
}

// GetUserByPlatformUsername returns a user by platform and username
func (m *MockUser) GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, nil
}

// GetUserByID returns a user by internal ID
func (m *MockUser) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	if user, ok := m.users[userID]; ok {
		return user, nil
	}
	return nil, nil
}

// UpdateUser updates an existing user
func (m *MockUser) UpdateUser(ctx context.Context, user domain.User) error {
	if _, ok := m.users[user.ID]; ok {
		m.users[user.ID] = &user
	}
	return nil
}

// DeleteUser deletes a user by ID
func (m *MockUser) DeleteUser(ctx context.Context, userID string) error {
	delete(m.users, userID)
	return nil
}

// GetInventory returns a user's inventory (stub)
func (m *MockUser) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return nil, nil
}

// UpdateInventory updates a user's inventory (stub)
func (m *MockUser) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return nil
}

// DeleteInventory deletes a user's inventory (stub)
func (m *MockUser) DeleteInventory(ctx context.Context, userID string) error {
	return nil
}

// GetItemByName returns an item by name (stub)
func (m *MockUser) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	return nil, nil
}

// GetItemsByNames returns items by names (stub)
func (m *MockUser) GetItemsByNames(ctx context.Context, names []string) ([]domain.Item, error) {
	return nil, nil
}

// GetItemByID returns an item by ID (stub)
func (m *MockUser) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	return nil, nil
}

// GetItemsByIDs returns items by IDs (stub)
func (m *MockUser) GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error) {
	return nil, nil
}

// GetAllItems returns all items (stub)
func (m *MockUser) GetAllItems(ctx context.Context) ([]domain.Item, error) {
	return nil, nil
}

// BeginTx begins a transaction (stub)
func (m *MockUser) BeginTx(ctx context.Context) (repository.UserTx, error) {
	return nil, nil
}

// GetLastCooldown returns the last cooldown timestamp (stub)
func (m *MockUser) GetLastCooldown(ctx context.Context, userID, action string) (*time.Time, error) {
	return nil, nil
}

// UpdateCooldown updates a cooldown timestamp (stub)
func (m *MockUser) UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error {
	return nil
}

// MergeUsersInTransaction merges two users in a transaction (stub)
func (m *MockUser) MergeUsersInTransaction(ctx context.Context, primaryUserID, secondaryUserID string, mergedUser domain.User, mergedInventory domain.Inventory) error {
	return nil
}

// GetRecentlyActiveUsers returns recently active users (stub)
func (m *MockUser) GetRecentlyActiveUsers(ctx context.Context, limit int) ([]domain.User, error) {
	return nil, nil
}

var _ repository.User = (*MockUser)(nil)
