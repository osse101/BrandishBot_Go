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
	file, threshold, runTests, htmlReport, smart, pkgs, explicitPkgs, err := c.parseConfig(args)
	if err != nil {
		return err
	}

	packages, err := c.resolvePackages(smart, pkgs, explicitPkgs)
	if err != nil {
		return err
	}

	if len(packages) == 0 && smart {
		PrintInfo("Smart mode enabled but no packages selected. Skipping tests.")
		return nil
	}

	PrintHeader(fmt.Sprintf("Checking coverage threshold (%.1f%%)...", threshold))

	if err := c.ensureCoverage(file, runTests, packages); err != nil {
		return err
	}

	// If we ran partial tests (packages != empty), the coverage profile only contains those packages.
	// This is fine for "check my changes" workflow.
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
func (c *CheckCoverageCommand) resolvePackages(smart bool, pkgs string, explicitPkgs []string) ([]string, error) {
	packages := explicitPkgs

	if smart {
		changed, err := getChangedPackages(false) // false = check local changes (staged + unstaged)
		if err != nil {
			return nil, fmt.Errorf("failed to get changed packages: %w", err)
		}
		if len(changed) == 0 {
			PrintInfo("Smart mode: No changes detected.")
		} else {
			PrintInfo("Smart mode: Testing changed packages: %v", changed)
			packages = append(packages, changed...)
		}
	}

	if pkgs != "" {
		pList := strings.Split(pkgs, ",")
		for _, p := range pList {
			p = strings.TrimSpace(p)
			if p != "" {
				packages = append(packages, p)
			}
		}
	}

	// Remove duplicates from packages
	if len(packages) > 0 {
		unique := make(map[string]bool)
		var deduped []string
		for _, p := range packages {
			if !unique[p] {
				unique[p] = true
				deduped = append(deduped, p)
			}
		}
		packages = deduped
	}

	return packages, nil
}

func (c *CheckCoverageCommand) parseConfig(args []string) (file string, threshold float64, runTests, htmlReport, smart bool, pkgs string, explicitPkgs []string, err error) {
	fs := flag.NewFlagSet("check-coverage", flag.ContinueOnError)
	runTestsPtr := fs.Bool("run", false, "Run tests before checking coverage")
	htmlReportPtr := fs.Bool("html", false, "Generate and open HTML coverage report")
	smartPtr := fs.Bool("smart", false, "Run tests only on changed packages")
	pkgsPtr := fs.String("pkgs", "", "Comma-separated list of packages to test")

	if err := fs.Parse(args); err != nil {
		return "", 0, false, false, false, "", nil, err
	}

	runTests = *runTestsPtr
	htmlReport = *htmlReportPtr
	smart = *smartPtr
	pkgs = *pkgsPtr
	file = "logs/coverage.out"
	thresholdStr := "80"

	positional := fs.Args()
	if len(positional) > 0 {
		file = filepath.Clean(positional[0])
	}
	if len(positional) > 1 {
		// Try to parse second arg as threshold
		if _, err := strconv.ParseFloat(positional[1], 64); err == nil {
			thresholdStr = positional[1]
			if len(positional) > 2 {
				explicitPkgs = positional[2:]
			}
		} else {
			// Second arg is not a number, treat as package?
			// But maintain backward compatibility: file threshold [pkgs...]
			// If existing users rely on "file threshold", we must support it.
			// If user types "check-coverage file pkg", then threshold defaults to 80?
			// This is tricky. Let's assume strict: file threshold [pkgs...]
			// If they omit threshold, they must use flags for packages.
			// But for now, let's just stick to strict positional.
			thresholdStr = positional[1]
			if len(positional) > 2 {
				explicitPkgs = positional[2:]
			}
		}
	}

	// Basic path validation to prevent escaping the project root or injection
	if strings.Contains(file, "..") || strings.HasPrefix(file, "/") {
		return "", 0, false, false, false, "", nil, fmt.Errorf("invalid path '%s': must be relative and within project", file)
	}

	threshold, err = strconv.ParseFloat(thresholdStr, 64)
	if err != nil {
		// Maybe user provided a package instead of threshold?
		// e.g. check-coverage logs/c.out ./pkg
		// In that case thresholdStr is "./pkg".
		// We could fallback to default threshold 80 and treat this as package.
		// But that's ambiguous.
		return "", 0, false, false, false, "", nil, fmt.Errorf("invalid threshold '%s'", thresholdStr)
	}

	return file, threshold, runTests, htmlReport, smart, pkgs, explicitPkgs, nil
}

func (c *CheckCoverageCommand) ensureCoverage(file string, runTests bool, packages []string) error {
	shouldRun := runTests

	// If specific packages are requested, we MUST run tests because the existing profile
	// (if any) likely covers everything or something else. We can't reuse it reliably
	// unless we know it matches exactly. Safer to always run.
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
	if len(packages) > 0 {
		testArgs = append(testArgs, packages...)
	} else {
		testArgs = append(testArgs, "./...")
	}

	testArgs = append(testArgs, "-coverprofile="+file, "-covermode=atomic", "-race")

	// #nosec G204 - file and packages are validated (packages from git or args)
	cmd := exec.Command("go", testArgs...)
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
