package progression

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
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
	VoteForUnlock(ctx context.Context, platform, platformID, nodeKey string) error
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
	ResetProgressionTree(ctx context.Context, resetBy string, reason string, preserveUserData bool) error
	InvalidateWeightCache() // Clears engagement weight cache (forces reload on next engagement)

	// Test helpers (should only be used in tests)
	InvalidateUnlockCacheForTest()

	// Shutdown gracefully shuts down the service
	Shutdown(ctx context.Context) error
}

type service struct {
	repo repository.Progression
	user repository.User
	bus  event.Bus

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
func NewService(repo repository.Progression, userRepo repository.User, bus event.Bus) Service {
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	svc := &service{
		repo:           repo,
		user:           userRepo,
		bus:            bus,
		modifierCache:  NewModifierCache(30 * time.Minute), // 30-min TTL
		unlockCache:    NewUnlockCache(),                   // No TTL - invalidate on unlock/relock
		unlockSem:      make(chan struct{}, 1),             // Buffer of 1 = only one unlock check at a time
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}

	// Subscribe to node unlock/relock events to invalidate caches
	if bus != nil {
		bus.Subscribe("progression.node_unlocked", svc.handleNodeUnlocked)
		bus.Subscribe("progression.node_relocked", svc.handleNodeRelocked)
	}

	return svc
}

// InvalidateUnlockCacheForTest clears the unlock cache for testing purposes
// This should only be used in tests where there's no event bus to trigger automatic invalidation
func (s *service) InvalidateUnlockCacheForTest() {
	s.unlockCache.InvalidateAll()
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
	// Check cache first (hottest query in the system)
	if unlocked, found := s.unlockCache.Get(featureKey, 1); found {
		return unlocked, nil
	}

	// Cache miss - query database
	unlocked, err := s.repo.IsNodeUnlocked(ctx, featureKey, 1)
	if err != nil {
		return false, err
	}

	// Cache the result
	s.unlockCache.Set(featureKey, 1, unlocked)

	return unlocked, nil
}

// IsItemUnlocked checks if an item is available
func (s *service) IsItemUnlocked(ctx context.Context, itemName string) (bool, error) {
	// Item names are prefixed with "item_"
	nodeKey := fmt.Sprintf("item_%s", itemName)

	// Check cache first
	if unlocked, found := s.unlockCache.Get(nodeKey, 1); found {
		return unlocked, nil
	}

	// Cache miss - query database
	unlocked, err := s.repo.IsNodeUnlocked(ctx, nodeKey, 1)
	if err != nil {
		return false, err
	}

	// Cache the result
	s.unlockCache.Set(nodeKey, 1, unlocked)

	return unlocked, nil
}

// VoteForUnlock allows a user to vote for next unlock (updated for voting sessions)
func (s *service) VoteForUnlock(ctx context.Context, platform, platformID, nodeKey string) error {
	log := logger.FromContext(ctx)

	// Convert platform_id to internal user ID
	user, err := s.user.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	userID := user.ID

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
		return domain.ErrUserAlreadyVoted
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

	log.Info("Vote recorded", "userID", userID, "platform", platform, "platformID", platformID, "nodeKey", nodeKey, "sessionID", session.ID)
	return nil
}

// GetActiveVotingSession returns the current voting session
func (s *service) GetActiveVotingSession(ctx context.Context) (*domain.ProgressionVotingSession, error) {
	session, err := s.repo.GetActiveSession(ctx)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}

	// Enrich options with estimates
	for i := range session.Options {
		if session.Options[i].NodeDetails != nil {
			estimate, err := s.EstimateUnlockTime(ctx, session.Options[i].NodeDetails.NodeKey)
			if err == nil && estimate != nil {
				session.Options[i].EstimatedUnlockDate = estimate.EstimatedUnlockDate
			}
		}
	}

	return session, nil
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
func (s *service) GetUserEngagement(ctx context.Context, platform, platformID string) (*domain.ContributionBreakdown, error) {
	// Convert platform_id to internal user ID
	user, err := s.user.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.repo.GetUserEngagement(ctx, user.ID)
}

// GetUserEngagementByUsername returns user's contribution breakdown by username
func (s *service) GetUserEngagementByUsername(ctx context.Context, platform, username string) (*domain.ContributionBreakdown, error) {
	// Convert username to internal user ID
	user, err := s.user.GetUserByPlatformUsername(ctx, platform, username)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.repo.GetUserEngagement(ctx, user.ID)
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

	// Get total node count to determine if all are unlocked
	allNodes, err := s.repo.GetAllNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all nodes: %w", err)
	}

	// Check if all nodes are unlocked at their max level
	allUnlocked := s.checkAllNodesUnlocked(ctx, allNodes, unlocks)

	contributionScore, err := s.GetEngagementScore(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get contribution score: %w", err)
	}

	activeSession, _ := s.GetActiveVotingSession(ctx) // Use service method to get enriched session
	unlockProgress, _ := s.GetUnlockProgress(ctx)     // Use service method (which currently just calls repo, but consistent)

	// Enrich unlock progress with estimate
	if unlockProgress != nil && unlockProgress.NodeID != nil {
		node, err := s.repo.GetNodeByID(ctx, *unlockProgress.NodeID)
		if err == nil && node != nil {
			estimate, err := s.EstimateUnlockTime(ctx, node.NodeKey)
			if err == nil && estimate != nil {
				unlockProgress.EstimatedUnlockDate = estimate.EstimatedUnlockDate
			}
		}
	}

	return &domain.ProgressionStatus{
		TotalUnlocked:        len(unlocks),
		TotalNodes:          len(allNodes),
		AllNodesUnlocked:    allUnlocked,
		ContributionScore:    contributionScore,
		ActiveSession:        activeSession,
		ActiveUnlockProgress: unlockProgress,
	}, nil
}

// checkAllNodesUnlocked returns true if all nodes are unlocked at their max level
func (s *service) checkAllNodesUnlocked(ctx context.Context, allNodes []*domain.ProgressionNode, unlocks []*domain.ProgressionUnlock) bool {
	if len(allNodes) == 0 {
		return false
	}

	// Build map of unlock levels by node ID
	unlockMap := make(map[int]int) // nodeID -> highest unlocked level
	for _, unlock := range unlocks {
		if existing, ok := unlockMap[unlock.NodeID]; !ok || unlock.CurrentLevel > existing {
			unlockMap[unlock.NodeID] = unlock.CurrentLevel
		}
	}

	// Check if all nodes are unlocked at max level
	for _, node := range allNodes {
		if level, ok := unlockMap[node.ID]; !ok || level < node.MaxLevel {
			return false
		}
	}
	return true
}

// AdminUnlock forces a node to unlock (for testing)
func (s *service) AdminUnlock(ctx context.Context, nodeKey string, level int) error {
	log := logger.FromContext(ctx)

	node, err := s.repo.GetNodeByKey(ctx, nodeKey)
	if err != nil || node == nil {
		// Get available node keys for helpful error message
		availableKeys := s.getAvailableNodeKeys(ctx)
		if len(availableKeys) > 0 {
			return fmt.Errorf("%w: %s. Valid nodes: %v", domain.ErrNodeNotFound, nodeKey, availableKeys)
		}
		return fmt.Errorf("%w: %s", domain.ErrNodeNotFound, nodeKey)
	}

	if level > node.MaxLevel {
		return fmt.Errorf("%w: level %d exceeds max level %d for node %s", domain.ErrMaxLevelExceeded, level, node.MaxLevel, nodeKey)
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

// AdminUnlockAll unlocks all progression nodes at their max level (for debugging)
func (s *service) AdminUnlockAll(ctx context.Context) error {
	log := logger.FromContext(ctx)

	// Get all nodes
	nodes, err := s.repo.GetAllNodes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all nodes: %w", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("%w: no nodes found", domain.ErrNodeNotFound)
	}

	// Unlock each node at its max level
	unlockedCount := 0
	for _, node := range nodes {
		if err := s.AdminUnlock(ctx, node.NodeKey, node.MaxLevel); err != nil {
			log.Warn("Failed to unlock node", "nodeKey", node.NodeKey, "error", err)
			continue
		}
		unlockedCount++
	}

	log.Info("Admin unlocked all nodes", "total", len(nodes), "unlocked", unlockedCount)
	return nil
}

// AdminRelock locks a node again (for testing)
func (s *service) AdminRelock(ctx context.Context, nodeKey string, level int) error {
	log := logger.FromContext(ctx)

	node, err := s.repo.GetNodeByKey(ctx, nodeKey)
	if err != nil || node == nil {
		// Get available node keys for helpful error message
		availableKeys := s.getAvailableNodeKeys(ctx)
		if len(availableKeys) > 0 {
			return fmt.Errorf("%w: %s. Valid nodes: %v", domain.ErrNodeNotFound, nodeKey, availableKeys)
		}
		return fmt.Errorf("%w: %s", domain.ErrNodeNotFound, nodeKey)
	}

	if err := s.repo.RelockNode(ctx, node.ID, level); err != nil {
		return fmt.Errorf("failed to relock node: %w", err)
	}

	log.Info("Admin relocked node", "nodeKey", nodeKey, "level", level)
	return nil
}

// getAvailableNodeKeys returns a list of valid node keys for error messages
func (s *service) getAvailableNodeKeys(ctx context.Context) []string {
	nodes, err := s.repo.GetAllNodes(ctx)
	if err != nil {
		return nil
	}
	keys := make([]string, 0, len(nodes))
	for _, node := range nodes {
		keys = append(keys, node.NodeKey)
	}
	return keys
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
		return nil, domain.ErrNoActiveSession
	}

	if session.Status != "voting" {
		return nil, domain.ErrNoActiveSession
	}

	// Find winning option
	winner := findWinningOption(session.Options)
	if winner == nil {
		return nil, domain.ErrNoActiveSession
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
			log.Error("Failed to complete unlock progress", "error", err)
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
	targetNode, err := s.repo.GetNodeByKey(ctx, nodeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}
	if targetNode == nil {
		return nil, domain.ErrNodeNotFound
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

// GetModifiedValue retrieves a feature value modified by progression nodes
// Returns the modified value or the baseValue on error (safe fallback)
func (s *service) GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error) {
	// 1. Check cache first
	if cached, ok := s.modifierCache.Get(featureKey); ok {
		return cached.Value, nil
	}

	// 2. Get modifier from repository
	modifier, err := s.GetModifierForFeature(ctx, featureKey)
	if err != nil {
		// Fallback to base value on error
		return baseValue, err
	}
	if modifier == nil {
		// No modifier configured for this feature
		return baseValue, nil
	}

	// 3. Calculate value
	value := ApplyModifier(modifier, baseValue)

	// 4. Cache the result
	s.modifierCache.Set(featureKey, value, modifier.CurrentLevel)

	return value, nil
}

// GetModifierForFeature retrieves the modifier configuration and current level for a feature
func (s *service) GetModifierForFeature(ctx context.Context, featureKey string) (*ValueModifier, error) {
	// Query repository for node with this feature_key
	node, currentLevel, err := s.repo.GetNodeByFeatureKey(ctx, featureKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get node for feature %s: %w", featureKey, err)
	}
	if node == nil || node.ModifierConfig == nil {
		// No modifier configured for this feature
		return nil, nil
	}

	// Build ValueModifier from node's ModifierConfig
	modifier := &ValueModifier{
		NodeKey:       node.NodeKey,
		ModifierType:  ModifierType(node.ModifierConfig.ModifierType),
		BaseValue:     node.ModifierConfig.BaseValue,
		PerLevelValue: node.ModifierConfig.PerLevelValue,
		CurrentLevel:  currentLevel,
		MaxValue:      node.ModifierConfig.MaxValue,
		MinValue:      node.ModifierConfig.MinValue,
	}

	return modifier, nil
}

// handleNodeUnlocked invalidates caches when any node is unlocked
func (s *service) handleNodeUnlocked(ctx context.Context, e event.Event) error {
	// Invalidate modifier cache - values may have changed
	s.modifierCache.InvalidateAll()

	// Invalidate unlock cache - new features may be available
	s.unlockCache.InvalidateAll()

	log := logger.FromContext(ctx)
	if payload, ok := e.Payload.(map[string]interface{}); ok {
		log.Info("Invalidated caches due to node unlock",
			"node_key", payload["node_key"],
			"level", payload["level"])
	}
	return nil
}

// handleNodeRelocked invalidates caches when any node is relocked
func (s *service) handleNodeRelocked(ctx context.Context, e event.Event) error {
	// Invalidate modifier cache - values have changed
	s.modifierCache.InvalidateAll()

	// Invalidate unlock cache - features may no longer be available
	s.unlockCache.InvalidateAll()

	log := logger.FromContext(ctx)
	if payload, ok := e.Payload.(map[string]interface{}); ok {
		log.Info("Invalidated caches due to node relock",
			"node_key", payload["node_key"],
			"level", payload["level"])
	}
	return nil
}

// GetEngagementVelocity calculates engagement velocity over a period
func (s *service) GetEngagementVelocity(ctx context.Context, days int) (*domain.VelocityMetrics, error) {
	if days <= 0 {
		days = 7
	}

	since := time.Now().AddDate(0, 0, -days)
	totals, err := s.repo.GetDailyEngagementTotals(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily totals: %w", err)
	}

	totalPoints := 0
	sampleSize := len(totals)

	if sampleSize == 0 {
		return &domain.VelocityMetrics{
			PointsPerDay: 0,
			Trend:        "stable",
			PeriodDays:   days,
			SampleSize:   0,
			TotalPoints:  0,
		}, nil
	}

	orderedDays := make([]time.Time, 0, len(totals))
	for day, points := range totals {
		totalPoints += points
		orderedDays = append(orderedDays, day)
	}

	// Sort days
	for i := 0; i < len(orderedDays)-1; i++ {
		for j := 0; j < len(orderedDays)-i-1; j++ {
			if orderedDays[j].After(orderedDays[j+1]) {
				orderedDays[j], orderedDays[j+1] = orderedDays[j+1], orderedDays[j]
			}
		}
	}

	avg := float64(totalPoints) / float64(days)

	// Trend detection
	trend := "stable"
	if sampleSize >= 2 {
		half := sampleSize / 2
		firstHalfSum := 0
		secondHalfSum := 0

		for i := 0; i < half; i++ {
			firstHalfSum += totals[orderedDays[i]]
		}
		for i := half; i < sampleSize; i++ {
			secondHalfSum += totals[orderedDays[i]]
		}

		firstHalfAvg := float64(firstHalfSum) / float64(half)
		secondHalfAvg := float64(secondHalfSum) / float64(sampleSize-half)

		if secondHalfAvg > firstHalfAvg*1.1 {
			trend = "increasing"
		} else if secondHalfAvg < firstHalfAvg*0.9 {
			trend = "decreasing"
		}
	}

	return &domain.VelocityMetrics{
		PointsPerDay: avg,
		Trend:        trend,
		PeriodDays:   days,
		SampleSize:   sampleSize,
		TotalPoints:  totalPoints,
	}, nil
}

// EstimateUnlockTime predicts when a node will unlock
func (s *service) EstimateUnlockTime(ctx context.Context, nodeKey string) (*domain.UnlockEstimate, error) {
	node, err := s.repo.GetNodeByKey(ctx, nodeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}
	if node == nil {
		return nil, fmt.Errorf("node not found: %s", nodeKey)
	}

	// Get current velocity (7 days default)
	velocity, err := s.GetEngagementVelocity(ctx, 7)
	if err != nil {
		return nil, err
	}

	// Get current progress
	var currentProgress int
	progress, _ := s.repo.GetActiveUnlockProgress(ctx)
	if progress != nil && progress.NodeID != nil && *progress.NodeID == node.ID {
		currentProgress = progress.ContributionsAccumulated
	}

	// Check if already unlocked (max level)
	isUnlocked, _ := s.repo.IsNodeUnlocked(ctx, nodeKey, node.MaxLevel)
	if isUnlocked {
		return &domain.UnlockEstimate{
			NodeKey:             nodeKey,
			EstimatedDays:       0,
			Confidence:          "high",
			RequiredPoints:      0,
			CurrentProgress:     node.UnlockCost,
			CurrentVelocity:     velocity.PointsPerDay,
			EstimatedUnlockDate: func() *time.Time { t := time.Now(); return &t }(),
		}, nil
	}

	required := node.UnlockCost - currentProgress
	if required <= 0 {
		required = 0
	}

	var estimatedDays float64
	var estimatedDate *time.Time

	if velocity.PointsPerDay > 0 {
		estimatedDays = float64(required) / velocity.PointsPerDay
		t := time.Now().Add(time.Duration(estimatedDays * 24 * float64(time.Hour)))
		estimatedDate = &t
	} else {
		estimatedDays = -1 // Infinite
	}

	confidence := "low"
	if velocity.SampleSize >= 7 {
		if velocity.Trend == "stable" || velocity.Trend == "increasing" {
			confidence = "high"
		} else {
			confidence = "medium"
		}
	} else if velocity.SampleSize >= 3 {
		confidence = "medium"
	}

	return &domain.UnlockEstimate{
		NodeKey:             nodeKey,
		EstimatedDays:       estimatedDays,
		Confidence:          confidence,
		RequiredPoints:      required,
		CurrentProgress:     currentProgress,
		CurrentVelocity:     velocity.PointsPerDay,
		EstimatedUnlockDate: estimatedDate,
	}, nil
}
