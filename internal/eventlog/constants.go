package eventlog

// Event type constants are now defined in internal/domain/events.go
// Import github.com/osse101/BrandishBot_Go/internal/domain to use:
//   - domain.EventTypeItemSold
//   - domain.EventTypeItemBought
//   - domain.EventTypeItemUpgraded
//   - domain.EventTypeItemDisassembled
//   - domain.EventTypeItemUsed
//   - domain.EventTypeSearchPerformed
//   - domain.EventTypeEngagement

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
