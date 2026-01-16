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

// ============================================================================
// Helper Functions for Inventory Operations
// ============================================================================

// buildLogFields creates consistent log fields for operation entry logging
func buildLogFields(mode userLookupMode, params inventoryOperationParams) []interface{} {
	fields := []interface{}{
		"platform", params.platform,
		"username", params.username,
	}

	// Add platformID first for consistency
	if mode == lookupByPlatformID && params.platformID != "" {
		fields = append([]interface{}{"platformID", params.platformID}, fields...)
	}

	// Add item fields if present
	if params.itemName != "" {
		fields = append(fields, "item", params.itemName)
	}
	if params.quantity > 0 {
		fields = append(fields, "quantity", params.quantity)
	}

	// Add optional fields
	if params.targetUsername != "" {
		fields = append(fields, "target", params.targetUsername)
	}

	return fields
}

// buildSuccessLogFields creates consistent log fields for success logging
func buildSuccessLogFields[T any](mode userLookupMode, params inventoryOperationParams, result T) []interface{} {
	fields := []interface{}{
		"username", params.username,
	}

	if params.itemName != "" {
		fields = append(fields, "item", params.itemName)
	}

	// Add result field with appropriate name based on type
	switch v := any(result).(type) {
	case int:
		if v != 0 {
			fields = append(fields, "result", v)
		}
	case string:
		if v != "" {
			fields = append(fields, "message", v)
		}
	}

	return fields
}

// validateByMode validates operation parameters based on lookup mode
func validateByMode(mode userLookupMode, params inventoryOperationParams) error {
	if mode == lookupByPlatformID {
		return validateInventoryInput(params.platform, params.platformID, params.username, params.quantity)
	}
	return validateInventoryInputByUsername(params.platform, params.username, params.quantity)
}

// lookupUserByMode retrieves a user based on lookup mode
func (s *service) lookupUserByMode(ctx context.Context, mode userLookupMode, params inventoryOperationParams) (*domain.User, error) {
	if mode == lookupByPlatformID {
		return s.getUserOrRegister(ctx, params.platform, params.platformID, params.username)
	}
	return s.repo.GetUserByPlatformUsername(ctx, params.platform, params.username)
}

// ============================================================================
// Generic Inventory Operation Handler
// ============================================================================

// withUserOpResult executes an inventory operation with any return type.
// This generic function consolidates the common patterns of validation,
// user lookup, operation execution, and logging for all inventory operations.
func withUserOpResult[T any](
	s *service,
	ctx context.Context,
	mode userLookupMode,
	params inventoryOperationParams,
	operationName string,
	zeroValue T,
	operation func(ctx context.Context, user *domain.User) (T, error),
) (T, error) {
	log := logger.FromContext(ctx)

	// Log operation start with consistent formatting
	logFields := buildLogFields(mode, params)
	log.Info(operationName+" called", logFields...)

	// Validate input based on mode
	if err := validateByMode(mode, params); err != nil {
		log.Warn(operationName+" validation failed", "error", err)
		return zeroValue, err
	}

	// Lookup user based on mode
	user, err := s.lookupUserByMode(ctx, mode, params)
	if err != nil {
		log.Warn(operationName+" user lookup failed", "error", err, "mode", mode)
		return zeroValue, err
	}

	// Execute the actual operation
	result, err := operation(ctx, user)
	if err != nil {
		// Don't log at Error level - internal methods already do that
		log.Debug(operationName+" operation failed", "error", err, "username", params.username)
		return zeroValue, err
	}

	// Log success with consistent formatting
	successFields := buildSuccessLogFields(mode, params, result)
	log.Info(operationName+" successful", successFields...)

	return result, nil
}

// ============================================================================
// Type-Specific Wrapper Functions
// ============================================================================

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

// withUserOp executes an inventory operation that returns only an error.
// This is a thin wrapper around withUserOpResult for operations without return values.
func (s *service) withUserOp(
	ctx context.Context,
	mode userLookupMode,
	params inventoryOperationParams,
	operationName string,
	operation func(ctx context.Context, user *domain.User) error,
) error {
	// Wrap the error-only operation to match the generic signature
	_, err := withUserOpResult(s, ctx, mode, params, operationName, struct{}{},
		func(ctx context.Context, user *domain.User) (struct{}, error) {
			return struct{}{}, operation(ctx, user)
		})
	return err
}

// withUserOpInt executes an inventory operation that returns (int, error).
// This is a thin wrapper around withUserOpResult for int-returning operations.
func (s *service) withUserOpInt(
	ctx context.Context,
	mode userLookupMode,
	params inventoryOperationParams,
	operationName string,
	operation func(ctx context.Context, user *domain.User) (int, error),
) (int, error) {
	return withUserOpResult(s, ctx, mode, params, operationName, 0, operation)
}

// withUserOpString executes an inventory operation that returns (string, error).
// This is a thin wrapper around withUserOpResult for string-returning operations.
func (s *service) withUserOpString(
	ctx context.Context,
	mode userLookupMode,
	params inventoryOperationParams,
	operationName string,
	operation func(ctx context.Context, user *domain.User) (string, error),
) (string, error) {
	return withUserOpResult(s, ctx, mode, params, operationName, "", operation)
}
