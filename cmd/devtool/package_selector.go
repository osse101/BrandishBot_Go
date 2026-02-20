package main

import (
	"fmt"
	"sort"
	"strings"
)

// PackageSelector handles the selection of packages for testing.
type PackageSelector struct {
	SmartMode  bool
	StagedOnly bool
	BaseRef    string
	Includes   []string
	Excludes   []string
}

// SelectPackages resolves the list of packages to test based on the configuration.
func (s *PackageSelector) SelectPackages() ([]string, error) {
	pkgSet := make(map[string]struct{})

	// 1. Add explicitly included packages
	for _, p := range s.Includes {
		s.addPackage(pkgSet, p)
	}

	// 2. Add changed packages in smart mode
	if s.SmartMode {
		if err := s.resolveSmartPackages(pkgSet); err != nil {
			return nil, err
		}
	}

	// 3. Remove excluded packages
	s.filterExcludedPackages(pkgSet)

	return s.toSortedSlice(pkgSet), nil
}

func (s *PackageSelector) resolveSmartPackages(pkgSet map[string]struct{}) error {
	changed, err := getChangedPackages(s.BaseRef, s.StagedOnly)
	if err != nil {
		return fmt.Errorf("failed to get changed packages: %w", err)
	}

	if len(changed) == 0 {
		// No changes detected, but we return nil error so that explicit includes can still run if present
		// If both are empty, the caller should decide what to do (e.g., skip tests or run all if intended).
		// For check-coverage, we handle "smart but no packages" by skipping.
		return nil
	}

	// Expand to include dependent packages
	expanded, err := GetDependentPackages(changed)
	if err != nil {
		return fmt.Errorf("failed to get dependent packages: %w", err)
	}

	for _, p := range expanded {
		pkgSet[p] = struct{}{}
	}
	return nil
}

func (s *PackageSelector) addPackage(pkgSet map[string]struct{}, pkg string) {
	pkg = s.normalizePkgPath(pkg)
	if pkg != "" {
		pkgSet[pkg] = struct{}{}
	}
}

func (s *PackageSelector) normalizePkgPath(p string) string {
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

func (s *PackageSelector) filterExcludedPackages(pkgSet map[string]struct{}) {
	if len(s.Excludes) == 0 {
		return
	}

	exPaths := make([]string, 0, len(s.Excludes))
	for _, ex := range s.Excludes {
		exPaths = append(exPaths, s.normalizePkgPath(ex))
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

func (s *PackageSelector) toSortedSlice(pkgSet map[string]struct{}) []string {
	packages := make([]string, 0, len(pkgSet))
	for p := range pkgSet {
		packages = append(packages, p)
	}
	sort.Strings(packages)
	return packages
}
