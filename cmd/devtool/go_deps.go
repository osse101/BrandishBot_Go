package main

import (
	"fmt"
	"sort"
	"strings"
)

// GetDependentPackages takes a list of changed packages (relative paths)
// and returns a list containing those packages plus any packages in the module
// that depend on them (transitively), converted back to relative paths.
func GetDependentPackages(packages []string) ([]string, error) {
	if len(packages) == 0 {
		return nil, nil
	}

	modulePath, err := getModulePath()
	if err != nil {
		return nil, err
	}

	reverseDeps, err := getReverseDependencyGraph()
	if err != nil {
		return nil, err
	}

	queue, visited := normalizePackages(packages, modulePath)
	affected := findTransitiveDependents(queue, visited, reverseDeps, modulePath)

	return formatResults(affected, modulePath), nil
}

func getModulePath() (string, error) {
	//nolint:forbidigo
	out, err := getCommandOutput("go", "list", "-m")
	if err != nil {
		return "", fmt.Errorf("failed to get module path: %w", err)
	}
	return strings.TrimSpace(out), nil
}

func getReverseDependencyGraph() (map[string][]string, error) {
	// Use -e to tolerate errors (e.g. if a deleted package is still imported)
	//nolint:forbidigo
	out, err := getCommandOutput("go", "list", "-e", "-f", "{{.ImportPath}} {{range .Imports}}{{.}} {{end}}", "./...")
	if err != nil {
		return nil, fmt.Errorf("failed to get dependency graph: %w", err)
	}

	reverseDeps := make(map[string][]string)
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		importer := parts[0]
		for _, imported := range parts[1:] {
			reverseDeps[imported] = append(reverseDeps[imported], importer)
		}
	}
	return reverseDeps, nil
}

func normalizePackages(packages []string, modulePath string) ([]string, map[string]bool) {
	queue := make([]string, 0, len(packages))
	visited := make(map[string]bool)

	for _, p := range packages {
		fullPath := p
		if strings.HasPrefix(p, "./") {
			fullPath = modulePath + strings.TrimPrefix(p, ".")
		} else if p == "." {
			fullPath = modulePath
		}

		if !visited[fullPath] {
			visited[fullPath] = true
			queue = append(queue, fullPath)
		}
	}
	return queue, visited
}

func findTransitiveDependents(queue []string, visited map[string]bool, reverseDeps map[string][]string, modulePath string) map[string]bool {
	affected := make(map[string]bool)
	// Add initial packages to affected
	for _, p := range queue {
		affected[p] = true
	}

	head := 0
	for head < len(queue) {
		current := queue[head]
		head++

		dependents := reverseDeps[current]
		for _, dep := range dependents {
			// Only care about dependents within our module
			if strings.HasPrefix(dep, modulePath) {
				if !visited[dep] {
					visited[dep] = true
					affected[dep] = true
					queue = append(queue, dep)
				}
			}
		}
	}
	return affected
}

func formatResults(affected map[string]bool, modulePath string) []string {
	result := make([]string, 0, len(affected))
	for p := range affected {
		// Only return packages that are inside the module
		if !strings.HasPrefix(p, modulePath) {
			continue
		}

		rel := strings.TrimPrefix(p, modulePath)
		if rel == "" {
			rel = "."
		} else {
			rel = "." + rel
		}
		result = append(result, rel)
	}
	sort.Strings(result)
	return result
}
