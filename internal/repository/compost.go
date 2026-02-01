package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Compost defines the interface for compost data access
type Compost interface {
	CreateDeposit(ctx context.Context, deposit *domain.CompostDeposit) error
	GetDeposit(ctx context.Context, id uuid.UUID) (*domain.CompostDeposit, error)
	GetActiveDepositsForUser(ctx context.Context, userID uuid.UUID) ([]domain.CompostDeposit, error)
	GetReadyDepositsForUser(ctx context.Context, userID uuid.UUID) ([]domain.CompostDeposit, error)
	HarvestDeposit(ctx context.Context, id uuid.UUID, gemsAwarded int) error
	HarvestAllReady(ctx context.Context, userID uuid.UUID) (int, error) // Returns total gems awarded

	// Transaction support
	BeginTx(ctx context.Context) (Tx, error)
	BeginCompostTx(ctx context.Context) (CompostTx, error)

	// User operations
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)

	// Inventory operations
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
}

// CompostTx extends Tx with compost-specific transactional operations
type CompostTx interface {
	Tx // Commit, Rollback

	// Compost operations within transaction
	CreateDeposit(ctx context.Context, deposit *domain.CompostDeposit) error
	GetReadyDepositsForUser(ctx context.Context, userID uuid.UUID) ([]domain.CompostDeposit, error)
	HarvestDeposit(ctx context.Context, id uuid.UUID, gemsAwarded int) error

	// Inventory operations within transaction
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
}
