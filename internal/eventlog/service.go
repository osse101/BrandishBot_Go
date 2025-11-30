package eventlog

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// Service handles event logging business logic
type Service interface {
	// Subscribe registers the event logger to listen to all events
	Subscribe(bus event.Bus) error

	// CleanupOldEvents removes events older than retention period
	CleanupOldEvents(ctx context.Context, retentionDays int) (int64, error)
}

type service struct {
	repo Repository
}

// NewService creates a new event logging service
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// Subscribe registers event handlers for all event types
func (s *service) Subscribe(bus event.Bus) error {
	// Subscribe to all domain event types
	eventTypes := []event.Type{
		"item.sold",
		"item.bought",
		"item.upgraded",
		"item.disassembled",
		"item.used",
		"search.performed",
		"engagement",
	}

	for _, eventType := range eventTypes {
		bus.Subscribe(eventType, s.handleEvent)
	}

	return nil
}

// handleEvent processes and logs events to the database
func (s *service) handleEvent(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	// Extract payload as map
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		log.Debug("Event payload is not a map, skipping log", "type", evt.Type)
		return nil
	}

	// Extract user_id if present
	var userID *string
	if uid, ok := payload["user_id"].(string); ok {
		userID = &uid
	}

	// Log event to database
	if err := s.repo.LogEvent(ctx, string(evt.Type), userID, payload, evt.Metadata); err != nil {
		log.Error("Failed to log event to database", "error", err, "type", evt.Type)
		return err
	}

	log.Debug("Event logged to database", "type", evt.Type, "user_id", userID)
	return nil
}

// CleanupOldEvents removes events older than the retention period
func (s *service) CleanupOldEvents(ctx context.Context, retentionDays int) (int64, error) {
	return s.repo.CleanupOldEvents(ctx, retentionDays)
}
