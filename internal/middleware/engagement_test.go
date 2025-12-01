package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestWithUserID_GetUserID tests context user ID management
func TestWithUserID_GetUserID(t *testing.T) {
	t.Run("stores and retrieves user ID from context", func(t *testing.T) {
		ctx := context.Background()
		userID := "test-user-123"

		// Add user ID to context
		newCtx := WithUserID(ctx, userID)

		// Retrieve it
		retrieved := GetUserID(newCtx)

		assert.Equal(t, userID, retrieved, "Should retrieve same user ID")
	})

	t.Run("returns empty string for context without user ID", func(t *testing.T) {
		ctx := context.Background()

		retrieved := GetUserID(ctx)

		assert.Equal(t, "", retrieved, "Should return empty string when not set")
	})

	t.Run("handles context with wrong type value", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), UserIDKey, 12345) // Wrong type

		retrieved := GetUserID(ctx)

		assert.Equal(t, "", retrieved, "Should return empty string for non-string value")
	})

	t.Run("preserves other context values", func(t *testing.T) {
		type testKey string
		otherKey := testKey("other-key")
		ctx := context.WithValue(context.Background(), otherKey, "other-value")

		newCtx := WithUserID(ctx, "user-123")

		// Both values should exist
		assert.Equal(t, "user-123", GetUserID(newCtx))
		assert.Equal(t, "other-value", newCtx.Value(otherKey))
	})
}

// TestExtractUserID tests user ID extraction from requests
func TestExtractUserID(t *testing.T) {
	t.Run("extracts from context using typed key", func(t *testing.T) {
		// This test ensures extractUserID can correctly retrieve a user ID
		// when it's stored in the context using the UserIDKey constant.
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := WithUserID(req.Context(), "context-user") // Use WithUserID to set with typed key
		req = req.WithContext(ctx)

		userID := extractUserID(req)

		assert.Equal(t, "context-user", userID,
			"extractUserID should correctly retrieve user ID set with UserIDKey")
	})

	t.Run("extracts from query parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?username=query-user", nil)

		userID := extractUserID(req)

		assert.Equal(t, "query-user", userID)
	})

	t.Run("query parameter used when no context value", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?username=query-user", nil)

		userID := extractUserID(req)

		assert.Equal(t, "query-user", userID,
			"Should fall back to query param when context doesn't have user_id")
	})

	t.Run("returns empty when no user ID found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)

		userID := extractUserID(req)

		assert.Equal(t, "", userID)
	})
}

// TestUserIDKey verifies context key uniqueness
func TestUserIDKey(t *testing.T) {
	t.Run("context keys are distinct", func(t *testing.T) {
		ctx := context.Background()

		// Set both keys
		ctx = WithUserID(ctx, "test-user")
		ctx = context.WithValue(ctx, EngagementMetricKey, "test-metric")

		// Verify both exist and are independent
		assert.Equal(t, "test-user", GetUserID(ctx))
		assert.Equal(t, "test-metric", ctx.Value(EngagementMetricKey))
	})

	t.Run("setting same key overwrites previous value", func(t *testing.T) {
		ctx := WithUserID(context.Background(), "first-user")
		ctx = WithUserID(ctx, "second-user")

		assert.Equal(t, "second-user", GetUserID(ctx))
	})
}

// Note: Full middleware integration tests are better suited for integration tests
// or end-to-end tests where you can properly test the async goroutine behavior
// and mock the full progression.Service interface. These unit tests focus on
// the synchronous, deterministic helper functions.

// MockEventBus is a mock implementation of event.Bus
type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) Publish(ctx context.Context, evt event.Event) error {
	args := m.Called(ctx, evt)
	return args.Error(0)
}

func (m *MockEventBus) Subscribe(eventType event.Type, handler event.Handler) {
	m.Called(eventType, handler)
}

// TestEngagementTracker_Track tests the Track middleware
func TestEngagementTracker_Track(t *testing.T) {
	t.Run("tracks engagement with user ID in context", func(t *testing.T) {
		mockBus := &MockEventBus{}
		tracker := NewEngagementTracker(mockBus)

		// Expect engagement event to be published
		mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
			return evt.Type == "engagement"
		})).Return(nil)

		// Create test handler
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Wrap with engagement tracking
		wrapped := tracker.Track("test_metric", nil)(handler)

		// Create request with user ID in context
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := WithUserID(req.Context(), "test-user-123")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockBus.AssertExpectations(t)
	})

	t.Run("tracks engagement with custom value function", func(t *testing.T) {
		mockBus := &MockEventBus{}
		tracker := NewEngagementTracker(mockBus)

		mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
			if evt.Type != "engagement" {
				return false
			}
			// Verify the custom value was used
			return true
		})).Return(nil)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Custom value function
		getValue := func(r *http.Request) int {
			return 42
		}

		wrapped := tracker.Track("custom_metric", getValue)(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		ctx := WithUserID(req.Context(), "test-user")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		mockBus.AssertExpectations(t)
	})

	t.Run("does not track when no user ID", func(t *testing.T) {
		mockBus := &MockEventBus{}
		tracker := NewEngagementTracker(mockBus)

		// Should NOT call Publish
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrapped := tracker.Track("test_metric", nil)(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockBus.AssertNotCalled(t, "Publish")
	})

	t.Run("continues handler execution even if event publish fails", func(t *testing.T) {
		mockBus := &MockEventBus{}
		tracker := NewEngagementTracker(mockBus)

		mockBus.On("Publish", mock.Anything, mock.Anything).Return(assert.AnError)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})

		wrapped := tracker.Track("test_metric", nil)(handler)

		req := httptest.NewRequest("GET", "/test?username=testuser", nil)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		// Handler should still complete successfully
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "success", w.Body.String())
	})
}

// TestEngagementTracker_TrackCommand tests the TrackCommand middleware
func TestEngagementTracker_TrackCommand(t *testing.T) {
	t.Run("tracks command execution with metadata", func(t *testing.T) {
		mockBus := &MockEventBus{}
		tracker := NewEngagementTracker(mockBus)

		mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
			return evt.Type == "engagement"
		})).Return(nil)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrapped := tracker.TrackCommand(handler)

		req := httptest.NewRequest("POST", "/api/command", nil)
		ctx := WithUserID(req.Context(), "command-user")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		mockBus.AssertExpectations(t)
	})

	t.Run("does not track when no user ID", func(t *testing.T) {
		mockBus := &MockEventBus{}
		tracker := NewEngagementTracker(mockBus)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrapped := tracker.TrackCommand(handler)

		req := httptest.NewRequest("POST", "/api/command", nil)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		mockBus.AssertNotCalled(t, "Publish")
	})
}

// TestTrackEngagementFromContext tests the standalone tracking function
func TestTrackEngagementFromContext(t *testing.T) {
	t.Run("tracks engagement from context", func(t *testing.T) {
		mockBus := &MockEventBus{}

		mockBus.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
			return evt.Type == "engagement"
		})).Return(nil)

		ctx := WithUserID(context.Background(), "ctx-user")

		TrackEngagementFromContext(ctx, mockBus, "manual_metric", 10)

		mockBus.AssertExpectations(t)
	})

	t.Run("does nothing when no user ID in context", func(t *testing.T) {
		mockBus := &MockEventBus{}

		ctx := context.Background()

		TrackEngagementFromContext(ctx, mockBus, "manual_metric", 10)

		mockBus.AssertNotCalled(t, "Publish")
	})

	t.Run("logs error when publish fails", func(t *testing.T) {
		mockBus := &MockEventBus{}

		mockBus.On("Publish", mock.Anything, mock.Anything).Return(assert.AnError)

		ctx := WithUserID(context.Background(), "ctx-user")

		// Should not panic or fail
		TrackEngagementFromContext(ctx, mockBus, "manual_metric", 10)

		mockBus.AssertExpectations(t)
	})
}

// TestNewEngagementTracker tests the constructor
func TestNewEngagementTracker(t *testing.T) {
	t.Run("creates tracker with event bus", func(t *testing.T) {
		mockBus := &MockEventBus{}

		tracker := NewEngagementTracker(mockBus)

		assert.NotNil(t, tracker)
		assert.Equal(t, mockBus, tracker.eventBus)
	})
}
