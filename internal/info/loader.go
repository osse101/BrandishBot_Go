package info

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Loader struct {
	dir   string
	cache map[string]*Feature
}

func NewLoader(dir string) *Loader {
	l := &Loader{
		dir:   dir,
		cache: make(map[string]*Feature),
	}
	_ = l.Load() // Load eagerly, ignore error for now as per previous pattern
	return l
}

func (l *Loader) Load() error {

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

func (l *Loader) GetFeature(name string) (*Feature, bool) {

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

	for featureName, feature := range l.cache {
		if topic, ok := feature.Topics[topicName]; ok {
			return &topic, featureName, true
		}
	}

	return nil, "", false
}

func (l *Loader) GetOverview() (*Feature, bool) {
	return l.GetFeature("overview")
}

func (l *Loader) GetFeatures() map[string]*Feature {
	return l.cache
}
