package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	testLogDir      = "logs"
	testFailLogFile = "test_failures.log"
)

type TestCommand struct{}

func (c *TestCommand) Name() string {
	return "test"
}

func (c *TestCommand) Description() string {
	return "Run tests with filtered output and failure logging"
}

type testEvent struct {
	Time    string  `json:"Time"`
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
	Output  string  `json:"Output"`
}

func (c *TestCommand) Run(args []string) error {
	PrintHeader("Running Tests...")

	if err := os.MkdirAll(testLogDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	failLogPath := filepath.Join(testLogDir, testFailLogFile)
	// Clean up previous failure log
	_ = os.Remove(failLogPath)

	testArgs := c.buildTestArgs(args)

	stdoutPipe, cmd, err := runCommandWithStdoutPipe("go", testArgs...)
	if err != nil {
		return fmt.Errorf("failed to start tests: %w", err)
	}
	cmd.Stderr = os.Stderr

	scanner := newScanner(stdoutPipe)

	var failLogFile *os.File

	// Track test output lines. map[Package]map[Test][]string
	testOutputs := make(map[string]map[string][]string)
	failedPackages := make(map[string]bool)

	for scanner.Scan() {
		line := scanner.Text()

		var event testEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Not a JSON line, print it directly (e.g. panic or build error)
			fmt.Println(line)
			continue
		}

		c.handleTestOutput(&event, testOutputs)
		failLogFile = c.handleTestAction(&event, failLogPath, failLogFile, testOutputs, failedPackages)
	}

	if failLogFile != nil {
		failLogFile.Close()
	}

	err = cmd.Wait()

	if len(failedPackages) > 0 {
		fmt.Println()
		PrintError("Tests failed in %d packages. See %s for details.", len(failedPackages), failLogPath)
		return fmt.Errorf("tests failed")
	}

	if err != nil {
		// If command failed but no packages were marked as failed (e.g., build error)
		return fmt.Errorf("tests failed: %w", err)
	}

	PrintSuccess("All tests passed!")
	return nil
}

func (c *TestCommand) buildTestArgs(args []string) []string {
	testArgs := []string{"test", "-json"}
	if len(args) > 0 {
		testArgs = append(testArgs, args...)
	} else {
		testArgs = append(testArgs, "./...")
	}

	// Always append race if not present
	hasRace := false
	for _, arg := range args {
		if arg == "-race" {
			hasRace = true
			break
		}
	}
	if !hasRace {
		testArgs = append(testArgs, "-race")
	}
	return testArgs
}

func (c *TestCommand) handleTestOutput(event *testEvent, testOutputs map[string]map[string][]string) {
	if event.Test != "" {
		c.handleTestLevelOutput(event, testOutputs)
	} else if event.Action == "output" {
		c.handlePackageLevelOutput(event, testOutputs)
	}
}

func (c *TestCommand) handleTestLevelOutput(event *testEvent, testOutputs map[string]map[string][]string) {
	if testOutputs[event.Package] == nil {
		testOutputs[event.Package] = make(map[string][]string)
	}
	if event.Action == "output" {
		if strings.Contains(event.Output, "--- SKIP:") || strings.Contains(event.Output, "=== SKIP:") {
			return // Filter out skipped subtests output
		}
		testOutputs[event.Package][event.Test] = append(testOutputs[event.Package][event.Test], event.Output)
	}
}

func (c *TestCommand) handlePackageLevelOutput(event *testEvent, testOutputs map[string]map[string][]string) {
	// Package level output
	if strings.Contains(event.Output, "[no test files]") {
		return // Ignore packages with no tests
	}
	if strings.Contains(event.Output, "(cached)") && strings.HasPrefix(event.Output, "ok") {
		return // Ignore cached passing packages
	}
	if strings.Contains(event.Output, "[no tests to run]") {
		return // Ignore packages with no tests entirely
	}
	if event.Output == "PASS\n" || strings.HasPrefix(event.Output, "ok  \t") || strings.HasPrefix(event.Output, "?   \t") {
		return // Filter out standard pass outputs so completely quiet packages don't trigger the success print
	}
	if testOutputs[event.Package] == nil {
		testOutputs[event.Package] = make(map[string][]string)
	}
	testOutputs[event.Package][""] = append(testOutputs[event.Package][""], event.Output)
}

func (c *TestCommand) getFailLogFile(failLogPath string, currentFile *os.File) *os.File {
	if currentFile != nil {
		return currentFile
	}
	f, err := os.OpenFile(failLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		PrintError("Failed to open fail log: %v", err)
		return nil
	}
	fmt.Fprintf(f, "=== TEST FAILURES ===\n\n")
	return f
}

func (c *TestCommand) handleTestAction(event *testEvent, failLogPath string, failLogFile *os.File, testOutputs map[string]map[string][]string, failedPackages map[string]bool) *os.File {
	switch event.Action {
	case "build-fail":
		PrintError("Build failed for %s", event.Package)
		failLogFile = c.getFailLogFile(failLogPath, failLogFile)
		if failLogFile != nil {
			fmt.Fprintf(failLogFile, "--- BUILD FAIL: %s ---\n", event.Package)
		}
		failedPackages[event.Package] = true

	case "fail":
		if event.Test == "" && event.Package != "" {
			// Package failed
			failedPackages[event.Package] = true
			PrintError("FAIL\t%s", event.Package)

			failLogFile = c.getFailLogFile(failLogPath, failLogFile)
			if failLogFile != nil {
				fmt.Fprintf(failLogFile, "--- FAIL: Package %s ---\n", event.Package)
				// Write package-level output
				for _, l := range testOutputs[event.Package][""] {
					fmt.Fprint(failLogFile, l)
				}
				fmt.Fprintf(failLogFile, "\n")
			}
		} else {
			// Test failed
			failLogFile = c.getFailLogFile(failLogPath, failLogFile)
			if failLogFile != nil {
				fmt.Fprintf(failLogFile, "--- FAIL: %s/%s ---\n", event.Package, event.Test)
				for _, l := range testOutputs[event.Package][event.Test] {
					fmt.Fprint(failLogFile, l)
				}
				fmt.Fprintf(failLogFile, "\n")
			}
		}
	}
	return failLogFile
}
