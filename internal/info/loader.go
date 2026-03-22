package info

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type Loader struct {
	dir     string
	cache   map[string]*Feature
	cacheMu sync.RWMutex
	loaded  bool
}

func NewLoader(dir string) *Loader {
	return &Loader{
		dir:   dir,
		cache: make(map[string]*Feature),
	}
}

func (l *Loader) Load() error {
	l.cacheMu.Lock()
	defer l.cacheMu.Unlock()

	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return fmt.Errorf("failed to read info directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")
		path := filepath.Join(l.dir, entry.Name())

		feature, err := l.loadFeatureFile(path)
		if err != nil {
			return fmt.Errorf("failed to load feature %s: %w", name, err)
		}

		l.cache[name] = feature
	}

	l.loaded = true
	return nil
}

func (l *Loader) loadFeatureFile(path string) (*Feature, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var feature Feature
	if err := yaml.Unmarshal(data, &feature); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &feature, nil
}

func (l *Loader) ensureLoaded() bool {
	if l.loaded {
		return true
	}

	// Upgrade to write lock to load
	l.cacheMu.RUnlock()
	err := l.Load()
	l.cacheMu.RLock()

	return err == nil
}

func (l *Loader) GetFeature(name string) (*Feature, bool) {
	l.cacheMu.RLock()
	defer l.cacheMu.RUnlock()

	if !l.ensureLoaded() {
		return nil, false
	}

	feature, ok := l.cache[name]
	return feature, ok
}

func (l *Loader) GetTopic(featureName, topicName string) (*Topic, bool) {
	feature, ok := l.GetFeature(featureName)
	if !ok {
		return nil, false
	}

	topic, ok := feature.Topics[topicName]
	if !ok {
		return nil, false
	}

	return &topic, true
}

func (l *Loader) SearchTopic(topicName string) (*Topic, string, bool) {
	l.cacheMu.RLock()
	defer l.cacheMu.RUnlock()

	if !l.ensureLoaded() {
		return nil, "", false
	}

	for featureName, feature := range l.cache {
		if topic, ok := feature.Topics[topicName]; ok {
			return &topic, featureName, true
		}
	}

	return nil, "", false
}

func (l *Loader) GetAllFeatures() map[string]*Feature {
	l.cacheMu.RLock()
	defer l.cacheMu.RUnlock()

	if !l.ensureLoaded() {
		return make(map[string]*Feature)
	}

	// Return a copy to prevent modification
	result := make(map[string]*Feature, len(l.cache))
	for k, v := range l.cache {
		result[k] = v
	}
	return result
}
