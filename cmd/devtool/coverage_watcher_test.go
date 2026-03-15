package main

import (
	"testing"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
)

func TestShouldProcess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		event    fsnotify.Event
		expected bool
	}{
		{
			name: "Process .go file creation",
			event: fsnotify.Event{
				Name: "main.go",
				Op:   fsnotify.Create,
			},
			expected: true,
		},
		{
			name: "Process .go file write",
			event: fsnotify.Event{
				Name: "internal/service.go",
				Op:   fsnotify.Write,
			},
			expected: true,
		},
		{
			name: "Process .mod file",
			event: fsnotify.Event{
				Name: "go.mod",
				Op:   fsnotify.Write,
			},
			expected: true,
		},
		{
			name: "Ignore .txt file",
			event: fsnotify.Event{
				Name: "README.txt",
				Op:   fsnotify.Write,
			},
			expected: false,
		},
		{
			name: "Ignore Chmod on .go file",
			event: fsnotify.Event{
				Name: "main.go",
				Op:   fsnotify.Chmod,
			},
			expected: false,
		},
		{
			name: "Ignore hidden files",
			event: fsnotify.Event{
				Name: ".env",
				Op:   fsnotify.Write,
			},
			expected: false,
		},
		{
			name: "Ignore directories (no extension)",
			event: fsnotify.Event{
				Name: "internal",
				Op:   fsnotify.Create,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := shouldProcess(tt.event)
			assert.Equal(t, tt.expected, result)
		})
	}
}
