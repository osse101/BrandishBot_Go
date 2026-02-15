package domain

import "time"

// Item internal name constants - stable code identifiers
const (
	ItemMoney    = "money"          // currency_money in future
	ItemLootbox0 = "lootbox_tier0"  // was lootbox0
	ItemLootbox1 = "lootbox_tier1"  // was lootbox1
	ItemLootbox2 = "lootbox_tier2"  // was lootbox2
	ItemLootbox3 = "lootbox_tier3"  // diamondbox
	ItemBlaster  = "weapon_blaster" // was blaster

	// Weapon items
	ItemBigBlaster  = "weapon_bigblaster"  // bigmissile - 10 min timeout
	ItemHugeBlaster = "weapon_hugeblaster" // hugemissile - 100 min timeout
	ItemThis        = "weapon_this"        // meme weapon - 101s timeout
	ItemDeez        = "weapon_deez"        // meme weapon - 202s timeout
	ItemMissile     = "weapon_missile"     // missile - 60s timeout (Tier 1 progression)
	ItemGrenade     = "item_grenade"       // grenade - 60s random timeout (Tier 2 progression)

	// Revive items
	ItemReviveSmall  = "revive_small"  // revives - 60s recovery
	ItemReviveMedium = "revive_medium" // revivem - 10 min recovery
	ItemReviveLarge  = "revive_large"  // revivel - 100 min recovery

	// Explosive items
	ItemMine = "explosive_mine" // mine - basic trap
	ItemTrap = "explosive_trap" // trap - upgraded trap
	ItemTNT  = "explosive_tnt"  // tnt - ultimate trap
	ItemBomb = "explosive_bomb" // large explosive

	// Utility items
	ItemStick        = "item_stick"        // basic crafting material
	ItemShield       = "item_shield"       // blocks weapon attacks
	ItemMirrorShield = "weapon_mirror"     // mirror shield - reflects attacks (Tier 4 progression)
	ItemShovel       = "item_shovel"       // shovel - generates sticks (Tier 2 progression)
	ItemVideoFilter  = "item_video_filter" // video filter - requires Streamer.bot (Tier 1 progression)
	ItemScrap        = "item_scrap"        // scrap - crafting material (Tier 2 progression)
	ItemScript       = "item_script"       // script - currency (Tier 2 progression)

	// Progression items
	ItemRareCandy = "xp_rarecandy" // instant job XP

	// Junk items
	ItemSludge = "compost_sludge" // compost byproduct
)

// Public item name constants - what clients use in commands (PublicName field)
const (
	PublicNameMoney      = "money"      // Currency
	PublicNameJunkbox    = "junkbox"    // Tier 0 lootbox (rusty)
	PublicNameLootbox    = "lootbox"    // Tier 1 lootbox (basic)
	PublicNameGoldbox    = "goldbox"    // Tier 2 lootbox (golden)
	PublicNameDiamondbox = "diamondbox" // Tier 3 lootbox (diamond)
	PublicNameMissile    = "missile"    // Blaster weapon

	// Weapon public names
	PublicNameBigMissile  = "bigmissile"  // Big blaster
	PublicNameHugeMissile = "hugemissile" // Huge blaster
	PublicNameThis        = "this"        // Meme weapon
	PublicNameDeez        = "deez"        // Meme weapon upgrade
	PublicNameGrenade     = "grenade"     // Grenade

	// Revive public names
	PublicNameReviveS = "revives" // Small revive
	PublicNameReviveM = "revivem" // Medium revive
	PublicNameReviveL = "revivel" // Large revive

	// Explosive public names
	PublicNameMine = "mine" // Basic explosive
	PublicNameTrap = "trap" // Upgraded explosive
	PublicNameTNT  = "tnt"  // Ultimate explosive

	// Utility public names
	PublicNameStick        = "stick"        // Basic material
	PublicNameShield       = "shield"       // Defensive item
	PublicNameMirrorShield = "mirrorshield" // Mirror shield
	PublicNameShovel       = "shovel"       // Shovel
	PublicNameVideoFilter  = "filter"       // Video filter

	// Progression public names
	PublicNameRareCandy = "rarecandy" // XP item
)

// Action name constants for cooldown tracking
const (
	ActionSearch = "search"
	ActionSlots  = "slots"
	// Future actions can be added here
	// ActionDaily  = "daily"
	// ActionQuest  = "quest"
)

// Duration constants for cooldowns and timing
const (
	SearchCooldownDuration = 30 * time.Minute
	SlotsCooldownDuration  = 10 * time.Minute
	// Future durations can be added here
	// DailyCooldownDuration  = 24 * time.Hour
)

// Platform constants
const (
	PlatformTwitch  = "twitch"
	PlatformYoutube = "youtube"
	PlatformDiscord = "discord"
	DiscordBotID    = "BrandishBot#6125"
)

// VotingStatus constants
const (
	VotingStatusVoting    = "voting"
	VotingStatusFrozen    = "frozen"
	VotingStatusCompleted = "completed"
)

// Message constants
const (
	MsgSearchNothingFound    = "You have found nothing"
	MsgSearchNearMiss        = "You found nothing... but you saw something glint in the distance!"
	MsgSearchCriticalSuccess = "You found a hidden stash!"
	MsgSearchCriticalFail    = "You tried to search, but disaster struck!"
	MsgFirstSearchBonus      = " (First Search of the Day!)"
	MsgStreakBonus           = " (🔥 %d Day Streak!)"
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

// Inventory filter type constants
const (
	FilterTypeUpgrade    = "upgrade"
	FilterTypeSellable   = "sellable"
	FilterTypeConsumable = "consumable"
)

// Economy constants
const (
	MaxTransactionQuantity = 10000
)

// Shared metadata keys used across multiple modules for event payloads
// These keys ensure consistency when publishing and consuming events
const (
	// MetadataKeyItemName is used to store item names in event metadata
	MetadataKeyItemName = "item_name"

	// MetadataKeyQuantity is used to store quantities in event metadata
	MetadataKeyQuantity = "quantity"

	// MetadataKeyMultiplier is used to store multiplier values in event metadata
	MetadataKeyMultiplier = "multiplier"

	// MetadataKeySource is used to store the source/origin in event metadata
	MetadataKeySource = "source"
)

// ============================================================================
// Item Quality Constants (Moved from item.go)
// ============================================================================

// QualityLevel represents the visual rarity and quality of an item
type QualityLevel string

const (
	QualityCommon    QualityLevel = "COMMON"
	QualityUncommon  QualityLevel = "UNCOMMON"
	QualityRare      QualityLevel = "RARE"
	QualityEpic      QualityLevel = "EPIC"
	QualityLegendary QualityLevel = "LEGENDARY"
	QualityPoor      QualityLevel = "POOR"
	QualityJunk      QualityLevel = "JUNK"
	QualityCursed    QualityLevel = "CURSED"
)

// GetTimeoutAdjustment returns the timeout adjustment in seconds based on quality level
// Distance from common * 10s
func (s QualityLevel) GetTimeoutAdjustment() time.Duration {
	qualityModifier := map[QualityLevel]time.Duration{
		QualityCursed:    -30 * time.Second,
		QualityJunk:      -20 * time.Second,
		QualityPoor:      -10 * time.Second,
		QualityCommon:    0 * time.Second,
		QualityUncommon:  10 * time.Second,
		QualityRare:      20 * time.Second,
		QualityEpic:      30 * time.Second,
		QualityLegendary: 40 * time.Second,
	}

	if modifier, ok := qualityModifier[s]; ok {
		return modifier
	}
	return 0
}

// Quality multipliers (Boosts item value and Gamble Score)
const (
	MultCommon    = 1.0
	MultUncommon  = 1.1
	MultRare      = 1.25
	MultEpic      = 1.5
	MultLegendary = 2.0
	MultPoor      = 0.8
	MultJunk      = 0.6
	MultCursed    = 0.4
)

// ============================================================================
// Event Type Constants (Moved from events.go)
// ============================================================================

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

	// EventTypePredictionProcessed is published when a prediction outcome is processed
	EventTypePredictionProcessed = "prediction.processed"

	// Quest events
	EventTypeWeeklyQuestReset     = "quest.weekly_reset"
	EventTypeQuestProgressUpdated = "quest.progress_updated"
	EventTypeQuestCompleted       = "quest.completed"
	EventTypeQuestClaimed         = "quest.claimed"

	// Economy events
	EventTypeWeeklySaleActive = "economy.weekly_sale_active"

	// Slots events
	EventSlotsCompleted = "slots.completed"

	// Gamble events (new)
	EventTypeGambleParticipated = "gamble.participated"

	// Harvest/Compost events
	EventTypeHarvestCompleted = "harvest.completed"
	EventTypeCompostHarvested = "compost.harvested"

	// Expedition events
	EventTypeExpeditionRewarded = "expedition.rewarded"

	// Prediction events
	EventTypePredictionParticipated = "prediction.participated"

	// Job XP critical (Epiphany bonus)
	EventTypeJobXPCritical = "job.xp_critical"
)

// ============================================================================
// Gamble Constants (Moved from gamble.go)
// ============================================================================

// GambleState represents the current state of a gamble
type GambleState string

const (
	GambleStateCreated   GambleState = "Created"
	GambleStateJoining   GambleState = "Joining"
	GambleStateOpening   GambleState = "Opening"
	GambleStateCompleted GambleState = "Completed"
	GambleStateRefunded  GambleState = "Refunded"
)

// Event types for Gamble
const (
	EventGambleStarted   = "GambleStarted"
	EventGambleCompleted = "GambleCompleted"
)

// ============================================================================
// Quest Constants (Moved from quest.go)
// ============================================================================

// Quest type constants
const (
	QuestTypeBuyItems        = "buy_items"        // Buy X items of target category
	QuestTypeSellItems       = "sell_items"       // Sell X items
	QuestTypeEarnMoney       = "earn_money"       // Earn X money from sales
	QuestTypeCraftRecipe     = "craft_recipe"     // Perform recipe (upgrade/disassemble) X times
	QuestTypePerformSearches = "perform_searches" // Perform X searches
	// Extensible: add new quest types as needed
)

// ============================================================================
// Item Constants (Moved from item.go)
// ============================================================================

// Item tag constants (from item_types / tags in items.json)
const (
	CompostableTag = "compostable"
	NoUseTag       = "no-use"
)

// Content type constants (from "type" field in items.json)
const (
	ContentTypeWeapon    = "weapon"
	ContentTypeExplosive = "explosive"
	ContentTypeDefense   = "defense"
	ContentTypeHealing   = "healing"
	ContentTypeMaterial  = "material"
	ContentTypeContainer = "container"
	ContentTypeUtility   = "utility"
	ContentTypeMagical   = "magical"
)

// ============================================================================
// Subscription Constants (Moved from subscription.go)
// ============================================================================

// Subscription status constants
const (
	SubscriptionStatusActive    = "active"
	SubscriptionStatusExpired   = "expired"
	SubscriptionStatusCancelled = "cancelled"
)

// Subscription event types (for event bus)
const (
	EventTypeSubscriptionActivated  = "subscription.activated"
	EventTypeSubscriptionRenewed    = "subscription.renewed"
	EventTypeSubscriptionUpgraded   = "subscription.upgraded"
	EventTypeSubscriptionDowngraded = "subscription.downgraded"
	EventTypeSubscriptionExpired    = "subscription.expired"
	EventTypeSubscriptionCancelled  = "subscription.cancelled"
)

// History event types
const (
	HistoryEventTypeSubscribed = "subscribed"
	HistoryEventTypeRenewed    = "renewed"
	HistoryEventTypeUpgraded   = "upgraded"
	HistoryEventTypeDowngraded = "downgraded"
	HistoryEventTypeCancelled  = "cancelled"
	HistoryEventTypeExpired    = "expired"
)

// Configuration constants
const (
	DefaultSubscriptionDuration = 30 * 24 * time.Hour // 30 days
	SubscriptionGracePeriod     = 24 * time.Hour      // 24-hour grace before expiration
)

// ============================================================================
// Expedition Constants (Moved from expedition.go)
// ============================================================================

// ExpeditionState represents the state of an expedition
type ExpeditionState string

const (
	ExpeditionStateCreated    ExpeditionState = "Created"
	ExpeditionStateRecruiting ExpeditionState = "Recruiting"
	ExpeditionStateInProgress ExpeditionState = "InProgress"
	ExpeditionStateCompleted  ExpeditionState = "Completed"
)

// Expedition event constants
const (
	EventExpeditionStarted   = "ExpeditionStarted"
	EventExpeditionCompleted = "ExpeditionCompleted"
	EventExpeditionTurn      = "ExpeditionTurn"
)

// EncounterType represents the type of encounter in an expedition
type EncounterType string

const (
	EncounterExplore       EncounterType = "explore"
	EncounterTravel        EncounterType = "travel"
	EncounterCombatSkirmsh EncounterType = "combat_skirmish"
	EncounterCombatElite   EncounterType = "combat_elite"
	EncounterCombatBoss    EncounterType = "combat_boss"
	EncounterCamp          EncounterType = "camp"
	EncounterHazard        EncounterType = "hazard"
	EncounterDiscovery     EncounterType = "discovery"
	EncounterEncounter     EncounterType = "encounter"
	EncounterTreasure      EncounterType = "treasure"
	EncounterMystic        EncounterType = "mystic"
	EncounterDrama         EncounterType = "drama"
)

// OutcomeType represents the outcome category of an encounter
type OutcomeType string

const (
	OutcomePositive OutcomeType = "positive"
	OutcomeNeutral  OutcomeType = "neutral"
	OutcomeNegative OutcomeType = "negative"
)

// ExpeditionSkill represents a skill used in expeditions, mapped 1:1 to jobs
type ExpeditionSkill string

const (
	SkillFortitude  ExpeditionSkill = "fortitude"
	SkillPerception ExpeditionSkill = "perception"
	SkillSurvival   ExpeditionSkill = "survival"
	SkillCunning    ExpeditionSkill = "cunning"
	SkillPersuasion ExpeditionSkill = "persuasion"
	SkillKnowledge  ExpeditionSkill = "knowledge"
)

// ============================================================================
// Stats Constants (Moved from stats.go)
// ============================================================================

// EventType represents the type of event being tracked
type EventType string

const (
	StatsEventUserRegistered  EventType = "user_registered"
	StatsEventItemAdded       EventType = "item_added"
	StatsEventItemRemoved     EventType = "item_removed"
	StatsEventItemUsed        EventType = "item_used"
	StatsEventItemSold        EventType = "item_sold"
	StatsEventItemBought      EventType = "item_bought"
	StatsEventItemTransferred EventType = "item_transferred"
	StatsEventMessageReceived EventType = "message_received"

	// Gamble events
	StatsEventGambleNearMiss     EventType = "gamble_near_miss"
	StatsEventGambleTieBreakLost EventType = "gamble_tie_break_lost"
	StatsEventGambleCriticalFail EventType = "gamble_critical_fail"
	StatsEventDailyStreak        EventType = "daily_streak"

	// Search events
	StatsEventSearch                EventType = "search"
	StatsEventSearchNearMiss        EventType = "search_near_miss"
	StatsEventSearchCriticalFail    EventType = "search_critical_fail"
	StatsEventSearchCriticalSuccess EventType = "search_critical_success"

	// Crafting events
	EventTypeCraftingCriticalSuccess EventType = "crafting_critical_success"
	EventTypeCraftingPerfectSalvage  EventType = "crafting_perfect_salvage"

	// Job events
	EventTypeJobLevelUp EventType = "job_level_up"

	// Lootbox events
	EventTypeLootboxJackpot EventType = "lootbox_jackpot"
	EventTypeLootboxBigWin  EventType = "lootbox_big_win"

	// Slots events
	EventTypeSlotsSpin        EventType = "slots_spin"
	EventTypeSlotsWin         EventType = "slots_win"
	EventTypeSlotsMegaJackpot EventType = "slots_mega_jackpot"
)
