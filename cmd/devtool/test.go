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
	return "Run tests with JSON output filtering and log failures"
}

type TestEvent struct {
	Time    string  `json:"Time"`
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
	Output  string  `json:"Output"`
}

func (c *TestCommand) Run(args []string) error {
	PrintHeader("Running tests...")

	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logPath := filepath.Join(logDir, "test_failures.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	// Find the go test command arguments
	cmdArgs := []string{"test", "-json"}

	// Determine if packages were provided
	hasPackages := false
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			hasPackages = true
			break
		}
	}
	if !hasPackages {
		cmdArgs = append(cmdArgs, "./...")
	}

	cmdArgs = append(cmdArgs, args...)

	// Add -race if not already there
	hasRace := false
	for _, arg := range cmdArgs {
		if arg == "-race" {
			hasRace = true
			break
		}
	}
	if !hasRace {
		cmdArgs = append(cmdArgs, "-race")
	}

	cmd := exec.Command("go", cmdArgs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start go test: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	// Increase max capacity to 1MB per line for long JSON payloads
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	testOutputs := make(map[string]*strings.Builder)
	var failedTests []string

	getBuilder := func(k string) *strings.Builder {
		if b, ok := testOutputs[k]; ok {
			return b
		}
		b := &strings.Builder{}
		testOutputs[k] = b
		return b
	}

	for scanner.Scan() {
		line := scanner.Text()

		var event TestEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Not a JSON event, could be build failure or warning
			if !strings.HasPrefix(line, "?") && !strings.Contains(line, "[no test files]") {
				fmt.Println(line)
				logFile.WriteString(line + "\n")
			}
			continue
		}

		key := event.Package
		if event.Test != "" {
			key = event.Package + "/" + event.Test
		}

		if event.Output != "" {
			getBuilder(key).WriteString(event.Output)
		}

		switch event.Action {
		case "pass":
			if event.Test == "" && event.Package != "" {
				// We don't want to show cached or no tests if we don't have to, but
				// to be consistent with go test output, it's nice to see package passes.
				output := getBuilder(event.Package).String()
				if !strings.Contains(output, "[no test files]") {
					status := "ok  "
					if strings.Contains(output, "(cached)") {
						fmt.Printf("%s\t%s\t(cached)\n", status, event.Package)
					} else {
						fmt.Printf("%s\t%s\t%.3fs\n", status, event.Package, event.Elapsed)
					}
				}
			}
		case "fail":
			if event.Test == "" && event.Package != "" {
				fmt.Printf("FAIL\t%s\t%.3fs\n", event.Package, event.Elapsed)

				logFile.WriteString(fmt.Sprintf("\n=== PACKAGE FAILED: %s ===\n", event.Package))
				logFile.WriteString(getBuilder(event.Package).String())
			} else if event.Test != "" {
				failedTests = append(failedTests, fmt.Sprintf("%s (%s)", event.Test, event.Package))

				logFile.WriteString(fmt.Sprintf("\n--- TEST FAILED: %s (%s) ---\n", event.Test, event.Package))
				logFile.WriteString(getBuilder(event.Package + "/" + event.Test).String())
			}
		}
	}

	err = cmd.Wait()

	if len(failedTests) > 0 {
		fmt.Println("\nFailed Tests:")
		for _, ft := range failedTests {
			PrintError("  " + ft)
		}
		PrintError("\nDetailed failures logged to %s", logPath)
	}

	if err != nil {
		return fmt.Errorf("tests failed")
	}

	PrintSuccess("\nAll tests passed successfully!")
	return nil
}
