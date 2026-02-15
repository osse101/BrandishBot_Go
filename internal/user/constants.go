package user

import "time"

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
// Search Mechanic - Probability Thresholds
// ============================================================================

// SearchSuccessRate defines the probability of finding an item when searching (80%)
const SearchSuccessRate = 0.8

// SearchCriticalRate defines the probability of a critical success when searching (5%)
const SearchCriticalRate = 0.05

// SearchNearMissRate defines the probability of a near-miss result when searching (5%)
const SearchNearMissRate = 0.05

// SearchCriticalFailRate defines the probability of a critical failure when searching (5%)
const SearchCriticalFailRate = 0.05

// ============================================================================
// Search Mechanic - Diminishing Returns
// ============================================================================

// SearchDailyDiminishmentThreshold is the number of searches per day after which returns are diminished
const SearchDailyDiminishmentThreshold = 6

// SearchDiminishedSuccessRate is the success rate when diminished returns are active (10%)
const SearchDiminishedSuccessRate = 0.1

// SearchDiminishedXPMultiplier is the XP multiplier when diminished returns are active (10%)
const SearchDiminishedXPMultiplier = 0.1

// SearchFirstDailyGuaranteedRoll is the roll value that guarantees success for first search (0.0)
const SearchFirstDailyGuaranteedRoll = 0.0

// ============================================================================
// Item Handler Constants
// ============================================================================

// BulkFeedbackThreshold defines the number of lootboxes required to trigger "Nice haul" message
const BulkFeedbackThreshold = 5

// BlasterTimeoutDuration is the default duration a user is timed out when hit by a blaster
const BlasterTimeoutDuration = 60 * time.Second

// TrapCooldownDuration is the cooldown after a trap triggers to prevent immediate re-trapping
const TrapCooldownDuration = 10 * time.Minute

// ============================================================================
// Resource Generation Constants
// ============================================================================

// ShovelSticksPerUse defines how many sticks are generated per shovel use
const ShovelSticksPerUse = 2

// ============================================================================
// Inventory Limits
// ============================================================================

// MaxStackSize is the maximum quantity allowed for a single item stack when merging inventories
const MaxStackSize = 999999

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
	ErrMsgInvalidPlatform       = "invalid platform '%s': must be one of: %s, %s, %s"
	ErrMsgItemHasNoEffect       = "item %s has no effect"
	ErrMsgNotFoundAsPublicOrInt = "%s (not found as public or internal name)"
	ErrMsgUnknownPlatform       = "unknown platform: %s"
	ErrMsgOwnerNotFound         = "owner"
	ErrMsgReceiverNotFound      = "receiver"
	ErrMsgInsufficientQuantity  = "has %d, needs %d"
)

// ============================================================================
// Metadata Keys - Job/Stats Events
// ============================================================================

const (
	MetadataKeyItemName   = "item_name"
	MetadataKeyMultiplier = "multiplier"
	MetadataKeySuccess    = "success"
	MetadataKeyDailyCount = "daily_count"
	MetadataKeyItem       = "item"
	MetadataKeyQuantity   = "quantity"
	MetadataKeyRoll       = "roll"
	MetadataKeyThreshold  = "threshold"
)

// ============================================================================
// XP Award Sources
// ============================================================================

const (
	XPSourceSearch = "search"
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
