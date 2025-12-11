package progression

import (
	"context"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Repository defines database operations for progression system
type Repository interface {
	// Node operations
	GetNodeByKey(ctx context.Context, nodeKey string) (*domain.ProgressionNode, error)
	GetNodeByID(ctx context.Context, id int) (*domain.ProgressionNode, error)
	GetAllNodes(ctx context.Context) ([]*domain.ProgressionNode, error)
	GetChildNodes(ctx context.Context, parentID int) ([]*domain.ProgressionNode, error)

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
	GetSessionByID(ctx context.Context, sessionID int) (*domain.ProgressionVotingSession, error)
	IncrementOptionVote(ctx context.Context, optionID int) error
	EndVotingSession(ctx context.Context, sessionID int, winningOptionID int) error
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
	GetEngagementWeights(ctx context.Context) (map[string]float64, error)

	// Reset operations
	ResetTree(ctx context.Context, resetBy string, reason string, preserveUserData bool) error
	RecordReset(ctx context.Context, reset *domain.ProgressionReset) error

	// Transaction support
	BeginTx(ctx context.Context) (repository.Tx, error)
}
