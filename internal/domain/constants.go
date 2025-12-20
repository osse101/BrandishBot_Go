package domain

import "time"

// Item internal name constants - stable code identifiers
const (
	ItemMoney    = "money"           // currency_money in future
	ItemLootbox0 = "lootbox_tier0"   // was lootbox0
	ItemLootbox1 = "lootbox_tier1"   // was lootbox1
	ItemLootbox2 = "lootbox_tier2"   // was lootbox2
	ItemBlaster  = "weapon_blaster"  // was blaster
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
	MsgSearchNothingFound    = "You have found nothing"
	MsgSearchNearMiss        = "You found nothing... but you saw something glint in the distance!"
	MsgSearchCriticalSuccess = "You found a hidden stash! (CRITICAL SUCCESS!)"
	MsgFirstSearchBonus      = " (First Search of the Day!)"
)

// SearchFailureMessages is a list of funny messages for failed searches
var SearchFailureMessages = []string{
	MsgSearchNothingFound,
	"You found a rock. It's just a rock.",
	"You tripped over a root and found nothing.",
	"You searched high and low, but mostly low, and found dust.",
	"A goblin stole the loot before you got there.",
	"You found a shiny coin! ...Wait, it's a chocolate wrapper.",
	"Nothing here but cobwebs.",
	"You found a 'IOU' note from a previous adventurer.",
}

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
