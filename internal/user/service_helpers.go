package user

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// getItemByNameCached retrieves an item from cache or DB
// Supports both internal names (lootbox_tier0) and public names (junkbox)
func (s *service) getItemByNameCached(ctx context.Context, name string) (*domain.Item, error) {
	// Try to resolve as public name first (e.g., "junkbox" -> "lootbox_tier0")
	if internalName, ok := s.namingResolver.ResolvePublicName(name); ok {
		name = internalName
	}

	s.itemCacheMu.RLock()
	if item, ok := s.itemCacheByName[name]; ok {
		s.itemCacheMu.RUnlock()
		return &item, nil
	}
	s.itemCacheMu.RUnlock()

	item, err := s.repo.GetItemByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if item != nil {
		s.itemCacheMu.Lock()
		s.itemCacheByName[name] = *item
		s.itemIDToName[item.ID] = name
		s.itemCacheMu.Unlock()
	}
	return item, nil
}

type txContextKey struct{}

// withTx executes a function within a transaction.
// It handles begin, commit, and rollback automatically.
// The operation function receives the transaction and should return an error if it fails.
// It also detects nested transactions and fails fast to prevent deadlocks.
func (s *service) withTx(ctx context.Context, operation func(txCtx context.Context, tx repository.UserTx) error) error {
	log := logger.FromContext(ctx)

	// Fail fast if we detect a nested transaction
	if ctx.Value(txContextKey{}) != nil {
		err := fmt.Errorf("developer error: nested transactions are not supported and cause deadlocks")
		log.Error("Nested transaction detected", "error", err)
		return err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Create a new context indicating we are inside a transaction
	txCtx := context.WithValue(ctx, txContextKey{}, true)

	if err := operation(txCtx, tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("Failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
