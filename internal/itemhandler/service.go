// Package itemhandler implements the item effect handler system.
// Each item type (weapon, lootbox, trap, etc.) has a dedicated handler
// that processes the item's effect when used by a player.
package itemhandler

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
)

// ActiveTarget represents a user selected for random targeting.
type ActiveTarget struct {
	UserID   string
	Username string
}

// HandlerArgs contains arguments for item handlers.
type HandlerArgs struct {
	Username       string
	Platform       string
	TargetUsername string
	JobName        string
}

// EffectContext provides the capabilities that item handlers need from the
// broader service layer, without coupling to a concrete service implementation.
type EffectContext interface {
	// Naming
	GetDisplayName(itemName string, quality domain.QualityLevel) string
	Pluralize(name string, quantity int) string

	// Combat
	TimeoutUser(ctx context.Context, username string, duration time.Duration, reason string) error
	ReduceTimeout(ctx context.Context, username string, reduction time.Duration) error
	ApplyShield(ctx context.Context, user *domain.User, quantity int, isMirror bool) error

	// Targeting
	GetRandomTarget(platform string) (username, userID string, err error)
	GetRandomTargets(platform string, count int) ([]ActiveTarget, error)
	RemoveActiveChatter(platform, userID string)

	// Items
	GetItemByName(ctx context.Context, name string) (*domain.Item, error)

	// Events
	RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, data interface{}) error
	PublishItemUsedEvent(ctx context.Context, userID, itemName string, quantity int, metadata map[string]interface{})

	// Lootbox
	OpenLootbox(ctx context.Context, lootboxName string, quantity int, boxQuality domain.QualityLevel) ([]lootbox.DroppedItem, error)

	// Traps
	GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error)
	CreateTrap(ctx context.Context, trap *domain.Trap) error
	GetActiveTrapForUpdate(ctx context.Context, targetID uuid.UUID) (*domain.Trap, error)
	TriggerTrap(ctx context.Context, trapID uuid.UUID) error

	// Bombs
	SetPendingBomb(ctx context.Context, platform, setterUsername string, timeout time.Duration) error

	// RNG
	RandomFloat() float64
}
