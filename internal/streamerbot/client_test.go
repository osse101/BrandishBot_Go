package streamerbot

import (
	"testing"
	"time"
)

// TestClient_NewClient verifies that the client is initialized correctly
func TestClient_NewClient(t *testing.T) {
	client := NewClient("", "test_password")

	if client.url != DefaultURL {
		t.Errorf("Expected default URL %s, got %s", DefaultURL, client.url)
	}

	if client.password != "test_password" {
		t.Errorf("Expected password 'test_password', got %s", client.password)
	}

	if client.wakeup == nil {
		t.Error("Expected wakeup channel to be initialized")
	}

	if cap(client.wakeup) != 1 {
		t.Errorf("Expected wakeup channel buffer size 1, got %d", cap(client.wakeup))
	}

	if client.dormant {
		t.Error("Expected dormant to be false initially")
	}
}

// TestClient_DoActionWhenDormant verifies wakeup is triggered when dormant
func TestClient_DoActionWhenDormant(t *testing.T) {
	client := NewClient("ws://localhost:9999/invalid", "")

	// Manually set client to dormant state
	client.mu.Lock()
	client.dormant = true
	client.mu.Unlock()

	// Trigger DoAction
	err := client.DoAction("test_action", map[string]string{"key": "value"})

	// Should return an error
	if err == nil {
		t.Fatal("Expected error from DoAction when dormant")
	}

	expectedErrMsg := "Streamer.bot is dormant, reconnection triggered"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}

	// Verify wakeup signal was sent
	select {
	case <-client.wakeup:
		// Good, wakeup was triggered
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected wakeup signal to be sent")
	}
}

// TestClient_DoActionWhenNotConnected verifies error when not connected but not dormant
func TestClient_DoActionWhenNotConnected(t *testing.T) {
	client := NewClient("ws://localhost:9999/invalid", "")

	// Client is not connected and not dormant
	err := client.DoAction("test_action", map[string]string{"key": "value"})

	if err == nil {
		t.Fatal("Expected error from DoAction when not connected")
	}

	expectedErrMsg := "not connected to Streamer.bot"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

// TestClient_WakeupBuffered verifies multiple wakeup calls don't block
func TestClient_WakeupBuffered(t *testing.T) {
	client := NewClient("", "")

	client.mu.Lock()
	client.dormant = true
	client.mu.Unlock()

	// First call should send wakeup
	err1 := client.DoAction("test1", nil)
	if err1 == nil {
		t.Fatal("Expected error from first DoAction")
	}

	// Second call should not block (buffered channel default case)
	err2 := client.DoAction("test2", nil)
	if err2 == nil {
		t.Fatal("Expected error from second DoAction")
	}

	// Verify channel only has one signal
	select {
	case <-client.wakeup:
		// First signal
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected at least one wakeup signal")
	}

	// Should not have a second signal (channel was buffered)
	select {
	case <-client.wakeup:
		t.Error("Should not have multiple wakeup signals")
	case <-time.After(100 * time.Millisecond):
		// Good, no second signal
	}
}

// TestMaxConsecutiveFailures verifies the constant is set correctly
func TestMaxConsecutiveFailures(t *testing.T) {
	if MaxConsecutiveFailures != 10 {
		t.Errorf("MaxConsecutiveFailures should be 10, got %d", MaxConsecutiveFailures)
	}
}

// TestConstants verifies backoff constants
func TestConstants(t *testing.T) {
	if DefaultReconnectDelay != 1*time.Second {
		t.Errorf("DefaultReconnectDelay should be 1s, got %v", DefaultReconnectDelay)
	}
	if MaxReconnectDelay != 30*time.Second {
		t.Errorf("MaxReconnectDelay should be 30s, got %v", MaxReconnectDelay)
	}
	if ReconnectMultiplier != 2.0 {
		t.Errorf("ReconnectMultiplier should be 2.0, got %v", ReconnectMultiplier)
	}
}
