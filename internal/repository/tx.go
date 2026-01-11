package repository

import (
	"context"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Tx defines the interface for transactional operations
type Tx interface {
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
	GetLastCooldownForUpdate(ctx context.Context, userID, action string) (*time.Time, error)
	UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}
