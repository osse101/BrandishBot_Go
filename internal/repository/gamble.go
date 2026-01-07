package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Gamble defines the interface for data access required by the gamble service
type Gamble interface {
	CreateGamble(ctx context.Context, gamble *domain.Gamble) error
	GetGamble(ctx context.Context, id uuid.UUID) (*domain.Gamble, error)
	JoinGamble(ctx context.Context, participant *domain.Participant) error
	UpdateGambleState(ctx context.Context, id uuid.UUID, state domain.GambleState) error
	UpdateGambleStateIfMatches(ctx context.Context, id uuid.UUID, expectedState, newState domain.GambleState) (int64, error)
	SaveOpenedItems(ctx context.Context, items []domain.GambleOpenedItem) error
	CompleteGamble(ctx context.Context, result *domain.GambleResult) error
	GetActiveGamble(ctx context.Context) (*domain.Gamble, error)

	// Transaction support
	BeginTx(ctx context.Context) (Tx, error)
	BeginGambleTx(ctx context.Context) (GambleTx, error)

	// Inventory operations (reused from other services)
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	GetItemByID(ctx context.Context, id int) (*domain.Item, error)
}

// GambleTx extends Tx with gamble-specific transactional operations
// This enables wrapping all ExecuteGamble operations in a single atomic transaction
type GambleTx interface {
	Tx // Commit, Rollback

	// Gamble operations within transaction
	UpdateGambleStateIfMatches(ctx context.Context, id uuid.UUID, expectedState, newState domain.GambleState) (int64, error)
	SaveOpenedItems(ctx context.Context, items []domain.GambleOpenedItem) error
	CompleteGamble(ctx context.Context, result *domain.GambleResult) error

	// Inventory operations within transaction
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
}
