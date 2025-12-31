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
					"unlock_cost": 0,
					"max_level": 1,
					"parent": null,
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
		assert.Nil(t, config.Nodes[0].Parent)
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

	t.Run("with parent reference", func(t *testing.T) {
		content := `{
			"version": "1.0",
			"description": "Tree with parent",
			"nodes": [
				{
					"key": "root",
					"name": "Root",
					"type": "feature",
					"description": "Root node",
					"unlock_cost": 0,
					"max_level": 1,
					"parent": null,
					"sort_order": 0
				},
				{
					"key": "child",
					"name": "Child",
					"type": "item",
					"description": "Child node",
					"unlock_cost": 100,
					"max_level": 1,
					"parent": "root",
					"sort_order": 1
				}
			]
		}`
		tmpFile := createTempFile(t, content)
		defer os.Remove(tmpFile)

		config, err := loader.Load(tmpFile)
		require.NoError(t, err)
		assert.Len(t, config.Nodes, 2)
		assert.Equal(t, "root", *config.Nodes[1].Parent)
	})
}

func TestTreeLoader_Validate(t *testing.T) {
	loader := NewTreeLoader()

	t.Run("valid tree", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "root", Name: "Root", Type: "feature", MaxLevel: 1, UnlockCost: 0},
				{Key: "child", Name: "Child", Type: "item", MaxLevel: 1, UnlockCost: 100, Parent: strPtr("root")},
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
				{Key: "dupe", Name: "First", Type: "feature", MaxLevel: 1},
				{Key: "dupe", Name: "Second", Type: "feature", MaxLevel: 1},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrDuplicateNodeKey))
		assert.Contains(t, err.Error(), "dupe")
	})

	t.Run("missing parent", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "child", Name: "Child", Type: "item", MaxLevel: 1, Parent: strPtr("nonexistent")},
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
				{Key: "", Name: "NoKey", Type: "feature", MaxLevel: 1},
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
				{Key: "node", Name: "", Type: "feature", MaxLevel: 1},
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
				{Key: "node", Name: "Node", Type: "", MaxLevel: 1},
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
				{Key: "node", Name: "Node", Type: "feature", MaxLevel: 0},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidConfig))
		assert.Contains(t, err.Error(), "max_level")
	})

	t.Run("negative unlock_cost", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "node", Name: "Node", Type: "feature", MaxLevel: 1, UnlockCost: -100},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidConfig))
		assert.Contains(t, err.Error(), "unlock_cost")
	})

	t.Run("deep valid tree", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "root", Name: "Root", Type: "feature", MaxLevel: 1},
				{Key: "level1", Name: "L1", Type: "item", MaxLevel: 1, Parent: strPtr("root")},
				{Key: "level2", Name: "L2", Type: "item", MaxLevel: 1, Parent: strPtr("level1")},
				{Key: "level3", Name: "L3", Type: "item", MaxLevel: 1, Parent: strPtr("level2")},
			},
		}
		err := loader.Validate(config)
		assert.NoError(t, err)
	})

	t.Run("multi-level node", func(t *testing.T) {
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "root", Name: "Root", Type: "feature", MaxLevel: 1},
				{Key: "upgrade", Name: "Upgrade", Type: "upgrade", MaxLevel: 5, UnlockCost: 100, Parent: strPtr("root")},
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
				{Key: "a", Name: "A", Type: "feature", MaxLevel: 1, Parent: strPtr("b")},
				{Key: "b", Name: "B", Type: "feature", MaxLevel: 1, Parent: strPtr("a")},
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
				{Key: "self", Name: "Self", Type: "feature", MaxLevel: 1, Parent: strPtr("self")},
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
				{Key: "a", Name: "A", Type: "feature", MaxLevel: 1, Parent: strPtr("c")},
				{Key: "b", Name: "B", Type: "feature", MaxLevel: 1, Parent: strPtr("a")},
				{Key: "c", Name: "C", Type: "feature", MaxLevel: 1, Parent: strPtr("b")},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrCycleDetected))
	})

	t.Run("no cycle with shared parent", func(t *testing.T) {
		// This is a valid tree: root has two children
		config := &TreeConfig{
			Version: "1.0",
			Nodes: []NodeConfig{
				{Key: "root", Name: "Root", Type: "feature", MaxLevel: 1},
				{Key: "child1", Name: "Child1", Type: "item", MaxLevel: 1, Parent: strPtr("root")},
				{Key: "child2", Name: "Child2", Type: "item", MaxLevel: 1, Parent: strPtr("root")},
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
	
	// Validate the loaded config
	err = loader.Validate(config)
	require.NoError(t, err, "Actual config should be valid")
	
	// Check expected nodes exist
	assert.Equal(t, "1.0", config.Version)
	assert.GreaterOrEqual(t, len(config.Nodes), 14, "Should have at least 14 nodes")
	
	// Verify root node
	var rootNode *NodeConfig
	for i := range config.Nodes {
		if config.Nodes[i].Key == "progression_system" {
			rootNode = &config.Nodes[i]
			break
		}
	}
	require.NotNil(t, rootNode, "Should have progression_system root node")
	assert.Nil(t, rootNode.Parent, "Root node should have no parent")
	assert.True(t, rootNode.AutoUnlock, "Root node should be auto_unlock")
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

func strPtr(s string) *string {
	return &s
}
