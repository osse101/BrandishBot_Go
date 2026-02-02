package discord

// Friendly message constants for Discord responses
const (
	// Economy
	MsgInsufficientFunds = "⚠️ **Not Enough Gold!**\nYou don't have enough coins for this transaction."

	// Items & Inventory
	MsgItemNotFound   = "❓ **Item Not Found**\nMaybe check the spelling?"
	MsgInventoryFull  = "🎒 **Inventory Full**\nYou're carrying too much stuff!"
	MsgNotEnoughItems = "🎒 **Not Enough Items**\nYou don't have enough of that item."

	// User
	MsgUserNotFound = "👤 **User Not Found**\nHave they registered yet?"

	// Cooldowns
	MsgCooldownActive = "⏳ **Whoa there!**\nYou need to wait a bit before doing that again."
	MsgFeatureLocked  = "🔒 **Feature Locked**"

	MsgGenericError = "❌ Something went wrong."
)
