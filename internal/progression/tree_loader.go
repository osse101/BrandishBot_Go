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
	Key         string  `json:"key"`          // node_key in DB
	Name        string  `json:"name"`         // display_name
	Type        string  `json:"type"`         // node_type: feature, item, upgrade
	Description string  `json:"description"`
	UnlockCost  int     `json:"unlock_cost"`
	MaxLevel    int     `json:"max_level"`
	Parent      *string `json:"parent"`     // null for root node
	SortOrder   int     `json:"sort_order"`
	AutoUnlock  bool    `json:"auto_unlock"` // If true, node is auto-unlocked (skips voting)
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
		if node.UnlockCost < 0 {
			return fmt.Errorf("%w: node '%s' has negative unlock_cost", ErrInvalidConfig, node.Key)
		}
	}

	// Validate parent references exist
	for _, node := range config.Nodes {
		if node.Parent != nil {
			if _, exists := nodesByKey[*node.Parent]; !exists {
				return fmt.Errorf("%w: node '%s' references parent '%s'", ErrMissingParent, node.Key, *node.Parent)
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
		if node.Parent != nil {
			if err := dfs(*node.Parent); err != nil {
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

	// Build map from config for parent ID resolution
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

			// Check if parent is processed (or node has no parent)
			var parentID *int
			if nodeConfig.Parent != nil {
				// Check if parent exists in DB
				if existing, ok := existingByKey[*nodeConfig.Parent]; ok {
					parentID = &existing.ID
				} else if id, ok := insertedNodeIDs[*nodeConfig.Parent]; ok {
					parentID = &id
				} else {
					// Parent not yet processed, skip for now
					continue
				}
			}

			// Check if node exists in DB
			if existing, ok := existingByKey[nodeConfig.Key]; ok {
				// Node exists - check if update needed
				needsUpdate := existing.DisplayName != nodeConfig.Name ||
					existing.Description != nodeConfig.Description ||
					existing.UnlockCost != nodeConfig.UnlockCost ||
					existing.MaxLevel != nodeConfig.MaxLevel ||
					existing.SortOrder != nodeConfig.SortOrder ||
					existing.NodeType != nodeConfig.Type

				if needsUpdate {
					// Update existing node
					err := updateNode(ctx, repo, existing.ID, &nodeConfig, parentID)
					if err != nil {
						return nil, fmt.Errorf("failed to update node '%s': %w", nodeConfig.Key, err)
					}
					result.NodesUpdated++
					log.Info("Updated progression node", "key", nodeConfig.Key)
				} else {
					result.NodesSkipped++
				}
			} else {
				// Insert new node
				nodeID, err := insertNode(ctx, repo, &nodeConfig, parentID)
				if err != nil {
					return nil, fmt.Errorf("failed to insert node '%s': %w", nodeConfig.Key, err)
				}
				insertedNodeIDs[nodeConfig.Key] = nodeID
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
func insertNode(ctx context.Context, repo Repository, config *NodeConfig, parentID *int) (int, error) {
	// The Repository interface doesn't have a direct insert method,
	// so we need to use a type that can do raw inserts.
	// For now, we'll use a pattern that works with the existing interface.
	
	// This requires adding a new method to the Repository interface
	inserter, ok := repo.(NodeInserter)
	if !ok {
		return 0, fmt.Errorf("repository does not support node insertion")
	}
	
	return inserter.InsertNode(ctx, &domain.ProgressionNode{
		NodeKey:      config.Key,
		NodeType:     config.Type,
		DisplayName:  config.Name,
		Description:  config.Description,
		ParentNodeID: parentID,
		MaxLevel:     config.MaxLevel,
		UnlockCost:   config.UnlockCost,
		SortOrder:    config.SortOrder,
	})
}

// updateNode updates an existing node in the database
func updateNode(ctx context.Context, repo Repository, nodeID int, config *NodeConfig, parentID *int) error {
	updater, ok := repo.(NodeUpdater)
	if !ok {
		return fmt.Errorf("repository does not support node updates")
	}
	
	return updater.UpdateNode(ctx, nodeID, &domain.ProgressionNode{
		NodeKey:      config.Key,
		NodeType:     config.Type,
		DisplayName:  config.Name,
		Description:  config.Description,
		ParentNodeID: parentID,
		MaxLevel:     config.MaxLevel,
		UnlockCost:   config.UnlockCost,
		SortOrder:    config.SortOrder,
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
