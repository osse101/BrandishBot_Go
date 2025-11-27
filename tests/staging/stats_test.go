//go:build staging

package staging

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestStatsEndpoints tests all stats-related endpoints
func TestStatsEndpoints(t *testing.T) {
	t.Run("SystemStats", func(t *testing.T) {
		resp, body := makeRequest(t, "GET", "/stats/system", nil)
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Should have system stats
		if len(result) == 0 {
			t.Error("Expected system stats, got empty response")
		}
	})

	t.Run("Leaderboard", func(t *testing.T) {
		resp, body := makeRequest(t, "GET", "/stats/leaderboard", nil)
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Should have leaderboard field
		if _, ok := result["leaderboard"]; !ok {
			t.Error("Expected 'leaderboard' field in response")
		}
	})

	t.Run("UserStats", func(t *testing.T) {
		userID := "test_user"
		resp, body := makeRequest(t, "GET", "/stats/user?user_id="+userID, nil)
		
		// 200 or 404 are both valid (404 if user doesn't exist)
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})
}

// TestRecordEvent tests the event recording endpoint
func TestRecordEvent(t *testing.T) {
	event := map[string]interface{}{
		"user_id":    "test_user_stats",
		"event_type": "message_sent",
		"platform":   "twitch",
		"metadata":   map[string]interface{}{},
	}

	resp, body := makeRequest(t, "POST", "/stats/event", event)
	
	// Should accept the event
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 200 or 201, got %d. Body: %s", resp.StatusCode, string(body))
	}
}
