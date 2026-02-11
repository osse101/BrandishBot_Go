package handler

// Generic HTTP error messages for client responses.
// These messages intentionally do not expose internal error details for security reasons.
// Both handlers and tests should reference these constants to maintain consistency.
const (
	// HTTP status messages
	ErrMsgMethodNotAllowed      = "Method not allowed"
	ErrMsgInvalidRequest        = "Invalid request body"
	ErrMsgInvalidRequestSummary = "Invalid request"

	// Query parameter error messages
	ErrMsgMissingQueryParam = "Missing %s query parameter"

	// Inventory operation error messages
	ErrMsgAddItemFailed      = "Failed to add item"
	ErrMsgRemoveItemFailed   = "Failed to remove item"
	ErrMsgGiveItemFailed     = "Failed to give item"
	ErrMsgGetInventoryFailed = "Failed to get inventory"

	// Economy operation error messages
	ErrMsgSellItemFailed = "Failed to sell item"
	ErrMsgBuyItemFailed  = "Failed to buy item"

	// Item usage error messages
	ErrMsgUseItemFailed = "Failed to use item"

	// Feature/progression error messages
	ErrMsgFeatureCheckFailed         = "Failed to check feature availability"
	ErrMsgFeatureLocked              = "Feature locked"
	ErrMsgGetProgressionTreeFailed   = "Failed to retrieve progression tree"
	ErrMsgGetAvailableUnlocksFailed  = "Failed to retrieve available unlocks"
	ErrMsgGetProgressionStatusFailed = "Failed to retrieve progression status"
	ErrMsgGetEngagementDataFailed    = "Failed to retrieve engagement data"
	ErrMsgGetLeaderboardFailed       = "Failed to retrieve leaderboard"
	ErrMsgGetVelocityMetricsFailed   = "Failed to retrieve velocity metrics"
	ErrMsgGetVotingSessionFailed     = "Failed to retrieve voting session"
	ErrMsgGetUnlockProgressFailed    = "Failed to retrieve unlock progress"

	// Crafting/upgrade error messages
	ErrMsgDisassembleItemFailed = "Failed to disassemble item"
	ErrMsgUpgradeItemFailed     = "Failed to upgrade item"

	// Search error messages
	ErrMsgSearchFailed = "Failed to perform search"

	// User management error messages
	ErrMsgRegisterUserFailed    = "Failed to register user"
	ErrMsgUsernameRequired      = "Username is required for new users"
	ErrMsgUserNotFoundHTTP      = "user not found"
	ErrMsgGetJobsFailed         = "Failed to retrieve jobs"
	ErrMsgGetUserJobsFailed     = "Failed to retrieve user jobs"
	ErrMsgMissingRequiredFields = "Missing required fields"

	// Message handling error messages
	ErrMsgHandleMessageFailed = "Failed to handle message"

	// Admin error messages
	ErrMsgReloadConfigFailed          = "Failed to reload configuration"
	ErrMsgAmountMustBePositive        = "amount must be positive"
	ErrMsgAmountExceedsMax            = "amount exceeds maximum (10000)"
	ErrMsgPlatformUsernameJobRequired = "platform, username, and job_key are required"

	// Gamble error messages
	ErrMsgInvalidGambleID    = "Invalid gamble ID"
	ErrMsgGambleNotFoundHTTP = "Gamble not found"

	// Inventory filter error messages
	ErrMsgInvalidFilterType = "Invalid filter type '%s'. Valid options: upgrade, sellable, consumable"
	ErrMsgFilterLocked      = "Filter '%s' is locked. Unlock it in the progression tree."

	// Parameter validation error messages
	ErrMsgInvalidLimit = "Invalid limit parameter"

	// Feature lock reason constants
	FeatureLockReasonProgression = "progression_locked"
	MsgLockedNodesFormat         = "LOCKED_NODES: %s"

	// Compost error messages
	ErrMsgCompostBinFull          = "Compost bin is full"
	ErrMsgCompostNotCompostable   = "That item cannot be composted"
	ErrMsgCompostMustHarvest      = "Bin is ready - harvest before depositing more"
	ErrMsgCompostNothingToHarvest = "Nothing to harvest"
)

// Success messages for API responses
// These are user-facing success messages returned in JSON responses
const (
	// Inventory operation success messages
	MsgItemAddedSuccess       = "Item added successfully"
	MsgItemTransferredSuccess = "Item transferred successfully"

	// Event and stats success messages
	MsgEventRecordedSuccess = "Event recorded successfully"

	// Progression success messages
	MsgAlreadyVoted              = "You have already voted"
	MsgVoteRecordedSuccess       = "Vote recorded successfully"
	MsgAllNodesUnlockedSuccess   = "All nodes unlocked successfully"
	MsgProgressionResetSuccess   = "Progression tree reset successfully"
	MsgVotingSessionStartSuccess = "Voting session started successfully"
	MsgContributionAddedSuccess  = "Contribution added successfully"
	MsgWeightCacheInvalidated    = "Engagement weight cache invalidated successfully"
	MsgNodeUnlockedSuccess       = "Node unlocked successfully"
	MsgNodeRelockedSuccess       = "Node relocked successfully"
	MsgInstantUnlockSuccess      = "Instant unlock successful"
	MsgVotingEndedSuccess        = "Voting ended successfully"

	// Gamble success messages
	MsgJoinedGambleSuccess = "Successfully joined gamble"

	// Linking success messages
	MsgConfirmWithinSeconds = "Confirm within 60 seconds"
	MsgPlatformUnlinked     = "Platform unlinked"

	// Info messages
	MsgNoActiveVotingSession  = "No active voting session"
	MsgNoActiveUnlockProgress = "No active unlock progress"

	// Admin success messages
	MsgConfigReloadedSuccess = "Alias configuration reloaded successfully"

	// Compost success messages
	MsgCompostDepositSuccess = "Items deposited into compost bin!"
	MsgCompostBinEmpty       = "Bin is empty"
)
