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

// withTx executes a function within a transaction.
// It handles begin, commit, and rollback automatically.
// The operation function receives the transaction and should return an error if it fails.
func (s *service) withTx(ctx context.Context, operation func(tx repository.UserTx) error) error {
	log := logger.FromContext(ctx)

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	if err := operation(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("Failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// userLookupMode specifies how to lookup a user
type userLookupMode int

const (
	lookupByPlatformID userLookupMode = iota
	lookupByUsername
)

// inventoryOperationParams holds parameters for inventory operations
type inventoryOperationParams struct {
	platform   string
	platformID string
	username   string
	itemName   string
	quantity   int
	// Additional fields for specific operations
	targetUsername string // for UseItem
}

// withUserOp executes an inventory operation that returns only an error
func (s *service) withUserOp(
	ctx context.Context,
	mode userLookupMode,
	params inventoryOperationParams,
	operationName string,
	operation func(ctx context.Context, user *domain.User) error,
) error {
	log := logger.FromContext(ctx)

	// Log call
	if mode == lookupByPlatformID {
		log.Info(operationName+" called", "platform", params.platform, "platformID", params.platformID, "username", params.username, "item", params.itemName, "quantity", params.quantity)
	} else {
		log.Info(operationName+" called", "platform", params.platform, "username", params.username, "item", params.itemName, "quantity", params.quantity)
	}

	// Validate input
	var err error
	if mode == lookupByPlatformID {
		err = validateInventoryInput(params.platform, params.platformID, params.username, params.quantity)
	} else {
		err = validateInventoryInputByUsername(params.platform, params.username, params.quantity)
	}
	if err != nil {
		return err
	}

	// Lookup user
	var user *domain.User
	if mode == lookupByPlatformID {
		user, err = s.getUserOrRegister(ctx, params.platform, params.platformID, params.username)
	} else {
		user, err = s.repo.GetUserByPlatformUsername(ctx, params.platform, params.username)
	}
	if err != nil {
		return err
	}

	// Execute operation
	if err := operation(ctx, user); err != nil {
		return err
	}

	// Log success
	if mode == lookupByPlatformID {
		log.Info(operationName+" successful", "username", params.username, "item", params.itemName, "quantity", params.quantity)
	} else {
		log.Info(operationName+" successful by username", "username", params.username, "item", params.itemName, "quantity", params.quantity)
	}

	return nil
}

// withUserOpInt executes an inventory operation that returns (int, error)
func (s *service) withUserOpInt(
	ctx context.Context,
	mode userLookupMode,
	params inventoryOperationParams,
	operationName string,
	operation func(ctx context.Context, user *domain.User) (int, error),
) (int, error) {
	log := logger.FromContext(ctx)

	// Log call
	if mode == lookupByPlatformID {
		log.Info(operationName+" called", "platform", params.platform, "platformID", params.platformID, "username", params.username, "item", params.itemName, "quantity", params.quantity)
	} else {
		log.Info(operationName+" called", "platform", params.platform, "username", params.username, "item", params.itemName, "quantity", params.quantity)
	}

	// Validate input
	var err error
	if mode == lookupByPlatformID {
		err = validateInventoryInput(params.platform, params.platformID, params.username, params.quantity)
	} else {
		err = validateInventoryInputByUsername(params.platform, params.username, params.quantity)
	}
	if err != nil {
		return 0, err
	}

	// Lookup user
	var user *domain.User
	if mode == lookupByPlatformID {
		user, err = s.getUserOrRegister(ctx, params.platform, params.platformID, params.username)
	} else {
		user, err = s.repo.GetUserByPlatformUsername(ctx, params.platform, params.username)
	}
	if err != nil {
		return 0, err
	}

	// Execute operation
	result, err := operation(ctx, user)
	if err != nil {
		return 0, err
	}

	// Log success
	if mode == lookupByPlatformID {
		log.Info(operationName+" successful", "username", params.username, "item", params.itemName, "result", result)
	} else {
		log.Info(operationName+" successful by username", "username", params.username, "item", params.itemName, "result", result)
	}

	return result, nil
}

// withUserOpString executes an inventory operation that returns (string, error)
func (s *service) withUserOpString(
	ctx context.Context,
	mode userLookupMode,
	params inventoryOperationParams,
	operationName string,
	operation func(ctx context.Context, user *domain.User) (string, error),
) (string, error) {
	log := logger.FromContext(ctx)

	// Log call
	logFields := []interface{}{"platform", params.platform, "username", params.username, "item", params.itemName, "quantity", params.quantity}
	if mode == lookupByPlatformID {
		logFields = append([]interface{}{"platformID", params.platformID}, logFields...)
	}
	if params.targetUsername != "" {
		logFields = append(logFields, "target", params.targetUsername)
	}
	log.Info(operationName+" called", logFields...)

	// Validate input
	var err error
	if mode == lookupByPlatformID {
		err = validateInventoryInput(params.platform, params.platformID, params.username, params.quantity)
	} else {
		err = validateInventoryInputByUsername(params.platform, params.username, params.quantity)
	}
	if err != nil {
		return "", err
	}

	// Lookup user
	var user *domain.User
	if mode == lookupByPlatformID {
		user, err = s.getUserOrRegister(ctx, params.platform, params.platformID, params.username)
	} else {
		user, err = s.repo.GetUserByPlatformUsername(ctx, params.platform, params.username)
	}
	if err != nil {
		return "", err
	}

	// Execute operation
	result, err := operation(ctx, user)
	if err != nil {
		return "", err
	}

	// Log success
	logFields = []interface{}{"username", params.username, "item", params.itemName, "message", result}
	if mode == lookupByPlatformID {
		log.Info(operationName+" successful", logFields...)
	} else {
		log.Info(operationName+" successful by username", logFields...)
	}

	return result, nil
}
