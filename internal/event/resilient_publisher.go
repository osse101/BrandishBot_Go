package event

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// ResilientConfig configures the ResilientPublisher
type ResilientConfig struct {
	MaxRetries     int
	RetryDelay     time.Duration
	DeadLetterPath string
}

// ResilientPublisher wraps an Event Bus to add retry logic and dead letter queuing
type ResilientPublisher struct {
	inner  Bus
	config ResilientConfig
	mu     sync.Mutex // Protects file writes
}

// NewResilientPublisher creates a new ResilientPublisher
func NewResilientPublisher(inner Bus, config ResilientConfig) *ResilientPublisher {
	return &ResilientPublisher{
		inner:  inner,
		config: config,
	}
}

// Publish attempts to publish an event. If it fails, it initiates a background retry loop.
// It returns nil to the caller immediately if the event is accepted for processing (even if the first attempt fails).
// This decouples the caller from the retry mechanism.
func (p *ResilientPublisher) Publish(ctx context.Context, event Event) error {
	// First attempt (synchronous or delegation)
	err := p.inner.Publish(ctx, event)
	if err == nil {
		return nil
	}

	// Log the initial failure
	logger.Warn("Failed to publish event, initiating async retry",
		"event_type", event.Type,
		"error", err,
		"retries", p.config.MaxRetries)

	// Launch background retry
	// We use a detached context or background context because the original request context might be cancelled
	go p.retryLoop(event)

	return nil
}

func (p *ResilientPublisher) retryLoop(event Event) {
	// Detached context for background work
	ctx := context.Background()

	for i := 1; i <= p.config.MaxRetries; i++ {
		// Backoff
		time.Sleep(p.config.RetryDelay * time.Duration(i)) // Simple linear backoff for now

		err := p.inner.Publish(ctx, event)
		if err == nil {
			logger.Info("Successfully published event after retry",
				"event_type", event.Type,
				"attempt", i)
			return
		}

		logger.Warn("Retry failed",
			"event_type", event.Type,
			"attempt", i,
			"error", err)
	}

	// All retries failed, send to dead letter queue
	p.writeToDeadLetter(event)
}

func (p *ResilientPublisher) writeToDeadLetter(event Event) {
	p.mu.Lock()
	defer p.mu.Unlock()

	f, err := os.OpenFile(p.config.DeadLetterPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Error("Failed to open dead letter file", "error", err, "path", p.config.DeadLetterPath)
		return
	}
	defer f.Close()

	// Add timestamp and error info to metadata if possible?
	// Event struct is shared, so maybe copy it? For now, just dump as is.
	// We might want to wrap it in a DeadLetter struct with Timestamp.
	type DeadLetterEntry struct {
		Timestamp time.Time `json:"timestamp"`
		Event     Event     `json:"event"`
	}

	entry := DeadLetterEntry{
		Timestamp: time.Now(),
		Event:     event,
	}

	if err := json.NewEncoder(f).Encode(entry); err != nil {
		logger.Error("Failed to write to dead letter file", "error", err)
	} else {
		logger.Info("Event written to dead letter queue", "event_type", event.Type)
	}
}

// Subscribe delegates to the inner bus
func (p *ResilientPublisher) Subscribe(eventType Type, handler Handler) {
	p.inner.Subscribe(eventType, handler)
}
