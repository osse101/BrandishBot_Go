package repository

import (
	"context"
	"time"
)

// EventLog defines the interface for event logging storage
type EventLog interface {
	// LogEvent stores an event in the database
	LogEvent(ctx context.Context, eventType string, userID *string, payload, metadata map[string]interface{}) error

	// GetEvents retrieves events based on filter criteria
	GetEvents(ctx context.Context, filter EventLogFilter) ([]EventLogEntry, error)

	// GetEventsByUser retrieves events for a specific user
	GetEventsByUser(ctx context.Context, userID string, limit int) ([]EventLogEntry, error)

	// GetEventsByType retrieves events of a specific type
	GetEventsByType(ctx context.Context, eventType string, limit int) ([]EventLogEntry, error)

	// CleanupOldEvents removes events older than the specified number of days
	CleanupOldEvents(ctx context.Context, retentionDays int) (int64, error)
}

// EventLogEntry represents a logged event (formerly Event in eventlog package)
type EventLogEntry struct {
	ID        int64                  `json:"id"`
	EventType string                 `json:"event_type"`
	UserID    *string                `json:"user_id,omitempty"`
	Payload   map[string]interface{} `json:"payload"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// EventLogFilter filters events for queries (formerly EventFilter)
type EventLogFilter struct {
	UserID    *string
	EventType *string
	Since     *time.Time
	Until     *time.Time
	Limit     int
}
