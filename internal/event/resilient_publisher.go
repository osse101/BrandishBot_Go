package event

import (
	"context"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// ResilientPublisher wraps an event bus with retry logic and dead-letter handling
type ResilientPublisher struct {
	bus          Bus
	retryQueue   chan retryEntry
	maxRetries   int
	retryDelay   time.Duration
	wg           sync.WaitGroup
	shutdown     chan struct{}
	deadLetter   *DeadLetterWriter
}

// retryEntry represents an event in the retry queue
type retryEntry struct {
	event      Event
	attempt    int
	lastError  error
}

// NewResilientPublisher creates a new ResilientPublisher with retry logic
func NewResilientPublisher(bus Bus, maxRetries int, retryDelay time.Duration, deadLetterPath string) (*ResilientPublisher, error) {
	dl, err := NewDeadLetterWriter(deadLetterPath)
	if err != nil {
		return nil, err
	}
	
	rp := &ResilientPublisher{
		bus:        bus,
		retryQueue: make(chan retryEntry, 1000), // Buffer 1000 events
		maxRetries: maxRetries,
		retryDelay: retryDelay,
		shutdown:   make(chan struct{}),
		deadLetter: dl,
	}
	
	// Start background retry worker
	rp.wg.Add(1)
	go rp.retryWorker()
	
	return rp, nil
}

// PublishWithRetry attempts to publish an event, queues for retry on failure
// This method never blocks or returns errors to ensure XP awards always succeed
func (rp *ResilientPublisher) PublishWithRetry(ctx context.Context, event Event) {
	if err := rp.bus.Publish(ctx, event); err != nil {
		log := logger.FromContext(ctx)
		log.Warn("Event publish failed, queuing for retry",
			"event_type", event.Type,
			"error", err)
		
		// Non-blocking send to retry queue
		select {
		case rp.retryQueue <- retryEntry{event: event, attempt: 1, lastError: err}:
			// Queued successfully
		default:
			log.Error("Retry queue full, event dropped to dead-letter",
				"event_type", event.Type)
			rp.deadLetter.Write(event, 0, err)
		}
	}
}

// retryWorker processes events from the retry queue
func (rp *ResilientPublisher) retryWorker() {
	defer rp.wg.Done()
	
	for {
		select {
		case entry := <-rp.retryQueue:
			rp.processRetry(entry)
		case <-rp.shutdown:
			// Drain remaining events before shutdown
			rp.drainQueue()
			return
		}
	}
}

// processRetry handles retry logic for a single event
func (rp *ResilientPublisher) processRetry(entry retryEntry) {
	// Exponential backoff: 2s, 4s, 8s, 16s, 32s
	delay := rp.retryDelay * time.Duration(1<<(entry.attempt-1))
	time.Sleep(delay)
	
	ctx := context.Background()
	log := logger.FromContext(ctx)
	
	if err := rp.bus.Publish(ctx, entry.event); err != nil {
		if entry.attempt >= rp.maxRetries {
			// Retry exhausted, write to dead-letter
			log.Error("Event retry exhausted, writing to dead-letter",
				"event_type", entry.event.Type,
				"attempts", entry.attempt,
				"error", err)
			rp.deadLetter.Write(entry.event, entry.attempt, err)
		} else {
			// Schedule next retry
			log.Warn("Event retry failed, scheduling next attempt",
				"event_type", entry.event.Type,
				"attempt", entry.attempt,
				"next_delay", delay*2,
				"error", err)
			
			// Use goroutine to avoid blocking the worker
			go func(nextEntry retryEntry) {
				select {
				case rp.retryQueue <- nextEntry:
					// Queued for next retry
				case <-rp.shutdown:
					// Shutting down, write to dead-letter
					rp.deadLetter.Write(nextEntry.event, nextEntry.attempt, nextEntry.lastError)
				}
			}(retryEntry{event: entry.event, attempt: entry.attempt + 1, lastError: err})
		}
	} else {
		log.Info("Event retry succeeded",
			"event_type", entry.event.Type,
			"attempt", entry.attempt)
	}
}

// drainQueue processes remaining events during shutdown
func (rp *ResilientPublisher) drainQueue() {
	log := logger.FromContext(context.Background())
	count := 0
	
	for {
		select {
		case entry := <-rp.retryQueue:
			// Try one final publish attempt
			ctx := context.Background()
			if err := rp.bus.Publish(ctx, entry.event); err != nil {
				log.Warn("Event dropped during shutdown",
					"event_type", entry.event.Type,
					"error", err)
				rp.deadLetter.Write(entry.event, entry.attempt, err)
			}
			count++
		default:
			if count > 0 {
				log.Info("Drained retry queue during shutdown", "events_processed", count)
			}
			return
		}
	}
}

// Shutdown gracefully shuts down the resilient publisher
func (rp *ResilientPublisher) Shutdown(ctx context.Context) error {
	close(rp.shutdown)
	
	// Wait for worker to finish with timeout
	done := make(chan struct{})
	go func() {
		rp.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// Completed successfully
	case <-ctx.Done():
		logger.FromContext(ctx).Warn("Resilient publisher shutdown timed out")
	}
	
	return rp.deadLetter.Close()
}
