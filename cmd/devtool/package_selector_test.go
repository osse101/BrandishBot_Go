package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackageSelector_SelectPackages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		includes []string
		excludes []string
		expected []string
	}{
		{
			name:     "Include specific package",
			includes: []string{"internal/domain"},
			excludes: nil,
			expected: []string{"./internal/domain"},
		},
		{
			name:     "Normalize dot slash",
			includes: []string{"./internal/handler"},
			excludes: nil,
			expected: []string{"./internal/handler"},
		},
		{
			name:     "Normalize three dots",
			includes: []string{"..."},
			excludes: nil,
			expected: []string{"./..."},
		},
		{
			name:     "Filter exact exclude",
			includes: []string{"internal/domain", "internal/handler"},
			excludes: []string{"internal/domain"},
			expected: []string{"./internal/handler"},
		},
		{
			name:     "Filter wildcard exclude",
			includes: []string{"internal/handler", "internal/handler/auth", "internal/domain"},
			excludes: []string{"./internal/handler/..."},
			expected: []string{"./internal/domain"},
		},
		{
			name:     "Multiple includes and excludes",
			includes: []string{"pkg/a", "pkg/b", "pkg/c"},
			excludes: []string{"pkg/b"},
			expected: []string{"./pkg/a", "./pkg/c"},
		},
	}

	for _, tt := range tests {
		tt := tt // capture loop variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			selector := &PackageSelector{
				SmartMode: false,
				Includes:  tt.includes,
				Excludes:  tt.excludes,
			}

			packages, err := selector.SelectPackages()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, packages)
		})
	}
}

func TestPackageSelector_filterExcludedPackages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pkgSet   map[string]struct{}
		excludes []string
		expected map[string]struct{}
	}{
		{
			name: "Exact match removal",
			pkgSet: map[string]struct{}{
				"./internal/domain":  {},
				"./internal/handler": {},
			},
			excludes: []string{"./internal/domain"},
			expected: map[string]struct{}{
				"./internal/handler": {},
			},
		},
		{
			name: "Wildcard directory removal",
			pkgSet: map[string]struct{}{
				"./internal/handler":      {},
				"./internal/handler/auth": {},
				"./internal/domain":       {},
			},
			excludes: []string{"./internal/handler/..."},
			expected: map[string]struct{}{
				"./internal/domain": {},
			},
		},
		{
			name: "No exclusions",
			pkgSet: map[string]struct{}{
				"./internal/domain": {},
			},
			excludes: nil,
			expected: map[string]struct{}{
				"./internal/domain": {},
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture loop variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			selector := &PackageSelector{
				Excludes: tt.excludes,
			}

			// Modifies pkgSet in place
			selector.filterExcludedPackages(tt.pkgSet)

			assert.Equal(t, tt.expected, tt.pkgSet)
		})
	}
}
