package gamble

import "time"

// ============================================================================
// Gamble Execution Thresholds
// ============================================================================

// ExecutionGracePeriod defines how early before the join deadline a gamble
// can be executed. This grace period helps handle clock skew and network delays.
const ExecutionGracePeriod = 5 * time.Second

// NearMissThreshold defines the percentage of the winner's score required to
// trigger a "Near Miss" event. A participant who scores >= 95% of the winner's
// score (but doesn't win) will have a near-miss event recorded.
const NearMissThreshold = 0.95

// CriticalFailThreshold defines the percentage of the average score required to
// trigger a "Critical Fail" event. A participant who scores <= 20% of the
// average score will have a critical fail event recorded.
const CriticalFailThreshold = 0.2

// InitialHighestValue is the sentinel value used to initialize the highest
// value tracker when determining gamble winners. Set to -1 to ensure any
// positive value will be higher.
const InitialHighestValue = -1

// ============================================================================
// Lootbox Validation
// ============================================================================

// LootboxPrefix is the required prefix for lootbox internal names
const LootboxPrefix = "lootbox"

// LootboxPrefixLength is the length of the lootbox prefix for validation
const LootboxPrefixLength = 7

// ============================================================================
// Progression Feature Keys
// ============================================================================

// ProgressionFeatureGambleWinBonus is the progression feature key used to
// query for gamble win value modifiers from the progression system
const ProgressionFeatureGambleWinBonus = "gamble_win_bonus"

// ============================================================================
// XP Award Metadata Keys
// ============================================================================

// MetadataKeySource identifies the source/reason for XP award
const MetadataKeySource = "source"

// MetadataKeyLootboxCount tracks the number of lootboxes in the gamble
const MetadataKeyLootboxCount = "lootbox_count"

// MetadataKeyIsWin indicates whether the XP award is for winning the gamble
const MetadataKeyIsWin = "is_win"

// ============================================================================
// Event Publishing
// ============================================================================

// EventSchemaVersion is the version of the event schema used for gamble events
const EventSchemaVersion = "1.0"

// ============================================================================
// Log Messages
// ============================================================================

// Log operation identifiers
const (
	LogMsgStartGambleCalled   = "StartGamble called"
	LogMsgJoinGambleCalled    = "JoinGamble called"
	LogMsgExecuteGambleCalled = "ExecuteGamble called"
)

// Log context for gamble events
const (
	LogContextGambleStartedEvent = "GambleStarted event"
)

// Log reasons and error contexts
const (
	LogReasonEventBusNil = "eventBus is nil"
)

// Warning/Info messages
const (
	LogMsgGambleAlreadyCompleted      = "Gamble already completed, skipping execution"
	LogMsgFailedToAwardGamblerXP      = "Failed to award Gambler XP"
	LogMsgGamblerLeveledUp            = "Gambler leveled up!"
	LogMsgShuttingDownGambleService   = "Shutting down gamble service, waiting for async operations..."
	LogMsgGambleServiceShutdownDone   = "Gamble service shutdown complete"
	LogMsgGambleServiceShutdownForced = "Gamble service shutdown forced by context cancellation"
)

// ============================================================================
// Error Messages (local to gamble service)
// ============================================================================

// Error context messages for wrapped errors
const (
	ErrContextFailedToResolveItemName = "failed to resolve item name"
	ErrContextFailedToGetItem         = "failed to get item"
	ErrContextFailedToGetUser         = "failed to get user"
	ErrContextFailedToGetInventory    = "failed to get inventory"
	ErrContextFailedToConsumeBet      = "failed to consume bet"
	ErrContextFailedToGetGamble       = "failed to get gamble"
	ErrContextFailedToBeginTx         = "failed to begin transaction"
	ErrContextFailedToUpdateInventory = "failed to update inventory"
	ErrContextFailedToJoinGamble      = "failed to join gamble"
	ErrContextFailedToCommitTx        = "failed to commit gamble transaction"
	ErrContextFailedToGetWinnerInv    = "failed to get winner inventory"
	ErrContextFailedToUpdateWinnerInv = "failed to update winner inventory"
	ErrContextFailedToCheckActive     = "failed to check active gamble"
	ErrContextFailedToCreateGamble    = "failed to create gamble"
	ErrContextFailedToAddInitiator    = "failed to add initiator as participant"
)

// Validation and state error messages
const (
	ErrMsgCannotExecuteBeforeDeadline       = "cannot execute gamble before join deadline"
	ErrMsgGambleAlreadyExecuted             = "gamble is already being executed or has been executed"
	ErrMsgItemNotFoundAsPublicOrInternalName = "not found as public or internal name"
)
