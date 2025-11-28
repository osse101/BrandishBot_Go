package domain

import "time"

// Item name constants - centralized item identifiers
const (
	ItemMoney    = "money"
	ItemLootbox0 = "lootbox0"
	ItemLootbox1 = "lootbox1"
	ItemLootbox2 = "lootbox2"
	ItemBlaster  = "blaster"
)

// Action name constants for cooldown tracking
const (
	ActionSearch = "search"
	// Future actions can be added here
	// ActionDaily  = "daily"
	// ActionQuest  = "quest"
)

// Duration constants for cooldowns and timing
const (
	SearchCooldownDuration = 30 * time.Minute
	// Future durations can be added here
	// DailyCooldownDuration  = 24 * time.Hour
)

// Platform constants
const (
	PlatformTwitch  = "twitch"
	PlatformYoutube = "youtube"
	PlatformDiscord = "discord"
)

// Message constants
const (
	MsgSearchNothingFound = "You have found nothing"
)

// Period constants
const (
	PeriodDaily = "daily"
)

// Error messages
const (
	ErrMsgTxClosed = "tx is closed"
)

// Economy constants
const (
	MaxTransactionQuantity = 10000
)
