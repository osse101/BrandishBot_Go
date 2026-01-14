package bootstrap

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/config"
	"github.com/osse101/BrandishBot_Go/internal/event"
)

// InitializeEventSystem creates and configures the event bus and resilient publisher.
// It applies default values for retry configuration if not specified in config,
// creates the dead-letter directory, and initializes the resilient publisher
// with exponential backoff retry logic.
// Returns the event bus, resilient publisher, and any error encountered.
func InitializeEventSystem(cfg *config.Config) (event.Bus, *event.ResilientPublisher, error) {
	// Initialize Event Bus
	eventBus := event.NewMemoryBus()

	// Apply config defaults for resilient publisher
	maxRetries := cfg.EventMaxRetries
	if maxRetries == 0 {
		maxRetries = 5 // Default to 5 retries
	}

	retryDelay := cfg.EventRetryDelay
	if retryDelay == 0 {
		retryDelay = 2 * time.Second // Default to 2s base delay
	}

	deadLetterPath := cfg.EventDeadLetterPath
	if deadLetterPath == "" {
		deadLetterPath = "logs/event_deadletter.jsonl" // Default path
	}

	// Ensure dead-letter directory exists
	if err := os.MkdirAll(filepath.Dir(deadLetterPath), 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create dead-letter directory: %w", err)
	}

	// Initialize Resilient Publisher with retry logic
	resilientPublisher, err := event.NewResilientPublisher(eventBus, maxRetries, retryDelay, deadLetterPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resilient publisher: %w", err)
	}

	slog.Info("Event system initialized",
		"max_retries", maxRetries,
		"retry_delay", retryDelay,
		"deadletter_path", deadLetterPath)

	return eventBus, resilientPublisher, nil
}
