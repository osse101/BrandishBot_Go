package event

import (
	"context"
	"errors"
	"testing"
)

func TestMemoryBus_PublishSubscribe(t *testing.T) {
	bus := NewMemoryBus()
	eventType := Type("test_event")
	handled := false

	bus.Subscribe(eventType, func(ctx context.Context, event Event) error {
		if event.Type != eventType {
			t.Errorf("Expected event type %s, got %s", eventType, event.Type)
		}
		if event.Payload.(string) != "payload" {
			t.Errorf("Expected payload 'payload', got %v", event.Payload)
		}
		handled = true
		return nil
	})

	err := bus.Publish(context.Background(), Event{
		Version: "1.0",
		Type:    eventType,
		Payload: "payload",
	})

	if err != nil {
		t.Errorf("Publish returned error: %v", err)
	}

	if !handled {
		t.Error("Handler was not called")
	}
}

func TestMemoryBus_PublishMultipleHandlers(t *testing.T) {
	bus := NewMemoryBus()
	eventType := Type("test_event")
	count := 0

	handler := func(ctx context.Context, event Event) error {
		count++
		return nil
	}

	bus.Subscribe(eventType, handler)
	bus.Subscribe(eventType, handler)

	err := bus.Publish(context.Background(), Event{Version: "1.0", Type: eventType})
	if err != nil {
		t.Errorf("Publish returned error: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 handlers to be called, got %d", count)
	}
}

func TestMemoryBus_PublishError(t *testing.T) {
	bus := NewMemoryBus()
	eventType := Type("test_event")

	bus.Subscribe(eventType, func(ctx context.Context, event Event) error {
		return errors.New("handler error")
	})

	err := bus.Publish(context.Background(), Event{Version: "1.0", Type: eventType})
	if err == nil {
		t.Error("Expected error from Publish, got nil")
	}
}
