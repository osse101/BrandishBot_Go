package lootbox

// ============================================================================
// Shine Rarity Thresholds
// ============================================================================

// ShineLegendaryThreshold defines the maximum roll (<=1%) for LEGENDARY shine.
const ShineLegendaryThreshold = 0.01

// ShineEpicThreshold defines the maximum roll (<=5%) for EPIC shine.
const ShineEpicThreshold = 0.05

// ShineRareThreshold defines the maximum roll (<=15%) for RARE shine.
const ShineRareThreshold = 0.15

// ShineUncommonThreshold defines the maximum roll (<=30%) for UNCOMMON shine.
const ShineUncommonThreshold = 0.30

// ShineCommonThreshold defines the maximum roll (<=70%) for COMMON shine.
// This is the largest bucket, making Common the most likely outcome.
const ShineCommonThreshold = 0.70

// ShinePoorThreshold defines the maximum roll (<=85%) for POOR shine.
const ShinePoorThreshold = 0.85

// ShineJunkThreshold defines the maximum roll (<=95%) for JUNK shine.
const ShineJunkThreshold = 0.95

// ============================================================================
// Drop Mechanics
// ============================================================================

// CriticalShineUpgradeChance is the probability (1%) that a dropped item will
// have its shine level upgraded by one tier. This creates exciting "Lucky!"
// moments where a common drop becomes uncommon, rare becomes epic, etc.
const CriticalShineUpgradeChance = 0.01

// GuaranteedDropThreshold defines the chance value (>=1.0) that ensures an
// item will always drop from the loot table. Used to distinguish between
// guaranteed drops and chance-based drops.
const GuaranteedDropThreshold = 1.0

// ZeroChanceThreshold defines the minimum chance value (<=0) for which a
// loot item will be skipped entirely during processing. Items with 0 or
// negative chance never drop.
const ZeroChanceThreshold = 0

// ============================================================================
// Configuration Keys
// ============================================================================

// ConfigKeyTables is the top-level JSON key used in loot tables configuration
// file to access the map of loot box name -> loot items.
const ConfigKeyTables = "tables"

// ============================================================================
// Error Messages
// ============================================================================

// Error context messages for wrapped errors during loot table loading
const (
	ErrContextFailedToLoadLootTables = "failed to load loot tables"
	ErrContextFailedToReadLootFile   = "failed to read loot tables file"
	ErrContextFailedToParseLootFile  = "failed to parse loot tables"
)

// Database operation error messages
const (
	ErrContextFailedToGetDroppedItems = "Failed to get dropped items"
)

// ============================================================================
// Log Messages
// ============================================================================

// Warning messages for missing or invalid data
const (
	LogMsgNoLootTableFound   = "No loot table found for lootbox"
	LogMsgDroppedItemNotInDB = "Dropped item not found in DB"
)

// Log field keys for structured logging
const (
	LogFieldLootbox = "lootbox"
	LogFieldItem    = "item"
	LogFieldError   = "error"
)
