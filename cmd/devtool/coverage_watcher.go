package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

func runWatchMode(config *CoverageConfig) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	if err := addRecursiveWatch(watcher, "."); err != nil {
		return fmt.Errorf("failed to add watch paths: %w", err)
	}

	PrintInfo("Watching for file changes...")
	PrintInfo("Press Ctrl+C to exit.")

	if err := runCoverageCheck(config); err != nil {
		PrintError("Initial check failed: %v", err)
	}

	watchEvents(watcher, config)
	return nil
}

func watchEvents(watcher *fsnotify.Watcher, config *CoverageConfig) {
	var (
		debounceTimer *time.Timer
		debounceDelay = 200 * time.Millisecond
	)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if !shouldProcess(event) {
				if event.Op&fsnotify.Create == fsnotify.Create {
					handleNewDir(watcher, event.Name)
				}
				continue
			}

			if debounceTimer != nil {
				debounceTimer.Stop()
			}

			debounceTimer = time.AfterFunc(debounceDelay, func() {
				handleChange(event.Name, config)
			})

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			PrintError("Watcher error: %v", err)
		}
	}
}

func shouldProcess(event fsnotify.Event) bool {
	if event.Op&fsnotify.Chmod == fsnotify.Chmod {
		return false
	}

	ext := filepath.Ext(event.Name)
	return ext == ".go" || ext == ".mod"
}

func handleNewDir(watcher *fsnotify.Watcher, name string) {
	info, err := os.Stat(name)
	if err == nil && info.IsDir() {
		if err := addRecursiveWatch(watcher, name); err != nil {
			PrintWarning("Failed to watch new directory %s: %v", name, err)
		}
	}
}

func handleChange(name string, config *CoverageConfig) {
	clearScreen()
	PrintInfo("Change detected: %s", name)
	if err := runCoverageCheck(config); err != nil {
		PrintError("Check failed: %v", err)
	} else {
		PrintSuccess("Last success: %s", time.Now().Format("15:04:05"))
	}
	PrintInfo("Watching for file changes...")
}

func addRecursiveWatch(watcher *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}

		// Ignore hidden directories and common ignore patterns
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".") && base != "." {
			return filepath.SkipDir
		}
		if IgnoreDirs(base) {
			return filepath.SkipDir
		}

		if err := watcher.Add(path); err != nil {
			// Ignore error if path is gone (race condition)
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("failed to watch %s: %w", path, err)
		}
		return nil
	})
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func IgnoreDirs(base string) bool {
	switch base {
	case ".git", ".idea", ".vscode", "logs", "bin", "vendor", "node_modules":
		return true
	}
	return false
}
