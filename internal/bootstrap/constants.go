package bootstrap

import "time"

// =============================================================================
// File System Permissions
// =============================================================================

const (
	// DirPermission is the standard permission for creating directories
	DirPermission = 0755

	// LogFilePermission is the permission for log files (read/write for owner, read for group/others)
	LogFilePermission = 0666
)

// =============================================================================
// Logger Configuration
// =============================================================================

const (
	// LogFileTimestampFormat is the timestamp format for log filenames (YYYY-MM-DD_HH-MM-SS)
	LogFileTimestampFormat = "2006-01-02_15-04-05"

	// LogFileNamePattern is the format string for log filenames
	LogFileNamePattern = "session_%s.log"

	// LogFileExtension is the file extension for log files
	LogFileExtension = ".log"

	// LogFileRetentionLimit is the maximum number of log files to keep
	LogFileRetentionLimit = 10

	// LogFileRetentionCount is the number of log files to retain after cleanup
	LogFileRetentionCount = 9
)

// Log level string constants
const (
	LogLevelDebug = "DEBUG"
	LogLevelInfo  = "INFO"
	LogLevelWarn  = "WARN"
	LogLevelError = "ERROR"
)

// Log messages for logger initialization
const (
	LogMsgLoggingInitialized  = "Logging initialized"
	LogMsgStartingBrandishBot = "Starting BrandishBot"
	LogMsgConfigurationLoaded = "Configuration loaded"
	LogMsgFailedCreateLogsDir = "failed to create logs directory"
	LogMsgFailedOpenLogFile   = "failed to open log file"
	LogMsgFailedDeleteOldLog  = "Failed to delete old log file %s: %v\n"
)

// =============================================================================
// Event System Configuration
// =============================================================================

const (
	// EventDefaultMaxRetries is the default number of retry attempts for failed event publishing
	EventDefaultMaxRetries = 5

	// EventDefaultRetryDelay is the default base delay between retry attempts (exponential backoff)
	EventDefaultRetryDelay = 2 * time.Second

	// EventDefaultDeadLetterPath is the default file path for dead-letter event logging
	EventDefaultDeadLetterPath = "logs/event_deadletter.jsonl"
)

// Log messages for event system initialization
const (
	LogMsgEventSystemInitialized         = "Event system initialized"
	LogMsgFailedCreateDeadLetterDir      = "failed to create dead-letter directory"
	LogMsgFailedCreateResilientPublisher = "failed to create resilient publisher"
)

// =============================================================================
// Config Sync Messages
// =============================================================================

const (
	// Config sync log messages
	LogMsgSyncingProgressionTree   = "Syncing progression tree from JSON config..."
	LogMsgSyncingItems             = "Syncing items from JSON config..."
	LogMsgSyncingRecipes           = "Syncing recipes from JSON config..."
	LogMsgProgressionTreeSynced    = "Progression tree synced successfully"
	LogMsgProgressionTreeUnchanged = "Progression tree config unchanged, sync skipped"
	LogMsgItemsSynced              = "Items synced successfully"
	LogMsgRecipesSynced            = "Recipes synced successfully"

	// Config sync error messages
	ErrMsgFailedLoadProgressionTree = "failed to load progression tree config"
	ErrMsgInvalidProgressionTree    = "invalid progression tree config"
	ErrMsgFailedSyncProgressionTree = "failed to sync progression tree to database"
	ErrMsgFailedLoadItems           = "failed to load items config"
	ErrMsgInvalidItems              = "invalid items config"
	ErrMsgFailedSyncItems           = "failed to sync items to database"
	ErrMsgFailedLoadRecipes         = "failed to load recipe config"
	ErrMsgInvalidRecipes            = "invalid recipe configuration"
	ErrMsgFailedSyncRecipes         = "failed to sync recipes to database"
)

// =============================================================================
// Event Handler Configuration
// =============================================================================

// Log messages for event handler registration
const (
	LogMsgMetricsCollectorRegistered = "Metrics collector registered"
	LogMsgEventLoggerInitialized     = "Event logger initialized"
	ErrMsgFailedRegisterMetrics      = "failed to register metrics collector"
	ErrMsgFailedSubscribeEventLogger = "failed to subscribe event logger"
)

// =============================================================================
// Shutdown Messages
// =============================================================================

const (
	LogMsgShuttingDownServer         = "Shutting down server..."
	LogMsgShuttingDownEventPublisher = "Shutting down event publisher..."
	LogMsgServerStopped              = "Server stopped"
	LogMsgServerForcedShutdown       = "Server forced to shutdown"
	LogMsgResilientPublisherFailed   = "Resilient publisher shutdown failed"

	// Service names for shutdown logging
	ServiceNameProgression = "progression"
	ServiceNameUser        = "user"
	ServiceNameEconomy     = "economy"
	ServiceNameCrafting    = "crafting"
	ServiceNameGamble      = "gamble"
)

// Shutdown log message format (service name will be prepended)
const (
	LogMsgServiceShutdownFailed = " service shutdown failed"
)
