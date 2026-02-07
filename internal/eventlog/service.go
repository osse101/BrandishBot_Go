package eventlog

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// Service handles event logging business logic
type Service interface {
	// Subscribe registers the event logger to listen to all events
	Subscribe(bus event.Bus) error

	// CleanupOldEvents removes events older than retention period
	CleanupOldEvents(ctx context.Context, retentionDays int) (int64, error)

	// GetEvents retrieves events based on filter criteria
	GetEvents(ctx context.Context, filter EventFilter) ([]Event, error)
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
		domain.EventTypeItemSold,
		domain.EventTypeItemBought,
		domain.EventTypeItemUpgraded,
		domain.EventTypeItemDisassembled,
		domain.EventTypeItemUsed,
		domain.EventTypeSearchPerformed,
		domain.EventTypeEngagement,
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
		log.Debug(LogMsgEventPayloadNotMap, LogFieldType, evt.Type)
		return nil
	}

	// Extract user_id if present
	var userID *string
	if uid, ok := payload[PayloadKeyUserID].(string); ok {
		userID = &uid
	}

	// Log event to database
	if err := s.repo.LogEvent(ctx, string(evt.Type), userID, payload, evt.Metadata); err != nil {
		log.Error(LogMsgFailedToLogEvent, LogFieldError, err, LogFieldType, evt.Type)
		return err
	}

	log.Debug(LogMsgEventLogged, LogFieldType, evt.Type, LogFieldUserID, userID)
	return nil
}

// CleanupOldEvents removes events older than the retention period
func (s *service) CleanupOldEvents(ctx context.Context, retentionDays int) (int64, error) {
	return s.repo.CleanupOldEvents(ctx, retentionDays)
}

// GetEvents retrieves events based on filter criteria
func (s *service) GetEvents(ctx context.Context, filter EventFilter) ([]Event, error) {
	return s.repo.GetEvents(ctx, filter)
}
