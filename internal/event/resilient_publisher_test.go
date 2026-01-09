package event

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// MockBus is a simple mock for the Bus interface
type MockBus struct {
	PublishFunc func(ctx context.Context, event Event) error
}

func (m *MockBus) Publish(ctx context.Context, event Event) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, event)
	}
	return nil
}

func (m *MockBus) Subscribe(eventType Type, handler Handler) {
	// No-op for this test
}

func TestResilientPublisher_Publish_Success(t *testing.T) {
	mockBus := &MockBus{
		PublishFunc: func(ctx context.Context, event Event) error {
			return nil
		},
	}

	publisher := NewResilientPublisher(mockBus, 3, time.Millisecond, "")
	err := publisher.Publish(context.Background(), Event{Type: "test"})

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
}

func TestResilientPublisher_Publish_RetrySuccess(t *testing.T) {
	attempts := 0
	mockBus := &MockBus{
		PublishFunc: func(ctx context.Context, event Event) error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary failure")
			}
			return nil
		},
	}

	publisher := NewResilientPublisher(mockBus, 5, time.Millisecond, "")
	err := publisher.Publish(context.Background(), Event{Type: "test"})

	if err != nil {
		t.Errorf("Expected success after retries, got error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestResilientPublisher_Publish_DeadLetter(t *testing.T) {
	mockBus := &MockBus{
		PublishFunc: func(ctx context.Context, event Event) error {
			return errors.New("persistent failure")
		},
	}

	dlqPath := "test_dlq.jsonl"
	defer os.Remove(dlqPath)

	publisher := NewResilientPublisher(mockBus, 2, time.Millisecond, dlqPath)
	err := publisher.Publish(context.Background(), Event{Type: "test", Payload: "some data"})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	// Verify file exists and has content
	content, readErr := os.ReadFile(dlqPath)
	if readErr != nil {
		t.Fatalf("Failed to read DLQ file: %v", readErr)
	}

	if !strings.Contains(string(content), "persistent failure") {
		t.Errorf("Expected DLQ to contain error message, got: %s", string(content))
	}
	if !strings.Contains(string(content), "some data") {
		t.Errorf("Expected DLQ to contain payload, got: %s", string(content))
	}
}

func TestResilientPublisher_Publish_ContextCancel(t *testing.T) {
	mockBus := &MockBus{
		PublishFunc: func(ctx context.Context, event Event) error {
			return errors.New("fail")
		},
	}

	publisher := NewResilientPublisher(mockBus, 5, 100*time.Millisecond, "")
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	err := publisher.Publish(ctx, Event{Type: "test"})

	if err == nil {
		t.Error("Expected error due to context cancellation, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		// Note: The implementation might return the publish error or the context error depending on when the check happens.
		// In our implementation:
		// 1. Try publish -> fails
		// 2. Check if i < maxRetries -> yes
		// 3. Select -> ctx.Done() matches -> return ctx.Err()
		// So it should be context.Canceled
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
}
