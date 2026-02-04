package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type TestSecurityCommand struct{}

func (c *TestSecurityCommand) Name() string {
	return "test-security"
}

func (c *TestSecurityCommand) Description() string {
	return "Run API security tests"
}

func (c *TestSecurityCommand) Run(args []string) error {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		PrintError("API_KEY not found in environment (check .env file)")
		return fmt.Errorf("API_KEY not found")
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	PrintHeader("Security Feature Tests")

	failures := 0

	// Test 1: Request without API key
	if !c.runTestCase(1, "Request without API key (should fail with 401)", baseURL, "", 401, map[string]interface{}{
		"username":    "testuser",
		"platform":    "twitch",
		"platform_id": "12345",
	}) {
		failures++
	}

	// Test 2: Request with wrong API key
	if !c.runTestCase(2, "Request with wrong API key (should fail with 401)", baseURL, "wrong_key", 401, map[string]interface{}{
		"username":    "testuser",
		"platform":    "twitch",
		"platform_id": "12345",
	}) {
		failures++
	}

	// Test 3: Request with valid API key
	if !c.runTestCase(3, "Request with valid API key (should succeed with 200/201)", baseURL, apiKey, 200, map[string]interface{}{
		"username":    "testuser",
		"platform":    "twitch",
		"platform_id": "12345",
	}) {
		failures++
	}

	// Test 4: Invalid platform
	if !c.runTestCase(4, "Invalid platform (should fail with 400)", baseURL, apiKey, 400, map[string]interface{}{
		"username":    "testuser",
		"platform":    "invalid_platform",
		"platform_id": "12345",
	}) {
		failures++
	}

	// Test 5: Username too long
	longUsername := strings.Repeat("A", 200)
	if !c.runTestCase(5, "Username too long (should fail with 400)", baseURL, apiKey, 400, map[string]interface{}{
		"username":    longUsername,
		"platform":    "twitch",
		"platform_id": "12345",
	}) {
		failures++
	}

	// Test 6: Username with control characters
	if !c.runTestCase(6, "Username with control characters (should fail with 400)", baseURL, apiKey, 400, map[string]interface{}{
		"username":    "test\nuser",
		"platform":    "twitch",
		"platform_id": "12345",
	}) {
		failures++
	}

	// Test 7: Valid platforms
	fmt.Println("Test 7: Valid platforms (should all succeed)")
	platforms := []string{"twitch", "youtube", "discord"}
	for _, p := range platforms {
		payload := map[string]interface{}{
			"username":    "user_" + p,
			"platform":    p,
			"platform_id": "12345",
		}
		statusCode := c.makeRequest(baseURL, apiKey, payload)
		if statusCode == 200 || statusCode == 201 {
			fmt.Printf("  - %s: %d\n", p, statusCode)
		} else {
			fmt.Printf("  - %s: %s%d%s\n", p, colorRed, statusCode, colorReset)
			failures++
		}
	}
	fmt.Println()

	if failures > 0 {
		PrintError("Security Tests Failed (%d failures)", failures)
		return fmt.Errorf("security tests failed")
	}

	PrintSuccess("Security Tests Complete")
	return nil
}

func (c *TestSecurityCommand) runTestCase(testNum int, description, baseURL, apiKey string, expectedStatus int, payload interface{}) bool {
	fmt.Printf("Test %d: %s\n", testNum, description)
	statusCode := c.makeRequest(baseURL, apiKey, payload)

	if statusCode == expectedStatus || (expectedStatus == 200 && statusCode == 201) {
		fmt.Printf(" - Result: %d (OK)\n", statusCode)
		fmt.Println()
		return true
	} else {
		fmt.Printf(" - Result: %s%d (Expected %d)%s\n", colorRed, statusCode, expectedStatus, colorReset)
		fmt.Println()
		return false
	}
}

func (c *TestSecurityCommand) makeRequest(baseURL, apiKey string, payload interface{}) int {
	body, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshaling payload: %v\n", err)
		return 0
	}

	req, err := http.NewRequest("POST", baseURL+"/test", bytes.NewBuffer(body))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return 0
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return 0
	}
	defer resp.Body.Close()

	return resp.StatusCode
}
