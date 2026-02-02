package bootstrap

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

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
		maxRetries = EventDefaultMaxRetries
	}

	retryDelay := cfg.EventRetryDelay
	if retryDelay == 0 {
		retryDelay = EventDefaultRetryDelay
	}

	deadLetterPath := cfg.EventDeadLetterPath
	if deadLetterPath == "" {
		deadLetterPath = EventDefaultDeadLetterPath
	}

	// Ensure dead-letter directory exists
	if err := os.MkdirAll(filepath.Dir(deadLetterPath), DirPermission); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", LogMsgFailedCreateDeadLetterDir, err)
	}

	// Initialize Resilient Publisher with retry logic
	resilientPublisher, err := event.NewResilientPublisher(eventBus, maxRetries, retryDelay, deadLetterPath)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", LogMsgFailedCreateResilientPublisher, err)
	}

	slog.Info(LogMsgEventSystemInitialized,
		"max_retries", maxRetries,
		"retry_delay", retryDelay,
		"deadletter_path", deadLetterPath)

	return eventBus, resilientPublisher, nil
}
