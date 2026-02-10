package worker

// ============================================================================
// Log Messages - Worker Pool
// ============================================================================

// LogMsgWorkerJobFailed is logged when a worker fails to process a job
const LogMsgWorkerJobFailed = "Worker job failed"

// ============================================================================
// Log Messages - Gamble Worker
// ============================================================================

// Log messages for gamble worker operations
const (
	LogMsgFailedToCheckActiveGambleOnStartup = "Failed to check active gamble on startup"
	LogMsgSchedulingGambleExecution          = "Scheduling gamble execution"
	LogMsgExecutingScheduledGamble           = "Executing scheduled gamble"
	LogMsgFailedToExecuteGamble              = "Failed to execute gamble"
)

// ============================================================================
// Log Messages - Expedition Worker
// ============================================================================

// Log messages for expedition worker operations
const (
	LogMsgFailedToCheckActiveExpeditionOnStartup = "Failed to check active expedition on startup"
	LogMsgSchedulingExpeditionExecution          = "Scheduling expedition execution"
	LogMsgExecutingScheduledExpedition           = "Executing scheduled expedition"
	LogMsgFailedToExecuteExpedition              = "Failed to execute expedition"
)

// ============================================================================
// Log Messages - Daily Reset Worker
// ============================================================================

// Log messages for daily reset worker operations
const (
	LogMsgDailyResetStarting      = "Daily reset starting"
	LogMsgDailyResetCompleted     = "Daily reset completed"
	LogMsgDailyResetFailed        = "Daily reset failed"
	LogMsgDailyResetScheduled     = "Daily reset scheduled"
	LogMsgDailyResetStandby       = "Daily reset entered standby (long-range wait)"
	LogMsgDailyResetApproach      = "Daily reset scheduled (final approach)"
	LogMsgDailyResetManualTrigger = "Daily reset manually triggered"
)

// ============================================================================
// Test Configuration
// ============================================================================

// Test pool configuration values used in pool_test.go
const (
	TestWorkerCount           = 2
	TestQueueSize             = 10
	TestExpectedJobCount      = 2
	TestWorkerProcessWaitTime = 100 // milliseconds
)
