//go:build staging

package staging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// TestUserRegistration tests user registration endpoint
func TestUserRegistration(t *testing.T) {
	username := "StagingTestUser"
	platform := domain.PlatformTwitch
	userID := fmt.Sprintf("staging_user_%d", time.Now().Unix())

	request := map[string]interface{}{
		"username":          username,
		"known_platform":    platform,
		"known_platform_id": userID,
		"new_platform":      platform,
		"new_platform_id":   userID,
	}

	resp, body := makeRequest(t, "POST", "/api/v1/user/register", request)

	// 200 for success, 400 if already exists
	require.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest,
		"Unexpected status: %d. Body: %s", resp.StatusCode, string(body))
}

// TestInventoryEndpoint tests getting user inventory
func TestInventoryEndpoint(t *testing.T) {
	// Register a fresh user
	ts := time.Now().Unix()
	username := fmt.Sprintf("InvUser_%d", ts)
	platform := "twitch"
	platformID := fmt.Sprintf("inv_pid_%d", ts)

	regReq := map[string]interface{}{
		"username":          username,
		"known_platform":    platform,
		"known_platform_id": platformID,
		"new_platform":      platform,
		"new_platform_id":   platformID,
	}
	regResp, _ := makeRequest(t, "POST", "/api/v1/user/register", regReq)
	require.True(t, regResp.StatusCode == http.StatusCreated || regResp.StatusCode == http.StatusOK,
		"Failed to register inventory user")

	path := fmt.Sprintf("/api/v1/user/inventory?platform=%s&platform_id=%s&username=%s", platform, platformID, username)
	resp, body := makeRequest(t, "GET", path, nil)

	require.Equal(t, http.StatusOK, resp.StatusCode, "Body: %s", string(body))

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &result), "Failed to unmarshal response")

	// Should have inventory-related fields
	assert.Contains(t, result, "items", "Expected 'items' field in inventory response")
}

// TestPricesEndpoint tests the prices endpoint
func TestPricesEndpoint(t *testing.T) {
	resp, body := makeRequest(t, "GET", "/api/v1/prices", nil)

	require.Equal(t, http.StatusOK, resp.StatusCode, "Body: %s", string(body))

	var result []interface{}
	require.NoError(t, json.Unmarshal(body, &result), "Failed to unmarshal response")

	// Should return pricing information
	if len(result) == 0 {
		t.Log("Warning: No prices returned, but endpoint working")
	}
}

// TestRecipesEndpoint tests the crafting recipes endpoint
func TestRecipesEndpoint(t *testing.T) {
	// Test requirements: recipe lookup needs item or user (with platform/ID); uses fresh registered user.
	// Register a fresh user
	ts := time.Now().Unix()
	username := fmt.Sprintf("RecipeUser_%d", ts)
	platform := "twitch" // domain.PlatformTwitch string value
	platformID := fmt.Sprintf("recipe_pid_%d", ts)

	regReq := map[string]interface{}{
		"username":          username,
		"known_platform":    platform,
		"known_platform_id": platformID,
		"new_platform":      platform,
		"new_platform_id":   platformID,
	}
	regResp, _ := makeRequest(t, "POST", "/api/v1/user/register", regReq)
	require.True(t, regResp.StatusCode == http.StatusCreated || regResp.StatusCode == http.StatusOK,
		"Failed to register recipe user")

	path := fmt.Sprintf("/api/v1/recipes?username=%s&platform=%s&platform_id=%s", username, platform, platformID)
	resp, body := makeRequest(t, "GET", path, nil)

	require.Equal(t, http.StatusOK, resp.StatusCode, "Body: %s", string(body))

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &result), "Failed to unmarshal response")

	// Should have recipes field
	assert.Contains(t, result, "recipes", "Expected 'recipes' field in response")
}
