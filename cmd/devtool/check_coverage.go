package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	file, threshold, runTests, htmlReport, smart, pkgs, baseRef, verbose, exclude, err := c.parseConfig(args)
	if err != nil {
		return err
	}

	packages, err := c.resolvePackages(smart, pkgs, baseRef, exclude)
	if err != nil {
		return err
	}

	if len(packages) == 0 && smart {
		PrintInfo("Smart mode enabled but no packages selected (or all excluded). Skipping tests.")
		return nil
	}

	PrintHeader(fmt.Sprintf("Checking coverage threshold (%.1f%%)...", threshold))

	if err := c.ensureCoverage(file, runTests, packages, verbose); err != nil {
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

func (c *CheckCoverageCommand) resolvePackages(smart bool, pkgs string, baseRef string, exclude string) ([]string, error) {
	pkgSet := make(map[string]struct{})

	// 1. Add explicitly requested packages (from -pkgs flag)
	for _, p := range c.splitCommaList(pkgs) {
		c.addPackage(pkgSet, p)
	}

	// 2. Add changed packages in smart mode
	if smart {
		if err := c.resolveSmartPackages(pkgSet, baseRef); err != nil {
			return nil, err
		}
	}

	// 3. Remove excluded packages
	c.filterExcludedPackages(pkgSet, exclude)

	return c.toSortedSlice(pkgSet), nil
}

func (c *CheckCoverageCommand) addPackage(pkgSet map[string]struct{}, pkg string) {
	pkg = c.normalizePkgPath(pkg)
	if pkg != "" {
		pkgSet[pkg] = struct{}{}
	}
}

func (c *CheckCoverageCommand) normalizePkgPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "./..." || strings.HasPrefix(p, "./") {
		return p
	}
	if p == "..." {
		return "./..."
	}
	// Basic normalization for local packages: ensure they start with ./
	// unless they look like absolute paths or full import paths (containing dots)
	if !strings.Contains(p, ".") && !strings.HasPrefix(p, "/") && p != "." {
		return "./" + p
	}
	if p == "." {
		return "./"
	}
	return p
}

func (c *CheckCoverageCommand) resolveSmartPackages(pkgSet map[string]struct{}, baseRef string) error {
	changed, err := getChangedPackages(baseRef, false)
	if err != nil {
		return fmt.Errorf("failed to get changed packages: %w", err)
	}

	if len(changed) == 0 {
		PrintInfo("Smart mode: No changes detected.")
		return nil
	}

	// Expand to include dependent packages
	expanded, err := GetDependentPackages(changed)
	if err != nil {
		return fmt.Errorf("failed to get dependent packages: %w", err)
	}

	PrintInfo("Smart mode: Testing changed packages and dependents: %v", expanded)
	for _, p := range expanded {
		pkgSet[p] = struct{}{}
	}
	return nil
}

func (c *CheckCoverageCommand) filterExcludedPackages(pkgSet map[string]struct{}, exclude string) {
	excluded := c.splitCommaList(exclude)
	if len(excluded) == 0 {
		return
	}

	exPaths := make([]string, 0, len(excluded))
	for _, ex := range excluded {
		exPaths = append(exPaths, c.normalizePkgPath(ex))
	}

	for p := range pkgSet {
		for _, ex := range exPaths {
			// Exact match or wildcard match (e.g. ./internal/... matches ./internal and ./internal/foo)
			isExcluded := (p == ex)
			if !isExcluded && strings.HasSuffix(ex, "/...") {
				base := strings.TrimSuffix(ex, "/...")
				if p == base || strings.HasPrefix(p, base+"/") {
					isExcluded = true
				}
			}

			if isExcluded {
				PrintInfo("Excluding package: %s", p)
				delete(pkgSet, p)
				break
			}
		}
	}
}

func (c *CheckCoverageCommand) splitCommaList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func (c *CheckCoverageCommand) toSortedSlice(pkgSet map[string]struct{}) []string {
	packages := make([]string, 0, len(pkgSet))
	for p := range pkgSet {
		packages = append(packages, p)
	}
	sort.Strings(packages)
	return packages
}

func (c *CheckCoverageCommand) parseConfig(args []string) (file string, threshold float64, runTests, htmlReport, smart bool, pkgs string, baseRef string, verbose bool, exclude string, err error) {
	fs := flag.NewFlagSet("check-coverage", flag.ContinueOnError)

	filePtr := fs.String("file", "logs/coverage.out", "Path to coverage output file")
	thresholdPtr := fs.Float64("threshold", 80.0, "Coverage threshold percentage")
	runTestsPtr := fs.Bool("run", false, "Run tests before checking coverage")
	htmlReportPtr := fs.Bool("html", false, "Generate and open HTML coverage report")
	smartPtr := fs.Bool("smart", false, "Run tests only on changed packages")
	pkgsPtr := fs.String("pkgs", "", "Comma-separated list of packages to test")
	baseRefPtr := fs.String("base", "", "Base reference for git diff (smart mode only)")
	verbosePtr := fs.Bool("v", false, "Verbose output")
	excludePtr := fs.String("exclude", "", "Comma-separated list of packages to exclude")

	if err := fs.Parse(args); err != nil {
		return "", 0, false, false, false, "", "", false, "", err
	}

	file = filepath.Clean(*filePtr)
	threshold = *thresholdPtr
	runTests = *runTestsPtr
	htmlReport = *htmlReportPtr
	smart = *smartPtr
	pkgs = *pkgsPtr
	baseRef = *baseRefPtr
	verbose = *verbosePtr
	exclude = *excludePtr

	// Deprecated positional args handling removed to simplify.
	if len(fs.Args()) > 0 {
		PrintWarning("Positional arguments are deprecated and ignored: %v. Use flags instead.", fs.Args())
	}

	// Basic path validation
	if strings.Contains(file, "..") || strings.HasPrefix(file, "/") {
		return "", 0, false, false, false, "", "", false, "", fmt.Errorf("invalid path '%s': must be relative and within project", file)
	}

	return file, threshold, runTests, htmlReport, smart, pkgs, baseRef, verbose, exclude, nil
}

func (c *CheckCoverageCommand) ensureCoverage(file string, runTests bool, packages []string, verbose bool) error {
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

	// Attempt to open in browser
	PrintInfo("Opening report in browser...")
	if err := OpenBrowser(htmlFile); err != nil {
		PrintWarning("Failed to open browser: %v", err)
		// Don't fail the command, just warn
	}

	return nil
}
