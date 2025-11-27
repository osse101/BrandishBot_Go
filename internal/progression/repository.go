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

	// Voting operations
	GetActiveVoting(ctx context.Context) (*domain.ProgressionVoting, error)
	StartVoting(ctx context.Context, nodeID int, level int, endsAt *time.Time) error
	GetVoting(ctx context.Context, nodeID int, level int) (*domain.ProgressionVoting, error)
	IncrementVote(ctx context.Context, nodeID int, level int) error
	EndVoting(ctx context.Context, nodeID int, level int) error

	// User vote tracking
	HasUserVoted(ctx context.Context, userID string, nodeID int, level int) (bool, error)
	RecordUserVote(ctx context.Context, userID string, nodeID int, level int) error

	// User progression
	UnlockUserProgression(ctx context.Context, userID string, progressionType string, key string, metadata map[string]interface{}) error
	IsUserProgressionUnlocked(ctx context.Context, userID string, progressionType string, key string) (bool, error)
	GetUserProgressions(ctx context.Context, userID string, progressionType string) ([]*domain.UserProgression, error)

	// Engagement tracking
	RecordEngagement(ctx context.Context, metric *domain.EngagementMetric) error
	GetEngagementScore(ctx context.Context, since *time.Time) (int, error)
	GetUserEngagement(ctx context.Context, userID string) (*domain.EngagementBreakdown, error)
	GetEngagementWeights(ctx context.Context) (map[string]float64, error)

	// Reset operations
	ResetTree(ctx context.Context, resetBy string, reason string, preserveUserData bool) error
	RecordReset(ctx context.Context, reset *domain.ProgressionReset) error

	// Transaction support
	BeginTx(ctx context.Context) (repository.Tx, error)
}
