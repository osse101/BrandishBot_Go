package sse

import "time"

// Buffer sizes
const (
	// BroadcastBufferSize is the buffer size for the broadcast channel
	BroadcastBufferSize = 100

	// ClientEventBuffer is the buffer size for each client's event channel
	ClientEventBuffer = 50

	// ClientChannelBuffer is the buffer size for register/unregister channels
	ClientChannelBuffer = 10
)

// SSE connection settings
const (
	// KeepaliveInterval is how often to send keepalive pings
	KeepaliveInterval = 30 * time.Second

	// WriteTimeout is the timeout for writing to client connections
	WriteTimeout = 10 * time.Second
)

// Event types for SSE
const (
	// EventTypeJobLevelUp is sent when a user levels up a job
	EventTypeJobLevelUp = "job.level_up"

	// EventTypeVotingStarted is sent when a new voting session begins
	EventTypeVotingStarted = "progression.voting_started"

	// EventTypeCycleCompleted is sent when a progression cycle completes (node unlocked + new voting)
	EventTypeCycleCompleted = "progression.cycle_completed"

	// EventTypeAllUnlocked is sent when all progression nodes have been unlocked
	EventTypeAllUnlocked = "progression.all_unlocked"

	// EventTypeGambleCompleted is sent when a gamble session completes
	EventTypeGambleCompleted = "gamble.completed"

	// EventTypeExpeditionStarted is sent when a new expedition starts recruiting
	EventTypeExpeditionStarted = "expedition.started"

	// EventTypeExpeditionTurn is sent for each turn of an expedition (live playback)
	EventTypeExpeditionTurn = "expedition.turn"

	// EventTypeExpeditionCompleted is sent when an expedition finishes
	EventTypeExpeditionCompleted = "expedition.completed"

	// EventTypeKeepalive is the keepalive ping event type
	EventTypeKeepalive = "keepalive"

	// EventTypeTimeoutApplied is sent when a timeout is applied to a user
	EventTypeTimeoutApplied = "timeout.applied"

	// EventTypeTimeoutCleared is sent when a timeout is cleared for a user
	EventTypeTimeoutCleared = "timeout.cleared"

	// EventTypeSubscription is sent for subscription lifecycle events
	EventTypeSubscription = "subscription"
)

// Log messages
const (
	LogMsgClientConnected    = "SSE client connected"
	LogMsgClientDisconnected = "SSE client disconnected"
	LogMsgEventBroadcast     = "Broadcasting SSE event"
	LogMsgWriteError         = "Failed to write SSE event"
	LogMsgFlushError         = "Failed to flush SSE response"
)
