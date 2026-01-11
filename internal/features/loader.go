package features

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FeatureData represents the loaded information for a feature
type FeatureData struct {
	Description string   `json:"description"`
	Commands    []string `json:"commands"`
}

// Loader handles loading feature data from files
type Loader struct {
	dir      string
	cache    map[string]FeatureData
	cacheMu  sync.RWMutex
	loaded   bool
}

// NewLoader creates a new feature loader
func NewLoader(dir string) *Loader {
	return &Loader{
		dir:   dir,
		cache: make(map[string]FeatureData),
	}
}

// Load reads all feature files from the directory
func (l *Loader) Load() error {
	l.cacheMu.Lock()
	defer l.cacheMu.Unlock()

	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return fmt.Errorf("failed to read feature directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".txt" {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".txt")
		path := filepath.Join(l.dir, entry.Name())

		data, err := l.parseFile(path)
		if err != nil {
			return fmt.Errorf("failed to parse feature file %s: %w", name, err)
		}

		l.cache[name] = data
	}

	l.loaded = true
	return nil
}

// GetFeature returns data for a specific feature
func (l *Loader) GetFeature(name string) (FeatureData, bool) {
	l.cacheMu.RLock()
	defer l.cacheMu.RUnlock()
	
	// Lazy load if not already loaded
	if !l.loaded {
		// Release read lock to acquire write lock in Load
		l.cacheMu.RUnlock() 
		if err := l.Load(); err != nil {
			l.cacheMu.RLock()
			return FeatureData{}, false
		}
		l.cacheMu.RLock()
	}

	data, ok := l.cache[name]
	return data, ok
}

// GetAllFeatures returns all loaded features
func (l *Loader) GetAllFeatures() map[string]FeatureData {
	l.cacheMu.RLock()
	defer l.cacheMu.RUnlock()
	
	if !l.loaded {
		l.cacheMu.RUnlock()
		_ = l.Load()
		l.cacheMu.RLock()
	}

	// Return a copy to prevent modification
	result := make(map[string]FeatureData, len(l.cache))
	for k, v := range l.cache {
		result[k] = v
	}
	return result
}

func (l *Loader) parseFile(path string) (FeatureData, error) {
	file, err := os.Open(path)
	if err != nil {
		return FeatureData{}, err
	}
	defer file.Close()

	var description strings.Builder
	var commands []string
	scanner := bufio.NewScanner(file)
	parsingCommands := false

	for scanner.Scan() {
		line := scanner.Text()
		
		if line == "---" {
			parsingCommands = true
			continue
		}

		if parsingCommands {
			if strings.TrimSpace(line) != "" {
				commands = append(commands, strings.TrimSpace(line))
			}
		} else {
			if description.Len() > 0 {
				description.WriteString("\n")
			}
			description.WriteString(line)
		}
	}

	if err := scanner.Err(); err != nil {
		return FeatureData{}, err
	}

	return FeatureData{
		Description: strings.TrimSpace(description.String()),
		Commands:    commands,
	}, nil
}
