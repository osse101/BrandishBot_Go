package user

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Helper function to set up the service for testing timeouts
func setupTimeoutService() Service {
	repo := NewFakeRepository()
	setupTestData(repo)
	// Using NewService from the package
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, nil, false)
	return svc
}

func TestAddTimeout(t *testing.T) {
	t.Run("Best Case - New Timeout", func(t *testing.T) {
		svc := setupTimeoutService()
		ctx := context.Background()

		err := svc.AddTimeout(ctx, domain.PlatformTwitch, "alice", 5*time.Second, "Test reason")
		require.NoError(t, err)

		timeout, err := svc.GetTimeoutPlatform(ctx, domain.PlatformTwitch, "alice")
		require.NoError(t, err)
		assert.Greater(t, timeout, time.Duration(0))
		assert.LessOrEqual(t, timeout, 5*time.Second)
	})

	t.Run("Edge Case - Accumulate Timeout", func(t *testing.T) {
		svc := setupTimeoutService()
		ctx := context.Background()

		// First timeout
		err := svc.AddTimeout(ctx, domain.PlatformTwitch, "bob", 2*time.Second, "First")
		require.NoError(t, err)

		// Second timeout should accumulate
		err = svc.AddTimeout(ctx, domain.PlatformTwitch, "bob", 3*time.Second, "Second")
		require.NoError(t, err)

		timeout, err := svc.GetTimeoutPlatform(ctx, domain.PlatformTwitch, "bob")
		require.NoError(t, err)
		// Should be > 2s and <= 5s
		assert.Greater(t, timeout, 2*time.Second)
		assert.LessOrEqual(t, timeout, 5*time.Second)
	})

	t.Run("Invalid Case - Zero Duration", func(t *testing.T) {
		// Zero duration adds a timeout that expires immediately.
		// Testing boundary/invalid logic.
		svc := setupTimeoutService()
		ctx := context.Background()

		err := svc.AddTimeout(ctx, domain.PlatformTwitch, "charlie", 0, "Zero")
		require.NoError(t, err)

		timeout, err := svc.GetTimeoutPlatform(ctx, domain.PlatformTwitch, "charlie")
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), timeout)
	})
}

func TestTimeoutUser(t *testing.T) {
	t.Run("Legacy Replace/Accumulate Behavior", func(t *testing.T) {
		svc := setupTimeoutService()
		ctx := context.Background()

		err := svc.TimeoutUser(ctx, "dave", 2*time.Second, "Legacy reason")
		require.NoError(t, err)

		timeout, err := svc.GetTimeout(ctx, "dave")
		require.NoError(t, err)
		assert.Greater(t, timeout, time.Duration(0))
		assert.LessOrEqual(t, timeout, 2*time.Second)
	})
}

func TestClearTimeout(t *testing.T) {
	t.Run("Best Case - Clear Existing Timeout", func(t *testing.T) {
		svc := setupTimeoutService()
		ctx := context.Background()

		err := svc.AddTimeout(ctx, domain.PlatformTwitch, "eve", 10*time.Second, "To be cleared")
		require.NoError(t, err)

		err = svc.ClearTimeout(ctx, domain.PlatformTwitch, "eve")
		require.NoError(t, err)

		timeout, err := svc.GetTimeoutPlatform(ctx, domain.PlatformTwitch, "eve")
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), timeout)
	})

	t.Run("Invalid Case - Clear Non-existent Timeout", func(t *testing.T) {
		svc := setupTimeoutService()
		ctx := context.Background()

		// Should not error, just return early
		err := svc.ClearTimeout(ctx, domain.PlatformTwitch, "frank")
		require.NoError(t, err)

		timeout, err := svc.GetTimeoutPlatform(ctx, domain.PlatformTwitch, "frank")
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), timeout)
	})
}

func TestGetTimeout(t *testing.T) {
	t.Run("Best Case - Get Active Timeout", func(t *testing.T) {
		svc := setupTimeoutService()
		ctx := context.Background()

		err := svc.AddTimeout(ctx, domain.PlatformDiscord, "grace", 10*time.Second, "Discord timeout")
		require.NoError(t, err)

		timeout, err := svc.GetTimeoutPlatform(ctx, domain.PlatformDiscord, "grace")
		require.NoError(t, err)
		assert.Greater(t, timeout, time.Duration(0))
		assert.LessOrEqual(t, timeout, 10*time.Second)
	})

	t.Run("Edge Case - Cross Platform Isolation", func(t *testing.T) {
		svc := setupTimeoutService()
		ctx := context.Background()

		err := svc.AddTimeout(ctx, domain.PlatformTwitch, "heidi", 10*time.Second, "Twitch timeout")
		require.NoError(t, err)

		// Check Discord platform for same user - should be 0
		timeout, err := svc.GetTimeoutPlatform(ctx, domain.PlatformDiscord, "heidi")
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), timeout)
	})

	t.Run("Boundary Case - No Timeout", func(t *testing.T) {
		svc := setupTimeoutService()
		ctx := context.Background()

		timeout, err := svc.GetTimeoutPlatform(ctx, domain.PlatformTwitch, "ivan")
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), timeout)
	})
}

func TestReduceTimeout(t *testing.T) {
	t.Run("Best Case - Partial Reduction", func(t *testing.T) {
		svc := setupTimeoutService()
		ctx := context.Background()

		err := svc.AddTimeout(ctx, domain.PlatformTwitch, "judy", 10*time.Second, "Full")
		require.NoError(t, err)

		err = svc.ReduceTimeoutPlatform(ctx, domain.PlatformTwitch, "judy", 4*time.Second)
		require.NoError(t, err)

		timeout, err := svc.GetTimeoutPlatform(ctx, domain.PlatformTwitch, "judy")
		require.NoError(t, err)
		assert.Greater(t, timeout, time.Duration(0))
		// Should be roughly 6 seconds left
		assert.LessOrEqual(t, timeout, 6*time.Second)
	})

	t.Run("Boundary Case - Full Reduction", func(t *testing.T) {
		svc := setupTimeoutService()
		ctx := context.Background()

		err := svc.AddTimeout(ctx, domain.PlatformTwitch, "karl", 5*time.Second, "Short")
		require.NoError(t, err)

		// Reduce by more than remaining
		err = svc.ReduceTimeoutPlatform(ctx, domain.PlatformTwitch, "karl", 10*time.Second)
		require.NoError(t, err)

		timeout, err := svc.GetTimeoutPlatform(ctx, domain.PlatformTwitch, "karl")
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), timeout)
	})

	t.Run("Invalid Case - Reduce Non-existent", func(t *testing.T) {
		svc := setupTimeoutService()
		ctx := context.Background()

		// Should not error
		err := svc.ReduceTimeoutPlatform(ctx, domain.PlatformTwitch, "leo", 5*time.Second)
		require.NoError(t, err)

		timeout, err := svc.GetTimeoutPlatform(ctx, domain.PlatformTwitch, "leo")
		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), timeout)
	})

	t.Run("Legacy Wrapper - ReduceTimeout", func(t *testing.T) {
		svc := setupTimeoutService()
		ctx := context.Background()

		err := svc.AddTimeout(ctx, domain.PlatformTwitch, "mia", 10*time.Second, "Full")
		require.NoError(t, err)

		err = svc.ReduceTimeout(ctx, "mia", 5*time.Second)
		require.NoError(t, err)

		timeout, err := svc.GetTimeout(ctx, "mia")
		require.NoError(t, err)
		assert.Greater(t, timeout, time.Duration(0))
		assert.LessOrEqual(t, timeout, 5*time.Second)
	})
}

func TestHandleBlaster_Timeout(t *testing.T) {
	t.Run("Integration Case - Blaster Applies Timeout", func(t *testing.T) {
		svc := setupTimeoutService()
		ctx := context.Background()
		item := domain.ItemBlaster

		// Setup: Give alice a blaster
		err := svc.AddItemByUsername(ctx, domain.PlatformTwitch, "alice", item, 1)
		require.NoError(t, err)

		// Use blaster on bob
		msg, err := svc.UseItem(ctx, domain.PlatformTwitch, "alice123", "alice", item, 1, "bob")
		require.NoError(t, err)

		// Verify message contains timeout info
		assert.Contains(t, msg, "Timed out for")

		// Verify bob actually received a timeout
		timeout, err := svc.GetTimeout(ctx, "bob")
		require.NoError(t, err)
		assert.Greater(t, timeout, time.Duration(0))
	})
}
