package progression

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// Service defines the progression system business logic
type Service interface {
	// Tree operations
	GetProgressionTree(ctx context.Context) ([]*domain.ProgressionTreeNode, error)
	GetAvailableUnlocks(ctx context.Context) ([]*domain.ProgressionNode, error)
	GetNode(ctx context.Context, id int) (*domain.ProgressionNode, error)

	// Feature checks
	IsFeatureUnlocked(ctx context.Context, featureKey string) (bool, error)
	IsItemUnlocked(ctx context.Context, itemName string) (bool, error)

	// Voting
	VoteForUnlock(ctx context.Context, userID string, nodeKey string) error
	GetActiveVotingSession(ctx context.Context) (*domain.ProgressionVotingSession, error)
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
	GetUserEngagement(ctx context.Context, userID string) (*domain.ContributionBreakdown, error)
	GetContributionLeaderboard(ctx context.Context, limit int) ([]domain.ContributionLeaderboardEntry, error)

	// Status
	GetProgressionStatus(ctx context.Context) (*domain.ProgressionStatus, error)
	GetRequiredNodes(ctx context.Context, nodeKey string) ([]*domain.ProgressionNode, error)

	// Admin functions
	AdminUnlock(ctx context.Context, nodeKey string, level int) error
	AdminRelock(ctx context.Context, nodeKey string, level int) error
	ResetProgressionTree(ctx context.Context, resetBy string, reason string, preserveUserData bool) error
	InvalidateWeightCache() // Clears engagement weight cache (forces reload on next engagement)

	// Shutdown gracefully shuts down the service
	Shutdown(ctx context.Context) error
}

type service struct {
	repo Repository
	bus  event.Bus

	// In-memory cache for unlock threshold checking
	mu               sync.RWMutex
	cachedTargetCost int // unlock_cost of target node
	cachedProgressID int // current unlock progress ID

	// Cache for engagement weights (reduces DB load)
	weightsMu     sync.RWMutex
	cachedWeights map[string]float64
	weightsExpiry time.Time

	// Semaphore to prevent concurrent unlock attempts
	unlockSem chan struct{}

	// Graceful shutdown support
	wg             sync.WaitGroup
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
}

// NewService creates a new progression service
func NewService(repo Repository, bus event.Bus) Service {
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	return &service{
		repo:           repo,
		bus:            bus,
		unlockSem:      make(chan struct{}, 1), // Buffer of 1 = only one unlock check at a time
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}
}

// GetProgressionTree returns the full tree with unlock status
func (s *service) GetProgressionTree(ctx context.Context) ([]*domain.ProgressionTreeNode, error) {
	log := logger.FromContext(ctx)

	// Get all nodes
	nodes, err := s.repo.GetAllNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	// Get all unlocks
	unlocks, err := s.repo.GetAllUnlocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get unlocks: %w", err)
	}

	// Build map of unlocks
	unlockMap := make(map[int]int) // nodeID -> level
	for _, unlock := range unlocks {
		unlockMap[unlock.NodeID] = unlock.CurrentLevel
	}

	// Build tree nodes with unlock status
	treeNodes := make([]*domain.ProgressionTreeNode, 0, len(nodes))
	for _, node := range nodes {
		level, isUnlocked := unlockMap[node.ID]

		// Get dependents (nodes that require this node)
		dependents, err := s.repo.GetDependents(ctx, node.ID)
		if err != nil {
			log.Warn("Failed to get dependent nodes", "nodeID", node.ID, "error", err)
		}

		childIDs := make([]int, 0, len(dependents))
		for _, child := range dependents {
			childIDs = append(childIDs, child.ID)
		}

		treeNode := &domain.ProgressionTreeNode{
			ProgressionNode: *node,
			IsUnlocked:      isUnlocked,
			UnlockedLevel:   level,
			Children:        childIDs,
		}
		treeNodes = append(treeNodes, treeNode)
	}

	return treeNodes, nil
}

// GetAvailableUnlocks returns nodes available for voting (prerequisites met)
func (s *service) GetAvailableUnlocks(ctx context.Context) ([]*domain.ProgressionNode, error) {
	log := logger.FromContext(ctx)

	// Get all nodes
	nodes, err := s.repo.GetAllNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	available := make([]*domain.ProgressionNode, 0)

	for _, node := range nodes {
		// Check if already unlocked at max level
		isUnlocked, err := s.repo.IsNodeUnlocked(ctx, node.NodeKey, node.MaxLevel)
		if err != nil {
			log.Warn("Failed to check unlock status", "nodeKey", node.NodeKey, "error", err)
			continue
		}
		if isUnlocked {
			continue // Already maxed out
		}

		// Get prerequisites for this node
		prerequisites, err := s.repo.GetPrerequisites(ctx, node.ID)
		if err != nil {
			log.Warn("Failed to get prerequisites", "nodeKey", node.NodeKey, "error", err)
			continue
		}

		// Check if all prerequisites are unlocked
		allPrereqsMet := true
		for _, prereq := range prerequisites {
			prereqUnlocked, err := s.repo.IsNodeUnlocked(ctx, prereq.NodeKey, 1)
			if err != nil || !prereqUnlocked {
				allPrereqsMet = false
				break
			}
		}

		if allPrereqsMet {
			available = append(available, node)
		}
	}

	return available, nil
}

// GetNode returns a single node by ID
func (s *service) GetNode(ctx context.Context, id int) (*domain.ProgressionNode, error) {
	return s.repo.GetNodeByID(ctx, id)
}

// IsFeatureUnlocked checks if a feature is available
func (s *service) IsFeatureUnlocked(ctx context.Context, featureKey string) (bool, error) {
	return s.repo.IsNodeUnlocked(ctx, featureKey, 1)
}

// IsItemUnlocked checks if an item is available
func (s *service) IsItemUnlocked(ctx context.Context, itemName string) (bool, error) {
	// Item names are prefixed with "item_"
	nodeKey := fmt.Sprintf("item_%s", itemName)
	return s.repo.IsNodeUnlocked(ctx, nodeKey, 1)
}

// VoteForUnlock allows a user to vote for next unlock (updated for voting sessions)
func (s *service) VoteForUnlock(ctx context.Context, userID string, nodeKey string) error {
	log := logger.FromContext(ctx)

	// Get active voting session
	session, err := s.repo.GetActiveSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active session: %w", err)
	}

	if session == nil || session.Status != SessionStatusVoting {
		return fmt.Errorf("no active voting session")
	}

	// Find option matching nodeKey
	var selectedOption *domain.ProgressionVotingOption
	for i := range session.Options {
		if session.Options[i].NodeDetails != nil && session.Options[i].NodeDetails.NodeKey == nodeKey {
			selectedOption = &session.Options[i]
			break
		}
	}

	if selectedOption == nil {
		return fmt.Errorf("node not in current voting options")
	}

	// Check if user already voted in this session
	hasVoted, err := s.repo.HasUserVotedInSession(ctx, userID, session.ID)
	if err != nil {
		return fmt.Errorf("failed to check vote status: %w", err)
	}
	if hasVoted {
		return fmt.Errorf("user already voted in this session")
	}

	// Increment vote and record user vote
	err = s.repo.IncrementOptionVote(ctx, selectedOption.ID)
	if err != nil {
		return fmt.Errorf("failed to increment vote: %w", err)
	}

	err = s.repo.RecordUserSessionVote(ctx, userID, session.ID, selectedOption.ID, selectedOption.NodeID)
	if err != nil {
		return fmt.Errorf("failed to record user vote: %w", err)
	}

	log.Info("Vote recorded", "userID", userID, "nodeKey", nodeKey, "sessionID", session.ID)
	return nil
}

// GetActiveVotingSession returns the current voting session
func (s *service) GetActiveVotingSession(ctx context.Context) (*domain.ProgressionVotingSession, error) {
	return s.repo.GetActiveSession(ctx)
}

// RecordEngagement records user engagement event
func (s *service) RecordEngagement(ctx context.Context, userID string, metricType string, value int) error {
	metric := &domain.EngagementMetric{
		UserID:      userID,
		MetricType:  metricType,
		MetricValue: value,
		RecordedAt:  time.Now(),
	}

	if err := s.repo.RecordEngagement(ctx, metric); err != nil {
		return err
	}

	// Try to get weights from cache first
	weight := s.getCachedWeight(metricType)

	// If not in cache or expired, fetch from DB
	if weight == 0.0 {
		weights, err := s.repo.GetEngagementWeights(ctx)
		if err != nil {
			// Log warning but don't fail, use default weight of 1.0 if not found
			logger.FromContext(ctx).Warn("Failed to get engagement weights, using default", "error", err)
			// We could fallback to hardcoded defaults here if critical
		} else {
			// Cache weights for future use (5 minute TTL)
			s.cacheWeights(weights)
			if w, ok := weights[metricType]; ok {
				weight = w
			}
		}
	}

	// Fallback defaults if still no weight found
	if weight == 0.0 {
		switch metricType {
		case "message":
			weight = 1.0
		case "command":
			weight = 2.0
		case "item_crafted":
			weight = 3.0 // Note: Migration sets this to 200, this is just code fallback
		default:
			weight = 1.0 // Safe default
		}
	}

	// If we have a weight, calculate score
	if weight > 0 {
		score := int(float64(value) * weight)
		if score > 0 {
			if err := s.AddContribution(ctx, score); err != nil {
				logger.FromContext(ctx).Warn("Failed to add contribution from engagement", "error", err)
			}
		}
	}

	return nil
}

// GetEngagementScore returns total community engagement score
func (s *service) GetEngagementScore(ctx context.Context) (int, error) {
	// Get score since last unlock (or beginning)
	return s.repo.GetEngagementScore(ctx, nil)
}

// GetUserEngagement returns user's contribution breakdown
func (s *service) GetUserEngagement(ctx context.Context, userID string) (*domain.ContributionBreakdown, error) {
	return s.repo.GetUserEngagement(ctx, userID)
}

// GetContributionLeaderboard retrieves top contributors
func (s *service) GetContributionLeaderboard(ctx context.Context, limit int) ([]domain.ContributionLeaderboardEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 10 // Default to top 10
	}
	return s.repo.GetContributionLeaderboard(ctx, limit)
}

// GetProgressionStatus returns current community progression status
func (s *service) GetProgressionStatus(ctx context.Context) (*domain.ProgressionStatus, error) {
	unlocks, err := s.repo.GetAllUnlocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get unlocks: %w", err)
	}

	contributionScore, err := s.GetEngagementScore(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get contribution score: %w", err)
	}

	activeSession, _ := s.repo.GetActiveSession(ctx)
	unlockProgress, _ := s.repo.GetActiveUnlockProgress(ctx)

	return &domain.ProgressionStatus{
		TotalUnlocked:        len(unlocks),
		ContributionScore:    contributionScore,
		ActiveSession:        activeSession,
		ActiveUnlockProgress: unlockProgress,
	}, nil
}

// AdminUnlock forces a node to unlock (for testing)
func (s *service) AdminUnlock(ctx context.Context, nodeKey string, level int) error {
	log := logger.FromContext(ctx)

	node, err := s.repo.GetNodeByKey(ctx, nodeKey)
	if err != nil || node == nil {
		return fmt.Errorf("node not found: %s", nodeKey)
	}

	if level > node.MaxLevel {
		return fmt.Errorf("level %d exceeds max level %d", level, node.MaxLevel)
	}

	engagementScore, err := s.GetEngagementScore(ctx)
	if err != nil {
		log.Warn("Failed to get engagement score for unlock", "error", err)
		engagementScore = 0
	}

	if err := s.repo.UnlockNode(ctx, node.ID, level, "admin", engagementScore); err != nil {
		return fmt.Errorf("failed to unlock node: %w", err)
	}

	log.Info("Admin unlocked node", "nodeKey", nodeKey, "level", level)
	return nil
}

// AdminRelock locks a node again (for testing)
func (s *service) AdminRelock(ctx context.Context, nodeKey string, level int) error {
	log := logger.FromContext(ctx)

	node, err := s.repo.GetNodeByKey(ctx, nodeKey)
	if err != nil || node == nil {
		return fmt.Errorf("node not found: %s", nodeKey)
	}

	if err := s.repo.RelockNode(ctx, node.ID, level); err != nil {
		return fmt.Errorf("failed to relock node: %w", err)
	}

	log.Info("Admin relocked node", "nodeKey", nodeKey, "level", level)
	return nil
}

// ResetProgressionTree performs annual reset
func (s *service) ResetProgressionTree(ctx context.Context, resetBy string, reason string, preserveUserData bool) error {
	log := logger.FromContext(ctx)
	log.Info("Resetting progression tree", "resetBy", resetBy, "reason", reason)

	return s.repo.ResetTree(ctx, resetBy, reason, preserveUserData)
}

// CheckAndUnlockCriteria checks if unlock criteria met
func (s *service) CheckAndUnlockCriteria(ctx context.Context) (*domain.ProgressionUnlock, error) {
	log := logger.FromContext(ctx)
	reqID := logger.GetRequestID(ctx)

	// Check if there's a node waiting to unlock
	unlock, err := s.CheckAndUnlockNode(ctx)
	if err != nil || unlock != nil {
		return unlock, err
	}

	// If no session exists, start one
	session, _ := s.repo.GetActiveSession(ctx)
	if session == nil {
		// Use shutdown context for async operation with timeout
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()

			ctx, cancel := context.WithTimeout(s.shutdownCtx, 1*time.Minute)
			defer cancel()

			// Inject request ID into context for tracing
			if reqID != "" {
				ctx = logger.WithRequestID(ctx, reqID)
			}

			if err := s.StartVotingSession(ctx, nil); err != nil {
				log.Error("Failed to auto-start voting session", "error", err)
			}
		}()
	}

	return nil, nil
}

// ForceInstantUnlock selects highest voted option and unlocks immediately
func (s *service) ForceInstantUnlock(ctx context.Context) (*domain.ProgressionUnlock, error) {
	log := logger.FromContext(ctx)

	// Get active session
	session, err := s.repo.GetActiveSession(ctx)
	if err != nil || session == nil {
		return nil, fmt.Errorf("no active voting session found")
	}

	if session.Status != "voting" {
		return nil, fmt.Errorf("voting session already ended")
	}

	// Find winning option
	winner := findWinningOption(session.Options)
	if winner == nil {
		return nil, fmt.Errorf("no voting options found")
	}

	// End voting session
	if err := s.repo.EndVotingSession(ctx, session.ID, winner.ID); err != nil {
		return nil, fmt.Errorf("failed to end voting: %w", err)
	}

	// Set unlock target
	progress, _ := s.repo.GetActiveUnlockProgress(ctx)
	if progress != nil {
		if err := s.repo.SetUnlockTarget(ctx, progress.ID, winner.NodeID, winner.TargetLevel, session.ID); err != nil {
			log.Warn("Failed to set unlock target during instant unlock", "error", err)
		}
	}

	// Unlock the node immediately
	engagementScore, err := s.GetEngagementScore(ctx)
	if err != nil {
		log.Warn("Failed to get engagement score for instant unlock", "error", err)
		engagementScore = 0
	}

	if err := s.repo.UnlockNode(ctx, winner.NodeID, winner.TargetLevel, "instant_override", engagementScore); err != nil {
		return nil, fmt.Errorf("failed to unlock node: %w", err)
	}

	// Mark progress complete and start new
	if progress != nil {
		if _, err := s.repo.CompleteUnlock(ctx, progress.ID, 0); err != nil {
			log.Error("Failed to complete unlock progress", "progressID", progress.ID, "error", err)
			// We don't return error here because the node IS unlocked, but we log the inconsistency
		}
	}

	// Start new voting session with the unlocked node context
	reqID := logger.GetRequestID(ctx)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		ctx, cancel := context.WithTimeout(s.shutdownCtx, 1*time.Minute)
		defer cancel()

		// Inject request ID into context for tracing
		if reqID != "" {
			ctx = logger.WithRequestID(ctx, reqID)
		}

		if err := s.StartVotingSession(ctx, &winner.NodeID); err != nil {
			log.Error("Failed to auto-start voting session after instant unlock", "error", err)
		}
	}()

	// Return the unlock
	return s.repo.GetUnlock(ctx, winner.NodeID, winner.TargetLevel)
}

// GetRequiredNodes returns a list of locked prerequisite nodes preventing the target node from being unlocked
func (s *service) GetRequiredNodes(ctx context.Context, nodeKey string) ([]*domain.ProgressionNode, error) {
	log := logger.FromContext(ctx)

	targetNode, err := s.repo.GetNodeByKey(ctx, nodeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}
	if targetNode == nil {
		return nil, fmt.Errorf("node not found: %s", nodeKey)
	}

	// Track which nodes we've already checked to avoid cycles
	visited := make(map[int]bool)
	var lockedPrereqs []*domain.ProgressionNode

	// Recursively check prerequisites
	var checkPrereqs func(nodeID int) error
	checkPrereqs = func(nodeID int) error {
		if visited[nodeID] {
			return nil // Already checked
		}
		visited[nodeID] = true

		prerequisites, err := s.repo.GetPrerequisites(ctx, nodeID)
		if err != nil {
			return fmt.Errorf("failed to get prerequisites for node %d: %w", nodeID, err)
		}

		for _, prereq := range prerequisites {
			// Check if this prerequisite is unlocked
			isUnlocked, err := s.repo.IsNodeUnlocked(ctx, prereq.NodeKey, 1)
			if err != nil {
				return fmt.Errorf("failed to check unlock status for %s: %w", prereq.NodeKey, err)
			}

			if !isUnlocked {
				// Add to locked list
				lockedPrereqs = append(lockedPrereqs, prereq)
				// Recursively check its prerequisites too
				if err := checkPrereqs(prereq.ID); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := checkPrereqs(targetNode.ID); err != nil {
		log.Error("Failed to check prerequisites", "error", err, "nodeKey", nodeKey)
		return nil, err
	}

	return lockedPrereqs, nil
}

// getCachedWeight retrieves weight from cache if not expired
func (s *service) getCachedWeight(metricType string) float64 {
	s.weightsMu.RLock()
	defer s.weightsMu.RUnlock()

	// Check if cache is expired
	if time.Now().After(s.weightsExpiry) {
		return 0.0
	}

	if s.cachedWeights == nil {
		return 0.0
	}

	return s.cachedWeights[metricType]
}

// cacheWeights stores engagement weights with 5-minute TTL
func (s *service) cacheWeights(weights map[string]float64) {
	s.weightsMu.Lock()
	defer s.weightsMu.Unlock()

	s.cachedWeights = weights
	s.weightsExpiry = time.Now().Add(5 * time.Minute) // 5 min TTL - weights rarely change
}

// InvalidateWeightCache clears the engagement weight cache
func (s *service) InvalidateWeightCache() {
	s.weightsMu.Lock()
	defer s.weightsMu.Unlock()

	s.cachedWeights = nil
	s.weightsExpiry = time.Time{} // Zero time = always expired
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
