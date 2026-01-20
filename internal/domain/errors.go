package domain

import "errors"

// Error message string constants - single source of truth for error messages
// Use these in assert.Contains() checks when testing error messages
const (
	// User errors
	ErrMsgUserNotFound = "user not found"

	// Item errors
	ErrMsgItemNotFound = "item not found"
	ErrMsgItemNotHandled = "item not handled"

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
	ErrMsgInvalidPlatform = "invalid platform"
	ErrMsgInvalidInput = "invalid input"

	// Gamble errors
	ErrMsgGambleAlreadyActive       = "a gamble is already active"
	ErrMsgGambleNotFound            = "gamble not found"
	ErrMsgNotInJoiningState         = "not in joining state"
	ErrMsgJoinDeadlinePassed        = "join deadline has passed"
	ErrMsgAtLeastOneLootboxRequired = "at least one lootbox bet is required"
	ErrMsgBetQuantityMustBePositive = "bet quantity must be positive"
	ErrMsgFailedToTransitionState   = "failed to transition gamble state"
	ErrMsgFailedToSaveOpenedItems   = "failed to save opened items"
	ErrMsgNotALootbox               = "not a lootbox"
	ErrMsgUserAlreadyJoined         = "user has already joined this gamble"

	// User service errors
	ErrMsgNotEnoughItems       = "not enough items"
	ErrMsgFailedToRegisterUser = "failed to register user"

	// Job errors
	ErrMsgDailyCapReached = "daily XP cap reached"

	// Database/System errors
	ErrMsgConnectionTimeout       = "connection timeout"
	ErrMsgDatabaseError           = "database error"
	ErrMsgDeadlockDetected        = "deadlock detected"
	ErrMsgFailedToGetUser         = "failed to get user"
	ErrMsgFailedToGetItem         = "failed to get item"
	ErrMsgFailedToGetItemDetails  = "failed to get item details"
	ErrMsgFailedToGetInventory    = "failed to get inventory"
	ErrMsgFailedToUpdateInventory = "failed to update inventory"
	ErrMsgFailedToBeginTx         = "failed to begin transaction"
	ErrMsgFailedToCommitTx        = "failed to commit transaction"
	ErrMsgFailedToRollbackTx      = "failed to rollback transaction"

	// Cooldown errors
	ErrMsgOnCooldown = "action on cooldown"

	// Feature errors
	ErrMsgFeatureLocked = "feature is locked"

	// Progression errors
	ErrMsgUserAlreadyVoted = "user has already voted"

	// Recipe/Crafting errors
	ErrMsgRecipeNotFound = "recipe not found"
	ErrMsgRecipeLocked   = "recipe is locked"
	ErrMsgInvalidRecipe  = "invalid recipe"

)

// Common domain errors
// These errors should be used consistently across all layers of the application.
// Wrap these errors with fmt.Errorf("%w: %s", domain.ErrXxx, details) for additional context.
var (
	// User errors
	ErrUserNotFound = errors.New(ErrMsgUserNotFound)

	// Item errors
	ErrItemNotFound = errors.New(ErrMsgItemNotFound)
	ErrItemNotHandled = errors.New(ErrMsgItemNotHandled)

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
	ErrUserAlreadyJoined         = errors.New(ErrMsgUserAlreadyJoined)

	// User service errors
	ErrNotEnoughItems       = errors.New(ErrMsgNotEnoughItems)
	ErrFailedToRegisterUser = errors.New(ErrMsgFailedToRegisterUser)

	// Job errors
	ErrDailyCapReached = errors.New(ErrMsgDailyCapReached)

	// Database/System errors
	ErrConnectionTimeout       = errors.New(ErrMsgConnectionTimeout)
	ErrDatabaseError           = errors.New(ErrMsgDatabaseError)
	ErrDeadlockDetected        = errors.New(ErrMsgDeadlockDetected)
	ErrFailedToGetUser         = errors.New(ErrMsgFailedToGetUser)
	ErrFailedToGetItem         = errors.New(ErrMsgFailedToGetItem)
	ErrFailedToGetItemDetails  = errors.New(ErrMsgFailedToGetItemDetails)
	ErrFailedToGetInventory    = errors.New(ErrMsgFailedToGetInventory)
	ErrFailedToUpdateInventory = errors.New(ErrMsgFailedToUpdateInventory)
	ErrFailedToBeginTx         = errors.New(ErrMsgFailedToBeginTx)
	ErrFailedToCommitTx        = errors.New(ErrMsgFailedToCommitTx)
	ErrFailedToRollbackTx      = errors.New(ErrMsgFailedToRollbackTx)

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
	ErrInvalidQuantity = errors.New(ErrMsgInvalidQuantity)
	ErrInvalidPlatform = errors.New(ErrMsgInvalidPlatform)

	// Progression errors
	ErrUserAlreadyVoted = errors.New(ErrMsgUserAlreadyVoted)
)
