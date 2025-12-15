package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	apiKey := "secret-key"
	middleware := AuthMiddleware(apiKey)

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
