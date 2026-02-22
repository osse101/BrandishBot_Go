package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type CheckCoverageCommand struct{}

type CoverageConfig struct {
	File       string
	Threshold  float64
	RunTests   bool
	HTMLReport bool
	Smart      bool
	Packages   []string
	BaseRef    string
	Verbose    bool
	Exclude    []string
	Watch      bool
}

func (c *CheckCoverageCommand) Name() string {
	return "check-coverage"
}

func (c *CheckCoverageCommand) Description() string {
	return "Run tests with coverage and check against threshold"
}

func (c *CheckCoverageCommand) Run(args []string) error {
	config, err := c.parseConfig(args)
	if err != nil {
		return err
	}

	if config.Watch {
		return c.runWatchMode(config)
	}

	return c.runCoverageCheck(config)
}

func (c *CheckCoverageCommand) runCoverageCheck(config *CoverageConfig) error {
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

	if err := c.ensureCoverage(config.File, config.RunTests, packages, config.Verbose); err != nil {
		return err
	}

	// If we ran partial tests (packages != empty), the coverage profile only contains those packages.
	// This is fine for "check my changes" workflow.
	coverage, err := c.getCoveragePercent(config.File)
	if err != nil {
		return err
	}

	PrintInfo("Total Coverage: %.1f%%", coverage)

	if config.HTMLReport {
		if err := c.generateHTMLReport(config.File); err != nil {
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

//nolint:gocyclo // Watcher event loop is self-contained and straight-forward
func (c *CheckCoverageCommand) runWatchMode(config *CoverageConfig) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	// Recursively add directories to watch
	if err := c.addRecursiveWatch(watcher, "."); err != nil {
		return fmt.Errorf("failed to add watch paths: %w", err)
	}

	PrintInfo("Watching for file changes...")
	PrintInfo("Press Ctrl+C to exit.")

	// Run initial check
	if err := c.runCoverageCheck(config); err != nil {
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
						if err := c.addRecursiveWatch(watcher, event.Name); err != nil {
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
					c.clearScreen()
					PrintInfo("Change detected: %s", event.Name)
					if err := c.runCoverageCheck(config); err != nil {
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

func (c *CheckCoverageCommand) addRecursiveWatch(watcher *fsnotify.Watcher, root string) error {
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
		if base == "vendor" || base == "node_modules" || base == "bin" || base == "dist" || base == "logs" || base == "mocks" {
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

func (c *CheckCoverageCommand) clearScreen() {
	fmt.Print("\033[H\033[2J")
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

func (c *CheckCoverageCommand) parseConfig(args []string) (*CoverageConfig, error) {
	fs := flag.NewFlagSet("check-coverage", flag.ContinueOnError)

	filePtr := fs.String("file", "logs/coverage.out", "Path to coverage output file")
	thresholdPtr := fs.Float64("threshold", 80.0, "Coverage threshold percentage")
	runTestsPtr := fs.Bool("run", false, "Run tests before checking coverage")
	htmlReportPtr := fs.Bool("html", false, "Generate HTML coverage report")
	smartPtr := fs.Bool("smart", false, "Run tests only on changed packages")
	pkgsPtr := fs.String("pkgs", "", "Comma-separated list of packages to test")
	baseRefPtr := fs.String("base", "", "Base reference for git diff (smart mode only)")
	verbosePtr := fs.Bool("v", false, "Verbose output")
	excludePtr := fs.String("exclude", "", "Comma-separated list of packages to exclude")
	watchPtr := fs.Bool("watch", false, "Watch for file changes and re-run tests")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	file := filepath.Clean(*filePtr)

	// Deprecated positional args handling removed to simplify.
	if len(fs.Args()) > 0 {
		PrintWarning("Positional arguments are deprecated and ignored: %v. Use flags instead.", fs.Args())
	}

	// Basic path validation
	if strings.Contains(file, "..") || strings.HasPrefix(file, "/") {
		return nil, fmt.Errorf("invalid path '%s': must be relative and within project", file)
	}

	return &CoverageConfig{
		File:       file,
		Threshold:  *thresholdPtr,
		RunTests:   *runTestsPtr,
		HTMLReport: *htmlReportPtr,
		Smart:      *smartPtr,
		Packages:   c.splitCommaList(*pkgsPtr),
		BaseRef:    *baseRefPtr,
		Verbose:    *verbosePtr,
		Exclude:    c.splitCommaList(*excludePtr),
		Watch:      *watchPtr,
	}, nil
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
	return nil
}
