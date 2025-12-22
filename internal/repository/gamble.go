package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

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
