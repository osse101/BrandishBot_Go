package domain

import "errors"

// Common domain errors
// These errors should be used consistently across all layers of the application.
// Wrap these errors with fmt.Errorf("%w: %s", domain.ErrXxx, details) for additional context.
var (
	// User errors
	ErrUserNotFound = errors.New("user not found")
	
	// Item errors
	ErrItemNotFound = errors.New("item not found")
	
	// Inventory errors
	ErrInsufficientQuantity = errors.New("insufficient quantity")
	ErrInventoryFull        = errors.New("inventory is full")
	
	// Economy errors
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrNotSellable       = errors.New("item is not sellable")
	ErrNotBuyable        = errors.New("item is not buyable")
	
	// Cooldown errors
	ErrOnCooldown = errors.New("action on cooldown")
	
	// Feature errors
	ErrFeatureLocked = errors.New("feature is locked")
	
	// Recipe/Crafting errors
	ErrRecipeNotFound  = errors.New("recipe not found")
	ErrRecipeLocked    = errors.New("recipe is locked")
	ErrInvalidRecipe   = errors.New("invalid recipe")
	
	// Validation errors
	ErrInvalidInput = errors.New("invalid input")

	// ErrInvalidPlatform is returned when a platform is not supported.
	ErrInvalidPlatform = errors.New("invalid platform")
)
