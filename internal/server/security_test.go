package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	apiKey := "secret-key"
	detector := NewSuspiciousActivityDetector()
	middleware := AuthMiddleware(apiKey, detector)

	tests := []struct {
		name           string
		providedKey    string
		path           string
		expectedStatus int
	}{
		{
			name:           "Valid API Key",
			providedKey:    apiKey,
			path:           "/api/test",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid API Key",
			providedKey:    "wrong-key",
			path:           "/api/test",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing API Key",
			providedKey:    "",
			path:           "/api/test",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Public Path - Healthz",
			providedKey:    "",
			path:           "/healthz",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Public Path - Metrics",
			providedKey:    "",
			path:           "/metrics",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.providedKey != "" {
				req.Header.Set("X-API-Key", tt.providedKey)
			}
			rec := httptest.NewRecorder()

			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestAuthMiddleware_RecordsFailures(t *testing.T) {
	apiKey := "secret-key"
	detector := NewSuspiciousActivityDetector()
	middleware := AuthMiddleware(apiKey, detector)

	// Create request with specific IP
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.1.5:12345"

	rec := httptest.NewRecorder()
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Execute failed request
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}

	// Check detector state
	ip := "192.168.1.5"
	count, exists := detector.failedAuthByIP[ip]
	if !exists {
		t.Errorf("expected IP %s to be in failedAuthByIP map", ip)
	}
	if count != 1 {
		t.Errorf("expected failure count 1, got %d", count)
	}

	// Trigger another failure
	handler.ServeHTTP(rec, req)
	if detector.failedAuthByIP[ip] != 2 {
		t.Errorf("expected failure count 2, got %d", detector.failedAuthByIP[ip])
	}
}
