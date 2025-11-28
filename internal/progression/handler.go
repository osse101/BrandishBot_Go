package progression

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// EventHandler handles progression-related events
type EventHandler struct {
	service Service
}

// NewEventHandler creates a new EventHandler
func NewEventHandler(service Service) *EventHandler {
	return &EventHandler{
		service: service,
	}
}

// Register registers the event handlers to the bus
func (h *EventHandler) Register(bus event.Bus) {
	bus.Subscribe("engagement", h.HandleEngagement)
}

// HandleEngagement handles engagement events
func (h *EventHandler) HandleEngagement(ctx context.Context, evt event.Event) error {
	metric, ok := evt.Payload.(*domain.EngagementMetric)
	if !ok {
		return fmt.Errorf("invalid payload type for engagement event: %T", evt.Payload)
	}

	log := logger.FromContext(ctx)
	log.Debug("Handling engagement event", "user_id", metric.UserID, "metric", metric.MetricType)

	if err := h.service.RecordEngagement(ctx, metric.UserID, metric.MetricType, metric.MetricValue); err != nil {
		return fmt.Errorf("failed to record engagement: %w", err)
	}

	return nil
}
