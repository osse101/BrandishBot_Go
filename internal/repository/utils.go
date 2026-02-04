package repository

import (
	"context"
	"errors"

	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// ErrTxClosed is returned when attempting to rollback an already-closed transaction.
// Implementations should wrap this error when the underlying transaction is closed.
var ErrTxClosed = errors.New("transaction already closed")

// SafeRollback rolls back a transaction and logs any error that isn't ErrTxClosed.
// Use this in defer to ensure proper cleanup without noisy logs for already-closed transactions.
func SafeRollback(ctx context.Context, tx Tx) {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, ErrTxClosed) {
		logger.FromContext(ctx).Error("Failed to rollback transaction", "error", err)
	}
}
