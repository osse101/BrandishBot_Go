package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

//nolint:gocyclo // Test runner logic handles parsing JSON stream of go test, inherently complex switch
func (c *TestCommand) Run(args []string) error {
	PrintHeader("Running Tests...")

	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	failLogPath := filepath.Join(logDir, "test_failures.log")
	// Clean up previous failure log
	_ = os.Remove(failLogPath)

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

	cmd := exec.Command("go", testArgs...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start tests: %w", err)
	}

	scanner := bufio.NewScanner(stdoutPipe)

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

		if event.Test != "" {
			if testOutputs[event.Package] == nil {
				testOutputs[event.Package] = make(map[string][]string)
			}
			if event.Action == "output" {
				testOutputs[event.Package][event.Test] = append(testOutputs[event.Package][event.Test], event.Output)
			}
		} else if event.Action == "output" {
			// Package level output
			if strings.Contains(event.Output, "[no test files]") {
				continue // Ignore packages with no tests
			}
			if strings.Contains(event.Output, "(cached)") && strings.HasPrefix(event.Output, "ok") {
				continue // Ignore cached passing packages
			}
			if testOutputs[event.Package] == nil {
				testOutputs[event.Package] = make(map[string][]string)
			}
			testOutputs[event.Package][""] = append(testOutputs[event.Package][""], event.Output)
		}

		switch event.Action {
		case "build-fail":
			PrintError("Build failed for %s", event.Package)
			if failLogFile == nil {
				failLogFile, err = os.OpenFile(failLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
				if err != nil {
					PrintError("Failed to open fail log: %v", err)
				} else {
					fmt.Fprintf(failLogFile, "=== TEST FAILURES ===\n\n")
				}
			}
			if failLogFile != nil {
				fmt.Fprintf(failLogFile, "--- BUILD FAIL: %s ---\n", event.Package)
			}
			failedPackages[event.Package] = true
		case "pass":
			if event.Test == "" && event.Package != "" {
				// Package passed
				if !failedPackages[event.Package] { // Make sure no subtest failed
					// Only print package pass if we didn't skip it
					if len(testOutputs[event.Package][""]) > 0 {
						PrintSuccess("ok\t%s\t%.3fs", event.Package, event.Elapsed)
					}
				}
			}
		case "fail":
			if event.Test == "" && event.Package != "" {
				// Package failed
				failedPackages[event.Package] = true
				PrintError("FAIL\t%s", event.Package)

				//nolint:all // Only log package-level failures if no specific subtests failed,
				// or just log the package output if there's no test info.
				// Often, `go test` output will already have the `FAIL: <test>` block logged in the `else` branch.
				// We only print the package fail header here, no duplicate dumping.
				if failLogFile == nil {
					failLogFile, err = os.OpenFile(failLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
					if err != nil {
						PrintError("Failed to open fail log: %v", err)
					} else {
						fmt.Fprintf(failLogFile, "=== TEST FAILURES ===\n\n")
					}
				}

				if failLogFile != nil {
					fmt.Fprintf(failLogFile, "--- FAIL: Package %s ---\n", event.Package)
					// Write all failed tests for this package
					// Actually, let's just dump the entire package output to the failure log
					for _, lines := range testOutputs[event.Package] {
						for _, l := range lines {
							fmt.Fprint(failLogFile, l)
						}
					}
					fmt.Fprintf(failLogFile, "\n")
				}
			} else {
				// Test failed
				if failLogFile == nil {
					failLogFile, err = os.OpenFile(failLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
					if err != nil {
						PrintError("Failed to open fail log: %v", err)
					} else {
						fmt.Fprintf(failLogFile, "=== TEST FAILURES ===\n\n")
					}
				}
				if failLogFile != nil {
					fmt.Fprintf(failLogFile, "--- FAIL: %s/%s ---\n", event.Package, event.Test)
					for _, l := range testOutputs[event.Package][event.Test] {
						fmt.Fprint(failLogFile, l)
					}
					fmt.Fprintf(failLogFile, "\n")
				}
			}
		}
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
