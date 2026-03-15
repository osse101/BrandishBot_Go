package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type CheckCoverageCommand struct{}

func (c *CheckCoverageCommand) Name() string {
	return "check-coverage"
}

func (c *CheckCoverageCommand) Description() string {
	return "Run tests with coverage and check against threshold"
}

func (c *CheckCoverageCommand) Run(args []string) error {
	config, err := parseCoverageConfig(args)
	if err != nil {
		return err
	}

	if config.Watch {
		return runWatchMode(config)
	}

	return runCoverageCheck(config)
}

func runCoverageCheck(config *CoverageConfig) error {
	selector := &PackageSelector{
		SmartMode:  config.Smart,
		BaseRef:    config.BaseRef,
		Includes:   config.Packages,
		Excludes:   config.Exclude,
		StagedOnly: false,
	}

	packages, err := selector.SelectPackages()
	if err != nil {
		return err
	}

	if len(packages) == 0 && config.Smart {
		PrintInfo("Smart mode enabled but no packages selected (or all excluded). Skipping tests.")
		return nil
	}

	PrintHeader(fmt.Sprintf("Checking coverage threshold (%.1f%%)...", config.Threshold))

	if err := ensureCoverage(config.File, config.RunTests, packages, config.Verbose); err != nil {
		return err
	}

	// Show top missing functions regardless of runTests flag (if file exists)
	if _, err := os.Stat(config.File); err == nil {
		if err := printTopMissingFunctions(config.File); err != nil {
			PrintWarning("Failed to analyze missing functions: %v", err)
		}
	}

	// If we ran partial tests (packages != empty), the coverage profile only contains those packages.
	// This is fine for "check my changes" workflow.
	coverage, err := getCoveragePercent(config.File)
	if err != nil {
		return err
	}

	PrintInfo("Total Coverage: %.1f%%", coverage)

	if config.HTMLReport {
		if err := generateHTMLReport(config.File); err != nil {
			PrintWarning("Failed to generate HTML report: %v", err)
		}
	}

	if coverage < config.Threshold {
		PrintError("Coverage is below threshold.")
		return fmt.Errorf("coverage below threshold")
	}

	PrintSuccess("Coverage meets threshold.")
	return nil
}

func ensureCoverage(file string, runTests bool, packages []string, verbose bool) error {
	shouldRun := runTests

	// Test requirements: specific package requests force a fresh test run to ensure accurate coverage profiling.
	if len(packages) > 0 {
		shouldRun = true
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		PrintInfo("Coverage file '%s' not found. Running tests...", file)
		shouldRun = true
	}

	if !shouldRun {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create coverage directory '%s': %w", dir, err)
	}

	PrintInfo("Running tests with coverage...")

	testArgs := []string{"test"}

	if verbose {
		testArgs = append(testArgs, "-v")
	}

	if len(packages) > 0 {
		testArgs = append(testArgs, packages...)
	} else {
		testArgs = append(testArgs, "./...")
	}

	testArgs = append(testArgs, "-coverprofile="+file, "-covermode=atomic", "-race")

	// Capture output for analysis
	var buf bytes.Buffer
	multiWriter := io.MultiWriter(os.Stdout, &buf)

	// #nosec G204 - file and packages are validated (packages from git or args)
	cmd := exec.Command("go", testArgs...)
	cmd.Stdout = multiWriter
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}

	// Print package coverage summary
	printPackageCoverageTable(buf.String())

	PrintSuccess("Tests passed and coverage profile generated.")
	return nil
}
