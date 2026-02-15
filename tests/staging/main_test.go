//go:build staging

package staging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

var (
	stagingURL string
	client     *http.Client
)

func TestMain(m *testing.M) {
	// Get API URL from environment or default to localhost
	stagingURL = os.Getenv("API_URL")
	if stagingURL == "" {
		stagingURL = "http://localhost:8080"
	}

	// Configure HTTP client with timeout
	client = &http.Client{
		Timeout: 10 * time.Second,
	}

	// Run tests
	os.Exit(m.Run())
}

// Helper function to make requests
func makeRequest(t *testing.T, method, path string, body interface{}) (*http.Response, []byte) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	url := fmt.Sprintf("%s%s", stagingURL, path)
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add API key
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		apiKey = "test-api-key" // Default for local testing if not specified
	}
	req.Header.Set("X-API-Key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request to %s: %v", url, err)
	}
	// Don't close body here, let caller do it or read it all

	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	return resp, respBody
}
