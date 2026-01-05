package progression

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// Sentinel errors for tree loader
var (
	ErrDuplicateNodeKey = errors.New("duplicate node key")
	ErrMissingParent    = errors.New("parent node not found")
	ErrCycleDetected    = errors.New("cycle detected in tree")
	ErrInvalidConfig    = errors.New("invalid configuration")
)

// TreeConfig represents the JSON configuration for the progression tree
type TreeConfig struct {
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Nodes       []NodeConfig `json:"nodes"`
}

// NodeConfig represents a single node in the progression tree JSON
type NodeConfig struct {
	Key         string   `json:"key"`          // node_key in DB
	Name        string   `json:"name"`         // display_name
	Type        string   `json:"type"`         // node_type: feature, item, upgrade
	Description string   `json:"description"`
	
	// Dynamic cost calculation inputs
	Tier        int      `json:"tier"`         // 0-4: Foundation → Endgame
	Size        string   `json:"size"`         // small, medium, large (1:2:4 multiplier)
	MaxLevel    int      `json:"max_level"`
	
	// Categorization
	Category    string   `json:"category"`     // Grouping: economy, combat, progression, etc.
	
	// Prerequisites (breaking: was single parent, now supports multiple)
	Prerequisites []string `json:"prerequisites"` // List of node keys that must be unlocked first (AND logic)
	
	SortOrder   int      `json:"sort_order"`
	AutoUnlock  bool     `json:"auto_unlock"`  // If true, node is auto-unlocked (skips voting)
}

// TreeLoader handles loading and validating progression tree configuration
type TreeLoader interface {
	Load(path string) (*TreeConfig, error)
	Validate(config *TreeConfig) error
	SyncToDatabase(ctx context.Context, config *TreeConfig, repo Repository) (*SyncResult, error)
}

// SyncResult contains the result of syncing the tree to the database
type SyncResult struct {
	NodesInserted int
	NodesUpdated  int
	NodesSkipped  int
	AutoUnlocked  int
}

type treeLoader struct{}

// NewTreeLoader creates a new TreeLoader instance
func NewTreeLoader() TreeLoader {
	return &treeLoader{}
}

// Load reads and parses a progression tree JSON file
func (t *treeLoader) Load(path string) (*TreeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read tree config file: %w", err)
	}

	var config TreeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse tree config: %w", err)
	}

	return &config, nil
}

// Validate checks the tree configuration for errors
func (t *treeLoader) Validate(config *TreeConfig) error {
	if config == nil {
		return fmt.Errorf("%w: config is nil", ErrInvalidConfig)
	}

	if len(config.Nodes) == 0 {
		return fmt.Errorf("%w: no nodes defined", ErrInvalidConfig)
	}

	// Build lookup maps
	nodesByKey := make(map[string]*NodeConfig, len(config.Nodes))
	
	// Check for duplicate keys and build index
	for i := range config.Nodes {
		node := &config.Nodes[i]
		
		if node.Key == "" {
			return fmt.Errorf("%w: node at index %d has empty key", ErrInvalidConfig, i)
		}
		
		if _, exists := nodesByKey[node.Key]; exists {
			return fmt.Errorf("%w: '%s'", ErrDuplicateNodeKey, node.Key)
		}
		nodesByKey[node.Key] = node
		
		// Validate required fields
		if node.Name == "" {
			return fmt.Errorf("%w: node '%s' has empty name", ErrInvalidConfig, node.Key)
		}
		if node.Type == "" {
			return fmt.Errorf("%w: node '%s' has empty type", ErrInvalidConfig, node.Key)
		}
		if node.MaxLevel <= 0 {
			return fmt.Errorf("%w: node '%s' has invalid max_level %d", ErrInvalidConfig, node.Key, node.MaxLevel)
		}
		
		// Validate tier
		if err := ValidateTier(node.Tier); err != nil {
			return fmt.Errorf("%w: node '%s' - %v", ErrInvalidConfig, node.Key, err)
		}
		
		// Validate size
		if err := ValidateSize(node.Size); err != nil {
			return fmt.Errorf("%w: node '%s' - %v", ErrInvalidConfig, node.Key, err)
		}
		
		// Validate category
		if node.Category == "" {
			return fmt.Errorf("%w: node '%s' has empty category", ErrInvalidConfig, node.Key)
		}
	}

	// Validate prerequisite references exist
	for _, node := range config.Nodes {
		for _, prereqKey := range node.Prerequisites {
			if _, exists := nodesByKey[prereqKey]; !exists {
				return fmt.Errorf("%w: node '%s' references prerequisite '%s'", ErrMissingParent, node.Key, prereqKey)
			}
		}
	}

	// Check for cycles using DFS
	if err := detectCycles(config.Nodes, nodesByKey); err != nil {
		return err
	}

	return nil
}

// detectCycles uses DFS to find cycles in the tree
func detectCycles(nodes []NodeConfig, nodesByKey map[string]*NodeConfig) error {
	// State: 0 = unvisited, 1 = visiting, 2 = visited
	state := make(map[string]int, len(nodes))
	
	var dfs func(key string) error
	dfs = func(key string) error {
		if state[key] == 1 {
			return fmt.Errorf("%w: at node '%s'", ErrCycleDetected, key)
		}
		if state[key] == 2 {
			return nil
		}
		
		state[key] = 1 // visiting
		
		node := nodesByKey[key]
		// Check all prerequisites for cycles
		for _, prereqKey := range node.Prerequisites {
			if err := dfs(prereqKey); err != nil {
				return err
			}
		}
		
		state[key] = 2 // visited
		return nil
	}
	
	for _, node := range nodes {
		if state[node.Key] == 0 {
			if err := dfs(node.Key); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// SyncToDatabase syncs the tree configuration to the database idempotently
func (t *treeLoader) SyncToDatabase(ctx context.Context, config *TreeConfig, repo Repository) (*SyncResult, error) {
	log := logger.FromContext(ctx)
	result := &SyncResult{}

	// Build a map of existing nodes from DB
	existingNodes, err := repo.GetAllNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing nodes: %w", err)
	}

	existingByKey := make(map[string]*domain.ProgressionNode, len(existingNodes))
	for _, node := range existingNodes {
		existingByKey[node.NodeKey] = node
	}

	// Build map from config for prerequisite ID resolution
	configByKey := make(map[string]*NodeConfig, len(config.Nodes))
	for i := range config.Nodes {
		configByKey[config.Nodes[i].Key] = &config.Nodes[i]
	}

	// Process nodes in order (parents first)
	// Since we validated no cycles, we can do multiple passes to resolve parents
	processed := make(map[string]bool)
	insertedNodeIDs := make(map[string]int) // key -> ID for newly inserted nodes

	for len(processed) < len(config.Nodes) {
		progressMade := false

		for _, nodeConfig := range config.Nodes {
			if processed[nodeConfig.Key] {
				continue
			}

			// Check if all prerequisites are processed
			allPrereqsProcessed := true
			for _, prereqKey := range nodeConfig.Prerequisites {
				// Check if prerequisite exists in DB or was just inserted
				if _, ok := existingByKey[prereqKey]; !ok {
					if _,ok := insertedNodeIDs[prereqKey]; !ok {
						// Prerequisite not yet processed, skip for now
						allPrereqsProcessed = false
						break
					}
				}
			}
			
			if !allPrereqsProcessed {
				continue
			}

			// Check if node exists in DB
			if existing, ok := existingByKey[nodeConfig.Key]; ok {
				// Node exists - check if update needed
			needsUpdate := existing.DisplayName != nodeConfig.Name ||
				existing.Description != nodeConfig.Description ||
				existing.MaxLevel != nodeConfig.MaxLevel ||
				existing.SortOrder != nodeConfig.SortOrder ||
				existing.NodeType != nodeConfig.Type
			
			// Note: We don't compare tier, size, category yet - those are new fields
			// Database migration will add them, repo will handle them

				if needsUpdate {
					// Update existing node
					err := updateNode(ctx, repo, existing.ID, &nodeConfig)
					if err != nil {
						return nil, fmt.Errorf("failed to update node '%s': %w", nodeConfig.Key, err)
					}
					
					// Update prerequisites in junction table
					if err := syncPrerequisites(ctx, repo, existing.ID, nodeConfig.Prerequisites, existingByKey, insertedNodeIDs); err != nil {
						return nil, fmt.Errorf("failed to sync prerequisites for '%s': %w", nodeConfig.Key, err)
					}
					
					result.NodesUpdated++
					log.Info("Updated progression node", "key", nodeConfig.Key)
				} else {
					result.NodesSkipped++
				}
			} else {
			// Insert new node
			nodeID, err := insertNode(ctx, repo, &nodeConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to insert node '%s': %w", nodeConfig.Key, err)
			}
			insertedNodeIDs[nodeConfig.Key] = nodeID
			
			// Sync prerequisites in junction table
			if err := syncPrerequisites(ctx, repo, nodeID, nodeConfig.Prerequisites, existingByKey, insertedNodeIDs); err != nil {
				return nil, fmt.Errorf("failed to sync prerequisites for '%s': %w", nodeConfig.Key, err)
			}
			
			result.NodesInserted++
			log.Info("Inserted progression node", "key", nodeConfig.Key, "id", nodeID)

			// Handle auto_unlock
			if nodeConfig.AutoUnlock {
				if err := repo.UnlockNode(ctx, nodeID, 1, "auto", 0); err != nil {
					log.Warn("Failed to auto-unlock node", "key", nodeConfig.Key, "error", err)
				} else {
					result.AutoUnlocked++
					log.Info("Auto-unlocked node", "key", nodeConfig.Key)
				}
			}
		}
			processed[nodeConfig.Key] = true
			progressMade = true
		}

		if !progressMade {
			return nil, fmt.Errorf("unable to process all nodes - possible circular dependency")
		}
	}

	log.Info("Progression tree sync completed",
		"inserted", result.NodesInserted,
		"updated", result.NodesUpdated,
		"skipped", result.NodesSkipped,
		"auto_unlocked", result.AutoUnlocked)

	return result, nil
}

// insertNode inserts a new node into the database
func insertNode(ctx context.Context, repo Repository, config *NodeConfig) (int, error) {
	inserter, ok := repo.(NodeInserter)
	if !ok {
		return 0, fmt.Errorf("repository does not support node insertion")
	}
	
	// Calculate unlock cost based on tier and size
	unlockCost, err := CalculateUnlockCost(config.Tier, NodeSize(config.Size))
	if err != nil {
		return 0, fmt.Errorf("failed to calculate unlock cost: %w", err)
	}
	
	return inserter.InsertNode(ctx, &domain.ProgressionNode{
		NodeKey:     config.Key,
		NodeType:    config.Type,
		DisplayName: config.Name,
		Description: config.Description,
		MaxLevel:    config.MaxLevel,
		UnlockCost:  unlockCost,
		SortOrder:   config.SortOrder,
		// New fields - will be added by migration
		// Tier:        config.Tier,
		// Size:        config.Size,
		// Category:    config.Category,
	})
}

// updateNode updates an existing node in the database
func updateNode(ctx context.Context, repo Repository, nodeID int, config *NodeConfig) error {
	updater, ok := repo.(NodeUpdater)
	if !ok {
		return fmt.Errorf("repository does not support node updates")
	}
	
	// Calculate unlock cost based on tier and size
	unlockCost, err := CalculateUnlockCost(config.Tier, NodeSize(config.Size))
	if err != nil {
		return fmt.Errorf("failed to calculate unlock cost: %w", err)
	}
	
	return updater.UpdateNode(ctx, nodeID, &domain.ProgressionNode{
		NodeKey:     config.Key,
		NodeType:    config.Type,
		DisplayName: config.Name,
		Description: config.Description,
		MaxLevel:    config.MaxLevel,
		UnlockCost:  unlockCost,
		SortOrder:   config.SortOrder,
		// New fields - will be added by migration
		// Tier:        config.Tier,
		// Size:        config.Size,
		// Category:    config.Category,
	})
}

// NodeInserter is an optional interface for inserting new nodes
type NodeInserter interface {
	InsertNode(ctx context.Context, node *domain.ProgressionNode) (int, error)
}

// NodeUpdater is an optional interface for updating existing nodes
type NodeUpdater interface {
	UpdateNode(ctx context.Context, nodeID int, node *domain.ProgressionNode) error
}
