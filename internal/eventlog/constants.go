package eventlog

// Event type constants - domain event types that are logged
const (
	EventTypeItemSold        = "item.sold"
	EventTypeItemBought      = "item.bought"
	EventTypeItemUpgraded    = "item.upgraded"
	EventTypeItemDisassembled = "item.disassembled"
	EventTypeItemUsed        = "item.used"
	EventTypeSearchPerformed = "search.performed"
	EventTypeEngagement      = "engagement"
)

// JSON payload field keys
const (
	PayloadKeyUserID = "user_id"
)

// Log messages - service events
const (
	LogMsgEventPayloadNotMap      = "Event payload is not a map, skipping log"
	LogMsgFailedToLogEvent        = "Failed to log event to database"
	LogMsgEventLogged             = "Event logged to database"
)

// Log messages - cleanup job
const (
	LogMsgCleanupJobStarting  = "Starting event log cleanup job"
	LogMsgCleanupJobFailed    = "Event log cleanup failed"
	LogMsgCleanupJobCompleted = "Event log cleanup completed"
)

// Log field keys - structured logging fields
const (
	LogFieldType         = "type"
	LogFieldUserID       = "user_id"
	LogFieldError        = "error"
	LogFieldRetentionDays = "retentionDays"
	LogFieldDuration     = "duration"
	LogFieldDeletedCount = "deletedCount"
)

// Data validation constants
const (
	MinDataLength = 0 // Minimum length for checking if JSON data exists
)
