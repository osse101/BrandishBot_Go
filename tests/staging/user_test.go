//go:build staging

package staging

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestUserRegistration tests user registration endpoint
func TestUserRegistration(t *testing.T) {
	userID := fmt.Sprintf("staging_user_%d", time.Now().Unix())
	
	request := map[string]interface{}{
		"user_id":   userID,
		"username":  "StagingTestUser",
		"platform":  "twitch",
	}

	resp, body := makeRequest(t, "POST", "/user/register", request)
	
	// 200 for success, 400 if already exists
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Unexpected status: %d. Body: %s", resp.StatusCode, string(body))
	}
}

// TestInventoryEndpoint tests getting user inventory
func TestInventoryEndpoint(t *testing.T) {
	userID := "test_user"
	
	resp, body := makeRequest(t, "GET", fmt.Sprintf("/user/inventory?user_id=%s", userID), nil)
	
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
	resp, body := makeRequest(t, "GET", "/prices", nil)
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Should return pricing information
	if len(result) == 0 {
		t.Error("Expected pricing data, got empty response")
	}
}

// TestRecipesEndpoint tests the crafting recipes endpoint
func TestRecipesEndpoint(t *testing.T) {
	resp, body := makeRequest(t, "GET", "/recipes", nil)
	
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
