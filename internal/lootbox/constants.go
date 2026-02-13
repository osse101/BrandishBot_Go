package lootbox

// ============================================================================
// Quality Rarity Thresholds
// ============================================================================

// QualityLegendaryThreshold defines the maximum roll (<=1%) for LEGENDARY quality.
const QualityLegendaryThreshold = 0.01

// QualityEpicThreshold defines the maximum roll (<=5%) for EPIC quality.
const QualityEpicThreshold = 0.05

// QualityRareThreshold defines the maximum roll (<=15%) for RARE quality.
const QualityRareThreshold = 0.15

// QualityUncommonThreshold defines the maximum roll (<=30%) for UNCOMMON quality.
const QualityUncommonThreshold = 0.30

// QualityCommonThreshold defines the maximum roll (<=70%) for COMMON quality.
// This is the largest bucket, making Common the most likely outcome.
const QualityCommonThreshold = 0.70

// QualityPoorThreshold defines the maximum roll (<=85%) for POOR quality.
const QualityPoorThreshold = 0.85

// QualityJunkThreshold defines the maximum roll (<=95%) for JUNK quality.
const QualityJunkThreshold = 0.95

// ============================================================================
// Drop Mechanics
// ============================================================================

// CriticalQualityUpgradeChance is the probability (1%) that a dropped item will
// have its quality level upgraded by one tier.
const CriticalQualityUpgradeChance = 0.01

// ============================================================================
// Configuration
// ============================================================================

// LootTablesSchemaPath is the path (relative to project root) for the v2 schema.
const LootTablesSchemaPath = "configs/schemas/loot_tables.schema.json"

// ConfigVersion2 is the expected version string for v2 loot table configs.
const ConfigVersion2 = "2.0"

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
	LogMsgOrphanedItem       = "Item not referenced in any pool (orphaned)"
)

// Log field keys for structured logging
const (
	LogFieldLootbox = "lootbox"
	LogFieldItem    = "item"
	LogFieldError   = "error"
)
