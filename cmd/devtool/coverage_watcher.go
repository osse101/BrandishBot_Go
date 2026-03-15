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

	// Recursively add directories to watch
	if err := addRecursiveWatch(watcher, "."); err != nil {
		return fmt.Errorf("failed to add watch paths: %w", err)
	}

	PrintInfo("Watching for file changes...")
	PrintInfo("Press Ctrl+C to exit.")

	// Run initial check
	if err := runCoverageCheck(config); err != nil {
		PrintError("Initial check failed: %v", err)
	}

	var debounceTimer *time.Timer
	debounceDuration := 200 * time.Millisecond

	done := make(chan bool)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Handle new directories
				if event.Op&fsnotify.Create == fsnotify.Create {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						if err := addRecursiveWatch(watcher, event.Name); err != nil {
							PrintWarning("Failed to watch new directory %s: %v", event.Name, err)
						}
					}
				}

				// Filter for interesting events
				if !strings.HasSuffix(event.Name, ".go") && !strings.HasSuffix(event.Name, ".mod") {
					continue
				}

				// Ignore Chmod
				if event.Op&fsnotify.Chmod == fsnotify.Chmod {
					continue
				}

				// Debounce
				if debounceTimer != nil {
					debounceTimer.Stop()
				}

				debounceTimer = time.AfterFunc(debounceDuration, func() {
					clearScreen()
					PrintInfo("Change detected: %s", event.Name)
					if err := runCoverageCheck(config); err != nil {
						PrintError("Check failed: %v", err)
					} else {
						// Print timestamp of last success
						PrintSuccess("Last success: %s", time.Now().Format("15:04:05"))
					}
					PrintInfo("Watching for file changes...")
				})

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				PrintError("Watcher error: %v", err)
			}
		}
	}()

	<-done
	return nil
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
