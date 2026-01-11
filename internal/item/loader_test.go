package item

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestItemLoader_Load(t *testing.T) {
	loader := NewLoader()

	t.Run("valid JSON file", func(t *testing.T) {
		content := `{
			"version": "1.0",
			"description": "Test items",
			"valid_tags": ["consumable", "tradeable"],
			"valid_handlers": ["lootbox"],
			"items": [
				{
					"internal_name": "test_item",
					"public_name": "testitem",
					"description": "A test item",
					"tier": 0,
					"max_stack": 10,
					"base_value": 100,
					"tags": ["consumable"],
					"handler": "lootbox",
					"default_display": "Test Box"
				}
			]
		}`
		tmpFile := createTempFile(t, content)
		defer os.Remove(tmpFile)

		config, err := loader.Load(tmpFile)
		require.NoError(t, err)
		assert.Equal(t, "1.0", config.Version)
		assert.Equal(t, "Test items", config.Description)
		assert.Len(t, config.ValidTags, 2)
		assert.Len(t, config.ValidHandlers, 1)
		assert.Len(t, config.Items, 1)
		assert.Equal(t, "test_item", config.Items[0].InternalName)
		assert.Equal(t, "testitem", config.Items[0].PublicName)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := loader.Load("/nonexistent/path.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read items config file")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		tmpFile := createTempFile(t, `{invalid json}`)
		defer os.Remove(tmpFile)

		_, err := loader.Load(tmpFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse items config")
	})
}

func TestItemLoader_Validate(t *testing.T) {
	loader := NewLoader()

	t.Run("valid config", func(t *testing.T) {
		config := &ItemConfig{
			Version:       "1.0",
			ValidTags:     []string{"consumable", "tradeable"},
			ValidHandlers: []string{"lootbox"},
			Items: []ItemDef{
				{
					InternalName:   "item1",
					PublicName:     "Item One",
					Description:    "Test item",
					Tier:           0,
					MaxStack:       10,
					BaseValue:      100,
					Tags:           []string{"consumable"},
					DefaultDisplay: "Item 1",
				},
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

	t.Run("empty items", func(t *testing.T) {
		config := &ItemConfig{Version: "1.0", Items: []ItemDef{}}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidConfig))
	})

	t.Run("duplicate internal names", func(t *testing.T) {
		config := &ItemConfig{
			Version:       "1.0",
			ValidTags:     []string{"consumable"},
			ValidHandlers: []string{},
			Items: []ItemDef{
				{InternalName: "dupe", PublicName: "First", DefaultDisplay: "First", Tags: []string{}},
				{InternalName: "dupe", PublicName: "Second", DefaultDisplay: "Second", Tags: []string{}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrDuplicateInternalName))
		assert.Contains(t, err.Error(), "dupe")
	})

	t.Run("invalid tag", func(t *testing.T) {
		config := &ItemConfig{
			Version:       "1.0",
			ValidTags:     []string{"consumable"},
			ValidHandlers: []string{},
			Items: []ItemDef{
				{
					InternalName:   "item1",
					PublicName:     "Item",
					DefaultDisplay: "Item",
					Tags:           []string{"invalid_tag"},
				},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidTag))
		assert.Contains(t, err.Error(), "invalid_tag")
	})

	t.Run("invalid handler", func(t *testing.T) {
		handler := "invalid_handler"
		config := &ItemConfig{
			Version:       "1.0",
			ValidTags:     []string{},
			ValidHandlers: []string{"lootbox"},
			Items: []ItemDef{
				{
					InternalName:   "item1",
					PublicName:     "Item",
					DefaultDisplay: "Item",
					Tags:           []string{},
					Handler:        &handler,
				},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidHandler))
		assert.Contains(t, err.Error(), "invalid_handler")
	})

	t.Run("empty internal name", func(t *testing.T) {
		config := &ItemConfig{
			Version:       "1.0",
			ValidTags:     []string{},
			ValidHandlers: []string{},
			Items: []ItemDef{
				{InternalName: "", PublicName: "Item", DefaultDisplay: "Item", Tags: []string{}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidConfig))
	})

	t.Run("empty public name", func(t *testing.T) {
		config := &ItemConfig{
			Version:       "1.0",
			ValidTags:     []string{},
			ValidHandlers: []string{},
			Items: []ItemDef{
				{InternalName: "item1", PublicName: "", DefaultDisplay: "Item", Tags: []string{}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidConfig))
	})

	t.Run("empty default display", func(t *testing.T) {
		config := &ItemConfig{
			Version:       "1.0",
			ValidTags:     []string{},
			ValidHandlers: []string{},
			Items: []ItemDef{
				{InternalName: "item1", PublicName: "Item", DefaultDisplay: "", Tags: []string{}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidConfig))
	})

	t.Run("negative max stack", func(t *testing.T) {
		config := &ItemConfig{
			Version:       "1.0",
			ValidTags:     []string{},
			ValidHandlers: []string{},
			Items: []ItemDef{
				{InternalName: "item1", PublicName: "Item", DefaultDisplay: "Item", MaxStack: -1, Tags: []string{}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidConfig))
	})

	t.Run("negative base value", func(t *testing.T) {
		config := &ItemConfig{
			Version:       "1.0",
			ValidTags:     []string{},
			ValidHandlers: []string{},
			Items: []ItemDef{
				{InternalName: "item1", PublicName: "Item", DefaultDisplay: "Item", BaseValue: -10, Tags: []string{}},
			},
		}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidConfig))
	})

	t.Run("valid handler", func(t *testing.T) {
		handler := "lootbox"
		config := &ItemConfig{
			Version:       "1.0",
			ValidTags:     []string{},
			ValidHandlers: []string{"lootbox"},
			Items: []ItemDef{
				{
					InternalName:   "item1",
					PublicName:     "Item",
					DefaultDisplay: "Item",
					Tags:           []string{},
					Handler:        &handler,
				},
			},
		}
		err := loader.Validate(config)
		assert.NoError(t, err)
	})
}

func TestItemLoader_LoadActualConfig(t *testing.T) {
	// Test that we can load the actual items.json
	loader := NewLoader()

	// Path relative to the test file location
	configPath := filepath.Join("..", "..", "configs", "items", "items.json")

	// Skip if file doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("items.json not found, skipping")
	}

	config, err := loader.Load(configPath)
	require.NoError(t, err, "Should load actual config file")

	// Validate the loaded config
	err = loader.Validate(config)
	require.NoError(t, err, "Actual config should be valid")

	// Check expected structure
	assert.Equal(t, "1.0", config.Version)
	assert.NotEmpty(t, config.ValidTags, "Should have valid tags defined")
	assert.NotEmpty(t, config.ValidHandlers, "Should have valid handlers defined")
	assert.NotEmpty(t, config.Items, "Should have items defined")

	// Verify specific items exist
	itemsByName := make(map[string]ItemDef)
	for _, item := range config.Items {
		itemsByName[item.InternalName] = item
	}

	// Check for expected items
	expectedItems := []string{"money", "lootbox_tier0", "lootbox_tier1", "lootbox_tier2", "weapon_blaster"}
	for _, expectedItem := range expectedItems {
		_, exists := itemsByName[expectedItem]
		assert.True(t, exists, "Expected item '%s' to exist", expectedItem)
	}
}

// Helper functions

func createTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "item_config_*.json")
	require.NoError(t, err)

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}
