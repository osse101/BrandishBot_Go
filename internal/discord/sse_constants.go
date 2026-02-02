package discord

import "time"

// SSE client configuration
const (
	// sseInitialBackoff is the initial backoff duration for reconnection
	sseInitialBackoff = 1 * time.Second

	// sseMaxBackoff is the maximum backoff duration for reconnection
	sseMaxBackoff = 30 * time.Second

	// sseBackoffMultiplier is the multiplier for exponential backoff
	sseBackoffMultiplier = 2.0

	// sseBufferSize is the buffer size for reading SSE events
	sseBufferSize = 64 * 1024 // 64KB
)

// SSE event types
const (
	// SSEEventTypeJobLevelUp is the event type for job level ups
	SSEEventTypeJobLevelUp = "job.level_up"

	// SSEEventTypeVotingStarted is the event type for voting session starts
	SSEEventTypeVotingStarted = "progression.voting_started"

	// SSEEventTypeCycleCompleted is the event type for progression cycle completion
	SSEEventTypeCycleCompleted = "progression.cycle_completed"

	// SSEEventTypeAllUnlocked is the event type for all progression nodes unlocked
	SSEEventTypeAllUnlocked = "progression.all_unlocked"

	// SSEEventTypeGambleCompleted is the event type for gamble completion
	SSEEventTypeGambleCompleted = "gamble.completed"
)

// SSE log messages
const (
	sseLogMsgClientConnected   = "SSE client connected"
	sseLogMsgClientStopped     = "SSE client stopped"
	sseLogMsgConnectionFailed  = "SSE connection failed"
	sseLogMsgParseError        = "Failed to parse SSE event"
	sseLogMsgHandlerError      = "SSE event handler error"
	sseLogMsgEventReceived     = "SSE event received"
	sseLogMsgNotificationSent  = "Discord notification sent"
	sseLogMsgNotificationError = "Failed to send Discord notification"
)
