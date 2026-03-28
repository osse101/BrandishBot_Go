package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type TestLogsCommand struct{}

func (c *TestLogsCommand) Name() string {
	return "test-logs"
}

func (c *TestLogsCommand) Description() string {
	return "Test log file rotation"
}

func (c *TestLogsCommand) Run(args []string) error {
	PrintHeader("Testing Log File Rotation")

	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Clean old log files
	files, err := os.ReadDir(logDir)
	if err == nil {
		for _, f := range files {
			if !f.IsDir() && filepath.Ext(f.Name()) == ".log" {
				os.Remove(filepath.Join(logDir, f.Name()))
			}
		}
	}

	// Build the app
	appPath, err := c.buildApp()
	if err != nil {
		return err
	}

	if err := c.runAppLoop(appPath); err != nil {
		return err
	}

	return c.verifyLogs(logDir)
}

func (c *TestLogsCommand) buildApp() (string, error) {
	PrintInfo("Building the application...")
	//nolint:forbidigo // Safe usage of wrapper
	if err := runCommandVerbose("make", "build"); err != nil {
		return "", fmt.Errorf("failed to build app: %w", err)
	}

	appName := "app"
	if runtime.GOOS == "windows" {
		appName = "app.exe"
	}
	appPath := filepath.Join("bin", appName)
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		return "", fmt.Errorf("app binary not found at %s", appPath)
	}
	return appPath, nil
}

func (c *TestLogsCommand) runAppLoop(appPath string) error {
	for i := 1; i <= 12; i++ {
		fmt.Printf("Run #%d\n", i)

		// Run app in background
		if err := runCommandAsyncAndKill(appPath, 3*time.Second); err != nil {
			PrintError("Failed to run app on run %d: %v", i, err)
		}

		time.Sleep(1 * time.Second)
	}
	return nil
}

func (c *TestLogsCommand) verifyLogs(logDir string) error {
	// Check logs
	files, err := os.ReadDir(logDir)
	if err != nil {
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	var logFiles []string
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".log" {
			logFiles = append(logFiles, f.Name())
		}
	}

	fmt.Printf("Total log files: %d\n", len(logFiles))

	if len(logFiles) == 10 {
		PrintSuccess("SUCCESS: Found 10 log files.")
	} else {
		PrintError("FAILURE: Found %d log files. Expected 10.", len(logFiles))
		return fmt.Errorf("expected 10 log files, got %d", len(logFiles))
	}

	for _, name := range logFiles {
		fmt.Println(name)
	}

	return nil
}

// runCommandAsyncAndKill starts a process and kills it after a timeout
func runCommandAsyncAndKill(path string, timeout time.Duration) error {
	// nolint:forbidigo
	cmd, err := runCommandAsync(filepath.Clean(path))
	if err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
		<-done // Wait for goroutine to exit
		return nil
	case err := <-done:
		return err
	}
}
