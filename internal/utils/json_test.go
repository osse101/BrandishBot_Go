package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadJSON tests the JSON loading functionality
func TestLoadJSON(t *testing.T) {
	t.Run("loads valid JSON file successfully", func(t *testing.T) {
		// Create a temp file with valid JSON
		tmpDir := t.TempDir()
		jsonFile := filepath.Join(tmpDir, "test.json")

		content := `{"name": "test", "value": 42}`
		err := os.WriteFile(jsonFile, []byte(content), 0600)
		require.NoError(t, err)

		// Load into a struct
		var result struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		err = LoadJSON(jsonFile, &result)

		// Verify
		assert.NoError(t, err)
		assert.Equal(t, "test", result.Name)
		assert.Equal(t, 42, result.Value)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		var result map[string]interface{}
		err := LoadJSON("/nonexistent/path/file.json", &result)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read file")
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		jsonFile := filepath.Join(tmpDir, "invalid.json")

		// Write invalid JSON
		err := os.WriteFile(jsonFile, []byte("{invalid json}"), 0600)
		require.NoError(t, err)

		var result map[string]interface{}
		err = LoadJSON(jsonFile, &result)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal JSON")
	})

	t.Run("handles empty JSON object", func(t *testing.T) {
		tmpDir := t.TempDir()
		jsonFile := filepath.Join(tmpDir, "empty.json")

		err := os.WriteFile(jsonFile, []byte("{}"), 0600)
		require.NoError(t, err)

		var result map[string]interface{}
		err = LoadJSON(jsonFile, &result)

		assert.NoError(t, err)
		assert.Empty(t, result)
	})
}

// TestSaveJSON tests the JSON saving functionality
func TestSaveJSON(t *testing.T) {
	t.Run("saves data to JSON file with proper formatting", func(t *testing.T) {
		tmpDir := t.TempDir()
		jsonFile := filepath.Join(tmpDir, "output.json")

		data := map[string]interface{}{
			"name":  "test",
			"value": 42,
			"tags":  []string{"a", "b"},
		}

		err := SaveJSON(jsonFile, data)
		require.NoError(t, err)

		// Verify file exists and has content
		content, err := os.ReadFile(jsonFile)
		require.NoError(t, err)

		// Should be indented (has newlines)
		assert.Contains(t, string(content), "\n")
		assert.Contains(t, string(content), "\"name\"")
		assert.Contains(t, string(content), "\"test\"")

		// Verify file permissions
		info, err := os.Stat(jsonFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
		data := map[string]string{"key": "value"}
		err := SaveJSON("/invalid/path/to/file.json", data)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write file")
	})

	t.Run("handles non-serializable data gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		jsonFile := filepath.Join(tmpDir, "invalid.json")

		// Channels cannot be marshaled to JSON
		data := map[string]interface{}{
			"channel": make(chan int),
		}

		err := SaveJSON(jsonFile, data)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal data")
	})
}

// TestLoadJSON_SaveJSON_RoundTrip verifies round-trip consistency
func TestLoadJSON_SaveJSON_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "roundtrip.json")

	original := struct {
		Name    string   `json:"name"`
		Count   int      `json:"count"`
		Enabled bool     `json:"enabled"`
		Tags    []string `json:"tags"`
	}{
		Name:    "test-item",
		Count:   100,
		Enabled: true,
		Tags:    []string{"tag1", "tag2"},
	}

	// Save
	err := SaveJSON(jsonFile, original)
	require.NoError(t, err)

	// Load
	var loaded struct {
		Name    string   `json:"name"`
		Count   int      `json:"count"`
		Enabled bool     `json:"enabled"`
		Tags    []string `json:"tags"`
	}
	err = LoadJSON(jsonFile, &loaded)
	require.NoError(t, err)

	// Verify data integrity
	assert.Equal(t, original.Name, loaded.Name)
	assert.Equal(t, original.Count, loaded.Count)
	assert.Equal(t, original.Enabled, loaded.Enabled)
	assert.Equal(t, original.Tags, loaded.Tags)
}
