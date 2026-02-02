package stats

import "time"

// ============================================================================
// Period Names
// ============================================================================

// Supported time period identifiers for stats queries
const (
	PeriodHourly  = "hourly"
	PeriodDaily   = "daily"
	PeriodWeekly  = "weekly"
	PeriodMonthly = "monthly"
	PeriodYearly  = "yearly"
	PeriodAll     = "all"
)

// ============================================================================
// Query Limits
// ============================================================================

// DefaultLeaderboardLimit is the default number of entries to return in
// leaderboard queries when no limit is specified or limit <= 0
const DefaultLeaderboardLimit = 10

// StreakEventQueryLimit is the number of recent streak events to fetch when
// checking or calculating daily streaks
const StreakEventQueryLimit = 1

// ============================================================================
// Streak Calculation
// ============================================================================

// DayOffsetYesterday is the number of days to subtract from current time
// to calculate yesterday's date for streak validation
const DayOffsetYesterday = -1

// AllTimeStartYear is the year used as the baseline for "all time" stats queries
const AllTimeStartYear = 2000

// AllTimeStartMonth is the month used as the baseline for "all time" stats queries
const AllTimeStartMonth = time.January

// AllTimeStartDay is the day used as the baseline for "all time" stats queries
const AllTimeStartDay = 1

// ============================================================================
// Metadata Keys
// ============================================================================

// MetadataKeyStreak is the event metadata key used to store/retrieve the
// current streak count in daily streak events
const MetadataKeyStreak = "streak"

// ============================================================================
// Error Messages
// ============================================================================

// Validation error messages
const (
	ErrMsgUserIDRequired = "user ID is required"
)

// Database operation error messages
const (
	ErrMsgGetStreakEventsFailed    = "failed to get streak events: %w"
	ErrMsgRecordStreakEventFailed  = "failed to record streak event: %w"
	ErrMsgGetUserEventCountsFailed = "failed to get user event counts: %w"
	ErrMsgGetTotalEventCountFailed = "failed to get total event count: %w"
	ErrMsgGetEventCountsFailed     = "failed to get event counts: %w"
	ErrMsgGetLeaderboardFailed     = "failed to get leaderboard: %w"
)

// General operation error messages
const (
	ErrMsgRecordEventFailed = "failed to record event: %w"
)

// ============================================================================
// Log Messages
// ============================================================================

// Service operation log messages
const (
	LogMsgEventRecorded        = "Event recorded"
	LogMsgRetrievedUserStats   = "Retrieved user stats"
	LogMsgRetrievedSystemStats = "Retrieved system stats"
	LogMsgRetrievedLeaderboard = "Retrieved leaderboard"
)

// Error log messages
const (
	LogMsgFailedToRecordEvent        = "Failed to record event"
	LogMsgFailedToCheckDailyStreak   = "Failed to check daily streak"
	LogMsgFailedToGetUserEventCounts = "Failed to get user event counts"
	LogMsgFailedToGetTotalEventCount = "Failed to get total event count"
	LogMsgFailedToGetEventCounts     = "Failed to get event counts"
	LogMsgFailedToGetLeaderboard     = "Failed to get leaderboard"
)
