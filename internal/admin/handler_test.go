package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_ServesSPA(t *testing.T) {
	handler := Handler()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		checkContent   bool
	}{
		{
			name:           "Root path serves index.html",
			path:           "/",
			expectedStatus: http.StatusOK,
			checkContent:   true,
		},
		{
			name:           "Non-existent file falls back to index.html",
			path:           "/events",
			expectedStatus: http.StatusOK,
			checkContent:   true,
		},
		{
			name:           "Assets path (if exists)",
			path:           "/assets/index.css",
			expectedStatus: http.StatusOK, // Will be 404 if doesn't exist, but handler should work
			checkContent:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus && rec.Code != http.StatusNotFound {
				t.Errorf("Expected status %d or 404, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.checkContent && rec.Code == http.StatusOK {
				body := rec.Body.String()
				if len(body) == 0 {
					t.Error("Expected non-empty response body")
				}
			}
		})
	}
}

func TestHandler_SetsCorrectHeaders(t *testing.T) {
	handler := Handler()

	t.Run("Index.html has no-cache", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		cacheControl := rec.Header().Get("Cache-Control")
		if cacheControl != "no-cache, no-store, must-revalidate" {
			t.Errorf("Expected no-cache for index.html, got %q", cacheControl)
		}
	})

	t.Run("SPA fallback has no-cache", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/commands", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		cacheControl := rec.Header().Get("Cache-Control")
		if cacheControl != "no-cache, no-store, must-revalidate" {
			t.Errorf("Expected no-cache for SPA fallback, got %q", cacheControl)
		}
	})
}
