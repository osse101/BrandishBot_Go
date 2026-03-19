package domain

import "time"

// Item internal name constants - stable code identifiers
const (
	ItemMoney    = "money"          // currency_money in future
	ItemLootbox0 = "lootbox_tier0"  // was lootbox0
	ItemLootbox1 = "lootbox_tier1"  // was lootbox1
	ItemLootbox2 = "lootbox_tier2"  // was lootbox2
	ItemLootbox3 = "lootbox_tier3"  // diamondbox
	ItemMissile  = "weapon_missile" // was blaster

	// Weapon items
	ItemBigMissile  = "weapon_bigmissile"  // bigmissile - 10 min timeout
	ItemHugeMissile = "weapon_hugemissile" // hugemissile - 100 min timeout
	ItemThis        = "weapon_this"        // meme weapon - 101s timeout
	ItemDeez        = "weapon_deez"        // meme weapon - 202s timeout
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
	MetadataKeyItemName   = "item_name"
	MetadataKeyQuantity   = "quantity"
	MetadataKeyMultiplier = "multiplier"
	MetadataKeySource     = "source"
	MetadataKeyRecorded   = "recorded"
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
// and metrics tracking. These represent events that can be published
// and consumed by multiple modules.
//
// Event types follow the pattern: <entity>.<action> (e.g., "item.sold")
const (
	// EventTypeItemSold is published when an item is sold through the economy system
	EventTypeItemSold = "item.sold"

	// EventTypeItemBought is published when an item is bought through the economy system
	EventTypeItemBought = "item.bought"

	// EventTypeItemAdded is published when an item is added to a user's inventory
	EventTypeItemAdded = "item.added"

	// EventTypeItemRemoved is published when an item is removed from a user's inventory
	EventTypeItemRemoved = "item.removed"

	// EventTypeItemTransferred is published when an item is transferred between users
	EventTypeItemTransferred = "item.transferred"

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

// ============================================================================
// Compost Constants (Moved from compost.go)
// ============================================================================

// CompostBinStatus represents the state of a compost bin
type CompostBinStatus string

const (
	CompostBinStatusIdle       CompostBinStatus = "idle"
	CompostBinStatusComposting CompostBinStatus = "composting"
	CompostBinStatusReady      CompostBinStatus = "ready"
	CompostBinStatusSludge     CompostBinStatus = "sludge"
)

// ============================================================================
// Duel Constants (Moved from duel.go)
// ============================================================================

// DuelState represents the state of a duel
type DuelState string

const (
	DuelStatePending    DuelState = "pending"
	DuelStateAccepted   DuelState = "accepted"
	DuelStateInProgress DuelState = "in_progress"
	DuelStateCompleted  DuelState = "completed"
	DuelStateDeclined   DuelState = "declined"
	DuelStateExpired    DuelState = "expired"
)

// ============================================================================
// Job Constants (Moved from job.go)
// ============================================================================

// Job keys for referencing specific jobs across various services
const (
	JobKeyBlacksmith = "job_blacksmith"
	JobKeyExplorer   = "job_explorer"
	JobKeyMerchant   = "job_merchant"
	JobKeyGambler    = "job_gambler"
	JobKeyFarmer     = "job_farmer"
	JobKeyScholar    = "job_scholar"
)

// ============================================================================
// Search Mechanic Constants (Moved from internal/user/constants.go)
// ============================================================================

// Search Mechanic - Probability Thresholds
const (
	// SearchSuccessRate defines the probability of finding an item when searching (80%)
	SearchSuccessRate = 0.8
	// SearchCriticalRate defines the probability of a critical success when searching (5%)
	SearchCriticalRate = 0.05
	// SearchNearMissRate defines the probability of a near-miss result when searching (5%)
	SearchNearMissRate = 0.05
	// SearchCriticalFailRate defines the probability of a critical failure when searching (5%)
	SearchCriticalFailRate = 0.05
)

// Search Mechanic - Diminishing Returns
const (
	// SearchDailyDiminishmentThreshold is the number of searches per day after which returns are diminished
	SearchDailyDiminishmentThreshold = 6
	// SearchDiminishedXPMultiplier is the XP multiplier when diminished returns are active (10%)
	SearchDiminishedXPMultiplier = 0.1
	// SearchFirstDailyGuaranteedRoll is the roll value that guarantees success for first search (0.0)
	SearchFirstDailyGuaranteedRoll = 0.0
)

// Search Mechanic - Region Constants
const (
	// SearchRegionItemDropChance is the probability of getting a region item instead of a lootbox (50%)
	SearchRegionItemDropChance = 0.5
	// SearchRegionConfigPath is the default path to the search regions config file
	SearchRegionConfigPath = "configs/search_regions.json"
)

// ============================================================================
// Item Handler Constants (Moved from internal/user/constants.go)
// ============================================================================

const (
	// BulkFeedbackThreshold defines the number of lootboxes required to trigger "Nice haul" message
	BulkFeedbackThreshold = 5
	// BlasterTimeoutDuration is the default duration a user is timed out when hit by a blaster
	BlasterTimeoutDuration = 60 * time.Second
	// TrapCooldownDuration is the cooldown after a trap triggers to prevent immediate re-trapping
	TrapCooldownDuration = 10 * time.Minute
)

// ============================================================================
// Resource Generation Constants (Moved from internal/user/constants.go)
// ============================================================================

const (
	// ShovelSticksPerUse defines how many sticks are generated per shovel use
	ShovelSticksPerUse = 2
)

// ============================================================================
// Inventory Limits (Moved from internal/user/constants.go)
// ============================================================================

const (
	// MaxStackSize is the maximum quantity allowed for a single item stack when merging inventories
	MaxStackSize = 999999
)

// ============================================================================
// Cache Configuration
// ============================================================================

// CacheSchemaVersion is the current version of the cache schema
// Increment this when the cached data structure changes to auto-invalidate old entries
const CacheSchemaVersion = "1.0"

// DefaultCacheSize is the default maximum number of cache entries
const DefaultCacheSize = 1000

// DefaultCacheTTL is the default time-to-live for cache entries
const DefaultCacheTTL = 5 * time.Minute

// ============================================================================
// Environment Variable Keys
// ============================================================================

// EnvUserCacheSize is the environment variable key for cache size configuration
const EnvUserCacheSize = "USER_CACHE_SIZE"

// EnvUserCacheTTL is the environment variable key for cache TTL configuration
const EnvUserCacheTTL = "USER_CACHE_TTL"

// ============================================================================
// Endpoint Handler Arguments
// ============================================================================

// Handler argument keys
const ArgsUsername = "username"
const ArgsTargetUsername = "target_username"
const ArgsJobName = "job_name"
const ArgsPlatform = "platform"

// ============================================================================
// Validation Error Messages
// ============================================================================

// ErrMsgQuantityMustBePositive is returned when quantity is zero or negative
const ErrMsgQuantityMustBePositive = "quantity must be positive"

// ErrMsgUsernameRequired is returned when username is empty
const ErrMsgUsernameRequired = "username is required"

// ErrMsgPlatformIDRequired is returned when platform ID is empty
const ErrMsgPlatformIDRequired = "platformID is required"

// ErrMsgUsernameCannotBeEmpty is returned when username is empty in search
const ErrMsgUsernameCannotBeEmpty = "username cannot be empty"

// ============================================================================
// Item Handler Error Messages
// ============================================================================

// ErrMsgItemNotFoundInInventory is returned when an item is not in the user's inventory
const ErrMsgItemNotFoundInInventory = "item not found in inventory"

// ErrMsgNotEnoughItemsInInventory is returned when user doesn't have enough of an item
const ErrMsgNotEnoughItemsInInventory = "not enough items in inventory"

// ErrMsgTargetUsernameRequired is returned when target username is missing for targeted items
const ErrMsgTargetUsernameRequired = "target username is required for weapon"

// ErrMsgTargetUsernameRequiredRevive is returned when target username is missing for revive
const ErrMsgTargetUsernameRequiredRevive = "target username is required for revive"

// ErrMsgJobNameRequired is returned when job name is missing for rare candy
const ErrMsgJobNameRequired = "job name is required for rare candy"

// ErrMsgFailedToApplyShield is returned when shield application fails
const ErrMsgFailedToApplyShield = "failed to apply shield"

// ErrMsgFailedToAwardXP is returned when XP award fails
const ErrMsgFailedToAwardXP = "failed to award XP"

// ErrMsgFilterServiceUnavailable is returned when video filter service is unavailable
const ErrMsgFilterServiceUnavailable = "video filter service is unavailable"

// ErrMsgNoActiveTargets is returned when no active users are available for targeting
const ErrMsgNoActiveTargets = "no active users to target"

// ============================================================================
// Log Messages - Operations
// ============================================================================

const (
	LogMsgRegisterUserCalled           = "RegisterUser called"
	LogMsgFindUserByPlatformIDCalled   = "FindUserByPlatformID called"
	LogMsgHandleIncomingMessageCalled  = "HandleIncomingMessage called"
	LogMsgAddItemCalled                = "AddItem called"
	LogMsgAddItemByUsernameCalled      = "AddItemByUsername called"
	LogMsgAddItemsCalled               = "AddItems called"
	LogMsgRemoveItemCalled             = "RemoveItem called"
	LogMsgRemoveItemByUsernameCalled   = "RemoveItemByUsername called"
	LogMsgUseItemCalled                = "UseItem called"
	LogMsgUseItemByUsernameCalled      = "UseItemByUsername called"
	LogMsgGetInventoryCalled           = "GetInventory called"
	LogMsgGetInventoryByUsernameCalled = "GetInventoryByUsername called"
	LogMsgTimeoutUserCalled            = "TimeoutUser called"
	LogMsgHandleSearchCalled           = "HandleSearch called"
	LogMsgGiveItemCalled               = "GiveItem called"
	LogMsgGiveItemByUsernameCalled     = "GiveItemByUsername called"
	LogMsgHandleBlasterCalled          = "handleBlaster called"
	LogMsgHandleWeaponCalled           = "handleWeapon called"
	LogMsgHandleReviveCalled           = "handleRevive called"
	LogMsgHandleShieldCalled           = "handleShield called"
	LogMsgHandleRareCandyCalled        = "handleRareCandy called"
	LogMsgHandleTrapCalled             = "handleTrap called"
	LogMsgResourceGeneratorCalled      = "ResourceGeneratorHandler called"
	LogMsgUtilityCalled                = "UtilityHandler called"
)

// ============================================================================
// Log Messages - Results & Events
// ============================================================================

const (
	LogMsgUserRegistered             = "User registered"
	LogMsgUserFound                  = "User found"
	LogMsgItemAddedSuccessfully      = "Item added successfully"
	LogMsgItemAddedSuccessByUsername = "Item added successfully by username"
	LogMsgItemsAddedSuccessfully     = "Items added successfully"
	LogMsgItemRemoved                = "Item removed"
	LogMsgItemRemovedByUsername      = "Item removed by username"
	LogMsgItemUsed                   = "Item used"
	LogMsgItemUsedByUsername         = "Item used by username"
	LogMsgItemTransferred            = "Item transferred"
	LogMsgItemGivenByUsername        = "Item given by username"
	LogMsgExistingTimeoutCancelled   = "Existing timeout cancelled"
	LogMsgUserTimedOut               = "User timed out"
	LogMsgUserTimeoutExpired         = "User timeout expired"
	LogMsgSearchCompleted            = "Search completed"
	LogMsgFirstSearchBonus           = "First search of the day - applying bonus"
	LogMsgDiminishedReturnsApplied   = "Diminished search returns applied"
	LogMsgSearchSuccessLootboxFound  = "Search successful - lootbox found"
	LogMsgSearchCriticalSuccess      = "Search CRITICAL success"
	LogMsgSearchSuccessNothingFound  = "Search successful - nothing found"
	LogMsgSearchNearMiss             = "Search NEAR MISS"
	LogMsgSearchCriticalFail         = "Search CRITICAL FAIL"
	LogMsgBlasterUsed                = "blaster used"
	LogMsgWeaponUsed                 = "weapon used"
	LogMsgReviveUsed                 = "revive used"
	LogMsgShieldApplied              = "shield applied"
	LogMsgRareCandyUsed              = "rare candy used"
	LogMsgTrapUsed                   = "trap used"
	LogMsgTrapTriggered              = "trap triggered"
	LogMsgUserCacheHit               = "User cache hit"
	LogMsgFoundExistingUser          = "Found existing user"
	LogMsgAutoRegisteringUser        = "Auto-registering new user"
	LogMsgUserAutoRegistered         = "User auto-registered"
	LogMsgExplorerLeveledUp          = "Explorer leveled up!"
	LogMsgUserServiceShuttingDown    = "User service shutting down, waiting for background tasks..."
	LogMsgPlatformDefaultingToTwitch = "Platform not specified, defaulting to twitch"
	LogMsgMergingUsers               = "Merging users"
	LogMsgUsersMergedSuccessfully    = "Users merged successfully"
	LogMsgUnlinkingPlatform          = "Unlinking platform"
	LogMsgPlatformUnlinked           = "Platform unlinked"
)

// ============================================================================
// Log Messages - Warnings
// ============================================================================

const (
	LogWarnItemNotFound                 = "Item not found"
	LogWarnItemNotInInventory           = "Item not in inventory"
	LogWarnNoHandlerForItem             = "No handler for item"
	LogWarnItemMissingForSlot           = "Item missing for slot"
	LogWarnTargetUsernameMissingBlaster = "target username missing for blaster"
	LogWarnTargetUsernameMissingWeapon  = "target username missing for weapon"
	LogWarnTargetUsernameMissingRevive  = "target username missing for revive"
	LogWarnBlasterNotInInventory        = "blaster not in inventory"
	LogWarnWeaponNotInInventory         = "weapon not in inventory"
	LogWarnReviveNotInInventory         = "revive not in inventory"
	LogWarnShieldNotInInventory         = "shield not in inventory"
	LogWarnRareCandyNotInInventory      = "rarecandy not in inventory"
	LogWarnNotEnoughBlasters            = "not enough blasters in inventory"
	LogWarnNotEnoughWeapons             = "not enough weapons in inventory"
	LogWarnNotEnoughRevives             = "not enough revives in inventory"
	LogWarnNotEnoughShields             = "not enough shields in inventory"
	LogWarnNotEnoughRareCandy           = "not enough rare candy in inventory"
	LogWarnNotEnoughTraps               = "not enough traps in inventory"
	LogWarnTrapNotInInventory           = "trap not in inventory"
	LogWarnTargetUsernameMissingTrap    = "target username missing for trap"
	LogWarnJobNameMissing               = "job name missing for rare candy"
	LogWarnFailedToGetSearchCounts      = "Failed to get search counts"
	LogWarnFailedToAwardExplorerXP      = "Failed to award Explorer XP"
	LogWarnFailedToGetUserStreak        = "Failed to get user streak"
	LogWarnFailedToTimeoutUser          = "Failed to timeout user"
	LogWarnFailedToReduceTimeout        = "Failed to reduce timeout"
	LogWarnFailedToApplyShield          = "Failed to apply shield"
	LogWarnFailedToAwardJobXP           = "Failed to award job XP"
	LogWarnFailedToRecordLootboxJackpot = "Failed to record lootbox jackpot event"
	LogWarnFailedToRecordLootboxBigWin  = "Failed to record lootbox big-win event"
	LogWarnFailedToDeleteSecondaryInv   = "Failed to delete secondary inventory"
)

// ============================================================================
// Log Messages - Errors
// ============================================================================

const (
	LogErrFailedToUpsertUser           = "Failed to upsert user"
	LogErrFailedToUpdateUser           = "Failed to update user"
	LogErrFailedToFindUserByPlatformID = "Failed to find user by platform ID"
	LogErrFailedToGetUser              = "Failed to get user"
	LogErrFailedToBeginTx              = "Failed to begin transaction"
	LogErrFailedToGetItem              = "Failed to get item"
	LogErrFailedToGetInventory         = "Failed to get inventory"
	LogErrFailedToUpdateInventory      = "Failed to update inventory"
	LogErrFailedToCommitTx             = "Failed to commit transaction"
	LogErrFailedToGetItemDetails       = "Failed to get item details"
	LogErrFailedToGetMissingItems      = "Failed to get missing items"
	LogErrHandlerError                 = "Handler error"
	LogErrFailedToUpdateInventoryAfter = "Failed to update inventory after use"
	LogErrFailedToGetLootbox0Item      = "Failed to get lootbox0 item"
	LogErrLootbox0NotFoundInDB         = "Lootbox0 item not found in database"
	LogErrFailedToOpenLootbox          = "Failed to open lootbox"
	LogErrFailedToAutoRegisterUser     = "Failed to auto-register user"
)

// ============================================================================
// User-Facing Messages - Lootbox Results
// ============================================================================

const (
	MsgLootboxEmpty     = "The lootbox was empty!"
	MsgLootboxOpened    = "Opened"
	MsgLootboxReceived  = " and received: "
	MsgLootboxValue     = " (Value: "
	MsgLootboxValueEnd  = ")"
	MsgLootboxJackpot   = " JACKPOT! 🎰✨"
	MsgLootboxBigWin    = " BIG WIN! 💰"
	MsgLootboxNiceHaul  = " Nice haul! 📦"
	MsgLootboxExhausted = " (Exhausted)"
)

// ============================================================================
// User-Facing Messages - Blaster
// ============================================================================

const (
	MsgBlasterUsedPrefix = " has BLASTED "
	MsgBlasterUsedSuffix = " times! They are timed out for "
	MsgBlasterTimeoutEnd = "."
	MsgBlasterReasonBy   = "Blasted by "
)

// ============================================================================
// User-Facing Messages - Search Results
// ============================================================================

const (
	MsgSearchFoundPrefix    = "You have found "
	MsgSearchFoundSuffix    = "x "
	MsgSearchCooldownMinute = "You can search again in %dm %ds."
	MsgSearchCooldownSecond = "You can search again in %ds."
)

// ============================================================================
// User-Facing Messages - Item Transfer
// ============================================================================

const (
	MsgGiveItemSuccess = "Successfully gave %d %s to %s"
)

// ============================================================================
// User-Facing Messages - Resource Generation
// ============================================================================

const (
	MsgShovelUsed        = " used a shovel and found "
	MsgStickUsed         = " planted a stick as a monument to their achievement!"
	MsgTrapSet           = "Trap set on %s! It will trigger when they speak."
	MsgTrapSelfTriggered = "BOOM! You stepped on %s's existing trap! Your trap has been placed anyway."
)

// ============================================================================
// Error Context Messages (for wrapped errors)
// ============================================================================

const (
	ErrContextFailedToGetUser              = "failed to get user"
	ErrContextFailedToBeginTx              = "failed to begin transaction"
	ErrContextFailedToGetItem              = "failed to get item"
	ErrContextFailedToGetInventory         = "failed to get inventory"
	ErrContextFailedToUpdateInventory      = "failed to update inventory"
	ErrContextFailedToCommitTx             = "failed to commit transaction"
	ErrContextFailedToGetItemDetails       = "failed to get item details"
	ErrContextFailedToGetMissingItems      = "failed to get missing items"
	ErrContextOwnerValidationFailed        = "owner validation failed"
	ErrContextReceiverValidationFailed     = "receiver validation failed"
	ErrContextSenderValidationFailed       = "sender validation failed"
	ErrContextFailedToGetOwnerInventory    = "failed to get owner inventory"
	ErrContextFailedToGetReceiverInventory = "failed to get receiver inventory"
	ErrContextFailedToUpdateOwnerInventory = "failed to update owner inventory"
	ErrContextFailedToUpdateReceiverInv    = "failed to update receiver inventory"
	ErrContextFailedToResolveItemName      = "failed to resolve item name '%s'"
	ErrContextFailedToGetRewardItem        = "failed to get reward item"
	ErrContextShutdownTimedOut             = "shutdown timed out"
	ErrContextUserNotFound                 = "user not found"
	ErrContextFailedToGetPrimaryInventory  = "failed to get primary inventory"
	ErrContextFailedToGetSecondaryInv      = "failed to get secondary inventory"
	ErrContextFailedToUpdatePrimaryInv     = "failed to update primary inventory"
	ErrContextFailedToMergeUsers           = "failed to merge users in transaction"
	ErrContextFailedToGetPrimaryUser       = "failed to get primary user"
	ErrContextFailedToGetSecondaryUser     = "failed to get secondary user"
	ErrContextPrimaryUserNotFound          = "primary user not found"
	ErrContextSecondaryUserNotFound        = "secondary user not found"
	ErrContextFailedToUpdateUser           = "failed to update user"
	ErrContextFailedToOpenLootbox          = "failed to open lootbox"
	ErrContextFailedToRegisterUser         = "failed to register user"
)

// ============================================================================
// Validation Error Messages (for users)
// ============================================================================

const (
	ErrMsgInvalidPlatformForUse      = "invalid platform '%s': must be one of: %s, %s, %s"
	ErrMsgItemHasNoEffect            = "item %s has no effect"
	ErrMsgNotFoundAsPublicOrInt      = "%s (not found as public or internal name)"
	ErrMsgUnknownPlatform            = "unknown platform: %s"
	ErrMsgOwnerNotFound              = "owner"
	ErrMsgReceiverNotFound           = "receiver"
	ErrMsgInsufficientQuantityForUse = "has %d, needs %d"
)

// ============================================================================
// Metadata Keys - Job/Stats Events
// ============================================================================

const (
	MetadataKeySuccess    = "success"
	MetadataKeyDailyCount = "daily_count"
	MetadataKeyItem       = "item"
	MetadataKeyRoll       = "roll"
	MetadataKeyThreshold  = "threshold"
)

// ============================================================================
// XP Award Sources
// ============================================================================

const (
	XPSourceSearch = ActionSearch
)

// ============================================================================
// Lootbox Opening - Display Constants
// ============================================================================

const (
	LootboxDisplayQuantityPrefix  = "x "
	LootboxQualityAnnotationOpen  = " ["
	LootboxQualityAnnotationClose = "!]"
	LootboxDropSeparator          = ", "
)
