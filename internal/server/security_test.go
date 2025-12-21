package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	apiKey := "secret-key"
	detector := NewSuspiciousActivityDetector()
	middleware := AuthMiddleware(apiKey, nil, detector)

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
	middleware := AuthMiddleware(apiKey, nil, detector)

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

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		xForwardedFor  string
		trustedProxies []string
		expectedIP     string
	}{
		{
			name:           "Direct connection, no trusted proxies",
			remoteAddr:     "1.2.3.4:1234",
			xForwardedFor:  "5.6.7.8",
			trustedProxies: nil,
			expectedIP:     "1.2.3.4",
		},
		{
			name:           "Trusted proxy",
			remoteAddr:     "10.0.0.1:1234",
			xForwardedFor:  "5.6.7.8",
			trustedProxies: []string{"10.0.0.1"},
			expectedIP:     "5.6.7.8",
		},
		{
			name:           "Untrusted proxy",
			remoteAddr:     "10.0.0.2:1234",
			xForwardedFor:  "5.6.7.8",
			trustedProxies: []string{"10.0.0.1"},
			expectedIP:     "10.0.0.2",
		},
		{
			name:           "Multiple X-Forwarded-For",
			remoteAddr:     "10.0.0.1:1234",
			xForwardedFor:  "5.6.7.8, 9.9.9.9",
			trustedProxies: []string{"10.0.0.1"},
			expectedIP:     "9.9.9.9",
		},
		{
			name:           "IPv6 Trusted Proxy",
			remoteAddr:     "[::1]:1234",
			xForwardedFor:  "2001:db8::1",
			trustedProxies: []string{"::1"},
			expectedIP:     "2001:db8::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}

			ip := extractIP(req, tt.trustedProxies)
			if ip != tt.expectedIP {
				t.Errorf("expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}
