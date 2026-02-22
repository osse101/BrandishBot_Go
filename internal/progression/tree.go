package progression

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

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
