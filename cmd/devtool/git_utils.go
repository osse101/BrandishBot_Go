package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// getChangedPackages returns a list of Go packages that have changed.
// If stagedOnly is true, it checks staged changes (for pre-commit).
// If stagedOnly is false, it checks local changes against HEAD.
// If go.mod or go.sum changed, returns ./... to test everything.
func getChangedPackages(stagedOnly bool) ([]string, error) {
	var out string
	var err error

	if stagedOnly {
		//nolint:forbidigo
		out, err = getCommandOutput("git", "diff", "--cached", "--name-only", "--diff-filter=ACMR")
	} else {
		//nolint:forbidigo
		out, err = getCommandOutput("git", "diff", "HEAD", "--name-only", "--diff-filter=ACMR")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	if out == "" {
		return []string{}, nil
	}

	files := strings.Split(out, "\n")
	packageSet := make(map[string]bool)
	testAll := false

	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}

		if file == "go.mod" || file == "go.sum" {
			testAll = true
			break
		}

		if strings.HasSuffix(file, ".go") {
			dir := filepath.Dir(file)
			// Convert to slash for Go package path consistency
			dir = filepath.ToSlash(dir)

			// Ensure path starts with ./ for go test
			if !strings.HasPrefix(dir, "./") && dir != "." {
				dir = "./" + dir
			} else if dir == "." {
				dir = "./"
			}
			packageSet[dir] = true
		}
	}

	if testAll {
		return []string{"./..."}, nil
	}

	packages := make([]string, 0, len(packageSet))
	for pkg := range packageSet {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)

	return packages, nil
}
