package streamerbot

import "time"

// Default configuration values
const (
	// DefaultURL is the default WebSocket URL for Streamer.bot
	DefaultURL = "ws://127.0.0.1:8080/"

	// DefaultReconnectDelay is the initial delay before attempting to reconnect
	DefaultReconnectDelay = 1 * time.Second

	// MaxReconnectDelay is the maximum delay between reconnection attempts
	MaxReconnectDelay = 30 * time.Second

	// ReconnectMultiplier is the multiplier for exponential backoff
	ReconnectMultiplier = 2.0

	// MaxConsecutiveFailures is the maximum number of connection attempts before giving up
	MaxConsecutiveFailures = 10

	// PingInterval is how often to send ping frames to keep connection alive
	PingInterval = 30 * time.Second

	// WriteTimeout is the timeout for writing messages
	WriteTimeout = 10 * time.Second

	// ReadBufferSize is the WebSocket read buffer size
	ReadBufferSize = 4096

	// WriteBufferSize is the WebSocket write buffer size
	WriteBufferSize = 4096
)

// Request types for Streamer.bot WebSocket API
const (
	RequestDoAction     = "DoAction"
	RequestAuthenticate = "Authenticate"
	RequestSubscribe    = "Subscribe"
	RequestGetInfo      = "GetInfo"
)

// Action names for BrandishBot events
const (
	ActionJobLevelUp         = "BrandishBot_JobLevelUp"
	ActionVotingStarted      = "BrandishBot_VotingStarted"
	ActionCycleCompleted     = "BrandishBot_CycleCompleted"
	ActionAllUnlocked        = "BrandishBot_AllUnlocked"
	ActionGambleCompleted    = "BrandishBot_GambleCompleted"
	ActionSlotsResult        = "BrandishBot_SlotsResult"
	ActionTimeoutUpdate      = "BrandishBot_TimeoutUpdate"
	ActionSubscriptionUpdate = "BrandishBot_SubscriptionUpdate"
)

// Response status values
const (
	StatusOK    = "ok"
	StatusError = "error"
)

// Log messages
const (
	LogMsgConnecting    = "Connecting to Streamer.bot WebSocket"
	LogMsgConnected     = "Connected to Streamer.bot WebSocket"
	LogMsgDisconnected  = "Disconnected from Streamer.bot WebSocket"
	LogMsgReconnecting  = "Reconnecting to Streamer.bot WebSocket"
	LogMsgAuthRequired  = "Streamer.bot requires authentication"
	LogMsgAuthSuccess   = "Streamer.bot authentication successful"
	LogMsgAuthFailed    = "Streamer.bot authentication failed"
	LogMsgSendingAction = "Sending DoAction to Streamer.bot"
	LogMsgActionSent    = "DoAction sent to Streamer.bot"
	LogMsgActionFailed  = "Failed to send DoAction to Streamer.bot"
	LogMsgReadError     = "Error reading from Streamer.bot WebSocket"
	LogMsgWriteError    = "Error writing to Streamer.bot WebSocket"
	LogMsgClientStopped = "Streamer.bot client stopped"
	LogMsgEventReceived = "Received event from internal bus"
	LogMsgGivingUp      = "Streamer.bot connection failed too many times, entering dormant mode"
	LogMsgDormantRetry  = "Streamer.bot dormant, retrying connection due to incoming event"
)
