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
}

type service struct {
	repo Repository
	bus  event.Bus
	
	// In-memory cache for unlock threshold checking
	mu               sync.RWMutex
	cachedTargetCost int  // unlock_cost of target node
	cachedProgressID int  // current unlock progress ID
}

// NewService creates a new progression service
func NewService(repo Repository, bus event.Bus) Service {
	return &service{
		repo: repo,
		bus:  bus,
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

		// Get children
		children, err := s.repo.GetChildNodes(ctx, node.ID)
		if err != nil {
			log.Warn("Failed to get child nodes", "nodeID", node.ID, "error", err)
		}

		childIDs := make([]int, 0, len(children))
		for _, child := range children {
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
		// Skip root (already unlocked)
		if node.ParentNodeID == nil {
			continue
		}

		// Check if already unlocked at max level
		isUnlocked, err := s.repo.IsNodeUnlocked(ctx, node.NodeKey, node.MaxLevel)
		if err != nil {
			log.Warn("Failed to check unlock status", "nodeKey", node.NodeKey, "error", err)
			continue
		}
		if isUnlocked {
			continue // Already maxed out
		}

		// Check if parent is unlocked
		parentNode, err := s.repo.GetNodeByID(ctx, *node.ParentNodeID)
		if err != nil {
			log.Warn("Failed to get parent node", "parentID", *node.ParentNodeID, "error", err)
			continue
		}

		parentUnlocked, err := s.repo.IsNodeUnlocked(ctx, parentNode.NodeKey, 1)
		if err != nil || !parentUnlocked {
			continue // Parent not unlocked
		}

		available = append(available, node)
	}

	return available, nil
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

	if session == nil || session.Status != "voting" {
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

	return s.repo.RecordEngagement(ctx, metric)
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
		// Use detached context for async operation
		go func() {
			detachedCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			defer cancel()
			
			// Inject request ID into detached context for tracing
			if reqID != "" {
				detachedCtx = logger.WithRequestID(detachedCtx, reqID)
			}

			if err := s.StartVotingSession(detachedCtx, nil); err != nil {
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
		s.repo.SetUnlockTarget(ctx, progress.ID, winner.NodeID, winner.TargetLevel, session.ID)
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
	go func() {
		detachedCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		
		// Inject request ID into detached context for tracing
		if reqID != "" {
			detachedCtx = logger.WithRequestID(detachedCtx, reqID)
		}

		if err := s.StartVotingSession(detachedCtx, &winner.NodeID); err != nil {
			log.Error("Failed to auto-start voting session after instant unlock", "error", err)
		}
	}()

	// Return the unlock
	return s.repo.GetUnlock(ctx, winner.NodeID, winner.TargetLevel)
}

// GetRequiredNodes returns a list of locked ancestor nodes that are preventing the target node from being unlocked
func (s *service) GetRequiredNodes(ctx context.Context, nodeKey string) ([]*domain.ProgressionNode, error) {
	log := logger.FromContext(ctx)

	targetNode, err := s.repo.GetNodeByKey(ctx, nodeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}
	if targetNode == nil {
		return nil, fmt.Errorf("node not found: %s", nodeKey)
	}

	var lockedAncestors []*domain.ProgressionNode
	currentNode := targetNode

	// Traverse up the tree
	for currentNode.ParentNodeID != nil {
		parentNode, err := s.repo.GetNodeByID(ctx, *currentNode.ParentNodeID)
		if err != nil {
			log.Error("Failed to get parent node during traversal", "error", err, "nodeID", currentNode.ID)
			break
		}
		if parentNode == nil {
			log.Warn("Parent node ID found but node does not exist", "parentID", *currentNode.ParentNodeID)
			break
		}

		// Check if parent is fully locked (level 0)
		isUnlocked, err := s.repo.IsNodeUnlocked(ctx, parentNode.NodeKey, 1)
		if err != nil {
			return nil, fmt.Errorf("failed to check unlock status for %s: %w", parentNode.NodeKey, err)
		}

		if !isUnlocked {
			// Prepend to list (so root is first) - actually, standard append is fine, user probably wants closest blocker first?
			// "Requires: [Parent], [Grandparent]" reads better as "requires Parent (which requires Grandparent)"
			// Let's keep it strictly ordered by traversal (closest ancestor first)
			lockedAncestors = append(lockedAncestors, parentNode)
		} else {
			// optimization: if parent is unlocked, then all its ancestors MUST be unlocked (in a strict tree)
			// so we can stop checking
			break
		}

		currentNode = parentNode
	}

	return lockedAncestors, nil
}
