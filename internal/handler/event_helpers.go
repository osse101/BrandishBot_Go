package handler

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/middleware"
)

// PublishEvent publishes an event to the event bus with standard error handling.
// This is a generic helper that can be used by any handler to publish events.
//
// Parameters:
//   - ctx: The request context
//   - eventBus: The event bus to publish to
//   - eventType: The type of event (e.g., "item.sold", "item.bought")
//   - payload: The event payload data
//
// Returns:
//   - error: nil if successful, error if publishing failed
//
// Example usage:
//
//	if err := PublishEvent(r.Context(), eventBus, "item.sold", map[string]interface{}{
//	    "user_id": username,
//	    "item_name": itemName,
//	    "quantity": quantity,
//	}); err != nil {
//	    // Error is already logged, handler can continue or return based on requirements
//	}
func PublishEvent(ctx context.Context, eventBus event.Bus, eventType event.Type, payload interface{}) error {
	log := logger.FromContext(ctx)

	if err := eventBus.Publish(ctx, event.Event{
		Version: "1.0",
		Type:    eventType,
		Payload: payload,
	}); err != nil {
		log.Error("Failed to publish event", "type", eventType, "error", err)
		return err
	}

	return nil
}

// TrackEngagement publishes engagement tracking for a user action.
// This is a convenience wrapper around middleware.TrackEngagementFromContext.
//
// Parameters:
//   - ctx: The request context
//   - eventBus: The event bus to publish to
//   - username: The username of the user performing the action
//   - eventType: The type of engagement event
//   - quantity: The quantity/count for the engagement metric
//
// Example usage:
//
//	TrackEngagement(r.Context(), eventBus, req.Username, "item_sold", itemsSold)
func TrackEngagement(ctx context.Context, eventBus event.Bus, username, eventType string, quantity int) {
	middleware.TrackEngagementFromContext(
		middleware.WithUserID(ctx, username),
		eventBus,
		eventType,
		quantity,
	)
}
