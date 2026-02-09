package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type CheckCoverageCommand struct{}

func (c *CheckCoverageCommand) Name() string {
	return "check-coverage"
}

func (c *CheckCoverageCommand) Description() string {
	return "Run tests with coverage and check against threshold"
}

func (c *CheckCoverageCommand) Run(args []string) error {
	fs := flag.NewFlagSet("check-coverage", flag.ContinueOnError)
	runTests := fs.Bool("run", false, "Run tests before checking coverage")
	htmlReport := fs.Bool("html", false, "Generate and open HTML coverage report")

	if err := fs.Parse(args); err != nil {
		return err
	}

	positional := fs.Args()
	file := "logs/coverage.out"
	thresholdStr := "80"

	if len(positional) > 0 {
		file = positional[0]
	}
	if len(positional) > 1 {
		thresholdStr = positional[1]
	}

	PrintHeader(fmt.Sprintf("Checking coverage threshold (%s%%)...", thresholdStr))

	// Check if we need to run tests
	shouldRun := *runTests
	if _, err := os.Stat(file); os.IsNotExist(err) {
		PrintInfo("Coverage file '%s' not found. Running tests...", file)
		shouldRun = true
	}

	if shouldRun {
		// Ensure directory exists
		dir := filepath.Dir(file)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create coverage directory '%s': %w", dir, err)
		}

		PrintInfo("Running tests with coverage...")
		// Note: mirroring 'make test' command
		cmd := exec.Command("go", "test", "./...", "-coverprofile="+file, "-covermode=atomic", "-race")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("tests failed: %w", err)
		}
		PrintSuccess("Tests passed and coverage profile generated.")
	}

	threshold, err := strconv.ParseFloat(thresholdStr, 64)
	if err != nil {
		return fmt.Errorf("invalid threshold '%s'", thresholdStr)
	}

	// Run go tool cover -func=file
	out, err := getCommandOutput("go", "tool", "cover", fmt.Sprintf("-func=%s", file))
	if err != nil {
		return fmt.Errorf("error running go tool cover: %w", err)
	}

	lines := strings.Split(out, "\n")
	var totalLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "total:") {
			totalLine = line
			break
		}
	}

	if totalLine == "" {
		return fmt.Errorf("could not determine coverage from output")
	}

	fields := strings.Fields(totalLine)
	if len(fields) < 3 {
		return fmt.Errorf("unexpected output format")
	}

	pctStr := fields[len(fields)-1] // Last field is percentage
	pctStr = strings.TrimSuffix(pctStr, "%")

	coverage, err := strconv.ParseFloat(pctStr, 64)
	if err != nil {
		return fmt.Errorf("could not parse coverage percentage '%s'", pctStr)
	}

	PrintInfo("Total Coverage: %.1f%%", coverage)

	if *htmlReport {
		htmlFile := strings.TrimSuffix(file, ".out") + ".html"
		PrintInfo("Generating HTML report: %s", htmlFile)
		cmd := exec.Command("go", "tool", "cover", "-html="+file, "-o", htmlFile)
		if err := cmd.Run(); err != nil {
			PrintWarning("Failed to generate HTML report: %v", err)
		} else {
			PrintSuccess("HTML report generated: %s", htmlFile)
		}
	}

	if coverage >= threshold {
		PrintSuccess("Coverage meets threshold.")
		return nil
	}

	PrintError("Coverage is below threshold.")
	return fmt.Errorf("coverage below threshold")
}
