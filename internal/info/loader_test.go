package info_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/info"
)

func setupTestDir(t *testing.T) string {
	dir := t.TempDir()

	// Create valid feature file
	validFeature := `discord:
  description: "Test Discord Description"
streamerbot:
  description: "Test Streamerbot Description"
topics:
  topic1:
    discord:
      description: "Topic 1 Discord"
    streamerbot:
      description: "Topic 1 Streamerbot"`
	err := os.WriteFile(filepath.Join(dir, "test_feature.yaml"), []byte(validFeature), 0644)
	assert.NoError(t, err)

	// Create another valid feature file
	anotherFeature := `discord:
  description: "Another Discord Description"
streamerbot:
  description: "Another Streamerbot Description"
topics:
  topic2:
    discord:
      description: "Topic 2 Discord"
    streamerbot:
      description: "Topic 2 Streamerbot"`
	err = os.WriteFile(filepath.Join(dir, "another_feature.yaml"), []byte(anotherFeature), 0644)
	assert.NoError(t, err)

	return dir
}

func setupInvalidTestDir(t *testing.T) string {
	dir := t.TempDir()

	// Create invalid yaml file
	invalidFeature := `discord: "Test Discord Description`
	err := os.WriteFile(filepath.Join(dir, "invalid_feature.yaml"), []byte(invalidFeature), 0644)
	assert.NoError(t, err)

	return dir
}

func TestLoader_Load(t *testing.T) {
	t.Run("Valid directory", func(t *testing.T) {
		dir := setupTestDir(t)
		loader := info.NewLoader(dir)
		err := loader.Load()
		assert.NoError(t, err)
	})

	t.Run("Invalid directory", func(t *testing.T) {
		loader := info.NewLoader("/non/existent/dir")
		err := loader.Load()
		assert.Error(t, err)
	})

	t.Run("Invalid yaml file", func(t *testing.T) {
		dir := setupInvalidTestDir(t)
		loader := info.NewLoader(dir)
		err := loader.Load()
		assert.Error(t, err)
	})
}

func TestLoader_GetFeature(t *testing.T) {
	dir := setupTestDir(t)
	loader := info.NewLoader(dir)

	t.Run("Existing feature", func(t *testing.T) {
		feature, ok := loader.GetFeature("test_feature")
		assert.True(t, ok)
		assert.NotNil(t, feature)
		assert.Equal(t, "Test Discord Description", feature.Discord.Description)
	})

	t.Run("Non-existing feature", func(t *testing.T) {
		feature, ok := loader.GetFeature("nonexistent")
		assert.False(t, ok)
		assert.Nil(t, feature)
	})
}

func TestLoader_GetTopic(t *testing.T) {
	dir := setupTestDir(t)
	loader := info.NewLoader(dir)

	t.Run("Existing feature and topic", func(t *testing.T) {
		topic, ok := loader.GetTopic("test_feature", "topic1")
		assert.True(t, ok)
		assert.NotNil(t, topic)
		assert.Equal(t, "Topic 1 Discord", topic.Discord.Description)
	})

	t.Run("Non-existing feature", func(t *testing.T) {
		topic, ok := loader.GetTopic("nonexistent", "topic1")
		assert.False(t, ok)
		assert.Nil(t, topic)
	})

	t.Run("Existing feature, non-existing topic", func(t *testing.T) {
		topic, ok := loader.GetTopic("test_feature", "nonexistent")
		assert.False(t, ok)
		assert.Nil(t, topic)
	})
}

func TestLoader_GetAllFeatures(t *testing.T) {
	dir := setupTestDir(t)
	loader := info.NewLoader(dir)

	t.Run("Returns all features", func(t *testing.T) {
		features := loader.GetAllFeatures()
		assert.Len(t, features, 2)
		assert.Contains(t, features, "test_feature")
		assert.Contains(t, features, "another_feature")
	})

	t.Run("Returns empty map for invalid dir on lazy load", func(t *testing.T) {
		loader := info.NewLoader("/non/existent/dir")
		features := loader.GetAllFeatures()
		assert.Empty(t, features)
	})
}
