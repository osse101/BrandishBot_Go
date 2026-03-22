package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
)

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

func splitCommaList(s string) []string {
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

func parseCoverageConfig(args []string) (*CoverageConfig, error) {
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
		Packages:   splitCommaList(*pkgsPtr),
		BaseRef:    *baseRefPtr,
		Verbose:    *verbosePtr,
		Exclude:    splitCommaList(*excludePtr),
		Watch:      *watchPtr,
	}, nil
}
