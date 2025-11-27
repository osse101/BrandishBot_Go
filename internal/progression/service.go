package progression

import (
	"context"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
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
	GetVotingStatus(ctx context.Context) (*domain.ProgressionVoting, error)

	// Unlocking
	CheckAndUnlockCriteria(ctx context.Context) (*domain.ProgressionUnlock, error) // Auto-check if criteria met
	ForceInstantUnlock(ctx context.Context) (*domain.ProgressionUnlock, error)     // Admin instant unlock

	// Engagement
	RecordEngagement(ctx context.Context, userID string, metricType string, value int) error
	GetEngagementScore(ctx context.Context) (int, error)
	GetUserEngagement(ctx context.Context, userID string) (*domain.EngagementBreakdown, error)

	// Status
	GetProgressionStatus(ctx context.Context) (*domain.ProgressionStatus, error)

	// Admin functions
	AdminUnlock(ctx context.Context, nodeKey string, level int) error
	AdminRelock(ctx context.Context, nodeKey string, level int) error
	ResetProgressionTree(ctx context.Context, resetBy string, reason string, preserveUserData bool) error
}

type service struct {
	repo Repository
}

// NewService creates a new progression service
func NewService(repo Repository) Service {
	return &service{
		repo: repo,
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

// VoteForUnlock allows a user to vote for next unlock
func (s *service) VoteForUnlock(ctx context.Context, userID string, nodeKey string) error {
	log := logger.FromContext(ctx)

	// Get node
	node, err := s.repo.GetNodeByKey(ctx, nodeKey)
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}
	if node == nil {
		return fmt.Errorf("node not found: %s", nodeKey)
	}

	// Determine target level (next unlockable level)
	targetLevel := 1
	for level := 1; level <= node.MaxLevel; level++ {
		unlocked, _ := s.repo.IsNodeUnlocked(ctx, nodeKey, level)
		if !unlocked {
			targetLevel = level
			break
		}
	}

	// Check if user already voted
	hasVoted, err := s.repo.HasUserVoted(ctx, userID, node.ID, targetLevel)
	if err != nil {
		return fmt.Errorf("failed to check vote status: %w", err)
	}
	if hasVoted {
		return fmt.Errorf("user has already voted for this unlock")
	}

	// Record vote
	if err := s.repo.RecordUserVote(ctx, userID, node.ID, targetLevel); err != nil {
		return fmt.Errorf("failed to record vote: %w", err)
	}

	// Increment vote count
	if err := s.repo.IncrementVote(ctx, node.ID, targetLevel); err != nil {
		return fmt.Errorf("failed to increment vote: %w", err)
	}

	log.Info("Vote recorded", "userID", userID, "nodeKey", nodeKey, "level", targetLevel)
	return nil
}

// GetVotingStatus returns current voting status
func (s *service) GetVotingStatus(ctx context.Context) (*domain.ProgressionVoting, error) {
	return s.repo.GetActiveVoting(ctx)
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
func (s *service) GetUserEngagement(ctx context.Context, userID string) (*domain.EngagementBreakdown, error) {
	return s.repo.GetUserEngagement(ctx, userID)
}

// GetProgressionStatus returns current community progression status
func (s *service) GetProgressionStatus(ctx context.Context) (*domain.ProgressionStatus, error) {
	unlocks, err := s.repo.GetAllUnlocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get unlocks: %w", err)
	}

	engagementScore, err := s.GetEngagementScore(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get engagement score: %w", err)
	}

	activeVoting, _ := s.repo.GetActiveVoting(ctx)

	return &domain.ProgressionStatus{
		TotalUnlocked:   len(unlocks),
		EngagementScore: engagementScore,
		ActiveVoting:    activeVoting,
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

	engagementScore, _ := s.GetEngagementScore(ctx)
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

// CheckAndUnlockCriteria checks if engagement criteria met and unlocks if so
func (s *service) CheckAndUnlockCriteria(ctx context.Context) (*domain.ProgressionUnlock, error) {
	// This would be called periodically (e.g., every 5 minutes)
	// For now, just return nil - will implement in next iteration
	return nil, nil
}

// ForceInstantUnlock selects highest voted option and unlocks immediately
func (s *service) ForceInstantUnlock(ctx context.Context) (*domain.ProgressionUnlock, error) {
	// Get active voting
	voting, err := s.repo.GetActiveVoting(ctx)
	if err != nil || voting == nil {
		return nil, fmt.Errorf("no active voting found")
	}

	// Close voting
	if err := s.repo.EndVoting(ctx, voting.NodeID, voting.TargetLevel); err != nil {
		return nil, fmt.Errorf("failed to end voting: %w", err)
	}

	// Unlock the node
	engagementScore, _ := s.GetEngagementScore(ctx)
	if err := s.repo.UnlockNode(ctx, voting.NodeID, voting.TargetLevel, "instant_override", engagementScore); err != nil {
		return nil, fmt.Errorf("failed to unlock node: %w", err)
	}

	// Return the unlock
	return s.repo.GetUnlock(ctx, voting.NodeID, voting.TargetLevel)
}
