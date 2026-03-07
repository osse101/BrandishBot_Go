//go:build staging

package staging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// TestStatsEndpoints tests all stats-related endpoints
func TestStatsEndpoints(t *testing.T) {
	t.Run("SystemStats", func(t *testing.T) {
		resp, body := makeRequest(t, "GET", "/api/v1/stats/system", nil)

		require.Equal(t, http.StatusOK, resp.StatusCode, "Body: %s", string(body))

		var result map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &result), "Failed to unmarshal response")

		// Should have system stats
		assert.NotEmpty(t, result, "Expected system stats, got empty response")
	})

	t.Run("Leaderboard", func(t *testing.T) {
		eventType := "message_sent"
		path := fmt.Sprintf("/api/v1/stats/leaderboard?event_type=%s", eventType)
		resp, body := makeRequest(t, "GET", path, nil)

		require.Equal(t, http.StatusOK, resp.StatusCode, "Body: %s", string(body))

		var result map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &result), "Failed to unmarshal response")

		// Should have entries field
		assert.Contains(t, result, "entries", "Expected 'entries' field in response")
	})

	t.Run("UserStats", func(t *testing.T) {
		// Use a valid UUID for testing
		userID := "00000000-0000-0000-0000-000000000001"
		platform := domain.PlatformTwitch
		path := fmt.Sprintf("/api/v1/stats/user?platform=%s&platform_id=%s", platform, userID)
		resp, body := makeRequest(t, "GET", path, nil)

		// 200 or 404 are both valid (404 if user doesn't exist)
		require.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound,
			"Expected status 200 or 404, got %d. Body: %s", resp.StatusCode, string(body))
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
	resp, body := makeRequest(t, "POST", "/api/v1/user/register", regRequest)
	require.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated,
		"Failed to register user for stats test: %d. Body: %s", resp.StatusCode, string(body))

	var userResp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &userResp), "Failed to unmarshal register response")

	registeredUserID, ok := userResp["internal_id"].(string)
	require.True(t, ok, "Response missing internal_id: %s", string(body))

	// Use valid UUID from registration
	eventType := "message_sent"
	event := map[string]interface{}{
		"user_id":    registeredUserID,
		"event_type": eventType,
		"platform":   platform,
		"metadata":   map[string]interface{}{},
	}

	resp, body = makeRequest(t, "POST", "/api/v1/stats/event", event)

	// Should accept the event
	require.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated,
		"Expected status 200 or 201, got %d. Body: %s", resp.StatusCode, string(body))
}
