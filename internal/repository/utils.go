package repository

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// SafeRollback rolls back a transaction and logs any error
func SafeRollback(ctx context.Context, tx Tx) {
	if err := tx.Rollback(ctx); err != nil {
		// Check for common "closed" errors to avoid noise
		if err.Error() != domain.ErrMsgTxClosed {
			logger.FromContext(ctx).Error("Failed to rollback transaction", "error", err)
		}
	}
}
