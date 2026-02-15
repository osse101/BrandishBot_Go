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
	file, threshold, runTests, htmlReport, err := c.parseConfig(args)
	if err != nil {
		return err
	}

	PrintHeader(fmt.Sprintf("Checking coverage threshold (%.1f%%)...", threshold))

	if err := c.ensureCoverage(file, runTests); err != nil {
		return err
	}

	coverage, err := c.getCoveragePercent(file)
	if err != nil {
		return err
	}

	PrintInfo("Total Coverage: %.1f%%", coverage)

	if htmlReport {
		if err := c.generateHTMLReport(file); err != nil {
			PrintWarning("Failed to generate HTML report: %v", err)
		}
	}

	if coverage < threshold {
		PrintError("Coverage is below threshold.")
		return fmt.Errorf("coverage below threshold")
	}

	PrintSuccess("Coverage meets threshold.")
	return nil
}

func (c *CheckCoverageCommand) parseConfig(args []string) (file string, threshold float64, runTests, htmlReport bool, err error) {
	fs := flag.NewFlagSet("check-coverage", flag.ContinueOnError)
	runTestsPtr := fs.Bool("run", false, "Run tests before checking coverage")
	htmlReportPtr := fs.Bool("html", false, "Generate and open HTML coverage report")

	if err := fs.Parse(args); err != nil {
		return "", 0, false, false, err
	}

	runTests = *runTestsPtr
	htmlReport = *htmlReportPtr
	file = "logs/coverage.out"
	thresholdStr := "80"

	positional := fs.Args()
	if len(positional) > 0 {
		file = filepath.Clean(positional[0])
	}
	if len(positional) > 1 {
		thresholdStr = positional[1]
	}

	// Basic path validation to prevent escaping the project root or injection
	if strings.Contains(file, "..") || strings.HasPrefix(file, "/") {
		return "", 0, false, false, fmt.Errorf("invalid path '%s': must be relative and within project", file)
	}

	threshold, err = strconv.ParseFloat(thresholdStr, 64)
	if err != nil {
		return "", 0, false, false, fmt.Errorf("invalid threshold '%s'", thresholdStr)
	}

	return file, threshold, runTests, htmlReport, nil
}

func (c *CheckCoverageCommand) ensureCoverage(file string, runTests bool) error {
	shouldRun := runTests
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
	// Note: mirroring 'make test' command
	// #nosec G204 - file is validated in parseConfig
	cmd := exec.Command("go", "test", "./...", "-coverprofile="+file, "-covermode=atomic", "-race")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}
	PrintSuccess("Tests passed and coverage profile generated.")
	return nil
}

func (c *CheckCoverageCommand) getCoveragePercent(file string) (float64, error) {
	// Run go tool cover -func=file
	//nolint:forbidigo // file is validated in parseConfig
	out, err := getCommandOutput("go", "tool", "cover", fmt.Sprintf("-func=%s", file)) // #nosec G204
	if err != nil {
		return 0, fmt.Errorf("error running go tool cover: %w", err)
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
		return 0, fmt.Errorf("could not determine coverage from output")
	}

	fields := strings.Fields(totalLine)
	if len(fields) < 3 {
		return 0, fmt.Errorf("unexpected output format")
	}

	pctStr := fields[len(fields)-1] // Last field is percentage
	pctStr = strings.TrimSuffix(pctStr, "%")

	coverage, err := strconv.ParseFloat(pctStr, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse coverage percentage '%s'", pctStr)
	}

	return coverage, nil
}

func (c *CheckCoverageCommand) generateHTMLReport(file string) error {
	htmlFile := filepath.Clean(strings.TrimSuffix(file, ".out") + ".html")

	// Extra validation for htmlFile
	if strings.Contains(htmlFile, "..") || strings.HasPrefix(htmlFile, "/") {
		return fmt.Errorf("invalid HTML report path '%s'", htmlFile)
	}

	PrintInfo("Generating HTML report: %s", htmlFile)
	// #nosec G204 - file and htmlFile are validated
	cmd := exec.Command("go", "tool", "cover", "-html="+file, "-o", htmlFile)
	if err := cmd.Run(); err != nil {
		return err
	}
	PrintSuccess("HTML report generated: %s", htmlFile)
	return nil
}
