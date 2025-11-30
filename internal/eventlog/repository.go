package eventlog

import (
	"context"
	"time"
)

// Event represents a logged event
type Event struct {
	ID        int64                  `json:"id"`
	EventType string                 `json:"event_type"`
	UserID    *string                `json:"user_id,omitempty"`
	Payload   map[string]interface{} `json:"payload"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// EventFilter filters events for queries
type EventFilter struct {
	UserID    *string
	EventType *string
	Since     *time.Time
	Until     *time.Time
	Limit     int
}

// Repository defines the interface for event logging storage
type Repository interface {
	// LogEvent stores an event in the database
	LogEvent(ctx context.Context, eventType string, userID *string, payload, metadata map[string]interface{}) error

	// GetEvents retrieves events based on filter criteria
	GetEvents(ctx context.Context, filter EventFilter) ([]Event, error)

	// GetEventsByUser retrieves events for a specific user
	GetEventsByUser(ctx context.Context, userID string, limit int) ([]Event, error)

	// GetEventsByType retrieves events of a specific type
	GetEventsByType(ctx context.Context, eventType string, limit int) ([]Event, error)

	// CleanupOldEvents removes events older than the specified number of days
	CleanupOldEvents(ctx context.Context, retentionDays int) (int64, error)
}
