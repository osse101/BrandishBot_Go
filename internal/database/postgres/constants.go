package postgres

// PostgreSQL Error Codes
const (
	// PgErrorCodeUniqueViolation is the PostgreSQL error code for unique constraint violations
	PgErrorCodeUniqueViolation = "23505"
)

// Event Versions
const (
	EventVersion1_0 = "1.0"
)

// Engagement Metric Types - used in progression engagement tracking
const (
	EngagementMetricMessage     = "message"
	EngagementMetricCommand     = "command"
	EngagementMetricItemCrafted = "item_crafted"
	EngagementMetricItemUsed    = "item_used"
)

// Default Engagement Weights - fallback values when DB weights are unavailable
const (
	DefaultEngagementWeightMessage     = 1.0
	DefaultEngagementWeightCommand     = 2.0
	DefaultEngagementWeightItemCrafted = 3.0
	DefaultEngagementWeightItemUsed    = 1.5
)

// Inventory Constants
const (
	// EmptyInventoryJSON is the default JSON structure for a new/empty inventory
	EmptyInventoryJSON = `{"slots": []}`
)

// Error Messages - Transaction Operations
const (
	ErrMsgFailedToBeginTransaction       = "failed to begin transaction"
	ErrMsgFailedToBeginGambleTransaction = "failed to begin gamble transaction"
	ErrMsgFailedToBeginTx                = "failed to begin tx for saving items"
	ErrMsgFailedToCommitTransaction      = "failed to commit transaction"
)

// Error Messages - User Operations
const (
	ErrMsgInvalidUserID               = "invalid user id"
	ErrMsgFailedToInsertUser          = "failed to insert user"
	ErrMsgFailedToUpdateUser          = "failed to update user"
	ErrMsgFailedToGetUserCoreData     = "failed to get user core data"
	ErrMsgFailedToGetUserByUsername   = "failed to get user by username"
	ErrMsgFailedToGetUserLinks        = "failed to get user links"
	ErrMsgUserNotFound                = "user not found"
	ErrMsgFailedToUpdateUserTimestamp = "failed to update user timestamp"
	ErrMsgFailedToDeleteUser          = "failed to delete user"
	ErrMsgInvalidPrimaryUserID        = "invalid primary user id"
	ErrMsgInvalidSecondaryUserID      = "invalid secondary user id"
)

// Error Messages - Inventory Operations
const (
	ErrMsgFailedToGetInventory             = "failed to get inventory"
	ErrMsgFailedToGetInventoryForUpdate    = "failed to get inventory for update"
	ErrMsgFailedToUpdateInventory          = "failed to update inventory"
	ErrMsgFailedToMarshalInventory         = "failed to marshal inventory"
	ErrMsgFailedToUnmarshalInventory       = "failed to unmarshal inventory"
	ErrMsgFailedToEnsureInventoryRow       = "failed to ensure inventory row"
	ErrMsgFailedToDeleteInventory          = "failed to delete inventory"
	ErrMsgFailedToDeleteSecondaryInventory = "failed to delete secondary inventory"
	ErrMsgFailedToUpdatePrimaryInventory   = "failed to update primary inventory"
)

// Error Messages - Item Operations
const (
	ErrMsgFailedToGetItemByName       = "failed to get item by name"
	ErrMsgFailedToGetItemByID         = "failed to get item by id"
	ErrMsgFailedToGetItemByPublicName = "failed to get item by public name"
	ErrMsgFailedToGetItemsByIDs       = "failed to get items by ids"
	ErrMsgFailedToGetItemsByNames     = "failed to get items by names"
	ErrMsgFailedToGetAllItems         = "failed to get all items"
	ErrMsgFailedToInsertItem          = "failed to insert item"
	ErrMsgFailedToUpdateItem          = "failed to update item"
	ErrMsgFailedToGetAllItemTypes     = "failed to get all item types"
	ErrMsgFailedToInsertItemType      = "failed to insert item type"
	ErrMsgFailedToClearItemTags       = "failed to clear item tags"
	ErrMsgFailedToAssignItemTag       = "failed to assign item tag"
	ErrMsgItemNotFound                = "item not found"
)

// Error Messages - Platform Operations
const (
	ErrMsgFailedToGetPlatformID      = "failed to get platform id"
	ErrMsgFailedToUpsertLink         = "failed to upsert link"
	ErrMsgFailedToUpdateTwitchLink   = "failed to update twitch link"
	ErrMsgFailedToUpdateYouTubeLink  = "failed to update youtube link"
	ErrMsgFailedToUpdateDiscordLink  = "failed to update discord link"
	ErrMsgFailedToDeletePlatformLink = "failed to delete platform link"
)

// Error Messages - Cooldown Operations
const (
	ErrMsgFailedToGetCooldown         = "failed to get cooldown"
	ErrMsgFailedToGetCooldownWithLock = "failed to get cooldown with lock"
	ErrMsgFailedToUpdateCooldown      = "failed to update cooldown"
)

// Error Messages - Recipe Operations
const (
	ErrMsgFailedToGetRecipeByTargetItemID      = "failed to get recipe by target item id"
	ErrMsgFailedToUnmarshalBaseCost            = "failed to unmarshal base cost"
	ErrMsgFailedToUnlockRecipe                 = "failed to unlock recipe"
	ErrMsgFailedToQueryUnlockedRecipes         = "failed to query unlocked recipes"
	ErrMsgFailedToQueryAllRecipes              = "failed to query all recipes"
	ErrMsgFailedToQueryDisassembleRecipe       = "failed to query disassemble recipe"
	ErrMsgFailedToQueryDisassembleOutputs      = "failed to query disassemble outputs"
	ErrMsgFailedToQueryAssociatedUpgradeRecipe = "failed to query associated upgrade recipe"
	ErrMsgFailedToQueryCraftingRecipes         = "failed to query crafting recipes"
	ErrMsgFailedToQueryDisassembleRecipes      = "failed to query disassemble recipes"
	ErrMsgFailedToGetOutputsForRecipe          = "failed to get outputs for recipe"
	ErrMsgFailedToQueryCraftingRecipeByKey     = "failed to query crafting recipe by key"
	ErrMsgFailedToQueryDisassembleRecipeByKey  = "failed to query disassemble recipe by key"
	ErrMsgFailedToInsertCraftingRecipe         = "failed to insert crafting recipe"
	ErrMsgFailedToInsertDisassembleRecipe      = "failed to insert disassemble recipe"
	ErrMsgFailedToUpdateCraftingRecipe         = "failed to update crafting recipe"
	ErrMsgFailedToUpdateDisassembleRecipe      = "failed to update disassemble recipe"
	ErrMsgFailedToClearDisassembleOutputs      = "failed to clear disassemble outputs"
	ErrMsgFailedToInsertDisassembleOutput      = "failed to insert disassemble output"
	ErrMsgFailedToUpsertRecipeAssociation      = "failed to upsert recipe association"
	ErrMsgFailedToMarshalBaseCost              = "failed to marshal base cost"
	ErrMsgNoAssociatedUpgradeRecipeFound       = "no associated upgrade recipe found for disassemble recipe %d | %w"
)

// Error Messages - Economy Operations
const (
	ErrMsgFailedToQuerySellableItems = "failed to query sellable items"
	ErrMsgFailedToQueryBuyableItems  = "failed to query buyable items"
)

// Error Messages - Gamble Operations
const (
	ErrMsgFailedToCreateGamble      = "failed to create gamble"
	ErrMsgFailedToGetGamble         = "failed to get gamble"
	ErrMsgFailedToGetParticipants   = "failed to get participants"
	ErrMsgFailedToUnmarshalBets     = "failed to unmarshal bets"
	ErrMsgFailedToMarshalBets       = "failed to marshal bets"
	ErrMsgFailedToJoinGamble        = "failed to join gamble"
	ErrMsgFailedToUpdateGambleState = "failed to update gamble state"
	ErrMsgFailedToInsertOpenedItem  = "failed to insert opened item"
	ErrMsgFailedToCompleteGamble    = "failed to complete gamble"
	ErrMsgFailedToGetActiveGamble   = "failed to get active gamble"
)

// Error Messages - Job Operations
const (
	ErrMsgFailedToQueryJobs         = "failed to query jobs"
	ErrMsgFailedToGetJob            = "failed to get job"
	ErrMsgFailedToQueryUserJobs     = "failed to query user jobs"
	ErrMsgFailedToGetUserJob        = "failed to get user job"
	ErrMsgFailedToUpsertUserJob     = "failed to upsert user job"
	ErrMsgFailedToRecordXPEvent     = "failed to record XP event"
	ErrMsgFailedToQueryJobBonuses   = "failed to query job bonuses"
	ErrMsgFailedToConvertBonusValue = "failed to convert bonus value for job %d"
	ErrMsgFailedToResetDailyXP      = "failed to reset daily XP"
	ErrMsgFailedToMarshalMetadata   = "failed to marshal metadata"
	ErrMsgJobNotFound               = "job not found: %s | %w"
)

// Error Messages - Stats Operations
const (
	ErrMsgFailedToMarshalEventData     = "failed to marshal event data"
	ErrMsgFailedToInsertEvent          = "failed to insert event"
	ErrMsgFailedToQueryEvents          = "failed to query events"
	ErrMsgFailedToQueryUserEvents      = "failed to query user events"
	ErrMsgFailedToUnmarshalEventData   = "failed to unmarshal event data"
	ErrMsgFailedToQueryTopUsers        = "failed to query top users"
	ErrMsgFailedToQueryEventCounts     = "failed to query event counts"
	ErrMsgFailedToQueryUserEventCounts = "failed to query user event counts"
	ErrMsgFailedToGetTotalEventCount   = "failed to get total event count"
)

// Error Messages - Progression Operations
const (
	ErrMsgFailedToGetNodeByKey            = "failed to get node by key"
	ErrMsgFailedToGetNodeByID             = "failed to get node by ID"
	ErrMsgFailedToQueryNodes              = "failed to query nodes"
	ErrMsgFailedToQueryPrerequisites      = "failed to query prerequisites"
	ErrMsgFailedToQueryDependents         = "failed to query dependents"
	ErrMsgFailedToInsertNode              = "failed to insert node"
	ErrMsgFailedToUpdateNode              = "failed to update node"
	ErrMsgFailedToGetUnlock               = "failed to get unlock"
	ErrMsgFailedToQueryUnlocks            = "failed to query unlocks"
	ErrMsgFailedToUnlockNode              = "failed to unlock node"
	ErrMsgFailedToRelockNode              = "failed to relock node"
	ErrMsgFailedToGetActiveVoting         = "failed to get active voting"
	ErrMsgFailedToStartVoting             = "failed to start voting"
	ErrMsgFailedToGetVoting               = "failed to get voting"
	ErrMsgFailedToIncrementVote           = "failed to increment vote"
	ErrMsgFailedToEndVoting               = "failed to end voting"
	ErrMsgFailedToRecordUserVote          = "failed to record user vote"
	ErrMsgFailedToUnlockUserProgression   = "failed to unlock user progression"
	ErrMsgFailedToQueryUserProgressions   = "failed to query user progressions"
	ErrMsgFailedToRecordEngagement        = "failed to record engagement"
	ErrMsgFailedToGetWeights              = "failed to get weights"
	ErrMsgFailedToQueryEngagementMetrics  = "failed to query engagement metrics"
	ErrMsgFailedToQueryUserEngagement     = "failed to query user engagement"
	ErrMsgFailedToQueryEngagementWeights  = "failed to query engagement weights"
	ErrMsgFailedToCountUnlocks            = "failed to count unlocks"
	ErrMsgFailedToGetEngagementScore      = "failed to get engagement score"
	ErrMsgFailedToRecordReset             = "failed to record reset"
	ErrMsgFailedToClearUnlocks            = "failed to clear unlocks"
	ErrMsgFailedToClearVoting             = "failed to clear voting"
	ErrMsgFailedToClearUserVotes          = "failed to clear user votes"
	ErrMsgFailedToClearUserProgression    = "failed to clear user progression"
	ErrMsgFailedToUnmarshalMetadata       = "failed to unmarshal metadata"
	ErrMsgFailedToGetNodeByFeatureKey     = "failed to get node by feature key"
	ErrMsgFailedToUnmarshalModifierConfig = "failed to unmarshal modifier config"
	ErrMsgFailedToQueryDailyTotals        = "failed to query daily totals"
	ErrMsgFailedToGetSyncMetadata         = "failed to get sync metadata"
	ErrMsgFailedToUpsertSyncMetadata      = "failed to upsert sync metadata"
	ErrMsgSyncMetadataNotFound            = "sync metadata not found"
)

// Error Messages - Voting Session Operations
const (
	ErrMsgFailedToCreateVotingSession   = "failed to create voting session"
	ErrMsgFailedToAddVotingOption       = "failed to add voting option"
	ErrMsgFailedToGetActiveSession      = "failed to get active session"
	ErrMsgFailedToGetSession            = "failed to get session"
	ErrMsgFailedToGetSessionOptions     = "failed to get session options"
	ErrMsgFailedToIncrementOptionVote   = "failed to increment option vote"
	ErrMsgFailedToEndVotingSession      = "failed to end voting session"
	ErrMsgFailedToGetSessionVoters      = "failed to get session voters"
	ErrMsgFailedToRecordUserSessionVote = "failed to record user session vote"
)

// Error Messages - Unlock Progress Operations
const (
	ErrMsgFailedToCreateUnlockProgress     = "failed to create unlock progress"
	ErrMsgFailedToGetActiveUnlockProgress  = "failed to get active unlock progress"
	ErrMsgFailedToAddContribution          = "failed to add contribution"
	ErrMsgFailedToSetUnlockTarget          = "failed to set unlock target"
	ErrMsgFailedToCompleteUnlock           = "failed to complete unlock"
	ErrMsgFailedToCreateNextUnlockProgress = "failed to create next unlock progress"
)

// Error Messages - Leaderboard Operations
const (
	ErrMsgFailedToGetContributionLeaderboard = "failed to get contribution leaderboard"
)

// Error Messages - Junction Table Operations
const (
	ErrMsgFailedToClearPrerequisites = "failed to clear prerequisites"
	ErrMsgFailedToInsertPrerequisite = "failed to insert prerequisite"
)

// Error Messages - Linking Operations
const (
	ErrMsgTokenNotFound                    = "token not found"
	ErrMsgNoClaimedTokenFound              = "no claimed token found"
	ErrMsgFailedToGetPlatformIDForPlatform = "failed to get platform id for %s"
)

// Error Messages - Conversion Operations
const (
	ErrMsgFailedToConvertNumericToFloat64 = "failed to convert numeric to float64"
)

// Log Messages - Job Operations
const (
	LogMsgResetDailyXP = "Reset daily XP"
)

// Log Messages - Event Publishing
const (
	LogMsgFailedToPublishNodeUnlockedEvent = "failed to publish node unlocked event"
	LogMsgFailedToPublishNodeRelockedEvent = "failed to publish node relocked event"
)

// Log Messages - Engagement Weights
const (
	LogMsgFailedToConvertWeightUsingDefault = "failed to convert weight for metric, using default"
)

// Database Operation Descriptions
const (
	OpDescGetInventory          = "get inventory"
	OpDescGetInventoryForUpdate = "get inventory for update"
)
