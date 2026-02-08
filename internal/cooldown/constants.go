package cooldown

import "time"

// =============================================================================
// Duration Constants
// =============================================================================

const (
	// DefaultCooldownDuration is the fallback cooldown when no specific duration is configured
	DefaultCooldownDuration = 5 * time.Minute
)

// =============================================================================
// Progression Feature Keys
// =============================================================================

const (
	// FeatureKeySearchCooldownReduction is the progression feature that reduces search cooldown
	FeatureKeySearchCooldownReduction = "search_cooldown_reduction"
)

// =============================================================================
// Hash Constants
// =============================================================================

const (
	// HashSeparator is the separator used when combining userID and action for advisory lock hashing
	HashSeparator = ":"

	// HashMaskPositiveInt64 is the bit mask to ensure advisory lock keys are positive int64 values
	// This masks the MSB to avoid overflow warnings and ensure PostgreSQL compatibility
	HashMaskPositiveInt64 = 0x7FFFFFFFFFFFFFFF
)

// =============================================================================
// SQL Query Constants
// =============================================================================

const (
	// SQLAdvisoryLock acquires a PostgreSQL advisory transaction lock
	SQLAdvisoryLock = "SELECT pg_advisory_xact_lock($1)"

	// SQLSelectLastUsed retrieves the last used timestamp for a user action
	SQLSelectLastUsed = `
		SELECT last_used_at
		FROM user_cooldowns
		WHERE user_id = $1 AND action_name = $2
	`

	// SQLDeleteCooldown removes a cooldown record for a user action
	SQLDeleteCooldown = `DELETE FROM user_cooldowns WHERE user_id = $1 AND action_name = $2`

	// SQLUpsertCooldown inserts or updates a cooldown timestamp
	SQLUpsertCooldown = `
		INSERT INTO user_cooldowns (user_id, action_name, last_used_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, action_name) DO UPDATE
		SET last_used_at = EXCLUDED.last_used_at
	`
)

// =============================================================================
// Error Message Constants
// =============================================================================

const (
	// ErrMsgCheckCooldownFailed is returned when checking cooldown state fails
	ErrMsgCheckCooldownFailed = "failed to check cooldown: %w"

	// ErrMsgBeginTransactionFailed is returned when transaction initialization fails
	ErrMsgBeginTransactionFailed = "failed to begin transaction: %w"

	// ErrMsgAcquireLockFailed is returned when advisory lock acquisition fails
	ErrMsgAcquireLockFailed = "failed to acquire advisory lock: %w"

	// ErrMsgGetCooldownTxFailed is returned when retrieving cooldown within transaction fails
	ErrMsgGetCooldownTxFailed = "failed to get cooldown within transaction: %w"

	// ErrMsgUpdateCooldownFailed is returned when updating cooldown timestamp fails
	ErrMsgUpdateCooldownFailed = "failed to update cooldown: %w"

	// ErrMsgCommitTransactionFailed is returned when transaction commit fails
	ErrMsgCommitTransactionFailed = "failed to commit cooldown transaction: %w"

	// ErrMsgResetCooldownFailed is returned when manual cooldown reset fails
	ErrMsgResetCooldownFailed = "failed to reset cooldown: %w"

	// ErrMsgGetLastUsedFailed is returned when retrieving last used timestamp fails
	ErrMsgGetLastUsedFailed = "failed to get last used: %w"
)

// =============================================================================
// Log Message Constants
// =============================================================================

const (
	// LogMsgDevModeBypass is logged when dev mode bypasses cooldown enforcement
	LogMsgDevModeBypass = "DEV_MODE: Bypassing cooldown enforcement"

	// LogMsgRaceConditionDetected is logged when concurrent cooldown requests create a race condition
	LogMsgRaceConditionDetected = "Race condition detected - concurrent request on cooldown"

	// LogMsgCooldownEnforced is logged when cooldown is successfully enforced and updated
	LogMsgCooldownEnforced = "Cooldown enforced successfully"
)

// =============================================================================
// Error Message Format Strings (for ErrOnCooldown.Error())
// =============================================================================

const (
	// ErrFmtCooldownWithMinutes formats cooldown error with minutes and seconds
	ErrFmtCooldownWithMinutes = "You can %s again in %dm %ds"

	// ErrFmtCooldownSecondsOnly formats cooldown error with seconds only
	ErrFmtCooldownSecondsOnly = "You can %s again in %ds"
)

// =============================================================================
// Time Conversion Constants
// =============================================================================

const (
	// SecondsPerMinute is used for time duration calculations
	SecondsPerMinute = 60
)
