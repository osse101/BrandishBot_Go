package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIgnoreDirs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dir      string
		expected bool
	}{
		{
			name:     "Ignore .git",
			dir:      ".git",
			expected: true,
		},
		{
			name:     "Ignore vendor",
			dir:      "vendor",
			expected: true,
		},
		{
			name:     "Ignore node_modules",
			dir:      "node_modules",
			expected: true,
		},
		{
			name:     "Do not ignore internal",
			dir:      "internal",
			expected: false,
		},
		{
			name:     "Do not ignore cmd",
			dir:      "cmd",
			expected: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := IgnoreDirs(tt.dir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitCommaList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "Single item",
			input:    "pkg/a",
			expected: []string{"pkg/a"},
		},
		{
			name:     "Multiple items",
			input:    "pkg/a,pkg/b",
			expected: []string{"pkg/a", "pkg/b"},
		},
		{
			name:     "Items with spaces",
			input:    "pkg/a, pkg/b , pkg/c",
			expected: []string{"pkg/a", "pkg/b", "pkg/c"},
		},
		{
			name:     "Empty items",
			input:    "pkg/a,,pkg/b",
			expected: []string{"pkg/a", "pkg/b"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := splitCommaList(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
