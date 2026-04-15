package info_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/info"
)

func createTestDir(t *testing.T) string {
	dir := t.TempDir()

	feature1YAML := `
name: feature1
title: Feature 1
topics:
  topic1:
    command: /topic1
    discord:
      description: Discord desc 1
    streamerbot:
      description: SB desc 1
`
	err := os.WriteFile(filepath.Join(dir, "feature1.yaml"), []byte(feature1YAML), 0644)
	require.NoError(t, err)

	feature2YAML := `
name: feature2
title: Feature 2
topics:
  topic2:
    command: /topic2
    discord:
      description: Discord desc 2
    streamerbot:
      description: SB desc 2
`
	err = os.WriteFile(filepath.Join(dir, "feature2.yaml"), []byte(feature2YAML), 0644)
	require.NoError(t, err)

    // Invalid YAML
	err = os.WriteFile(filepath.Join(dir, "invalid.yaml"), []byte("invalid:\n  - yaml: ["), 0644)
	require.NoError(t, err)

    // Directory that should be ignored
    err = os.Mkdir(filepath.Join(dir, "ignored_dir"), 0755)
    require.NoError(t, err)

    // Non-YAML file that should be ignored
    err = os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte("ignore me"), 0644)
    require.NoError(t, err)

	return dir
}

func TestNewLoader(t *testing.T) {
	loader := info.NewLoader("testdir")
	assert.NotNil(t, loader)
}

func TestLoader_Load(t *testing.T) {
	dir := createTestDir(t)


    // Test successful load (ignoring invalid yaml to prevent failure)
    // Actually, `loadFeatureFile` will return error on invalid yaml, and `Load` will return error.
    // Let's create a clean dir for success case.

    cleanDir := t.TempDir()
    err := os.WriteFile(filepath.Join(cleanDir, "valid.yaml"), []byte("name: valid\n"), 0644)
    require.NoError(t, err)

    loaderClean := info.NewLoader(cleanDir)
	err = loaderClean.Load()
	assert.NoError(t, err)

    // Test invalid dir
    loaderInvalidDir := info.NewLoader("/non/existent/dir/12345")
    err = loaderInvalidDir.Load()
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "failed to read info directory")

    // Test invalid yaml
    loaderInvalidYaml := info.NewLoader(dir)
    err = loaderInvalidYaml.Load()
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "failed to load feature invalid")
}

func TestLoader_GetFeature(t *testing.T) {
    cleanDir := t.TempDir()
    err := os.WriteFile(filepath.Join(cleanDir, "feature1.yaml"), []byte("name: feature1\n"), 0644)
    require.NoError(t, err)

	loader := info.NewLoader(cleanDir)

    // Implicitly tests ensureLoaded
	feature, ok := loader.GetFeature("feature1")
	assert.True(t, ok)
	assert.NotNil(t, feature)
	assert.Equal(t, "feature1", feature.Name)

	feature, ok = loader.GetFeature("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, feature)
}

func TestLoader_GetTopic(t *testing.T) {
    cleanDir := t.TempDir()
    yamlContent := `
name: feature1
topics:
  topic1:
    command: /topic1
`
    err := os.WriteFile(filepath.Join(cleanDir, "feature1.yaml"), []byte(yamlContent), 0644)
    require.NoError(t, err)

	loader := info.NewLoader(cleanDir)

	topic, ok := loader.GetTopic("feature1", "topic1")
	assert.True(t, ok)
	assert.NotNil(t, topic)
	assert.Equal(t, "/topic1", topic.Command)

	topic, ok = loader.GetTopic("feature1", "nonexistent")
	assert.False(t, ok)
	assert.Nil(t, topic)

	topic, ok = loader.GetTopic("nonexistent", "topic1")
	assert.False(t, ok)
	assert.Nil(t, topic)
}

func TestLoader_GetAllFeatures(t *testing.T) {
    cleanDir := t.TempDir()
    err := os.WriteFile(filepath.Join(cleanDir, "feature1.yaml"), []byte("name: feature1\n"), 0644)
    require.NoError(t, err)
    err = os.WriteFile(filepath.Join(cleanDir, "feature2.yaml"), []byte("name: feature2\n"), 0644)
    require.NoError(t, err)

	loader := info.NewLoader(cleanDir)

	features := loader.GetAllFeatures()
	assert.Len(t, features, 2)
	assert.Contains(t, features, "feature1")
	assert.Contains(t, features, "feature2")
}

func TestLoader_EnsureLoaded_Error(t *testing.T) {
    // Test that when ensureLoaded fails, Get methods return false
    loader := info.NewLoader("/invalid/dir/that/does/not/exist")

    feature, ok := loader.GetFeature("any")
    assert.False(t, ok)
    assert.Nil(t, feature)

    topic, ok := loader.GetTopic("any", "topic")
    assert.False(t, ok)
    assert.Nil(t, topic)

    features := loader.GetAllFeatures()
    assert.Empty(t, features)
}
