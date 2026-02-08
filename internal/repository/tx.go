package repository

import (
	"context"
)

// Tx defines the interface for transaction lifecycle operations.
// Domain-specific transaction interfaces (UserTx, CraftingTx, etc.) embed this
// and add their own data access methods.
type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}
