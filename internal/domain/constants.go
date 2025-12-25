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

// Public item name constants - what clients use in commands (PublicName field)
const (
	PublicNameMoney   = "money"   // Currency
	PublicNameJunkbox = "junkbox" // Tier 0 lootbox (rusty)
	PublicNameLootbox = "lootbox" // Tier 1 lootbox (basic)
	PublicNameGoldbox = "goldbox" // Tier 2 lootbox (golden)
	PublicNameMissile = "missile" // Blaster weapon
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
	MsgSearchCriticalFail    = "You tried to search, but disaster struck! (CRITICAL FAIL!)"
	MsgFirstSearchBonus      = " (First Search of the Day!)"
)

// SearchCriticalFailMessages is a list of funny messages for critical failures
var SearchCriticalFailMessages = []string{
	"You found a bee hive. They found you.",
	"You fell into a hole. It's dark down here.",
	"A mimic bit your hand! Ouch!",
	"You dropped your wallet while searching. Now you have less than nothing.",
	"You found a cursed amulet that smells like wet dog.",
	"You searched so hard you pulled a muscle.",
	"A bird pooped on your head. Unlucky.",
	"You tripped and fell face-first into the mud.",
	"You disturbed a sleeping bear. Run!",
	"You found a trap! ...With your foot.",
}

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
