package metrics

import (
	"context"

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
		"item.sold",
		"item.bought",
		"item.upgraded",
		"item.disassembled",
		"item.used",
		"search.performed",
		"engagement",
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
		log.Debug("Event payload is not a map", "type", evt.Type)
		return nil
	}

	// Record business metrics based on event type
	switch evt.Type {
	case "item.sold":
		if itemName, ok := payload["item_name"].(string); ok {
			ItemsSold.WithLabelValues(itemName).Inc()
		}
		// Track money earned from sales
		if moneyGained, ok := payload["money_gained"].(int); ok {
			MoneyEarned.Add(float64(moneyGained))
		}

	case "item.bought":
		if itemName, ok := payload["item_name"].(string); ok {
			ItemsBought.WithLabelValues(itemName).Inc()
		}
		// Note: We don't have money_spent in the payload yet
		// This would require modifying the economy.Service.BuyItem to return cost

	case "item.upgraded":
		sourceItem, okSource := payload["source_item"].(string)
		resultItem, okResult := payload["result_item"].(string)
		if okSource && okResult {
			ItemsUpgraded.WithLabelValues(sourceItem, resultItem).Inc()
		}

	case "item.disassembled":
		if itemName, ok := payload["item"].(string); ok {
			ItemsDisassembled.WithLabelValues(itemName).Inc()
		}

	case "item.used":
		if itemName, ok := payload["item"].(string); ok {
			ItemsUsed.WithLabelValues(itemName).Inc()
		}

	case "search.performed":
		SearchesPerformed.Inc()
	}

	log.Debug("Metrics recorded for event", "type", evt.Type)
	return nil
}
