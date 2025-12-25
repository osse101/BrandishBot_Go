//go:build staging

package staging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
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
		eventType := "message_sent"
		path := fmt.Sprintf("/stats/leaderboard?event_type=%s", eventType)
		resp, body := makeRequest(t, "GET", path, nil)
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Should have entries field
		if _, ok := result["entries"]; !ok {
			t.Error("Expected 'entries' field in response")
		}
	})

	t.Run("UserStats", func(t *testing.T) {
		// Use a valid UUID for testing
		userID := "00000000-0000-0000-0000-000000000001"
		platform := domain.PlatformTwitch
		path := fmt.Sprintf("/stats/user?user_id=%s&platform=%s", userID, platform)
		resp, body := makeRequest(t, "GET", path, nil)
		
		// 200 or 404 are both valid (404 if user doesn't exist)
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404, got %d. Body: %s", resp.StatusCode, string(body))
		}
	})
}

// TestRecordEvent tests the event recording endpoint
func TestRecordEvent(t *testing.T) {
	// Register a user first to avoid FK constraint violation
	username := "StatsTestUser"
	platform := domain.PlatformTwitch
	platformID := "stats_test_twitch_id"

	regRequest := map[string]interface{}{
		"username":          username,
		"known_platform":    platform,
		"known_platform_id": platformID,
		"new_platform":      platform,
		"new_platform_id":   platformID,
	}
	resp, body := makeRequest(t, "POST", "/user/register", regRequest)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to register user for stats test: %d. Body: %s", resp.StatusCode, string(body))
	}

	var userResp map[string]interface{}
	if err := json.Unmarshal(body, &userResp); err != nil {
		t.Fatalf("Failed to unmarshal register response: %v", err)
	}
	registeredUserID := userResp["internal_id"].(string)

	// Use valid UUID from registration
	eventType := "message_sent"
	event := map[string]interface{}{
		"user_id":    registeredUserID,
		"event_type": eventType,
		"platform":   platform,
		"metadata":   map[string]interface{}{},
	}

	resp, body = makeRequest(t, "POST", "/stats/event", event)
	
	// Should accept the event
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 200 or 201, got %d. Body: %s", resp.StatusCode, string(body))
	}
}
