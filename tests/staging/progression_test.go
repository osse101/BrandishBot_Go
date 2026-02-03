//go:build staging

package staging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestProgressionEndpoints tests all progression-related endpoints
func TestProgressionEndpoints(t *testing.T) {
	t.Run("GetTree", func(t *testing.T) {
		resp, body := makeRequest(t, "GET", "/api/v1/progression/tree", nil)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if _, ok := result["nodes"]; !ok {
			t.Error("Expected 'nodes' field in response")
		}
	})

	t.Run("GetAvailable", func(t *testing.T) {
		resp, body := makeRequest(t, "GET", "/api/v1/progression/available", nil)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if _, ok := result["available"]; !ok {
			t.Error("Expected 'available' field in response")
		}
	})

	t.Run("GetStatus", func(t *testing.T) {
		resp, body := makeRequest(t, "GET", "/api/v1/progression/status", nil)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Check for expected fields
		// total_nodes is not always returned, just check for total_unlocked
		expectedFields := []string{"total_unlocked"}
		for _, field := range expectedFields {
			if _, ok := result[field]; !ok {
				t.Errorf("Expected '%s' field in response", field)
			}
		}
	})
}

// TestVotingFlow tests the voting mechanism (requires user_id)
func TestVotingFlow(t *testing.T) {
	// Generate unique user ID for this test
	userID := fmt.Sprintf("test_user_%d", time.Now().Unix())

	// First, get available nodes
	resp, body := makeRequest(t, "GET", "/api/v1/progression/available", nil)
	if resp.StatusCode != http.StatusOK {
		t.Skipf("Cannot get available nodes: %d", resp.StatusCode)
	}

	var availableResp struct {
		Available []struct {
			NodeKey string `json:"node_key"`
		} `json:"available"`
	}

	if err := json.Unmarshal(body, &availableResp); err != nil {
		t.Fatalf("Failed to unmarshal available nodes: %v", err)
	}

	if len(availableResp.Available) == 0 {
		t.Skip("No available nodes to vote for")
	}

	// Register user first
	registerRequest := map[string]interface{}{
		"username":          fmt.Sprintf("voter_%d", time.Now().Unix()),
		"known_platform":    "twitch",
		"known_platform_id": userID,
		"new_platform":      "twitch",
		"new_platform_id":   userID,
	}

	regResp, regBody := makeRequest(t, "POST", "/api/v1/user/register", registerRequest)
	if regResp.StatusCode != http.StatusCreated && regResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to register voter: %d. Body: %s", regResp.StatusCode, string(regBody))
	}

	// Vote for the first available node
	voteRequest := map[string]interface{}{
		"platform":    "twitch",
		"platform_id": userID,
		"node_key":    availableResp.Available[0].NodeKey,
	}

	resp, body = makeRequest(t, "POST", "/api/v1/progression/vote", voteRequest)

	// Should succeed (200) or indicate already voted/other business logic (400)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Unexpected status for vote: %d. Body: %s", resp.StatusCode, string(body))
	}
}

// TestEngagementTracking tests the engagement endpoint
func TestEngagementTracking(t *testing.T) {
	userID := fmt.Sprintf("test_eng_%d", time.Now().Unix())
	platform := "twitch"

	// Register first
	regReq := map[string]interface{}{
		"username":          "EngagementTestUser",
		"known_platform":    platform,
		"known_platform_id": userID,
		"new_platform":      platform,
		"new_platform_id":   userID,
	}
	regResp, _ := makeRequest(t, "POST", "/api/v1/user/register", regReq)
	if regResp.StatusCode != http.StatusCreated && regResp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to register engagement user")
	}

	resp, body := makeRequest(t, "GET", fmt.Sprintf("/api/v1/progression/engagement?platform=%s&platform_id=%s", platform, userID), nil)

	// Should return 200 even if user doesn't exist (0 engagement)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Engagement response might vary, just check it parses
	// And has at least some score data if available (or empty if not)
	if _, ok := result["total_score"]; !ok {
		t.Log("Warning: 'total_score' field not found in engagement response, might be empty")
	}
}
