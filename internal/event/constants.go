package event

import "time"

// Event schema versioning
const (
	// EventSchemaVersion is the current event schema version
	EventSchemaVersion = "1.0"
)

// Retry configuration constants
const (
	// RetryQueueBufferSize is the buffer size for the retry queue
	RetryQueueBufferSize = 1000

	// RetryInitialDelaySeconds is the initial retry delay in seconds (2s)
	RetryInitialDelaySeconds = 2

	// RetryMaxAttempts is the default maximum number of retry attempts
	RetryMaxAttempts = 5
)

// Dead letter file configuration
const (
	// DeadLetterFilePermissions is the file permission mode for dead-letter files
	DeadLetterFilePermissions = 0644
)

// Log message constants
const (
	// Log messages for event publishing
	LogMsgEventPublishFailed     = "Event publish failed, queuing for retry"
	LogMsgRetryQueueFull         = "Retry queue full, event dropped to dead-letter"
	LogMsgDeadLetterWriteFailed  = "Failed to write to dead letter"
	LogMsgEventRetryExhausted    = "Event retry exhausted, writing to dead-letter"
	LogMsgEventRetryFailed       = "Event retry failed, scheduling next attempt"
	LogMsgEventRetrySucceeded    = "Event retry succeeded"
	LogMsgEventDroppedShutdown   = "Event dropped during shutdown"
	LogMsgQueueDrainedShutdown   = "Drained retry queue during shutdown"
	LogMsgShutdownTimeout        = "Resilient publisher shutdown timed out"
	LogMsgDeadLetterWriteFailedS = "Failed to write to dead letter shutdown"

	// Log message for handler errors
	LogMsgHandlerErrorFormat = "encountered %d errors while handling event %s: %v"
)

// CalculateRetryDelay calculates the exponential backoff delay for retry attempts.
// Implements exponential backoff: 2s, 4s, 8s, 16s, 32s
// Formula: initialDelay * 2^(attempt-1)
func CalculateRetryDelay(baseDelay time.Duration, attempt int) time.Duration {
	return baseDelay * time.Duration(1<<(attempt-1))
}
