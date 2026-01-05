package progression

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeLoader_Load(t *testing.T) {
	loader := NewTreeLoader()

	t.Run("valid JSON file", func(t *testing.T) {
		// Create temp file with valid JSON
		content := `{
			"version": "1.0",
			"description": "Test tree",
			"nodes": [
				{
					"key": "root",
					"name": "Root Node",
					"type": "feature",
					"description": "The root",
					"tier": 0,
					"size": "medium",
					"category": "core",
					"max_level": 1,
					"prerequisites": [],
					"sort_order": 0,
					"auto_unlock": true
				}
			]
		}`
		tmpFile := createTempFile(t, content)
		defer os.Remove(tmpFile)

		config, err := loader.Load(tmpFile)
		require.NoError(t, err)
		assert.Equal(t, "1.0", config.Version)
		assert.Equal(t, "Test tree", config.Description)
		assert.Len(t, config.Nodes, 1)
		assert.Equal(t, "root", config.Nodes[0].Key)
		assert.Empty(t, config.Nodes[0].Prerequisites)
		assert.True(t, config.Nodes[0].AutoUnlock)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := loader.Load("/nonexistent/path.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read tree config file")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		tmpFile := createTempFile(t, `{invalid json}`)
		defer os.Remove(tmpFile)

		_, err := loader.Load(tmpFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse tree config")
	})

	t.Run("with prerequisites", func(t *testing.T) {
		content := `{
			"version": "1.0",
			"description": "Tree with prerequisites",
			"nodes": [
				{
					"key": "root",
					"name": "Root",
					"type": "feature",
					"description": "Root node",
					"tier": 0,
					"size": "medium",
					"category": "core",
					"max_level": 1,
					"prerequisites": [],
					"sort_order": 0
				},
				{
					"key": "child",
					"name": "Child",
					"type": "item",
					"description": "Child node",
					"tier": 1,
					"size": "small",
					"category": "items",
					"max_level": 1,
					"prerequisites": ["root"],
					"sort_order": 1
				}
			]
		}`
		tmpFile := createTempFile(t, content)
		defer os.Remove(tmpFile)

		config, err := loader.Load(tmpFile)
		require.NoError(t, err)
		assert.Len(t, config.Nodes, 2)
		assert.Equal(t, []string{"root"}, config.Nodes[1].Prerequisites)
	})
}

func TestTreeLoader_Validate(t *testing.T) {
	loader := NewTreeLoader()

	t.Run("valid tree", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "root", Name: "Root", Type: "feature", Tier: 0, Size: "medium", Category: "core", MaxLevel: 1, Prerequisites: []string{}},
				{Key: "child", Name: "Child", Type: "item", Tier: 1, Size: "small", Category: "items", MaxLevel: 1, Prerequisites: []string{"root"}},
			},
		}
		err := loader.Validate(config)
		assert.NoError(t, err)
	})

	t.Run("nil config", func(t *testing.T) {
		err := loader.Validate(nil)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidConfig))
	})

	t.Run("empty nodes", func(t *testing.T) {
		config := &TreeConfig{Version: "1.0", Nodes: []NodeConfig{}}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidConfig))
	})

	t.Run("duplicate keys", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "dupe", Name: "First", Type: "feature", Tier: 1, Size: "medium", Category: "test", MaxLevel: 1, Prerequisites: []string{}},
				{Key: "dupe", Name: "Second", Type: "feature", Tier: 1, Size: "medium", Category: "test", MaxLevel: 1, Prerequisites: []string{}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrDuplicateNodeKey))
		assert.Contains(t, err.Error(), "dupe")
	})

	t.Run("missing prerequisite", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "child", Name: "Child", Type: "item", Tier: 1, Size: "small", Category: "items", MaxLevel: 1, Prerequisites: []string{"nonexistent"}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrMissingParent))
		assert.Contains(t, err.Error(), "nonexistent")
	})

	t.Run("empty key", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "", Name: "NoKey", Type: "feature", Tier: 1, Size: "medium", Category: "test", MaxLevel: 1, Prerequisites: []string{}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidConfig))
	})

	t.Run("empty name", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "node", Name: "", Type: "feature", Tier: 1, Size: "medium", Category: "test", MaxLevel: 1, Prerequisites: []string{}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidConfig))
	})

	t.Run("empty type", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "node", Name: "Node", Type: "", Tier: 1, Size: "medium", Category: "test", MaxLevel: 1, Prerequisites: []string{}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidConfig))
	})

	t.Run("invalid max_level", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "node", Name: "Node", Type: "feature", Tier: 1, Size: "medium", Category: "test", MaxLevel: 0, Prerequisites: []string{}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidConfig))
		assert.Contains(t, err.Error(), "max_level")
	})

	t.Run("invalid tier", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "node", Name: "Node", Type: "feature", Tier: 5, Size: "medium", Category: "test", MaxLevel: 1, Prerequisites: []string{}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidConfig))
		assert.Contains(t, err.Error(), "tier")
	})

	t.Run("invalid size", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "node", Name: "Node", Type: "feature", Tier: 1, Size: "huge", Category: "test", MaxLevel: 1, Prerequisites: []string{}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidConfig))
		assert.Contains(t, err.Error(), "size")
	})

	t.Run("empty category", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "node", Name: "Node", Type: "feature", Tier: 1, Size: "medium", Category: "", MaxLevel: 1, Prerequisites: []string{}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidConfig))
		assert.Contains(t, err.Error(), "category")
	})

	t.Run("deep valid tree", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "root", Name: "Root", Type: "feature", Tier: 0, Size: "medium", Category: "core", MaxLevel: 1, Prerequisites: []string{}},
				{Key: "level1", Name: "L1", Type: "item", Tier: 1, Size: "small", Category: "items", MaxLevel: 1, Prerequisites: []string{"root"}},
				{Key: "level2", Name: "L2", Type: "item", Tier: 2, Size: "small", Category: "items", MaxLevel: 1, Prerequisites: []string{"level1"}},
				{Key: "level3", Name: "L3", Type: "item", Tier: 3, Size: "small", Category: "items", MaxLevel: 1, Prerequisites: []string{"level2"}},
			},
		}
		err := loader.Validate(config)
		assert.NoError(t, err)
	})

	t.Run("multi-level node", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "root", Name: "Root", Type: "feature", Tier: 0, Size: "medium", Category: "core", MaxLevel: 1, Prerequisites: []string{}},
				{Key: "upgrade", Name: "Upgrade", Type: "upgrade", Tier: 2, Size: "medium", Category: "upgrades", MaxLevel: 5, Prerequisites: []string{"root"}},
			},
		}
		err := loader.Validate(config)
		assert.NoError(t, err)
	})

	t.Run("multiple prerequisites", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "root1", Name: "Root1", Type: "feature", Tier: 0, Size: "medium", Category: "core", MaxLevel: 1, Prerequisites: []string{}},
				{Key: "root2", Name: "Root2", Type: "feature", Tier: 0, Size: "medium", Category: "core", MaxLevel: 1, Prerequisites: []string{}},
				{Key: "child", Name: "Child", Type: "item", Tier: 1, Size: "large", Category: "items", MaxLevel: 1, Prerequisites: []string{"root1", "root2"}},
			},
		}
		err := loader.Validate(config)
		assert.NoError(t, err)
	})
}

func TestTreeLoader_CycleDetection(t *testing.T) {
	loader := NewTreeLoader()

	t.Run("simple cycle A->B->A", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "a", Name: "A", Type: "feature", Tier: 1, Size: "medium", Category: "test", MaxLevel: 1, Prerequisites: []string{"b"}},
				{Key: "b", Name: "B", Type: "feature", Tier: 1, Size: "medium", Category: "test", MaxLevel: 1, Prerequisites: []string{"a"}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrCycleDetected))
	})

	t.Run("self-reference", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "self", Name: "Self", Type: "feature", Tier: 1, Size: "medium", Category: "test", MaxLevel: 1, Prerequisites: []string{"self"}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrCycleDetected))
	})

	t.Run("longer cycle A->B->C->A", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "a", Name: "A", Type: "feature", Tier: 1, Size: "medium", Category: "test", MaxLevel: 1, Prerequisites: []string{"c"}},
				{Key: "b", Name: "B", Type: "feature", Tier: 1, Size: "medium", Category: "test", MaxLevel: 1, Prerequisites: []string{"a"}},
				{Key: "c", Name: "C", Type: "feature", Tier: 1, Size: "medium", Category: "test", MaxLevel: 1, Prerequisites: []string{"b"}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrCycleDetected))
	})

	t.Run("cycle with multiple prerequisites", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "a", Name: "A", Type: "feature", Tier: 1, Size: "medium", Category: "test", MaxLevel: 1, Prerequisites: []string{"b", "c"}},
				{Key: "b", Name: "B", Type: "feature", Tier: 1, Size: "medium", Category: "test", MaxLevel: 1, Prerequisites: []string{}},
				{Key: "c", Name: "C", Type: "feature", Tier: 1, Size: "medium", Category: "test", MaxLevel: 1, Prerequisites: []string{"a"}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrCycleDetected))
	})

	t.Run("no cycle with shared prerequisites", func(t *testing.T) {
		// This is a valid tree: two nodes both depend on root
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "root", Name: "Root", Type: "feature", Tier: 0, Size: "medium", Category: "core", MaxLevel: 1, Prerequisites: []string{}},
				{Key: "child1", Name: "Child1", Type: "item", Tier: 1, Size: "small", Category: "items", MaxLevel: 1, Prerequisites: []string{"root"}},
				{Key: "child2", Name: "Child2", Type: "item", Tier: 1, Size: "small", Category: "items", MaxLevel: 1, Prerequisites: []string{"root"}},
			},
		}
		err := loader.Validate(config)
		assert.NoError(t, err)
	})
}

func TestTreeLoader_LoadActualConfig(t *testing.T) {
	// Test that we can load the actual progression_tree.json
	loader := NewTreeLoader()
	
	// Path relative to the test file location
	configPath := filepath.Join("..", "..", "configs", "progression_tree.json")
	
	// Skip if file doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("progression_tree.json not found, skipping")
	}
	
	config, err := loader.Load(configPath)
	require.NoError(t, err, "Should load actual config file")
	
	// Check if config has been migrated to new schema
	// If any node lacks tier/size/category, skip validation (not yet migrated)
	migrated := true
	for i := range config.Nodes {
		if config.Nodes[i].Tier == 0 && config.Nodes[i].Size == "" && config.Nodes[i].Category == "" {
			migrated = false
			break
		}
	}
	
	if !migrated {
		t.Skip("progression_tree.json hasn't been migrated to new schema yet, skipping validation")
	}
	
	// Validate the loaded config
	err = loader.Validate(config)
	require.NoError(t, err, "Actual config should be valid")
	
	// Check expected structure
	// Version 2.0 indicates migrated schema
	if config.Version != "1.0" && config.Version != "2.0" {
		t.Errorf("Unexpected version: %s", config.Version)
	}
	
	// Verify root node exists if config has been updated
	var rootNode *NodeConfig
	for i := range config.Nodes {
		if config.Nodes[i].Key == "progression_system" {
			rootNode = &config.Nodes[i]
			break
		}
	}
	
	if rootNode != nil {
		// If root node exists, it should have new schema
		assert.Empty(t, rootNode.Prerequisites, "Root node should have no prerequisites")
		assert.True(t, rootNode.AutoUnlock, "Root node should be auto_unlock")
	}
}

// Helper functions

func createTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "tree_config_*.json")
	require.NoError(t, err)
	
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	
	err = tmpFile.Close()
	require.NoError(t, err)
	
	return tmpFile.Name()
}
