package crafting

// ==================== Crafting Mechanics ====================

// Masterwork constants control the probability and multiplier for critical crafts
const (
	// MasterworkChance determines the probability of a masterwork craft occurring (10% = 1 in 10 crafts)
	MasterworkChance = 0.10

	// MasterworkMultiplier is applied to output quantity when masterwork procs (2x output)
	MasterworkMultiplier = 2
)

// Perfect Salvage constants control the probability and multiplier for disassembly bonuses
const (
	// PerfectSalvageChance is the probability of a "Perfect Salvage" occurring during disassembly
	PerfectSalvageChance = 0.10

	// PerfectSalvageMultiplier is the bonus multiplier for materials when Perfect Salvage triggers
	PerfectSalvageMultiplier = 1.5
)

// ==================== Configuration File Names ====================

// Recipe configuration file names used by the loader
const (
	ConfigFileCrafting    = "crafting.json"
	ConfigFileDisassemble = "disassemble.json"
)

// Recipe sync metadata names stored in database
const (
	MetadataNameCrafting    = "recipes_crafting.json"
	MetadataNameDisassemble = "recipes_disassemble.json"
)

// ==================== Recipe Type Prefixes ====================

// Recipe type prefixes for orphaned recipe identification
const (
	RecipeTypePrefixCrafting    = "crafting:"
	RecipeTypePrefixDisassemble = "disassemble:"
)

// ==================== Error Messages ====================

// Validation error messages
const (
	ErrMsgUserNotFound           = "user not found"
	ErrMsgItemNotFoundFmt        = "item not found: %s"
	ErrMsgItemNotFoundPublicFmt  = "item not found: %s (not found as public or internal name)"
	ErrMsgResolveItemFailedFmt   = "failed to resolve item name '%s': %w"
	ErrMsgInsufficientMaterialFmt = "insufficient material (itemID: %d)"
	ErrMsgInsufficientCraftFmt   = "insufficient materials to craft %s"
	ErrMsgInsufficientItemsFmt   = "insufficient items to disassemble %s (need %d, have %d)"
)

// Recipe error messages
const (
	ErrMsgNoRecipeFmt             = "no recipe found for item: %s"
	ErrMsgRecipeNotUnlockedFmt    = "recipe for %s is not unlocked"
	ErrMsgNoDisassembleRecipeFmt  = "no disassemble recipe found for item: %s"
	ErrMsgDisassembleNotUnlockedFmt = "disassemble recipe for %s is not unlocked"
)

// Database operation error messages
const (
	ErrMsgGetUserFailed                = "failed to get user: %w"
	ErrMsgGetItemFailed                = "failed to get item: %w"
	ErrMsgGetRecipeFailed              = "failed to get recipe: %w"
	ErrMsgCheckRecipeUnlockFailed      = "failed to check recipe unlock: %w"
	ErrMsgGetUnlockedRecipesFailed     = "failed to get unlocked recipes: %w"
	ErrMsgGetAllRecipesFailed          = "failed to get all recipes: %w"
	ErrMsgBeginTransactionFailed       = "failed to begin transaction: %w"
	ErrMsgGetInventoryFailed           = "failed to get inventory: %w"
	ErrMsgUpdateInventoryFailed        = "failed to update inventory: %w"
	ErrMsgCommitTransactionFailed      = "failed to commit transaction: %w"
	ErrMsgGetDisassembleRecipeFailed   = "failed to get disassemble recipe: %w"
	ErrMsgGetAssociatedRecipeFailed    = "failed to get associated upgrade recipe: %w"
	ErrMsgGetOutputItemsFailed         = "failed to get output items: %w"
	ErrMsgOutputItemNotFoundFmt        = "output item not found: %d"
)

// Recipe loader error messages
const (
	ErrMsgReadCraftingConfigFailed        = "failed to read crafting config file: %w"
	ErrMsgParseCraftingConfigFailed       = "failed to parse crafting config: %w"
	ErrMsgReadDisassembleConfigFailed     = "failed to read disassemble config file: %w"
	ErrMsgParseDisassembleConfigFailed    = "failed to parse disassemble config: %w"
	ErrMsgGetItemsForValidationFailed     = "failed to get items for validation: %w"
	ErrMsgCheckCraftingFileChangeFailed   = "failed to check crafting file change: %w"
	ErrMsgCheckDisassembleFileChangeFailed = "failed to check disassemble file change: %w"
	ErrMsgGetItemsFailed                  = "failed to get items: %w"
	ErrMsgSyncCraftingRecipesFailed       = "failed to sync crafting recipes: %w"
	ErrMsgSyncDisassembleRecipesFailed    = "failed to sync disassemble recipes: %w"
	ErrMsgGetExistingCraftingRecipesFailed = "failed to get existing crafting recipes: %w"
	ErrMsgUpdateCraftingRecipeFmt         = "failed to update crafting recipe '%s': %w"
	ErrMsgInsertCraftingRecipeFmt         = "failed to insert crafting recipe '%s': %w"
	ErrMsgGetExistingDisassembleRecipesFailed = "failed to get existing disassemble recipes: %w"
	ErrMsgGetCraftingRecipesForAssocFailed = "failed to get crafting recipes for associations: %w"
	ErrMsgUpdateDisassembleRecipeFmt      = "failed to update disassemble recipe '%s': %w"
	ErrMsgClearOutputsFmt                 = "failed to clear outputs for recipe '%s': %w"
	ErrMsgInsertOutputFmt                 = "failed to insert output for recipe '%s': %w"
	ErrMsgUpsertAssociationFmt            = "failed to upsert association for '%s': %w"
	ErrMsgInsertDisassembleRecipeFmt      = "failed to insert disassemble recipe '%s': %w"
	ErrMsgStatConfigFileFailed            = "failed to stat config file: %w"
	ErrMsgReadConfigFileFailed            = "failed to read config file: %w"
)

// ==================== Log Messages ====================

// Service operation log messages
const (
	LogMsgUpgradeItemCalled        = "UpgradeItem called"
	LogMsgItemsUpgraded            = "Items upgraded"
	LogMsgMasterworkTriggered      = "Masterwork craft triggered!"
	LogMsgGetRecipeCalled          = "GetRecipe called"
	LogMsgRecipeRetrieved          = "Recipe retrieved"
	LogMsgGetUnlockedRecipesCalled = "GetUnlockedRecipes called"
	LogMsgUnlockedRecipesRetrieved = "Unlocked recipes retrieved"
	LogMsgGetAllRecipesCalled      = "GetAllRecipes called"
	LogMsgDisassembleItemCalled    = "DisassembleItem called"
	LogMsgItemsDisassembled        = "Items disassembled"
	LogMsgPerfectSalvageTriggered  = "Perfect Salvage triggered!"
	LogMsgShuttingDown             = "Shutting down crafting service, waiting for async operations..."
	LogMsgShutdownComplete         = "Crafting service shutdown complete"
	LogMsgShutdownForced           = "Crafting service shutdown forced by context cancellation"
	LogMsgAwardXPFailed            = "Failed to award Blacksmith XP"
	LogMsgBlacksmithLeveledUp      = "Blacksmith leveled up!"
)

// Recipe loader log messages
const (
	LogMsgRecipeConfigUnchanged            = "Recipe config files unchanged, skipping sync"
	LogMsgRecipeSyncCompleted              = "Recipe sync completed"
	LogMsgOrphanedRecipesFound             = "Found orphaned recipes in database (in DB but not in config)"
	LogMsgUpdateCraftingSyncMetadataFailed = "Failed to update crafting sync metadata"
	LogMsgUpdateDisassembleSyncMetadataFailed = "Failed to update disassemble sync metadata"
	LogMsgUpdatedCraftingRecipe            = "Updated crafting recipe"
	LogMsgInsertedCraftingRecipe           = "Inserted crafting recipe"
	LogMsgUpdatedDisassembleRecipe         = "Updated disassemble recipe"
	LogMsgInsertedDisassembleRecipe        = "Inserted disassemble recipe"
)

// ==================== Metadata Keys ====================

// Event metadata keys for stats recording
const (
	MetadataKeyItemName        = "item_name"
	MetadataKeyOriginalQty     = "original_quantity"
	MetadataKeyMasterworkCount = "masterwork_count"
	MetadataKeyBonusQty        = "bonus_quantity"
	MetadataKeyQuantity        = "quantity"
	MetadataKeyPerfectCount    = "perfect_count"
	MetadataKeyMultiplier      = "multiplier"
	MetadataKeySource          = "source"
)

// XP award source identifiers
const (
	XPSourceUpgrade     = "upgrade"
	XPSourceDisassemble = "disassemble"
)
