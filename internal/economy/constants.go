package economy

// ==================== Error Messages ====================

// General error messages
const (
	ErrMsgUserNotFound      = "user not found"
	ErrMsgInsufficientFunds = "insufficient funds"
	ErrMsgMoneyItemNotFound = "money item not found"
)

// Formatted error messages for items
const (
	ErrMsgResolveItemFailedFmt         = "failed to resolve item name '%s': %w"
	ErrMsgItemNotFoundPublicFmt        = "%s (not found as public or internal name): %w"
	ErrMsgItemNotFoundFmt              = "item not found: %s: %w"
	ErrMsgItemNotInInventoryFmt        = "item %s not in inventory: %w"
	ErrMsgItemNotBuyableFmt            = "item %s is not buyable: %w"
	ErrMsgInsufficientFundsToBuyOneFmt = "insufficient funds to buy even one %s (cost: %d, balance: %d): %w"
)

// Formatted error messages for validation
const (
	ErrMsgInvalidQuantityFmt    = "invalid quantity: %d: %w"
	ErrMsgQuantityExceedsMaxFmt = "quantity %d exceeds maximum allowed (%d): %w"
)

// Database operation error messages
const (
	ErrMsgGetUserFailed           = "failed to get user: %w"
	ErrMsgGetItemFailed           = "failed to get item: %w"
	ErrMsgGetMoneyItemFailed      = "failed to get money item: %w"
	ErrMsgBeginTransactionFailed  = "failed to begin transaction: %w"
	ErrMsgGetInventoryFailed      = "failed to get inventory: %w"
	ErrMsgUpdateInventoryFailed   = "failed to update inventory: %w"
	ErrMsgCommitTransactionFailed = "failed to commit transaction: %w"
	ErrMsgCheckBuyableFailed      = "failed to check if item is buyable: %w"
)

// Shutdown error messages
const (
	ErrMsgShutdownTimedOut = "shutdown timed out: %w"
)

// ==================== Log Messages ====================

// Service operation log messages
const (
	LogMsgGetSellablePricesCalled = "GetSellablePrices called"
	LogMsgGetBuyablePricesCalled  = "GetBuyablePrices called"
	LogMsgSellItemCalled          = "SellItem called"
	LogMsgItemSold                = "Item sold"
	LogMsgBuyItemCalled           = "BuyItem called"
	LogMsgItemPurchased           = "Item purchased"
	LogMsgAdjustedPurchaseQty     = "Adjusted purchase quantity due to funds"
)

// Background task log messages
const (
	LogMsgEconomyShuttingDown = "Economy service shutting down, waiting for background tasks..."
	ErrMsgAwardMerchantXPFailed = "Failed to award Merchant XP: UserID %d: %w"
	LogMsgMerchantLeveledUp     = "Merchant leveled up!"
)

// ==================== Metadata Keys ====================

// Event metadata keys for stats and job XP tracking
const (
	MetadataKeyAction   = "action"
	MetadataKeyItemName = "item_name"
	MetadataKeyValue    = "value"
)

// ==================== Transaction Action Types ====================

// XP award source identifiers for economy actions
const (
	ActionTypeSell = "sell"
	ActionTypeBuy  = "buy"
)
