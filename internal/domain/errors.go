package domain

import "errors"

// Error message string constants - single source of truth for error messages
// Use these in assert.Contains() checks when testing error messages
const (
	// User errors
	ErrMsgUserNotFound = "user not found"

	// Item errors
	ErrMsgItemNotFound = "item not found"

	// Inventory errors
	ErrMsgInsufficientQuantity = "insufficient quantity"
	ErrMsgInventoryFull        = "inventory is full"

	// Economy errors
	ErrMsgInsufficientFunds = "insufficient funds"
	ErrMsgNotSellable       = "item is not sellable"
	ErrMsgNotBuyable        = "is not buyable"

	// Validation errors (used for partial matches)
	ErrMsgInvalidQuantity = "quantity" // Used in contains checks for various quantity errors

	// Gamble errors
	ErrMsgGambleAlreadyActive = "a gamble is already active"

	// Database/System errors
	ErrMsgConnectionTimeout = "connection timeout"
	ErrMsgDatabaseError     = "database error"
	ErrMsgDeadlockDetected  = "deadlock detected"

	// Cooldown errors
	ErrMsgOnCooldown = "action on cooldown"

	// Feature errors
	ErrMsgFeatureLocked = "feature is locked"

	// Recipe/Crafting errors
	ErrMsgRecipeNotFound = "recipe not found"
	ErrMsgRecipeLocked   = "recipe is locked"
	ErrMsgInvalidRecipe  = "invalid recipe"

	// Platform errors
	ErrMsgInvalidPlatform = "invalid platform"

	// Input errors
	ErrMsgInvalidInput = "invalid input"
)

// Common domain errors
// These errors should be used consistently across all layers of the application.
// Wrap these errors with fmt.Errorf("%w: %s", domain.ErrXxx, details) for additional context.
var (
	// User errors
	ErrUserNotFound = errors.New(ErrMsgUserNotFound)

	// Item errors
	ErrItemNotFound = errors.New(ErrMsgItemNotFound)

	// Inventory errors
	ErrInsufficientQuantity = errors.New(ErrMsgInsufficientQuantity)
	ErrInventoryFull        = errors.New(ErrMsgInventoryFull)

	// Economy errors
	ErrInsufficientFunds = errors.New(ErrMsgInsufficientFunds)
	ErrNotSellable       = errors.New(ErrMsgNotSellable)
	ErrNotBuyable        = errors.New(ErrMsgNotBuyable)

	// Cooldown errors
	ErrOnCooldown = errors.New(ErrMsgOnCooldown)

	// Feature errors
	ErrFeatureLocked = errors.New(ErrMsgFeatureLocked)

	// Recipe/Crafting errors
	ErrRecipeNotFound = errors.New(ErrMsgRecipeNotFound)
	ErrRecipeLocked   = errors.New(ErrMsgRecipeLocked)
	ErrInvalidRecipe  = errors.New(ErrMsgInvalidRecipe)

	// Validation errors
	ErrInvalidInput = errors.New(ErrMsgInvalidInput)

	// Platform errors
	ErrInvalidPlatform = errors.New(ErrMsgInvalidPlatform)
)
