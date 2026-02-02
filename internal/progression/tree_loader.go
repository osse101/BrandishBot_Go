package progression

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/validation"
)

// Sentinel errors for tree loader
var (
	ErrDuplicateNodeKey = errors.New("duplicate node key")
	ErrMissingParent    = errors.New("parent node not found")
	ErrCycleDetected    = errors.New("cycle detected in tree")
	ErrInvalidConfig    = errors.New("invalid configuration")
)

// Schema paths
const (
	ProgressionTreeSchemaPath = "configs/schemas/progression_tree.schema.json"
)

// TreeConfig represents the JSON configuration for the progression tree
type TreeConfig struct {
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Nodes       []NodeConfig `json:"nodes"`
}

// NodeConfig represents a single node in the progression tree JSON
type NodeConfig struct {
	Key         string `json:"key"`  // node_key in DB
	Name        string `json:"name"` // display_name
	Type        string `json:"type"` // node_type: feature, item, upgrade
	Description string `json:"description"`

	// Dynamic cost calculation inputs
	Tier     int    `json:"tier"` // 0-4: Foundation → Endgame
	Size     string `json:"size"` // small, medium, large (1:2:4 multiplier)
	MaxLevel int    `json:"max_level"`

	// Categorization
	Category string `json:"category"` // Grouping: economy, combat, progression, etc.

	// Prerequisites (breaking: was single parent, now supports multiple)
	Prerequisites []string `json:"prerequisites"` // List of node keys that must be unlocked first (AND logic)

	SortOrder  int  `json:"sort_order"`
	AutoUnlock bool `json:"auto_unlock"` // If true, node is auto-unlocked (skips voting)

	// Modifier configuration for upgrade nodes
	ModifierConfig *domain.ModifierConfig `json:"modifier_config,omitempty"`
}

// TreeLoader handles loading and validating progression tree configuration
type TreeLoader interface {
	Load(path string) (*TreeConfig, error)
	Validate(config *TreeConfig) error
	SyncToDatabase(ctx context.Context, config *TreeConfig, repo repository.Progression, path string) (*SyncResult, error)
}

// SyncResult contains the result of syncing the tree to the database
type SyncResult struct {
	NodesInserted int
	NodesUpdated  int
	NodesSkipped  int
	AutoUnlocked  int
}

type treeLoader struct {
	schemaValidator validation.SchemaValidator
}

// NewTreeLoader creates a new TreeLoader instance
func NewTreeLoader() TreeLoader {
	return &treeLoader{
		schemaValidator: validation.NewSchemaValidator(),
	}
}

// Load reads and parses a progression tree JSON file
func (t *treeLoader) Load(path string) (*TreeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read tree config file: %w", err)
	}

	// Validate against schema first
	if err := t.schemaValidator.ValidateBytes(data, ProgressionTreeSchemaPath); err != nil {
		return nil, fmt.Errorf("schema validation failed for %s: %w", path, err)
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
		if err := t.validateNodeConfig(i, node, nodesByKey); err != nil {
			return err
		}
		nodesByKey[node.Key] = node
	}

	// Validate prerequisites (both static and dynamic)
	for _, node := range config.Nodes {
		for _, prereqStr := range node.Prerequisites {
			isDynamic, dynamicPrereq, staticKey, err := ParsePrerequisite(prereqStr)
			if err != nil {
				return fmt.Errorf("%w: node '%s' has invalid prerequisite '%s': %w",
					ErrInvalidConfig, node.Key, prereqStr, err)
			}

			if isDynamic {
				// Validate dynamic prerequisite parameters
				if err := ValidateDynamicPrerequisite(dynamicPrereq); err != nil {
					return fmt.Errorf("%w: node '%s' dynamic prerequisite invalid: %w",
						ErrInvalidConfig, node.Key, err)
				}
			} else {
				// Validate static prerequisite references valid node
				if _, exists := nodesByKey[staticKey]; !exists {
					return fmt.Errorf("%w: node '%s' references prerequisite '%s'",
						ErrMissingParent, node.Key, staticKey)
				}
			}
		}
	}

	// Check for cycles using DFS (only for static prerequisites)
	return detectCycles(config.Nodes, nodesByKey)
}

func (t *treeLoader) validateNodeConfig(index int, node *NodeConfig, nodesByKey map[string]*NodeConfig) error {
	if node.Key == "" {
		return fmt.Errorf("%w: node at index %d has empty key", ErrInvalidConfig, index)
	}

	if _, exists := nodesByKey[node.Key]; exists {
		return fmt.Errorf("%w: '%s'", ErrDuplicateNodeKey, node.Key)
	}

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
		return fmt.Errorf("%w: node '%s' - %w", ErrInvalidConfig, node.Key, err)
	}

	// Validate size
	if err := ValidateSize(node.Size); err != nil {
		return fmt.Errorf("%w: node '%s' - %w", ErrInvalidConfig, node.Key, err)
	}

	// Validate category
	if node.Category == "" {
		return fmt.Errorf("%w: node '%s' has empty category", ErrInvalidConfig, node.Key)
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
		// Check only static prerequisites for cycles (skip dynamic ones)
		for _, prereqStr := range node.Prerequisites {
			isDynamic, _, staticKey, err := ParsePrerequisite(prereqStr)
			if err != nil {
				// Skip invalid prerequisites (they were validated earlier)
				continue
			}

			// Skip dynamic prerequisites - they don't form cycles
			if isDynamic {
				continue
			}

			if err := dfs(staticKey); err != nil {
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
func (t *treeLoader) SyncToDatabase(ctx context.Context, config *TreeConfig, repo repository.Progression, path string) (*SyncResult, error) {
	log := logger.FromContext(ctx)

	hasChanged, err := hasFileChanged(ctx, repo, path)
	if err != nil {
		return nil, fmt.Errorf("failed to check if file changed: %w", err)
	}

	if !hasChanged {
		log.Info("Progression tree config file unchanged, skipping sync", "path", path)
		return &SyncResult{}, nil
	}

	existingByKey, err := t.loadExistingNodes(ctx, repo)
	if err != nil {
		return nil, err
	}

	result := &SyncResult{}
	processed := make(map[string]bool)
	insertedNodeIDs := make(map[string]int)

	for len(processed) < len(config.Nodes) {
		progressMade := false
		for _, nodeConfig := range config.Nodes {
			if processed[nodeConfig.Key] || !t.arePrerequisitesMet(nodeConfig, existingByKey, insertedNodeIDs) {
				continue
			}

			if err := t.syncOneNode(ctx, repo, &nodeConfig, existingByKey, insertedNodeIDs, result); err != nil {
				return nil, err
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

	if err := updateSyncMetadata(ctx, repo, path); err != nil {
		log.Warn("Failed to update sync metadata", "error", err)
	}

	return result, nil
}

func (t *treeLoader) loadExistingNodes(ctx context.Context, repo repository.Progression) (map[string]*domain.ProgressionNode, error) {
	existingNodes, err := repo.GetAllNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing nodes: %w", err)
	}

	existingByKey := make(map[string]*domain.ProgressionNode, len(existingNodes))
	for _, node := range existingNodes {
		existingByKey[node.NodeKey] = node
	}
	return existingByKey, nil
}

func (t *treeLoader) arePrerequisitesMet(node NodeConfig, existingByKey map[string]*domain.ProgressionNode, insertedNodeIDs map[string]int) bool {
	for _, prereqStr := range node.Prerequisites {
		isDynamic, _, staticKey, err := ParsePrerequisite(prereqStr)
		if err != nil {
			// If parsing fails, skip (validation should have caught this earlier)
			return false
		}

		// Skip dynamic prerequisites - they don't block node insertion
		if isDynamic {
			continue
		}

		// Check if static prerequisite exists
		if _, ok := existingByKey[staticKey]; !ok {
			if _, ok := insertedNodeIDs[staticKey]; !ok {
				return false
			}
		}
	}
	return true
}

func (t *treeLoader) syncOneNode(ctx context.Context, repo repository.Progression, nodeConfig *NodeConfig, existingByKey map[string]*domain.ProgressionNode, insertedNodeIDs map[string]int, result *SyncResult) error {
	log := logger.FromContext(ctx)

	if existing, ok := existingByKey[nodeConfig.Key]; ok {
		needsUpdate := existing.DisplayName != nodeConfig.Name ||
			existing.Description != nodeConfig.Description ||
			existing.MaxLevel != nodeConfig.MaxLevel ||
			existing.SortOrder != nodeConfig.SortOrder ||
			existing.NodeType != nodeConfig.Type ||
			existing.Tier != nodeConfig.Tier ||
			existing.Size != nodeConfig.Size ||
			existing.Category != nodeConfig.Category ||
			!modifierConfigsEqual(existing.ModifierConfig, nodeConfig.ModifierConfig)

		if needsUpdate {
			if err := updateNode(ctx, repo, existing.ID, nodeConfig); err != nil {
				return fmt.Errorf("failed to update node '%s': %w", nodeConfig.Key, err)
			}
			if err := syncPrerequisites(ctx, repo, existing.ID, nodeConfig.Prerequisites, existingByKey, insertedNodeIDs); err != nil {
				return fmt.Errorf("failed to sync prerequisites for '%s': %w", nodeConfig.Key, err)
			}
			if err := syncDynamicPrerequisites(ctx, repo, existing.ID, nodeConfig.Prerequisites); err != nil {
				return fmt.Errorf("failed to sync dynamic prerequisites for '%s': %w", nodeConfig.Key, err)
			}
			result.NodesUpdated++
			log.Info("Updated progression node", "key", nodeConfig.Key)
		} else {
			result.NodesSkipped++
		}
	} else {
		nodeID, err := insertNode(ctx, repo, nodeConfig)
		if err != nil {
			return fmt.Errorf("failed to insert node '%s': %w", nodeConfig.Key, err)
		}
		insertedNodeIDs[nodeConfig.Key] = nodeID

		if err := syncPrerequisites(ctx, repo, nodeID, nodeConfig.Prerequisites, existingByKey, insertedNodeIDs); err != nil {
			return fmt.Errorf("failed to sync prerequisites for '%s': %w", nodeConfig.Key, err)
		}

		if err := syncDynamicPrerequisites(ctx, repo, nodeID, nodeConfig.Prerequisites); err != nil {
			return fmt.Errorf("failed to sync dynamic prerequisites for '%s': %w", nodeConfig.Key, err)
		}

		result.NodesInserted++
		log.Info("Inserted progression node", "key", nodeConfig.Key, "id", nodeID)

		if nodeConfig.AutoUnlock {
			if err := repo.UnlockNode(ctx, nodeID, 1, "auto", 0); err != nil {
				log.Warn("Failed to auto-unlock node", "key", nodeConfig.Key, "error", err)
			} else {
				result.AutoUnlocked++
				log.Info("Auto-unlocked node", "key", nodeConfig.Key)
			}
		}
	}
	return nil
}

func modifierConfigsEqual(a, b *domain.ModifierConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.FeatureKey == b.FeatureKey &&
		a.ModifierType == b.ModifierType &&
		a.BaseValue == b.BaseValue &&
		a.PerLevelValue == b.PerLevelValue &&
		floatPtrsEqual(a.MaxValue, b.MaxValue) &&
		floatPtrsEqual(a.MinValue, b.MinValue)
}

func floatPtrsEqual(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// insertNode inserts a new node into the database
func insertNode(ctx context.Context, repo repository.Progression, config *NodeConfig) (int, error) {
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
		NodeKey:        config.Key,
		NodeType:       config.Type,
		DisplayName:    config.Name,
		Description:    config.Description,
		MaxLevel:       config.MaxLevel,
		UnlockCost:     unlockCost,
		SortOrder:      config.SortOrder,
		Tier:           config.Tier,
		Size:           config.Size,
		Category:       config.Category,
		ModifierConfig: config.ModifierConfig,
	})
}

// updateNode updates an existing node in the database
func updateNode(ctx context.Context, repo repository.Progression, nodeID int, config *NodeConfig) error {
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
		NodeKey:        config.Key,
		NodeType:       config.Type,
		DisplayName:    config.Name,
		Description:    config.Description,
		MaxLevel:       config.MaxLevel,
		UnlockCost:     unlockCost,
		SortOrder:      config.SortOrder,
		Tier:           config.Tier,
		Size:           config.Size,
		Category:       config.Category,
		ModifierConfig: config.ModifierConfig,
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

// hasFileChanged checks if the config file has changed since last sync
func hasFileChanged(ctx context.Context, repo repository.Progression, configPath string) (bool, error) {
	// Get file info
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return false, fmt.Errorf("failed to stat config file: %w", err)
	}

	// Calculate file hash
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false, fmt.Errorf("failed to read config file: %w", err)
	}

	hash := sha256.Sum256(data)
	fileHash := hex.EncodeToString(hash[:])

	// Get last sync metadata
	syncMeta, err := repo.GetSyncMetadata(ctx, "progression_tree.json")
	if err != nil {
		// First sync - no metadata exists
		return true, nil
	}

	// Compare hash and mod time
	if syncMeta.FileHash != fileHash || !syncMeta.FileModTime.Equal(fileInfo.ModTime()) {
		return true, nil
	}

	return false, nil
}

// updateSyncMetadata updates the sync metadata after a successful sync
func updateSyncMetadata(ctx context.Context, repo repository.Progression, configPath string) error {
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return fmt.Errorf("failed to stat config file: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	hash := sha256.Sum256(data)
	fileHash := hex.EncodeToString(hash[:])

	return repo.UpsertSyncMetadata(ctx, &domain.SyncMetadata{
		ConfigName:   "progression_tree.json",
		LastSyncTime: time.Now(),
		FileHash:     fileHash,
		FileModTime:  fileInfo.ModTime(),
	})
}
