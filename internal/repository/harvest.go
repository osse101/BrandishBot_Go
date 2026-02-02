package repository

import (
	"context"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// HarvestRepository handles harvest state persistence
type HarvestRepository interface {
	// GetHarvestState retrieves the harvest state for a user
	GetHarvestState(ctx context.Context, userID string) (*domain.HarvestState, error)

	// CreateHarvestState initializes harvest state for a new user
	CreateHarvestState(ctx context.Context, userID string) (*domain.HarvestState, error)

	// Transaction support
	BeginTx(ctx context.Context) (HarvestTx, error)
}

// HarvestTx defines the interface for harvest transactions
type HarvestTx interface {
	Tx

	// GetHarvestStateWithLock retrieves the harvest state with FOR UPDATE lock
	GetHarvestStateWithLock(ctx context.Context, userID string) (*domain.HarvestState, error)

	// UpdateHarvestState updates the last harvested timestamp
	UpdateHarvestState(ctx context.Context, userID string, lastHarvestedAt time.Time) error

	// Inventory operations within transaction
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
}
