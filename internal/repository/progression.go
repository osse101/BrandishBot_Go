package repository

import (
	"context"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Progression defines database operations for progression system
type Progression interface {
	// Node operations
	GetNodeByKey(ctx context.Context, nodeKey string) (*domain.ProgressionNode, error)
	GetNodeByID(ctx context.Context, id int) (*domain.ProgressionNode, error)
	GetAllNodes(ctx context.Context) ([]*domain.ProgressionNode, error)
	GetNodeByFeatureKey(ctx context.Context, featureKey string) (*domain.ProgressionNode, int, error) // Returns node with ModifierConfig and current unlock level

	// Prerequisites operations (v2.0 - junction table)
	GetPrerequisites(ctx context.Context, nodeID int) ([]*domain.ProgressionNode, error) // Get prerequisites FOR this node
	GetDependents(ctx context.Context, nodeID int) ([]*domain.ProgressionNode, error)    // Get nodes that depend ON this node

	// Unlock operations
	GetUnlock(ctx context.Context, nodeID int, level int) (*domain.ProgressionUnlock, error)
	GetAllUnlocks(ctx context.Context) ([]*domain.ProgressionUnlock, error)
	IsNodeUnlocked(ctx context.Context, nodeKey string, level int) (bool, error)
	UnlockNode(ctx context.Context, nodeID int, level int, unlockedBy string, engagementScore int) error
	RelockNode(ctx context.Context, nodeID int, level int) error

	// Voting session operations
	CreateVotingSession(ctx context.Context) (int, error)
	AddVotingOption(ctx context.Context, sessionID, nodeID, targetLevel int) error
	GetActiveSession(ctx context.Context) (*domain.ProgressionVotingSession, error)
	GetActiveOrFrozenSession(ctx context.Context) (*domain.ProgressionVotingSession, error) // Get session with status 'voting' or 'frozen'
	GetMostRecentSession(ctx context.Context) (*domain.ProgressionVotingSession, error)    // Bug #1: Get most recent session (any status)
	GetSessionByID(ctx context.Context, sessionID int) (*domain.ProgressionVotingSession, error)
	IncrementOptionVote(ctx context.Context, optionID int) error
	EndVotingSession(ctx context.Context, sessionID int, winningOptionID *int) error
	FreezeVotingSession(ctx context.Context, sessionID int) error  // Pause voting until unlock completes
	ResumeVotingSession(ctx context.Context, sessionID int) error  // Resume frozen voting session
	GetSessionVoters(ctx context.Context, sessionID int) ([]string, error)
	HasUserVotedInSession(ctx context.Context, userID string, sessionID int) (bool, error)
	RecordUserSessionVote(ctx context.Context, userID string, sessionID, optionID, nodeID int) error

	// Unlock progress tracking
	CreateUnlockProgress(ctx context.Context) (int, error)
	GetActiveUnlockProgress(ctx context.Context) (*domain.UnlockProgress, error)
	AddContribution(ctx context.Context, progressID int, amount int) error
	SetUnlockTarget(ctx context.Context, progressID int, nodeID int, targetLevel int, sessionID int) error
	CompleteUnlock(ctx context.Context, progressID int, rolloverPoints int) (int, error)

	// User progression
	UnlockUserProgression(ctx context.Context, userID string, progressionType string, key string, metadata map[string]interface{}) error
	IsUserProgressionUnlocked(ctx context.Context, userID string, progressionType string, key string) (bool, error)
	GetUserProgressions(ctx context.Context, userID string, progressionType string) ([]*domain.UserProgression, error)

	// Contribution tracking
	RecordEngagement(ctx context.Context, metric *domain.EngagementMetric) error
	GetEngagementScore(ctx context.Context, since *time.Time) (int, error)
	GetUserEngagement(ctx context.Context, userID string) (*domain.ContributionBreakdown, error)
	GetContributionLeaderboard(ctx context.Context, limit int) ([]domain.ContributionLeaderboardEntry, error)
	GetEngagementWeights(ctx context.Context) (map[string]float64, error)
	GetDailyEngagementTotals(ctx context.Context, since time.Time) (map[time.Time]int, error)

	// Reset operations
	ResetTree(ctx context.Context, resetBy string, reason string, preserveUserData bool) error
	RecordReset(ctx context.Context, reset *domain.ProgressionReset) error

	// Sync metadata operations
	GetSyncMetadata(ctx context.Context, configName string) (*domain.SyncMetadata, error)
	UpsertSyncMetadata(ctx context.Context, metadata *domain.SyncMetadata) error

	// Transaction support
	BeginTx(ctx context.Context) (Tx, error)
}
