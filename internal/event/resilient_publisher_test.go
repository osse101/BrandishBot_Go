package event

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBus is a test double for event.Bus
type mockBus struct {
	mu           sync.Mutex
	calls        []Event
	failCount    int32 // Atomic counter for failures
	shouldFail   func(attempt int) bool
	publishDelay time.Duration
}

func (m *mockBus) Publish(ctx context.Context, event Event) error {
	m.mu.Lock()
	m.calls = append(m.calls, event)
	callCount := len(m.calls)
	m.mu.Unlock()

	if m.publishDelay > 0 {
		time.Sleep(m.publishDelay)
	}

	if m.shouldFail != nil && m.shouldFail(callCount) {
		atomic.AddInt32(&m.failCount, 1)
		return errors.New("mock publish error")
	}
	return nil
}

func (m *mockBus) Subscribe(eventType Type, handler Handler) {
	// Not used in these tests
}

func (m *mockBus) GetCalls() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Event{}, m.calls...)
}

func (m *mockBus) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

// Test 1: Successful publish without retry
func TestResilientPublisher_SuccessfulPublish(t *testing.T) {
	tmpFile := t.TempDir() + "/deadletter.jsonl"
	bus := &mockBus{}

	rp, err := NewResilientPublisher(bus, 3, 100*time.Millisecond, tmpFile)
	require.NoError(t, err)
	defer rp.Shutdown(context.Background())

	// Publish event
	testEvent := Event{
		Type:    Type("test_event"),
		Payload: map[string]interface{}{"test": "data"},
	}
	rp.PublishWithRetry(context.Background(), testEvent)

	// Event should publish immediately
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, bus.CallCount(), "Event should be published once")
	assert.Equal(t, testEvent.Type, bus.GetCalls()[0].Type)

	// No dead-letter entry
	content, _ := os.ReadFile(tmpFile)
	assert.Empty(t, content, "No dead-letter entries expected")
}

// Test 2: Failed publish → retry →success
func TestResilientPublisher_RetrySuccess(t *testing.T) {
	tmpFile := t.TempDir() + "/deadletter.jsonl"

	// Bus fails on first attempt, succeeds on second
	bus := &mockBus{
		shouldFail: func(attempt int) bool {
			return attempt == 1
		},
	}

	rp, err := NewResilientPublisher(bus, 3, 100*time.Millisecond, tmpFile)
	require.NoError(t, err)
	defer rp.Shutdown(context.Background())

	testEvent := Event{
		Type:    Type("test_event"),
		Payload: map[string]interface{}{"id": "123"},
	}
	rp.PublishWithRetry(context.Background(), testEvent)

	// Wait for retry (first attempt + 100ms delay + second attempt)
	time.Sleep(300 * time.Millisecond)

	assert.Equal(t, 2, bus.CallCount(), "Should attempt twice: initial + retry")

	// No dead-letter entry
	content, _ := os.ReadFile(tmpFile)
	assert.Empty(t, content, "No dead-letter entries for successful retry")
}

// Test 3: Retry exhaustion → dead letter
func TestResilientPublisher_RetryExhaustion(t *testing.T) {
	tmpFile := t.TempDir() + "/deadletter.jsonl"

	// Bus always fails
	bus := &mockBus{
		shouldFail: func(attempt int) bool {
			return true
		},
	}

	rp, err := NewResilientPublisher(bus, 3, 50*time.Millisecond, tmpFile)
	require.NoError(t, err)
	defer rp.Shutdown(context.Background())

	testEvent := Event{
		Type:    Type("test_event"),
		Payload: map[string]interface{}{"id": "456"},
	}
	rp.PublishWithRetry(context.Background(), testEvent)

	// Wait for all retries (50ms + 100ms + 200ms + processing)
	time.Sleep(500 * time.Millisecond)

	// Should attempt: initial + 3 retries = 4 total
	assert.GreaterOrEqual(t, bus.CallCount(), 4, "Should exhaust all retries")

	// Verify dead-letter entry
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.NotEmpty(t, content, "Dead-letter file should have entry")

	// Parse and verify dead-letter format (DeadLetterEntry with nested Event)
	var dlEntry struct {
		Timestamp string `json:"timestamp"`
		Event     struct {
			Type    string                 `json:"type"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"event"`
		Attempts  int    `json:"attempts"`
		LastError string `json:"last_error"`
	}
	err = json.Unmarshal(content, &dlEntry)
	require.NoError(t, err, "Dead-letter should be valid JSON")

	assert.Equal(t, "test_event", dlEntry.Event.Type)
	assert.NotNil(t, dlEntry.Event.Payload)
	assert.NotEmpty(t, dlEntry.LastError)
	assert.GreaterOrEqual(t, dlEntry.Attempts, 1)
}

// Test 4: Queue overflow → immediate dead letter
func TestResilientPublisher_QueueOverflow(t *testing.T) {
	tmpFile := t.TempDir() + "/deadletter.jsonl"

	// Bus always fails, but takes time to process
	bus := &mockBus{
		shouldFail: func(attempt int) bool {
			return true
		},
		publishDelay: 50 * time.Millisecond, // Slow down to fill queue
	}

	// Small queue for easier testing
	rp := &ResilientPublisher{
		bus:        bus,
		retryQueue: make(chan retryEntry, 5), // Small queue
		maxRetries: 3,
		retryDelay: 50 * time.Millisecond,
		shutdown:   make(chan struct{}),
	}
	dl, err := NewDeadLetterWriter(tmpFile)
	require.NoError(t, err)
	rp.deadLetter = dl

	rp.wg.Add(1)
	go rp.retryWorker()
	defer rp.Shutdown(context.Background())

	// Flood the queue
	for i := 0; i < 10; i++ {
		testEvent := Event{
			Type:    Type("overflow_event"),
			Payload: map[string]interface{}{"id": i},
		}
		rp.PublishWithRetry(context.Background(), testEvent)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Dead-letter should have entries from queue overflow
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.NotEmpty(t, content, "Dead-letter should have overflow entries")
}

// Test 5: Graceful shutdown with pending retries
func TestResilientPublisher_GracefulShutdown(t *testing.T) {
	tmpFile := t.TempDir() + "/deadletter.jsonl"

	callCount := int32(0)
	// Bus fails first 2, succeeds after
	bus := &mockBus{
		shouldFail: func(attempt int) bool {
			count := atomic.AddInt32(&callCount, 1)
			return count <= 2
		},
	}

	rp, err := NewResilientPublisher(bus, 5, 50*time.Millisecond, tmpFile)
	require.NoError(t, err)

	// Queue some events that will fail initially
	for i := 0; i < 3; i++ {
		testEvent := Event{
			Type:    Type("shutdown_test"),
			Payload: map[string]interface{}{"id": i},
		}
		rp.PublishWithRetry(context.Background(), testEvent)
	}

	// Give time for initial failures and queuing
	time.Sleep(100 * time.Millisecond)

	// Shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = rp.Shutdown(ctx)
	assert.NoError(t, err, "Shutdown should complete successfully")

	// Worker should have processed remaining queue
	assert.GreaterOrEqual(t, bus.CallCount(), 3, "Should process queued events during shutdown")
}

// Test 6: Exponential backoff timing
func TestResilientPublisher_ExponentialBackoff(t *testing.T) {
	tmpFile := t.TempDir() + "/deadletter.jsonl"

	attempts := make([]time.Time, 0, 5)
	attemptMu := sync.Mutex{}

	// Bus tracks timing of attempts
	bus := &mockBus{
		shouldFail: func(attempt int) bool {
			attemptMu.Lock()
			attempts = append(attempts, time.Now())
			attemptMu.Unlock()
			return attempt < 4 // Fail first 3 attempts
		},
	}

	baseDelay := 100 * time.Millisecond
	rp, err := NewResilientPublisher(bus, 5, baseDelay, tmpFile)
	require.NoError(t, err)
	defer rp.Shutdown(context.Background())

	testEvent := Event{
		Type:    Type("backoff_test"),
		Payload: map[string]interface{}{"test": "backoff"},
	}
	rp.PublishWithRetry(context.Background(), testEvent)

	// Wait for retries to complete
	time.Sleep(1 * time.Second)

	attemptMu.Lock()
	defer attemptMu.Unlock()

	require.GreaterOrEqual(t, len(attempts), 3, "Should have at least 3 attempts")

	// Verify exponential backoff delays (with tolerance for timing variance)
	if len(attempts) >= 3 {
		delay1 := attempts[1].Sub(attempts[0])
		delay2 := attempts[2].Sub(attempts[1])

		// First retry: ~100ms delay
		assert.InDelta(t, baseDelay.Milliseconds(), delay1.Milliseconds(), 50,
			"First retry delay should be ~100ms")

		// Second retry: ~200ms delay (2x)
		assert.InDelta(t, (baseDelay * 2).Milliseconds(), delay2.Milliseconds(), 50,
			"Second retry delay should be ~200ms")
	}
}

// Test 7: Concurrent publishes
func TestResilientPublisher_ConcurrentPublishes(t *testing.T) {
	tmpFile := t.TempDir() + "/deadletter.jsonl"

	bus := &mockBus{}
	rp, err := NewResilientPublisher(bus, 3, 50*time.Millisecond, tmpFile)
	require.NoError(t, err)
	defer rp.Shutdown(context.Background())

	// Publish concurrently from multiple goroutines
	const numGoroutines = 10
	const eventsPerGoroutine = 5

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				testEvent := Event{
					Type:    Type("concurrent_test"),
					Payload: map[string]interface{}{"goroutine": goroutineID, "event": j},
				}
				rp.PublishWithRetry(context.Background(), testEvent)
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond) // Let events process

	// Should have all events published
	assert.Equal(t, numGoroutines*eventsPerGoroutine, bus.CallCount(),
		"All concurrent events should be published")
}
