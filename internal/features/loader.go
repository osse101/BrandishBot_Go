package features

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type FeatureData struct {
	Description string   `json:"description"`
	Commands    []string `json:"commands"`
}

type Loader struct {
	dir    string
	data   map[string]FeatureData
	mu     sync.RWMutex
	loaded bool
}

func NewLoader(dir string) *Loader {
	return &Loader{
		dir:  dir,
		data: make(map[string]FeatureData),
	}
}

func (l *Loader) Load() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return fmt.Errorf(ErrMsgReadDirectoryFailed, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != FeatureFileExtension {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), FeatureFileExtension)
		path := filepath.Join(l.dir, entry.Name())

		data, err := l.parseFile(path)
		if err != nil {
			return fmt.Errorf(ErrMsgParseFileFailed, name, err)
		}

		l.data[name] = data
	}

	l.loaded = true
	return nil
}

func (l *Loader) GetFeature(name string) (FeatureData, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Lazy load if not already loaded
	if !l.loaded {
		// Release read lock to acquire write lock in Load
		l.mu.RUnlock()
		if err := l.Load(); err != nil {
			l.mu.RLock()
			return FeatureData{}, false
		}
		l.mu.RLock()
	}

	data, ok := l.data[name]
	return data, ok
}

func (l *Loader) GetAllFeatures() map[string]FeatureData {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if !l.loaded {
		l.mu.RUnlock()
		_ = l.Load()
		l.mu.RLock()
	}

	// Return a copy to prevent modification
	result := make(map[string]FeatureData, len(l.data))
	for k, v := range l.data {
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
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)
	parsingCommands := false

	for scanner.Scan() {
		line := scanner.Text()

		if line == FeatureFileDelimiter {
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
