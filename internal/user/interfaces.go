package user

import (
	"context"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// InventoryItem represents an item in a user's inventory with display information
type InventoryItem struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

// InventoryService handles inventory operations
type InventoryService interface {
	// Inventory operations by platform ID
	UseItem(ctx context.Context, platform, platformID, username, itemName string, quantity int, targetUsername string) (string, error)
	GetInventory(ctx context.Context, platform, platformID, username, filter string) ([]InventoryItem, error)
	GiveItem(ctx context.Context, ownerPlatform, ownerPlatformID, ownerUsername, receiverPlatform, receiverUsername, itemName string, quantity int) error

	// Inventory operations by username
	AddItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) error
	RemoveItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) (int, error)
	GetInventoryByUsername(ctx context.Context, platform, username, filter string) ([]InventoryItem, error)
}

// ManagementService handles user lifecycle operations
type ManagementService interface {
	RegisterUser(ctx context.Context, user domain.User) (domain.User, error)
	UpdateUser(ctx context.Context, user domain.User) error
	FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error)
}

// AccountLinkingService handles account linking operations
type AccountLinkingService interface {
	MergeUsers(ctx context.Context, primaryUserID, secondaryUserID string) error
	UnlinkPlatform(ctx context.Context, userID, platform string) error
	GetLinkedPlatforms(ctx context.Context, platform, platformID string) ([]string, error)
}

// GameplayService handles gameplay features
type GameplayService interface {
	HandleSearch(ctx context.Context, platform, platformID, username string) (string, error)
	HandleIncomingMessage(ctx context.Context, platform, platformID, username, message string) (*domain.MessageResult, error)
	TimeoutUser(ctx context.Context, username string, duration time.Duration, reason string) error
	GetTimeout(ctx context.Context, username string) (time.Duration, error)
	ReduceTimeout(ctx context.Context, username string, reduction time.Duration) error
	ApplyShield(ctx context.Context, user *domain.User, quantity int, isMirror bool) error
}

// Service is the full interface that composes all sub-interfaces.
// Existing code can continue using this interface for backwards compatibility.
// New code should depend on the smallest interface that meets its needs.
type Service interface {
	InventoryService
	ManagementService
	AccountLinkingService
	GameplayService

	// Service lifecycle
	GetCacheStats() CacheStats
	Shutdown(ctx context.Context) error
}
