package event

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// ResilientPublisher wraps an event.Bus to provide retry logic and dead-letter queues.
type ResilientPublisher struct {
	bus            Bus
	maxRetries     int
	retryDelay     time.Duration
	deadLetterPath string
	mu             sync.Mutex // Protects file writing
}

// NewResilientPublisher creates a new ResilientPublisher.
// If deadLetterPath is empty, it defaults to "./dead_letter.jsonl".
func NewResilientPublisher(bus Bus, maxRetries int, retryDelay time.Duration, deadLetterPath string) *ResilientPublisher {
	if deadLetterPath == "" {
		deadLetterPath = "./dead_letter.jsonl"
	}
	return &ResilientPublisher{
		bus:            bus,
		maxRetries:     maxRetries,
		retryDelay:     retryDelay,
		deadLetterPath: deadLetterPath,
	}
}

// Publish attempts to publish an event with retries.
// If all retries fail, it writes the event to a dead-letter queue.
func (p *ResilientPublisher) Publish(ctx context.Context, event Event) error {
	var err error

	for i := 0; i <= p.maxRetries; i++ {
		err = p.bus.Publish(ctx, event)
		if err == nil {
			return nil
		}

		// Wait before retrying, but respect context cancellation
		if i < p.maxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(p.retryDelay):
				// Continue to next retry
			}
		}
	}

	// If we get here, all retries failed. Write to dead letter queue.
	dlqErr := p.writeToDeadLetterQueue(event, err)
	if dlqErr != nil {
		return fmt.Errorf("failed to publish event %s after %d retries: %v; also failed to write to DLQ: %v", event.Type, p.maxRetries, err, dlqErr)
	}

	return fmt.Errorf("failed to publish event %s after %d retries: %v; event written to DLQ", event.Type, p.maxRetries, err)
}

// Subscribe delegates to the underlying bus.
func (p *ResilientPublisher) Subscribe(eventType Type, handler Handler) {
	p.bus.Subscribe(eventType, handler)
}

// writeToDeadLetterQueue appends the failed event to a JSONL file.
func (p *ResilientPublisher) writeToDeadLetterQueue(event Event, publishErr error) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create a wrapper struct to include the error
	type DeadLetterEntry struct {
		Timestamp time.Time `json:"timestamp"`
		Event     Event     `json:"event"`
		Error     string    `json:"error"`
	}

	entry := DeadLetterEntry{
		Timestamp: time.Now(),
		Event:     event,
		Error:     publishErr.Error(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal dead letter entry: %w", err)
	}

	// append to file
	f, err := os.OpenFile(p.deadLetterPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open dead letter file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write to dead letter file: %w", err)
	}
	if _, err := f.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline to dead letter file: %w", err)
	}

	return nil
}
