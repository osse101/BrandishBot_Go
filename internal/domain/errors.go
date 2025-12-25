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
	ErrMsgNotInInventory       = "not in inventory"

	// Economy errors
	ErrMsgInsufficientFunds = "insufficient funds"
	ErrMsgNotSellable       = "item is not sellable"
	ErrMsgNotBuyable        = "is not buyable"

	// Validation errors (used for partial matches)
	ErrMsgInvalidQuantity = "quantity" // Used in contains checks for various quantity errors

	// Gamble errors
	ErrMsgGambleAlreadyActive        = "a gamble is already active"
	ErrMsgGambleNotFound             = "gamble not found"
	ErrMsgNotInJoiningState          = "not in joining state"
	ErrMsgJoinDeadlinePassed         = "join deadline has passed"
	ErrMsgAtLeastOneLootboxRequired  = "at least one lootbox bet is required"
	ErrMsgBetQuantityMustBePositive  = "bet quantity must be positive"
	ErrMsgFailedToTransitionState    = "failed to transition gamble state"
	ErrMsgFailedToSaveOpenedItems    = "failed to save opened items"
	ErrMsgNotALootbox                = "not a lootbox"

	// User service errors
	ErrMsgNotEnoughItems       = "not enough items"
	ErrMsgFailedToRegisterUser = "failed to register user"

	// Database/System errors
	ErrMsgConnectionTimeout = "connection timeout"
	ErrMsgDatabaseError     = "database error"
	ErrMsgDeadlockDetected  = "deadlock detected"
	ErrMsgFailedToGetUser   = "failed to get user"
	ErrMsgFailedToGetItem   = "failed to get item"
	ErrMsgFailedToGetInventory   = "failed to get inventory"
	ErrMsgFailedToUpdateInventory   = "failed to update inventory"
	ErrMsgFailedToBeginTx   = "failed to begin transaction"
	ErrMsgFailedToCommitTx   = "failed to commit transaction"
	ErrMsgFailedToRollbackTx   = "failed to rollback transaction"

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
	ErrNotInInventory       = errors.New(ErrMsgNotInInventory)

	// Economy errors
	ErrInsufficientFunds = errors.New(ErrMsgInsufficientFunds)
	ErrNotSellable       = errors.New(ErrMsgNotSellable)
	ErrNotBuyable        = errors.New(ErrMsgNotBuyable)

	// Gamble errors
	ErrGambleAlreadyActive       = errors.New(ErrMsgGambleAlreadyActive)
	ErrGambleNotFound            = errors.New(ErrMsgGambleNotFound)
	ErrNotInJoiningState         = errors.New(ErrMsgNotInJoiningState)
	ErrJoinDeadlinePassed        = errors.New(ErrMsgJoinDeadlinePassed)
	ErrAtLeastOneLootboxRequired = errors.New(ErrMsgAtLeastOneLootboxRequired)
	ErrBetQuantityMustBePositive = errors.New(ErrMsgBetQuantityMustBePositive)
	ErrNotALootbox               = errors.New(ErrMsgNotALootbox)

	// User service errors
	ErrNotEnoughItems       = errors.New(ErrMsgNotEnoughItems)
	ErrFailedToRegisterUser = errors.New(ErrMsgFailedToRegisterUser)

	// Database/System errors
	ErrConnectionTimeout = errors.New(ErrMsgConnectionTimeout)
	ErrDatabaseError     = errors.New(ErrMsgDatabaseError)
	ErrDeadlockDetected  = errors.New(ErrMsgDeadlockDetected)
	ErrFailedToGetUser   = errors.New(ErrMsgFailedToGetUser)
	ErrFailedToGetItem   = errors.New(ErrMsgFailedToGetItem)
	ErrFailedToGetInventory   = errors.New(ErrMsgFailedToGetInventory)
	ErrFailedToUpdateInventory   = errors.New(ErrMsgFailedToUpdateInventory)
	ErrFailedToBeginTx   = errors.New(ErrMsgFailedToBeginTx)
	ErrFailedToCommitTx   = errors.New(ErrMsgFailedToCommitTx)
	ErrFailedToRollbackTx   = errors.New(ErrMsgFailedToRollbackTx)

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
