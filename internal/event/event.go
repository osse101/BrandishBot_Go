package event

import (
	"context"
	"fmt"
	"sync"
)

// Type represents the type of an event
type Type string

// Event represents a generic event in the system
type Event struct {
	Type     Type
	Payload  interface{}
	Metadata map[string]interface{}
}

// Handler is a function that handles an event
type Handler func(ctx context.Context, event Event) error

// Bus defines the interface for an event bus
type Bus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(eventType Type, handler Handler)
}

// MemoryBus is an in-memory implementation of the Event Bus
type MemoryBus struct {
	handlers map[Type][]Handler
	mu       sync.RWMutex
}

// NewMemoryBus creates a new MemoryBus
func NewMemoryBus() *MemoryBus {
	return &MemoryBus{
		handlers: make(map[Type][]Handler),
	}
}

// Publish publishes an event to all subscribers
func (b *MemoryBus) Publish(ctx context.Context, event Event) error {
	b.mu.RLock()
	handlers, ok := b.handlers[event.Type]
	b.mu.RUnlock()

	if !ok {
		return nil
	}

	// For now, we execute handlers synchronously.
	// In the future, or with configuration, we could dispatch these to a worker pool
	// or run them in goroutines.
	var errs []error
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("encountered %d errors while handling event %s: %v", len(errs), event.Type, errs)
	}

	return nil
}

// Subscribe subscribes a handler to an event type
func (b *MemoryBus) Subscribe(eventType Type, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
}
