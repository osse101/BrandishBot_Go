package domain

// Event type constants used across the application for event bus subscriptions
// and metrics tracking. These represent domain events that can be published
// and consumed by multiple modules.
//
// Event types follow the pattern: <entity>.<action> (e.g., "item.sold")
const (
	// EventTypeItemSold is published when an item is sold through the economy system
	EventTypeItemSold = "item.sold"

	// EventTypeItemBought is published when an item is bought through the economy system
	EventTypeItemBought = "item.bought"

	// EventTypeItemUpgraded is published when an item is upgraded through crafting
	EventTypeItemUpgraded = "item.upgraded"

	// EventTypeItemDisassembled is published when an item is disassembled through crafting
	EventTypeItemDisassembled = "item.disassembled"

	// EventTypeItemUsed is published when a consumable item is used
	EventTypeItemUsed = "item.used"

	// EventTypeSearchPerformed is published when a user performs a search action
	EventTypeSearchPerformed = "search.performed"

	// EventTypeEngagement is published when a user interaction occurs (commands, messages, etc.)
	EventTypeEngagement = "engagement"

	// EventTypeDailyResetComplete is published when the daily job XP reset completes
	EventTypeDailyResetComplete = "daily_reset.complete"

	// EventTrapPlaced is published when a trap is placed on a user
	EventTrapPlaced = "trap.placed"

	// EventTrapTriggered is published when a trap is triggered
	EventTrapTriggered = "trap.triggered"

	// EventTrapSelfTriggered is published when a user triggers their own trap
	EventTrapSelfTriggered = "trap.self_triggered"

	// EventTypeTimeoutApplied is published when a timeout is applied to a user
	EventTypeTimeoutApplied = "timeout.applied"

	// EventTypeTimeoutCleared is published when a timeout is cleared for a user
	EventTypeTimeoutCleared = "timeout.cleared"
)
