package middleware

// Event Constants
const (
	// EventVersion is the engagement event schema version
	EventVersion = "1.0"

	// EventTypeEngagement is the event type for engagement tracking
	EventTypeEngagement = "engagement"
)

// Metric Constants
const (
	// DefaultMetricValue is the default engagement metric value when no custom value is provided
	DefaultMetricValue = 1

	// MetricTypeCommand is the metric type for command execution tracking
	MetricTypeCommand = "command"
)

// HTTP Request Parameter Names
const (
	// QueryParamUsername is the query parameter name for username
	QueryParamUsername = "username"
)

// Metadata Keys
const (
	// MetadataKeyEndpoint is the metadata key for API endpoint path
	MetadataKeyEndpoint = "endpoint"

	// MetadataKeyMethod is the metadata key for HTTP method
	MetadataKeyMethod = "method"
)

// Default Values
const (
	// EmptyUserID represents an empty or missing user ID
	EmptyUserID = ""
)

// Log Messages
const (
	// LogMsgEngagementEventPublishFailed indicates engagement event publishing failed
	LogMsgEngagementEventPublishFailed = "Failed to publish engagement event"

	// LogMsgCommandEngagementEventPublishFailed indicates command engagement event publishing failed
	LogMsgCommandEngagementEventPublishFailed = "Failed to publish command engagement event"

	// LogMsgEngagementEventFromContextFailed indicates engagement event from context publishing failed
	LogMsgEngagementEventFromContextFailed = "Failed to publish engagement event from context"
)
