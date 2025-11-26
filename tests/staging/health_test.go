//go:build staging

package staging

import (
	"net/http"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	resp, _ := makeRequest(t, "GET", "/healthz", nil)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
