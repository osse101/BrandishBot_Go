package progression

import (
	"context"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// JobService defines the interface for the job system (read-only operations)
type JobService interface {
	GetJobLevel(ctx context.Context, userID, jobKey string) (int, error)
}

// Service defines the progression system business logic
type Service interface {
	// Tree operations
	GetProgressionTree(ctx context.Context) ([]*domain.ProgressionTreeNode, error)
	GetAvailableUnlocks(ctx context.Context) ([]*domain.ProgressionNode, error)
	GetNode(ctx context.Context, id int) (*domain.ProgressionNode, error)

	// Feature checks
	IsFeatureUnlocked(ctx context.Context, featureKey string) (bool, error)
	IsItemUnlocked(ctx context.Context, itemName string) (bool, error)
	AreItemsUnlocked(ctx context.Context, itemNames []string) (map[string]bool, error)
	IsNodeUnlocked(ctx context.Context, nodeKey string, level int) (bool, error) // Bug #2: Check if specific node/level is unlocked

	// Voting
	VoteForUnlock(ctx context.Context, platform, platformID, username string, optionIndex int) error
	GetActiveVotingSession(ctx context.Context) (*domain.ProgressionVotingSession, error)
	GetMostRecentVotingSession(ctx context.Context) (*domain.ProgressionVotingSession, error) // Bug #1: Get most recent session (any status)
	StartVotingSession(ctx context.Context, unlockedNodeID *int) error
	EndVoting(ctx context.Context) (*domain.ProgressionVotingOption, error)

	// Unlocking
	CheckAndUnlockCriteria(ctx context.Context) (*domain.ProgressionUnlock, error) // Auto-check if criteria met
	CheckAndUnlockNode(ctx context.Context) (*domain.ProgressionUnlock, error)     // Check specific node threshold
	ForceInstantUnlock(ctx context.Context) (*domain.ProgressionUnlock, error)     // Admin instant unlock
	GetUnlockProgress(ctx context.Context) (*domain.UnlockProgress, error)
	AddContribution(ctx context.Context, amount int) error

	// Contribution tracking
	RecordEngagement(ctx context.Context, userID string, metricType string, value int) error
	GetEngagementScore(ctx context.Context) (int, error)
	GetUserEngagement(ctx context.Context, platform, platformID string) (*domain.ContributionBreakdown, error)
	GetUserEngagementByUsername(ctx context.Context, platform, username string) (*domain.ContributionBreakdown, error)
	GetContributionLeaderboard(ctx context.Context, limit int) ([]domain.ContributionLeaderboardEntry, error)
	GetEngagementVelocity(ctx context.Context, days int) (*domain.VelocityMetrics, error)
	EstimateUnlockTime(ctx context.Context, nodeKey string) (*domain.UnlockEstimate, error)

	// Value modification
	GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error)
	GetModifierForFeature(ctx context.Context, featureKey string) (*ValueModifier, error)

	// Status
	GetProgressionStatus(ctx context.Context) (*domain.ProgressionStatus, error)
	GetRequiredNodes(ctx context.Context, nodeKey string) ([]*domain.ProgressionNode, error)

	// Admin functions
	AdminUnlock(ctx context.Context, nodeKey string, level int) error
	AdminUnlockAll(ctx context.Context) error
	AdminRelock(ctx context.Context, nodeKey string, level int) error
	AdminFreezeVoting(ctx context.Context) error // Freeze voting session (pause until unlock)
	AdminStartVoting(ctx context.Context) error  // Resume frozen vote OR start new if nodes available
	ResetProgressionTree(ctx context.Context, resetBy string, reason string, preserveUserData bool) error
	InvalidateWeightCache() // Clears engagement weight cache (forces reload on next engagement)

	// Initialization
	InitializeProgressionState(ctx context.Context) error // Called on startup to ensure valid state

	// Test helpers (should only be used in tests)
	InvalidateUnlockCacheForTest()

	// Shutdown gracefully shuts down the service
	Shutdown(ctx context.Context) error
}

type service struct {
	repo       repository.Progression
	user       repository.User
	bus        event.Bus
	jobService JobService
	publisher  *event.ResilientPublisher

	// In-memory cache for unlock threshold checking
	mu               sync.RWMutex
	cachedTargetCost int // unlock_cost of target node
	cachedProgressID int // current unlock progress ID

	// Cache for engagement weights (reduces DB load)
	weightsMu     sync.RWMutex
	cachedWeights map[string]float64
	weightsExpiry time.Time

	// Cache for modifier values (reduces DB load for feature values)
	modifierCache *ModifierCache

	// Cache for node unlock status (reduces DB load for feature checks)
	unlockCache *UnlockCache

	// Semaphore to prevent concurrent unlock attempts
	unlockSem chan struct{}

	// Graceful shutdown support
	wg             sync.WaitGroup
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
}

// NewService creates a new progression service
func NewService(repo repository.Progression, userRepo repository.User, bus event.Bus, publisher *event.ResilientPublisher, jobService JobService) Service {
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	svc := &service{
		repo:           repo,
		user:           userRepo,
		bus:            bus,
		jobService:     jobService,
		publisher:      publisher,
		modifierCache:  NewModifierCache(30 * time.Minute), // 30-min TTL
		unlockCache:    NewUnlockCache(),                   // No TTL - invalidate on unlock/relock
		unlockSem:      make(chan struct{}, 1),             // Buffer of 1 = only one unlock check at a time
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}

	// Subscribe to node unlock/relock events to invalidate caches
	if bus != nil {
		bus.Subscribe(event.ProgressionNodeUnlocked, svc.handleNodeUnlocked)
		bus.Subscribe(event.ProgressionNodeRelocked, svc.handleNodeRelocked)
	}

	return svc
}

// InvalidateUnlockCacheForTest clears the unlock cache for testing purposes
// This should only be used in tests where there's no event bus to trigger automatic invalidation
func (s *service) InvalidateUnlockCacheForTest() {
	s.unlockCache.InvalidateAll()
}

// Shutdown gracefully shuts down the progression service
func (s *service) Shutdown(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("Shutting down progression service")

	// Cancel shutdown context to signal goroutines to stop
	s.shutdownCancel()

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("Progression service shutdown complete")
		return nil
	case <-ctx.Done():
		log.Warn("Progression service shutdown timed out")
		return ctx.Err()
	}
}
