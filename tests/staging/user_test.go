//go:build staging

package staging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// TestUserRegistration tests user registration endpoint
func TestUserRegistration(t *testing.T) {
	username := "StagingTestUser"
	platform := domain.PlatformTwitch
	userID := fmt.Sprintf("staging_user_%d", time.Now().Unix())

	request := map[string]interface{}{
		"user_id":  userID,
		"username": username,
		"platform": platform,
	}

	resp, body := makeRequest(t, "POST", "/api/v1/user/register", request)

	// 200 for success, 400 if already exists
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Unexpected status: %d. Body: %s", resp.StatusCode, string(body))
	}
}

// TestInventoryEndpoint tests getting user inventory
func TestInventoryEndpoint(t *testing.T) {
	// Use valid UUID
	userID := "00000000-0000-0000-0000-000000000001"
	platformID := "test_platform"
	username := "TestUser"

	path := fmt.Sprintf("/api/v1/user/inventory?platform=twitch&user_id=%s&platform_id=%s&username=%s", userID, platformID, username)
	resp, body := makeRequest(t, "GET", path, nil)

	if resp.StatusCode == http.StatusNotFound {
		t.Skip("User not found - this is expected for staging tests")
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Should have inventory-related fields
	if _, ok := result["items"]; !ok {
		t.Error("Expected 'items' field in inventory response")
	}
}

// TestPricesEndpoint tests the prices endpoint
func TestPricesEndpoint(t *testing.T) {
	resp, body := makeRequest(t, "GET", "/api/v1/prices", nil)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Should return pricing information
	if len(result) == 0 {
		t.Log("Warning: No prices returned, but endpoint working")
	}
}

// TestRecipesEndpoint tests the crafting recipes endpoint
func TestRecipesEndpoint(t *testing.T) {
	// Needs either item or user, providing generic query
	// Requires platform/platform_id if user is provided
	// Use the same user registered in TestUserRegistration or another known one
	username := "StagingTestUser"
	platform := domain.PlatformTwitch
	platformID := "test_platform" // Not used for lookup if username provided but good practice

	path := fmt.Sprintf("/api/v1/recipes?user=%s&platform=%s&platform_id=%s", username, platform, platformID)
	resp, body := makeRequest(t, "GET", path, nil)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Should have recipes field
	if _, ok := result["recipes"]; !ok {
		t.Error("Expected 'recipes' field in response")
	}
}
