package item

// ==================== Configuration File Names ====================

// Item configuration file names
const (
	// ConfigFileName is the name of the items configuration file
	ConfigFileName = "items.json"
)

// ==================== Error Messages ====================

// File operation error messages
const (
	ErrMsgReadConfigFileFailed = "failed to read items config file: %w"
	ErrMsgParseConfigFailed    = "failed to parse items config: %w"
	ErrMsgStatConfigFileFailed = "failed to stat config file: %w"
	ErrMsgReadForHashFailed    = "failed to read config file: %w"
)

// Validation error messages (fragments used with error wrapping)
const (
	ErrMsgConfigNil           = "config is nil"
	ErrMsgNoItemsDefined      = "no items defined"
	ErrMsgEmptyInternalName   = "has empty internal_name"
	ErrMsgEmptyPublicName     = "has empty public_name"
	ErrMsgEmptyDefaultDisplay = "has empty default_display"
	ErrMsgNegativeMaxStack    = "has negative max_stack"
	ErrMsgNegativeBaseValue   = "has negative base_value"
)

// Database operation error messages
const (
	ErrMsgCheckFileChangeFailed  = "failed to check if file changed: %w"
	ErrMsgGetExistingItemsFailed = "failed to get existing items: %w"
	ErrMsgGetItemTypesFailed     = "failed to get item types: %w"
	ErrMsgUpdateItemFailed       = "failed to update item '%s': %w"
	ErrMsgInsertItemFailed       = "failed to insert item '%s': %w"
	ErrMsgSyncTagsFailed         = "failed to sync tags for '%s': %w"
	ErrMsgSyncTagsNewItemFailed  = "failed to sync tags for new item '%s': %w"
	ErrMsgCreateItemTypeFailed   = "failed to create item type '%s': %w"
	ErrMsgClearTagsFailed        = "failed to clear existing tags: %w"
	ErrMsgAssignTagFailed        = "failed to assign tag: %w"
)

// ==================== Log Messages ====================

// Sync operation log messages
const (
	LogMsgConfigUnchanged      = "Items config file unchanged, skipping sync"
	LogMsgSyncCompleted        = "Items sync completed"
	LogMsgUpdatedItem          = "Updated item"
	LogMsgInsertedItem         = "Inserted item"
	LogMsgUpdateMetadataFailed = "Failed to update sync metadata"
)

// ==================== Format Strings for Error Construction ====================

// These format strings are used with fmt.Errorf for detailed error messages
const (
	ErrFmtItemAtIndexEmpty     = "%w: item at index %d has empty internal_name"
	ErrFmtItemHasEmptyPublic   = "%w: item '%s' has empty public_name"
	ErrFmtItemHasEmptyDisplay  = "%w: item '%s' has empty default_display"
	ErrFmtItemNegativeMaxStack = "%w: item '%s' has negative max_stack"
	ErrFmtItemNegativeValue    = "%w: item '%s' has negative base_value"
)
