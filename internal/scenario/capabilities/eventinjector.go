package capabilities

import (
	"github.com/osse101/BrandishBot_Go/internal/scenario"
)

// EventInjectorCapabilityInfo returns the capability info for event injection
func EventInjectorCapabilityInfo() scenario.CapabilityInfo {
	return scenario.CapabilityInfo{
		Type:        scenario.CapabilityEventInjector,
		Name:        "Event Injector",
		Description: "Allows injection of events to trigger feature behavior without actual user actions",
		Actions: []scenario.ActionInfo{
			{
				Action:      scenario.ActionInjectEvent,
				Name:        "Inject Event",
				Description: "Injects a specific event to trigger feature handlers",
				Parameters: []scenario.ParameterInfo{
					{
						Name:        "event_type",
						Type:        "string",
						Required:    true,
						Description: "Type of event to inject (e.g., 'search', 'item_bought', 'recipe_crafted')",
					},
					{
						Name:        "count",
						Type:        "number",
						Required:    false,
						Description: "Number of times to inject the event (default: 1)",
					},
					{
						Name:        "metadata",
						Type:        "object",
						Required:    false,
						Description: "Additional metadata for the event",
					},
				},
				Example: map[string]interface{}{
					"event_type": "search",
					"count":      10,
				},
			},
			{
				Action:      scenario.ActionTriggerSearch,
				Name:        "Trigger Search",
				Description: "Triggers search events for quest progress",
				Parameters: []scenario.ParameterInfo{
					{
						Name:        "count",
						Type:        "number",
						Required:    false,
						Description: "Number of searches to trigger (default: 1)",
					},
				},
				Example: map[string]interface{}{
					"count": 10,
				},
			},
		},
	}
}

// EventInjectorParams represents parameters for an event injection action
type EventInjectorParams struct {
	EventType string
	Count     int
	Metadata  map[string]interface{}
}

// ParseEventInjectorParams extracts event injection parameters from a step
func ParseEventInjectorParams(params map[string]interface{}) (*EventInjectorParams, error) {
	result := &EventInjectorParams{
		Count:    1, // Default
		Metadata: make(map[string]interface{}),
	}

	// Event type (required for inject_event, optional for specific actions)
	if eventType, ok := params["event_type"]; ok {
		if et, ok := eventType.(string); ok {
			result.EventType = et
		} else {
			return nil, scenario.NewParameterError("event_type", "must be a string")
		}
	}

	// Count (optional)
	if count, ok := params["count"]; ok {
		switch c := count.(type) {
		case float64:
			result.Count = int(c)
		case int:
			result.Count = c
		case int64:
			result.Count = int(c)
		default:
			return nil, scenario.NewParameterError("count", "must be a number")
		}
	}

	// Metadata (optional)
	if metadata, ok := params["metadata"]; ok {
		if m, ok := metadata.(map[string]interface{}); ok {
			result.Metadata = m
		}
	}

	return result, nil
}

// SearchEventParams represents parameters for a search trigger action
type SearchEventParams struct {
	Count int
}

// ParseSearchEventParams extracts search event parameters from a step
func ParseSearchEventParams(params map[string]interface{}) (*SearchEventParams, error) {
	result := &SearchEventParams{
		Count: 1, // Default
	}

	if count, ok := params["count"]; ok {
		switch c := count.(type) {
		case float64:
			result.Count = int(c)
		case int:
			result.Count = c
		case int64:
			result.Count = int(c)
		default:
			return nil, scenario.NewParameterError("count", "must be a number")
		}
	}

	return result, nil
}
