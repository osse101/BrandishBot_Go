package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityLoggingMiddleware_RateLimiting(t *testing.T) {
	detector := NewSuspiciousActivityDetector()
	middleware := SecurityLoggingMiddleware(nil, detector)

	// Create a handler that always returns OK
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ip := "192.168.1.100"
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = ip + ":1234"

	// Simulate requests up to the limit
	// Limit is 1000
	for i := 0; i < 1000; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d failed with status %d", i, rec.Code)
		}
	}

	// Next request should be blocked
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429 Too Many Requests, got %d", rec.Code)
	}

	// Verify detector state
	detector.mu.Lock()
	count := detector.requestCountByIP[ip]
	detector.mu.Unlock()

	if count != 1001 {
		t.Errorf("expected count 1001, got %d", count)
	}
}
