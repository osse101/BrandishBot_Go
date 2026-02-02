package naming

// ============================================================================
// Shine Level Constants
// ============================================================================

// ShineCommonLevel is the default shine level that does not display a prefix
// in the formatted display name. Items with COMMON shine show only their name.
// Deprecated: use domain.ShineCommon instead
const ShineCommonLevel = "COMMON"

// ShineFormatTemplate is the format string for displaying items with non-common
// shine levels. Format: "<SHINE_LEVEL> <item_name>"
const ShineFormatTemplate = "%s %s"

// ============================================================================
// Date Parsing Constants
// ============================================================================

// DateSeparator is the character used to separate month and day in MM-DD format
const DateSeparator = "-"

// DatePartsCount is the expected number of parts when splitting MM-DD format
const DatePartsCount = 2

// DateComparisonMultiplier is the multiplier used to create comparable integer
// values from month/day pairs. Formula: (month * 100 + day) allows direct
// comparison of dates within a year to determine if current date falls within
// a theme period (handles year-wrap scenarios like 12-15 to 01-05).
const DateComparisonMultiplier = 100

// ============================================================================
// Configuration Schema Constants
// ============================================================================

// SchemaItemAliases is the schema identifier for item aliases configuration
const SchemaItemAliases = "item-aliases"

// SchemaItemThemes is the schema identifier for item themes configuration
const SchemaItemThemes = "item-themes"

// ============================================================================
// JSON Configuration Keys
// ============================================================================

// JSONKeyVersion is the top-level JSON key for configuration version
const JSONKeyVersion = "version"

// JSONKeySchema is the top-level JSON key for schema identifier
const JSONKeySchema = "schema"

// JSONKeyAliases is the top-level JSON key for alias pool mappings
const JSONKeyAliases = "aliases"

// JSONKeyThemes is the top-level JSON key for theme period definitions
const JSONKeyThemes = "themes"

// ============================================================================
// Error Messages
// ============================================================================

// Error context messages for wrapped errors during configuration loading
const (
	ErrContextFailedToLoadAliases = "failed to load aliases"
	ErrContextFailedToLoadThemes  = "failed to load themes"
	ErrContextFailedToParseConfig = "failed to parse config %s"
	ErrContextFailedToDecodeData  = "failed to decode data for %s"
)

// Configuration validation error messages
const (
	ErrMsgMissingVersionField = "%s missing version field"
	ErrMsgInvalidSchema       = "invalid schema in %s: expected '%s', got '%s'"
)
