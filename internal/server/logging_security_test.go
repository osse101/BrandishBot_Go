package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingMiddleware_RedactsSecrets(t *testing.T) {
	// Setup logger to write to buffer
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug, // Must be Debug to log headers
	}
	l := slog.New(slog.NewTextHandler(&buf, opts))
	slog.SetDefault(l)

	// Dummy handler
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	handler := loggingMiddleware(next)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "secret-key-123")
	req.Header.Set("Authorization", "Bearer mytoken")
	req.Header.Set("User-Agent", "TestAgent")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	logOutput := buf.String()

	// Check if headers are logged at all (pre-condition)
	if !strings.Contains(logOutput, "Request headers") {
		t.Fatalf("Log output missing headers log: %s", logOutput)
	}

	// Check for leaks
	if strings.Contains(logOutput, "secret-key-123") {
		t.Errorf("SECURITY FAIL: Log output contains X-API-Key value: %s", logOutput)
	}

	if strings.Contains(logOutput, "Bearer mytoken") {
		t.Errorf("SECURITY FAIL: Log output contains Authorization value: %s", logOutput)
	}

	// Check that non-sensitive headers are still present
	if !strings.Contains(logOutput, "TestAgent") {
		t.Errorf("Log output missing non-sensitive header: %s", logOutput)
	}
}
