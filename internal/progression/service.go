package progression

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// JobService defines the interface for the job system
type JobService interface {
	AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error)
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
	nodes, err := s.repo.GetAllNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	available := make([]*domain.ProgressionNode, 0)
	for _, node := range nodes {
		if s.isNodeAvailable(ctx, node) {
			available = append(available, node)
		}
	}

	return available, nil
}

func (s *service) isNodeAvailable(ctx context.Context, node *domain.ProgressionNode) bool {
	log := logger.FromContext(ctx)

	// 1. Check if already maxed out
	unlocked, err := s.repo.IsNodeUnlocked(ctx, node.NodeKey, node.MaxLevel)
	if err != nil {
		log.Warn("Failed to check unlock status", "nodeKey", node.NodeKey, "error", err)
		return false
	}
	if unlocked {
		return false
	}

	// 2. Check static prerequisites
	if met := s.checkStaticPrereqs(ctx, node); !met {
		return false
	}

	// 3. Check dynamic prerequisites
	if met := s.checkDynamicPrereqs(ctx, node); !met {
		return false
	}

	return true
}

func (s *service) checkStaticPrereqs(ctx context.Context, node *domain.ProgressionNode) bool {
	prerequisites, err := s.repo.GetPrerequisites(ctx, node.ID)
	if err != nil {
		logger.FromContext(ctx).Warn("Failed to get prerequisites", "nodeKey", node.NodeKey, "error", err)
		return false
	}

	for _, prereq := range prerequisites {
		unlocked, err := s.repo.IsNodeUnlocked(ctx, prereq.NodeKey, 1)
		if err != nil || !unlocked {
			return false
		}
	}
	return true
}

func (s *service) checkDynamicPrereqs(ctx context.Context, node *domain.ProgressionNode) bool {
	log := logger.FromContext(ctx)
	dynamicPrereqsJSON, err := s.repo.GetNodeDynamicPrerequisites(ctx, node.ID)
	if err != nil {
		log.Warn("Failed to get dynamic prerequisites", "nodeKey", node.NodeKey, "error", err)
		return false
	}

	if len(dynamicPrereqsJSON) == 0 || string(dynamicPrereqsJSON) == "[]" {
		return true
	}

	var dynamicPrereqs []domain.DynamicPrerequisite
	if err := json.Unmarshal(dynamicPrereqsJSON, &dynamicPrereqs); err != nil {
		log.Warn("Failed to parse dynamic prerequisites", "nodeKey", node.NodeKey, "error", err)
		return false
	}

	for _, dynPrereq := range dynamicPrereqs {
		met, err := s.checkDynamicPrerequisite(ctx, dynPrereq)
		if err != nil || !met {
			return false
		}
	}
	return true
}

// GetAvailableUnlocksWithFutureTarget returns nodes available for voting now, plus nodes that will become available
// once the current unlock target is unlocked. This prevents voting gaps when a node with dependents completes.
func (s *service) GetAvailableUnlocksWithFutureTarget(ctx context.Context) ([]*domain.ProgressionNode, error) {
	log := logger.FromContext(ctx)

	// Get currently available nodes
	available, err := s.GetAvailableUnlocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current available unlocks: %w", err)
	}

	// Get current unlock progress to check if there's an active target
	progress, err := s.repo.GetActiveUnlockProgress(ctx)
	if err != nil {
		log.Warn("Failed to get active unlock progress, using only current available", "error", err)
		return available, nil
	}

	// If no target is set, return only currently available
	if progress == nil || progress.NodeID == nil {
		return available, nil
	}

	// Get the target node being worked towards
	targetNode, err := s.repo.GetNodeByID(ctx, *progress.NodeID)
	if err != nil || targetNode == nil {
		log.Warn("Failed to get target node, using only current available", "nodeID", progress.NodeID, "error", err)
		return available, nil
	}

	// Get all nodes that have the target as a prerequisite (future-available nodes)
	futureAvailable, err := s.getNodesDependentOn(ctx, targetNode.ID, targetNode.NodeKey)
	if err != nil {
		log.Warn("Failed to get dependent nodes, using only current available", "error", err)
		return available, nil
	}

	// Combine both sets, avoiding duplicates
	seen := make(map[int]bool)
	combined := make([]*domain.ProgressionNode, 0, len(available)+len(futureAvailable))

	for _, node := range available {
		if !seen[node.ID] {
			combined = append(combined, node)
			seen[node.ID] = true
		}
	}

	for _, node := range futureAvailable {
		if !seen[node.ID] {
			combined = append(combined, node)
			seen[node.ID] = true
		}
	}

	log.Debug("GetAvailableUnlocksWithFutureTarget results",
		"currentAvailable", len(available),
		"futureAvailable", len(futureAvailable),
		"combined", len(combined),
		"targetNodeKey", targetNode.NodeKey)

	return combined, nil
}

// getNodesDependentOn returns all nodes that have the specified node as a prerequisite
// (i.e., nodes that will become available once the specified node is unlocked)
func (s *service) getNodesDependentOn(ctx context.Context, _ int, nodeKey string) ([]*domain.ProgressionNode, error) {
	log := logger.FromContext(ctx)

	// Get all nodes
	allNodes, err := s.repo.GetAllNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all nodes: %w", err)
	}

	dependent := make([]*domain.ProgressionNode, 0)

	for _, node := range allNodes {
		// Skip if already fully unlocked
		isUnlocked, err := s.repo.IsNodeUnlocked(ctx, node.NodeKey, node.MaxLevel)
		if err != nil || isUnlocked {
			continue
		}

		// Get prerequisites for this node
		prerequisites, err := s.repo.GetPrerequisites(ctx, node.ID)
		if err != nil {
			log.Warn("Failed to get prerequisites", "nodeKey", node.NodeKey, "error", err)
			continue
		}

		// Check if the target node is a prerequisite
		for _, prereq := range prerequisites {
			if prereq.NodeKey == nodeKey {
				dependent = append(dependent, node)
				break
			}
		}
	}

	return dependent, nil
}

// checkDynamicPrerequisite evaluates a dynamic prerequisite
func (s *service) checkDynamicPrerequisite(ctx context.Context, prereq domain.DynamicPrerequisite) (bool, error) {
	switch prereq.Type {
	case "nodes_unlocked_below_tier":
		count, err := s.repo.CountUnlockedNodesBelowTier(ctx, prereq.Tier)
		if err != nil {
			return false, fmt.Errorf("failed to count unlocked nodes below tier %d: %w", prereq.Tier, err)
		}
		return count >= prereq.Count, nil

	case "total_nodes_unlocked":
		count, err := s.repo.CountTotalUnlockedNodes(ctx)
		if err != nil {
			return false, fmt.Errorf("failed to count total unlocked nodes: %w", err)
		}
		return count >= prereq.Count, nil

	default:
		return false, fmt.Errorf("unknown dynamic prerequisite type: %s", prereq.Type)
	}
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

func (s *service) IsNodeUnlocked(ctx context.Context, nodeKey string, level int) (bool, error) {
	// Check cache first
	if unlocked, found := s.unlockCache.Get(nodeKey, level); found {
		return unlocked, nil
	}

	// Cache miss - query database
	unlocked, err := s.repo.IsNodeUnlocked(ctx, nodeKey, level)
	if err != nil {
		return false, err
	}

	// Cache the result
	s.unlockCache.Set(nodeKey, level, unlocked)

	return unlocked, nil
}

// AreItemsUnlocked checks if multiple items are unlocked in a single batch operation.
// Returns a map of itemName -> unlocked status.
// This is much more efficient than calling IsItemUnlocked N times.
func (s *service) AreItemsUnlocked(ctx context.Context, itemNames []string) (map[string]bool, error) {
	if len(itemNames) == 0 {
		return make(map[string]bool), nil
	}

	result := make(map[string]bool, len(itemNames))
	uncachedKeys := make([]string, 0)
	uncachedNames := make([]string, 0)

	// Check cache first for all items
	for _, itemName := range itemNames {
		nodeKey := fmt.Sprintf("item_%s", itemName)
		if unlocked, found := s.unlockCache.Get(nodeKey, 1); found {
			result[itemName] = unlocked
		} else {
			uncachedKeys = append(uncachedKeys, nodeKey)
			uncachedNames = append(uncachedNames, itemName)
		}
	}

	// If all were cached, return early
	if len(uncachedKeys) == 0 {
		return result, nil
	}

	// Query DB for uncached items and populate cache
	for i, nodeKey := range uncachedKeys {
		unlocked, err := s.repo.IsNodeUnlocked(ctx, nodeKey, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to check unlock status for %s: %w", nodeKey, err)
		}
		s.unlockCache.Set(nodeKey, 1, unlocked)
		result[uncachedNames[i]] = unlocked
	}

	return result, nil
}

func (s *service) VoteForUnlock(ctx context.Context, platform, platformID, username string, optionIndex int) error {
	log := logger.FromContext(ctx)

	// 1. Resolve or auto-register user
	user, err := s.resolveUserByPlatform(ctx, platform, platformID, username)
	if err != nil {
		return err
	}

	// 2. Validate session and option
	session, selectedOption, err := s.validateVotingSession(ctx, optionIndex)
	if err != nil {
		return err
	}

	// 3. Record vote atomically
	if err := s.repo.CheckAndRecordVoteAtomic(ctx, user.ID, session.ID, selectedOption.ID, selectedOption.NodeID); err != nil {
		return err
	}

	// 4. Record engagement
	if err := s.RecordEngagement(ctx, user.ID, "vote_cast", 1); err != nil {
		log.Warn("Failed to record vote engagement", "userID", user.ID, "error", err)
	}

	log.Info("Vote recorded", "userID", user.ID, "platform", platform, "platformID", platformID, "optionIndex", optionIndex, "nodeKey", selectedOption.NodeDetails.NodeKey, "sessionID", session.ID)
	return nil
}

func (s *service) resolveUserByPlatform(ctx context.Context, platform, platformID, username string) (*domain.User, error) {
	log := logger.FromContext(ctx)
	user, err := s.user.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to resolve user: %w", err)
	}

	if user != nil {
		return user, nil
	}

	// Auto-registration
	if username == "" {
		return nil, fmt.Errorf("user not found and no username provided for auto-registration")
	}

	log.Info("Auto-registering new user from vote", "platform", platform, "platformID", platformID, "username", username)
	newUser := domain.User{Username: username}
	switch platform {
	case domain.PlatformTwitch:
		newUser.TwitchID = platformID
	case domain.PlatformYoutube:
		newUser.YoutubeID = platformID
	case domain.PlatformDiscord:
		newUser.DiscordID = platformID
	}

	if err := s.user.UpsertUser(ctx, &newUser); err != nil {
		return nil, fmt.Errorf("failed to auto-register user: %w", err)
	}

	user, err = s.user.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("failed to fetch newly registered user")
	}
	return user, nil
}

func (s *service) validateVotingSession(ctx context.Context, optionIndex int) (*domain.ProgressionVotingSession, *domain.ProgressionVotingOption, error) {
	session, err := s.repo.GetActiveSession(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get active session: %w", err)
	}

	if session == nil || session.Status != domain.VotingStatusVoting {
		return nil, nil, fmt.Errorf("no active voting session")
	}

	if optionIndex < 1 || optionIndex > len(session.Options) {
		return nil, nil, fmt.Errorf("invalid option index: %d (must be between 1 and %d)", optionIndex, len(session.Options))
	}

	return session, &session.Options[optionIndex-1], nil
}

// enrichSessionWithEstimates adds unlock time estimates to session options
func (s *service) enrichSessionWithEstimates(ctx context.Context, session *domain.ProgressionVotingSession) {
	if session == nil {
		return
	}
	for i := range session.Options {
		if session.Options[i].NodeDetails != nil {
			estimate, err := s.EstimateUnlockTime(ctx, session.Options[i].NodeDetails.NodeKey)
			if err == nil && estimate != nil {
				session.Options[i].EstimatedUnlockDate = estimate.EstimatedUnlockDate
			}
		}
	}
}

// GetActiveVotingSession returns the current voting session
func (s *service) GetActiveVotingSession(ctx context.Context) (*domain.ProgressionVotingSession, error) {
	session, err := s.repo.GetActiveSession(ctx)
	if err != nil {
		return nil, err
	}
	s.enrichSessionWithEstimates(ctx, session)
	return session, nil
}

func (s *service) GetMostRecentVotingSession(ctx context.Context) (*domain.ProgressionVotingSession, error) {
	session, err := s.repo.GetMostRecentSession(ctx)
	if err != nil {
		return nil, err
	}
	s.enrichSessionWithEstimates(ctx, session)
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
		baseScore := float64(value) * weight

		// Apply progression rate modifier (stacks multiplicatively across all three upgrades)
		// upgrade_progression_basic, upgrade_progression_two, upgrade_progression_three
		modifiedScore, err := s.GetModifiedValue(ctx, "progression_rate", baseScore)
		if err != nil {
			// Log warning but continue with base score if modifier fails
			logger.FromContext(ctx).Warn("Failed to apply progression_rate modifier, using base score", "error", err)
			modifiedScore = baseScore
		}

		// Apply Scholar bonus (per-user contribution multiplier)
		scholarMultiplier := s.calculateScholarBonus(ctx, userID)
		if scholarMultiplier > 1.0 {
			log := logger.FromContext(ctx)
			log.Info("Applying Scholar bonus",
				"user_id", userID,
				"base_score", modifiedScore,
				"multiplier", scholarMultiplier)
			modifiedScore = modifiedScore * scholarMultiplier
		}

		score := int(modifiedScore)
		if score > 0 {
			if err := s.AddContribution(ctx, score); err != nil {
				logger.FromContext(ctx).Warn("Failed to add contribution from engagement", "error", err)
			} else {
				// Award Scholar XP asynchronously (don't block engagement recording)
				s.awardScholarXP(ctx, userID, metricType, value)
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

// calculateScholarBonus calculates the contribution multiplier from Scholar job
// Returns 1.0 + (level × 0.10), e.g., level 5 = 1.5x multiplier
func (s *service) calculateScholarBonus(ctx context.Context, userID string) float64 {
	log := logger.FromContext(ctx)

	if s.jobService == nil {
		return 1.0
	}

	level, err := s.jobService.GetJobLevel(ctx, userID, job.JobKeyScholar)
	if err != nil {
		log.Warn("Failed to get Scholar level, using 1.0x multiplier", "error", err)
		return 1.0
	}

	if level == 0 {
		return 1.0
	}

	// 10% bonus per level
	multiplier := 1.0 + (float64(level) * job.ScholarBonusPerLevel / 100.0)
	return multiplier
}

// awardScholarXP awards XP to Scholar job for any engagement action
// Runs asynchronously to avoid blocking engagement recording
func (s *service) awardScholarXP(ctx context.Context, userID, metricType string, value int) {
	if s.jobService == nil {
		return
	}

	log := logger.FromContext(ctx)

	// Award XP asynchronously (don't block engagement recording)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		metadata := map[string]interface{}{
			"metric_type": metricType,
			"value":       value,
		}

		result, err := s.jobService.AwardXP(ctx, userID, job.JobKeyScholar, job.ScholarXPPerEngagement, "engagement", metadata)
		if err != nil {
			log.Warn("Failed to award Scholar XP", "error", err, "user_id", userID)
		} else if result != nil && result.LeveledUp {
			log.Info("Scholar leveled up!", "user_id", userID, "new_level", result.NewLevel)
		}
	}()
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
	allUnlocked := s.checkAllNodesUnlocked(allNodes, unlocks)

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

	isTransitioning := false
	if activeSession == nil && unlockProgress != nil && unlockProgress.UnlockedAt != nil {
		isTransitioning = true
	}

	return &domain.ProgressionStatus{
		TotalUnlocked:        len(unlocks),
		TotalNodes:           len(allNodes),
		AllNodesUnlocked:     allUnlocked,
		ContributionScore:    contributionScore,
		ActiveSession:        activeSession,
		ActiveUnlockProgress: unlockProgress,
		IsTransitioning:      isTransitioning,
	}, nil
}

// checkAllNodesUnlocked returns true if all nodes are unlocked at their max level
func (s *service) checkAllNodesUnlocked(allNodes []*domain.ProgressionNode, unlocks []*domain.ProgressionUnlock) bool {
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
		return fmt.Errorf("%w: %s", domain.ErrNodeNotFound, nodeKey)
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
		return nil, domain.ErrNoActiveSession
	}

	if session.Status != domain.VotingStatusVoting {
		return nil, domain.ErrNoActiveSession
	}

	// Find winning option
	winner := findWinningOption(session.Options)
	if winner == nil {
		return nil, domain.ErrNoActiveSession
	}

	// End voting session
	winnerID := winner.ID
	if err := s.repo.EndVotingSession(ctx, session.ID, &winnerID); err != nil {
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
// Supports stacking multiple modifiers with the same feature_key (multiplicative)
func (s *service) GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error) {
	// 1. Check cache first
	if cached, ok := s.modifierCache.Get(featureKey); ok {
		return cached.Value, nil
	}

	// 2. Get ALL modifiers for this feature
	modifiers, err := s.GetAllModifiersForFeature(ctx, featureKey)
	if err != nil {
		// Fallback to base value on error
		return baseValue, err
	}
	if len(modifiers) == 0 {
		// No modifiers configured for this feature
		return baseValue, nil
	}

	// 3. Apply all modifiers (stacks multiplicatively)
	value := baseValue
	totalLevel := 0
	for _, modifier := range modifiers {
		value = ApplyModifier(modifier, value)
		totalLevel += modifier.CurrentLevel
	}

	// 4. Cache with total level across all modifiers
	s.modifierCache.Set(featureKey, value, totalLevel)

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

// GetAllModifiersForFeature retrieves ALL modifiers for a feature key
func (s *service) GetAllModifiersForFeature(ctx context.Context, featureKey string) ([]*ValueModifier, error) {
	nodes, levels, err := s.repo.GetAllNodesByFeatureKey(ctx, featureKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes for feature %s: %w", featureKey, err)
	}

	modifiers := make([]*ValueModifier, 0, len(nodes))
	for i, node := range nodes {
		if node.ModifierConfig != nil {
			modifier := &ValueModifier{
				NodeKey:       node.NodeKey,
				ModifierType:  ModifierType(node.ModifierConfig.ModifierType),
				BaseValue:     node.ModifierConfig.BaseValue,
				PerLevelValue: node.ModifierConfig.PerLevelValue,
				CurrentLevel:  levels[i],
				MaxValue:      node.ModifierConfig.MaxValue,
				MinValue:      node.ModifierConfig.MinValue,
			}
			modifiers = append(modifiers, modifier)
		}
	}

	return modifiers, nil
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
