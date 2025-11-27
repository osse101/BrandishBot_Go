package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// SafeRollback rolls back a transaction and logs any error that isn't ErrTxClosed
func SafeRollback(ctx context.Context, tx pgx.Tx) {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		logger.FromContext(ctx).Error("Failed to rollback transaction", "error", err)
	}
}
