package metrics

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// EventMetricsCollector subscribes to events and records metrics
type EventMetricsCollector struct{}

// NewEventMetricsCollector creates a new event metrics collector
func NewEventMetricsCollector() *EventMetricsCollector {
	return &EventMetricsCollector{}
}

// Register subscribes to all events
func (e *EventMetricsCollector) Register(bus event.Bus) error {
	// Subscribe to all event types we care about
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
		bus.Subscribe(eventType, e.HandleEvent)
	}

	return nil
}

// HandleEvent processes events and updates metrics
func (e *EventMetricsCollector) HandleEvent(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	// Always increment event counter
	EventsPublished.WithLabelValues(string(evt.Type)).Inc()

	// Extract payload as map
	payload, ok := evt.Payload.(map[string]interface{})
	if !ok {
		log.Debug(LogMsgEventPayloadNotMap, "type", evt.Type)
		return nil
	}

	// Record business metrics based on event type
	switch evt.Type {
	case domain.EventTypeItemSold:
		if itemName, ok := payload[PayloadFieldItemName].(string); ok {
			ItemsSold.WithLabelValues(itemName).Inc()
		}
		// Track money earned from sales
		if moneyGained, ok := payload[PayloadFieldMoneyGained].(int); ok {
			MoneyEarned.Add(float64(moneyGained))
		}

	case domain.EventTypeItemBought:
		if itemName, ok := payload[PayloadFieldItemName].(string); ok {
			ItemsBought.WithLabelValues(itemName).Inc()
		}
		// Note: We don't have money_spent in the payload yet
		// This would require modifying the economy.Service.BuyItem to return cost

	case domain.EventTypeItemUpgraded:
		sourceItem, okSource := payload[PayloadFieldSourceItem].(string)
		resultItem, okResult := payload[PayloadFieldResultItem].(string)
		if okSource && okResult {
			ItemsUpgraded.WithLabelValues(sourceItem, resultItem).Inc()
		}

	case domain.EventTypeItemDisassembled:
		if itemName, ok := payload[PayloadFieldItem].(string); ok {
			ItemsDisassembled.WithLabelValues(itemName).Inc()
		}

	case domain.EventTypeItemUsed:
		if itemName, ok := payload[PayloadFieldItem].(string); ok {
			ItemsUsed.WithLabelValues(itemName).Inc()
		}

	case domain.EventTypeSearchPerformed:
		SearchesPerformed.Inc()
	}

	log.Debug(LogMsgMetricsRecorded, "type", evt.Type)
	return nil
}
