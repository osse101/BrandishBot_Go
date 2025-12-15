package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	apiKey := "test-secret-key"
	middleware := AuthMiddleware(apiKey)

	// Mock handler that returns 200 OK
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		path           string
		requestHeaders map[string]string
		expectedStatus int
	}{
		{
			name:           "Valid API Key",
			path:           "/api/sensitive",
			requestHeaders: map[string]string{"X-API-Key": apiKey},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid API Key",
			path:           "/api/sensitive",
			requestHeaders: map[string]string{"X-API-Key": "wrong-key"},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing API Key",
			path:           "/api/sensitive",
			requestHeaders: map[string]string{},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Public Path - Healthz",
			path:           "/healthz",
			requestHeaders: map[string]string{},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Public Path - Metrics",
			path:           "/metrics",
			requestHeaders: map[string]string{},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Public Path - Swagger",
			path:           "/swagger/index.html",
			requestHeaders: map[string]string{},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			for k, v := range tt.requestHeaders {
				req.Header.Set(k, v)
			}

			rr := httptest.NewRecorder()
			middleware(nextHandler).ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.expectedStatus)
			}
		})
	}
}
