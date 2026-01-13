package progression

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeLoader_SyncToDatabase_ChangeDetection(t *testing.T) {
	loader := NewTreeLoader()
	repo := NewMockRepository()
	ctx := context.Background()

	// Initial config content
	initialContent := `{
		"version": "1.0",
		"description": "Initial Tree",
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
			}
		]
	}`

	// Create temp file
	tmpFile := createTempFile(t, initialContent)
	defer os.Remove(tmpFile)

	// Step 1: First Sync (New File)
	// Expectation: Sync should proceed
	t.Run("first sync (new file)", func(t *testing.T) {
		config, err := loader.Load(tmpFile)
		require.NoError(t, err)

		result, err := loader.SyncToDatabase(ctx, config, repo, tmpFile)
		require.NoError(t, err)
		assert.Equal(t, 1, result.NodesInserted)

		// Verify metadata was created
		meta, err := repo.GetSyncMetadata(ctx, "progression_tree.json")
		require.NoError(t, err)
		assert.NotNil(t, meta)
		assert.NotEmpty(t, meta.FileHash)
	})

	// Step 2: Second Sync (Unchanged File)
	// Expectation: Sync should be skipped
	t.Run("second sync (unchanged)", func(t *testing.T) {
		config, err := loader.Load(tmpFile)
		require.NoError(t, err)

		result, err := loader.SyncToDatabase(ctx, config, repo, tmpFile)
		require.NoError(t, err)

		// Result should indicate skipped
		// Note: SyncToDatabase returns empty result if skipped
		assert.Equal(t, 0, result.NodesInserted)
		assert.Equal(t, 0, result.NodesUpdated)
		assert.Equal(t, 0, result.NodesSkipped)

		// Verify metadata is unchanged (optional, but good sanity check)
		meta, err := repo.GetSyncMetadata(ctx, "progression_tree.json")
		require.NoError(t, err)
		assert.NotNil(t, meta)
	})

	// Step 3: Third Sync (Changed File)
	// Expectation: Sync should proceed
	t.Run("third sync (changed file)", func(t *testing.T) {
		// Modify file - wait 1s to ensure modtime changes if FS has low resolution
		// (though hash check should catch it regardless)
		time.Sleep(10 * time.Millisecond)

		newContent := `{
			"version": "1.0",
			"description": "Updated Tree",
			"nodes": [
				{
					"key": "root",
					"name": "Root Updated",
					"type": "feature",
					"description": "Root node updated",
					"tier": 0,
					"size": "medium",
					"category": "core",
					"max_level": 1,
					"prerequisites": [],
					"sort_order": 0
				}
			]
		}`
		err := os.WriteFile(tmpFile, []byte(newContent), 0644)
		require.NoError(t, err)

		config, err := loader.Load(tmpFile)
		require.NoError(t, err)
		assert.Equal(t, "Root Updated", config.Nodes[0].Name)

		result, err := loader.SyncToDatabase(ctx, config, repo, tmpFile)
		require.NoError(t, err)

		// Should be updated now
		assert.Equal(t, 0, result.NodesInserted) // Updates don't count as inserts
		assert.Equal(t, 1, result.NodesUpdated)  // Should be 1 update

		// Check metadata updated
		meta, err := repo.GetSyncMetadata(ctx, "progression_tree.json")
		require.NoError(t, err)
		assert.NotNil(t, meta)
	})
}
