package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const (
	colorGreen  = "\033[0;32m"
	colorRed    = "\033[0;31m"
	colorYellow = "\033[1;33m"
	colorReset  = "\033[0m"
)

func runTestSecurity() {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		fmt.Printf("%sError: API_KEY not found in environment (check .env file)%s\n", colorRed, colorReset)
		os.Exit(1)
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	fmt.Printf("%s=== Security Feature Tests ===%s\n\n", colorYellow, colorReset)

	failures := 0

	// Test 1: Request without API key
	if !runTestCase(1, "Request without API key (should fail with 401)", baseURL, "", 401, map[string]interface{}{
		"username":    "testuser",
		"platform":    "twitch",
		"platform_id": "12345",
	}) {
		failures++
	}

	// Test 2: Request with wrong API key
	if !runTestCase(2, "Request with wrong API key (should fail with 401)", baseURL, "wrong_key", 401, map[string]interface{}{
		"username":    "testuser",
		"platform":    "twitch",
		"platform_id": "12345",
	}) {
		failures++
	}

	// Test 3: Request with valid API key
	if !runTestCase(3, "Request with valid API key (should succeed with 200/201)", baseURL, apiKey, 200, map[string]interface{}{
		"username":    "testuser",
		"platform":    "twitch",
		"platform_id": "12345",
	}) {
		failures++
	}

	// Test 4: Invalid platform
	if !runTestCase(4, "Invalid platform (should fail with 400)", baseURL, apiKey, 400, map[string]interface{}{
		"username":    "testuser",
		"platform":    "invalid_platform",
		"platform_id": "12345",
	}) {
		failures++
	}

	// Test 5: Username too long
	longUsername := strings.Repeat("A", 200)
	if !runTestCase(5, "Username too long (should fail with 400)", baseURL, apiKey, 400, map[string]interface{}{
		"username":    longUsername,
		"platform":    "twitch",
		"platform_id": "12345",
	}) {
		failures++
	}

	// Test 6: Username with control characters
	if !runTestCase(6, "Username with control characters (should fail with 400)", baseURL, apiKey, 400, map[string]interface{}{
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
		statusCode := makeRequest(baseURL, apiKey, payload)
		if statusCode == 200 || statusCode == 201 {
			fmt.Printf("  - %s: %d\n", p, statusCode)
		} else {
			fmt.Printf("  - %s: %s%d%s\n", p, colorRed, statusCode, colorReset)
			failures++
		}
	}
	fmt.Println()

	if failures > 0 {
		fmt.Printf("%s=== Security Tests Failed (%d failures) ===%s\n", colorRed, failures, colorReset)
		os.Exit(1)
	}

	fmt.Printf("%s=== Security Tests Complete ===%s\n", colorGreen, colorReset)
}

func runTestCase(testNum int, description, baseURL, apiKey string, expectedStatus int, payload interface{}) bool {
	fmt.Printf("Test %d: %s\n", testNum, description)
	statusCode := makeRequest(baseURL, apiKey, payload)

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

func makeRequest(baseURL, apiKey string, payload interface{}) int {
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
