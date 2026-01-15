package event

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// DeadLetterSchemaVersion is the current version of the dead-letter log format
// Increment this when changing the DeadLetterEntry structure
const DeadLetterSchemaVersion = "1.0"

// DeadLetterWriter handles writing failed events to a dead-letter file
type DeadLetterWriter struct {
	file *os.File
	mu   sync.Mutex
}

// DeadLetterEntry represents an event that failed to publish after all retries
type DeadLetterEntry struct {
	SchemaVersion string    `json:"schema_version"` // Format version for future migrations
	Timestamp     time.Time `json:"timestamp"`
	Event         Event     `json:"event"`
	Attempts      int       `json:"attempts"`
	LastError     string    `json:"last_error,omitempty"`
}

// NewDeadLetterWriter creates a new DeadLetterWriter
func NewDeadLetterWriter(path string) (*DeadLetterWriter, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, DeadLetterFilePermissions)
	if err != nil {
		return nil, err
	}
	return &DeadLetterWriter{file: f}, nil
}

// Write writes a failed event to the dead-letter file
func (dlw *DeadLetterWriter) Write(event Event, attempts int, lastError error) error {
	dlw.mu.Lock()
	defer dlw.mu.Unlock()

	entry := DeadLetterEntry{
		SchemaVersion: DeadLetterSchemaVersion,
		Timestamp:     time.Now(),
		Event:         event,
		Attempts:      attempts,
	}

	if lastError != nil {
		entry.LastError = lastError.Error()
	}

	data, _ := json.Marshal(entry)
	_, err := dlw.file.Write(append(data, '\n'))
	return err
}

// Close closes the dead-letter file
func (dlw *DeadLetterWriter) Close() error {
	return dlw.file.Close()
}
