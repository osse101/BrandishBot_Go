package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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
		ctx := context.WithValue(context.Background(), "other-key", "other-value")

		newCtx := WithUserID(ctx, "user-123")

		// Both values should exist
		assert.Equal(t, "user-123", GetUserID(newCtx))
		assert.Equal(t, "other-value", newCtx.Value("other-key"))
	})
}

// TestExtractUserID tests user ID extraction from requests
func TestExtractUserID(t *testing.T) {
	t.Run("extracts from context using string key", func(t *testing.T) {
		// Note: extractUserID uses string literal "user_id" not UserIDKey
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), "user_id", "context-user")
		req = req.WithContext(ctx)

		userID := extractUserID(req)

		assert.Equal(t, "context-user", userID,
			"extractUserID should work with raw string key for backward compatibility")
	})

	t.Run("does not extract from UserIDKey constant", func(t *testing.T) {
		// This documents current behavior - extractUserID uses string, not constant
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := WithUserID(req.Context(), "context-user")
		req = req.WithContext(ctx)

		userID := extractUserID(req)

		// Current implementation won't find it because it uses string literal
		// This test documents the inconsistency
		assert.Equal(t, "", userID,
			"extractUserID doesn't use UserIDKey constant - potential bug")
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
