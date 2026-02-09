package linking

import "time"

// ============================================================================
// Token Configuration
// ============================================================================

const (
	// TokenLength is the number of characters in a link token
	TokenLength = 6

	// TokenExpiration is how long a link token remains valid
	TokenExpiration = 10 * time.Minute

	// TokenRandomBytes is the number of random bytes to generate for token creation
	TokenRandomBytes = 4
)

// ============================================================================
// Token States
// ============================================================================

const (
	// StatePending indicates a token is waiting for Step 2 (claim)
	StatePending = "pending"

	// StateClaimed indicates a token is waiting for Step 3 (confirm)
	StateClaimed = "claimed"

	// StateConfirmed indicates the link is complete
	StateConfirmed = "confirmed"

	// StateExpired indicates the token has timed out
	StateExpired = "expired"
)

// ============================================================================
// Unlink Configuration
// ============================================================================

const (
	// UnlinkTimeout is how long an unlink confirmation remains valid
	UnlinkTimeout = 60 * time.Second

	// UnlinkCacheKeyFormat is the format string for unlink cache keys
	UnlinkCacheKeyFormat = "%s:%s:%s"
)

// ============================================================================
// Error Messages (Client-Facing)
// ============================================================================

const (
	// ErrMsgTokenNotFound is returned when a token cannot be found
	ErrMsgTokenNotFound = "token not found"

	// ErrMsgTokenAlreadyUsed is returned when attempting to claim an already-used token
	ErrMsgTokenAlreadyUsed = "token already used or expired"

	// ErrMsgTokenExpired is returned when a token has expired
	ErrMsgTokenExpired = "token expired"

	// ErrMsgLinkTokenExpired is returned when confirming an expired link
	//nolint:gosec // G101: False positive - this is an error message, not a credential
	ErrMsgLinkTokenExpired = "link token expired"

	// ErrMsgCannotLinkSameAccount is returned when source and target are identical
	ErrMsgCannotLinkSameAccount = "cannot link same account to itself"

	// ErrMsgNoPendingLink is returned when no claimed token exists to confirm
	ErrMsgNoPendingLink = "no pending link to confirm"

	// ErrMsgNoPendingUnlink is returned when no unlink confirmation exists
	ErrMsgNoPendingUnlink = "no pending unlink confirmation"

	// ErrMsgUserNotFound is returned when a user cannot be found for unlinking
	ErrMsgUserNotFound = "user not found"
)

// ============================================================================
// Error Context Messages (Wrapped Errors)
// ============================================================================

const (
	// ErrContextFailedToGenerateToken wraps token generation errors
	ErrContextFailedToGenerateToken = "failed to generate token: %w"

	// ErrContextFailedToCreateToken wraps token creation errors
	ErrContextFailedToCreateToken = "failed to create token: %w"

	// ErrContextFailedToClaimToken wraps token claim errors
	ErrContextFailedToClaimToken = "failed to claim token: %w"

	// ErrContextFailedToRegisterSourceUser wraps source user registration errors
	ErrContextFailedToRegisterSourceUser = "failed to register source user: %w"

	// ErrContextFailedToLinkAccounts wraps account linking errors
	ErrContextFailedToLinkAccounts = "failed to link accounts: %w"

	// ErrContextFailedToUpdateTokenState wraps token state update errors
	ErrContextFailedToUpdateTokenState = "failed to update token state: %w"

	// ErrContextFailedToMergeAccounts wraps account merge errors
	ErrContextFailedToMergeAccounts = "failed to merge accounts: %w"

	// ErrContextFailedToUnlink wraps unlink operation errors
	ErrContextFailedToUnlink = "failed to unlink: %w"
)

// ============================================================================
// Log Messages
// ============================================================================

const (
	// LogMsgFailedToInvalidateOldTokens is logged when old token cleanup fails
	LogMsgFailedToInvalidateOldTokens = "Failed to invalidate old tokens"

	// LogMsgLinkTokenCreated is logged when a new link token is generated
	//nolint:gosec // G101: False positive - this is a log message, not a credential
	LogMsgLinkTokenCreated = "Link token created"

	// LogMsgFailedToExpireToken is logged when token expiration update fails
	LogMsgFailedToExpireToken = "Failed to expire token"

	// LogMsgLinkTokenClaimed is logged when a token is claimed by target platform
	LogMsgLinkTokenClaimed = "Link token claimed"

	// LogMsgAccountsLinked is logged when accounts are linked successfully
	LogMsgAccountsLinked = "Accounts linked"

	// LogMsgAccountsMerged is logged when two existing users are merged
	LogMsgAccountsMerged = "Accounts merged"

	// LogMsgPlatformUnlinked is logged when a platform is unlinked from a user
	LogMsgPlatformUnlinked = "Platform unlinked"
)

// ============================================================================
// Log Context Keys
// ============================================================================

const (
	// LogKeyPlatform is the log key for platform identifier
	LogKeyPlatform = "platform"

	// LogKeyToken is the log key for token string
	LogKeyToken = "token"

	// LogKeyTargetPlatform is the log key for target platform identifier
	LogKeyTargetPlatform = "target_platform"

	// LogKeyUserID is the log key for user ID
	LogKeyUserID = "user_id"

	// LogKeyPlatforms is the log key for platforms list
	LogKeyPlatforms = "platforms"

	// LogKeyPrimaryID is the log key for primary user ID in merge
	LogKeyPrimaryID = "primary_id"

	// LogKeySecondaryID is the log key for secondary user ID in merge
	LogKeySecondaryID = "secondary_id"

	// LogKeyError is the log key for error values
	LogKeyError = "error"
)
