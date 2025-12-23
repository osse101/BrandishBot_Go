package handler

// Generic HTTP error messages for client responses.
// These messages intentionally do not expose internal error details for security reasons.
// Both handlers and tests should reference these constants to maintain consistency.
const (
	// Inventory operation error messages
	ErrMsgAddItemFailed      = "Failed to add item"
	ErrMsgRemoveItemFailed   = "Failed to remove item"
	ErrMsgGiveItemFailed     = "Failed to give item"
	ErrMsgGetInventoryFailed = "Failed to get inventory"

	// Economy operation error messages
	ErrMsgSellItemFailed = "Failed to sell item"
	ErrMsgBuyItemFailed  = "Failed to buy item"

	// Item usage error messages
	ErrMsgUseItemFailed = "Failed to use item"

	// Feature/progression error messages
	ErrMsgFeatureCheckFailed = "Failed to check feature availability"

	// Crafting/upgrade error messages
	ErrMsgDisassembleItemFailed = "Failed to disassemble item"
	ErrMsgUpgradeItemFailed     = "Failed to upgrade item"

	// Search error messages
	ErrMsgSearchFailed = "Failed to perform search"

	// User management error messages
	ErrMsgRegisterUserFailed = "Failed to register user"

	// Message handling error messages
	ErrMsgHandleMessageFailed = "Failed to handle message"
)
