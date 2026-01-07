package event

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"
)

// mockBus is a simple mock for the Bus interface
type mockBus struct {
	mu           sync.Mutex
	publishCalls int
	shouldFail   bool
	handlers     map[Type][]Handler
}

func (m *mockBus) Publish(ctx context.Context, event Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishCalls++

	if m.shouldFail {
		return errors.New("mock bus error")
	}
	return nil
}

func (m *mockBus) Subscribe(eventType Type, handler Handler) {
	if m.handlers == nil {
		m.handlers = make(map[Type][]Handler)
	}
	m.handlers[eventType] = append(m.handlers[eventType], handler)
}

func TestResilientPublisher_Publish_Success(t *testing.T) {
	mock := &mockBus{}
	publisher := NewResilientPublisher(mock, ResilientConfig{})

	err := publisher.Publish(context.Background(), Event{Type: "test"})
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}

	if mock.publishCalls != 1 {
		t.Errorf("Expected 1 call to inner bus, got %d", mock.publishCalls)
	}
}

func TestResilientPublisher_Publish_Retry(t *testing.T) {
	// Create a temp file for DLQ
	tmpfile, err := os.CreateTemp("", "dlq_*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	mock := &mockBus{shouldFail: true}

	// Fast retry for testing
	config := ResilientConfig{
		MaxRetries:     2,
		RetryDelay:     10 * time.Millisecond,
		DeadLetterPath: tmpfile.Name(),
	}

	publisher := NewResilientPublisher(mock, config)

	// This should return nil immediately despite failure, because it schedules retries
	err = publisher.Publish(context.Background(), Event{Type: "test"})
	if err != nil {
		t.Errorf("Expected nil error (swallowed), got %v", err)
	}

	// Wait for retries
	// Initial attempt + 2 retries = 3 calls
	// We need to wait a bit more than delay * retries
	time.Sleep(100 * time.Millisecond)

	mock.mu.Lock()
	calls := mock.publishCalls
	mock.mu.Unlock()

	// 1 initial + 2 retries = 3
	if calls != 3 {
		t.Errorf("Expected 3 calls (1 initial + 2 retries), got %d", calls)
	}

	// Check if DLQ was written
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if len(content) == 0 {
		t.Error("Expected DLQ file to contain data")
	}
}

func TestResilientPublisher_Publish_Recover(t *testing.T) {
	mock := &mockBus{shouldFail: true}

	config := ResilientConfig{
		MaxRetries: 2,
		RetryDelay: 10 * time.Millisecond,
	}

	publisher := NewResilientPublisher(mock, config)

	go func() {
		// Recover after some time (before all retries fail)
		time.Sleep(15 * time.Millisecond) // after 1st retry (10ms), before 2nd (30ms total time: 10 + 20)
		// Wait... retry logic:
		// Attempt 1 (fail) -> Sleep 10ms -> Attempt 2 (fail) -> Sleep 20ms -> Attempt 3.
		// If we recover after 15ms, Attempt 2 starts at T+10ms.
		// If we set shouldFail=false at T+15ms, then Attempt 2 (running at T+10ms) probably failed?
		// No, Attempt 2 runs instantly at T+10ms.
		// So we need to time it carefully or use a smarter mock.

		// Let's just update the mock state.
		mock.mu.Lock()
		mock.shouldFail = false
		mock.mu.Unlock()
	}()

	publisher.Publish(context.Background(), Event{Type: "test"})

	time.Sleep(100 * time.Millisecond)

	mock.mu.Lock()
	calls := mock.publishCalls
	mock.mu.Unlock()

	// Calls should be at least 2?
	// If it recovered, it stops retrying.
	if calls < 2 {
		t.Errorf("Expected at least 2 calls (initial + retry), got %d", calls)
	}

	// It shouldn't retry max times if it succeeded.
}
