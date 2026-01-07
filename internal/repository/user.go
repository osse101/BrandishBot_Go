package repository

import (
	"context"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// User defines the interface for user persistence
type User interface {
	UpsertUser(ctx context.Context, user *domain.User) error
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error)
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
	UpdateUser(ctx context.Context, user domain.User) error
	DeleteUser(ctx context.Context, userID string) error
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
	DeleteInventory(ctx context.Context, userID string) error
	GetItemByName(ctx context.Context, itemName string) (*domain.Item, error)
	GetItemsByNames(ctx context.Context, names []string) ([]domain.Item, error)
	GetItemByID(ctx context.Context, id int) (*domain.Item, error)
	GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error)

	BeginTx(ctx context.Context) (UserTx, error)

	GetLastCooldown(ctx context.Context, userID, action string) (*time.Time, error)
	UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error

	// Account linking - atomic transaction for merge
	MergeUsersInTransaction(ctx context.Context, primaryUserID, secondaryUserID string, mergedUser domain.User, mergedInventory domain.Inventory) error
}

// UserTx defines the interface for user transactions
type UserTx interface {
	Tx
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
}
