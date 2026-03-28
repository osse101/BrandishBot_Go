package user

import (
	"context"
	"sync"
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
	svc := NewService(repo, repo, nil, nil, nil, NewMockNamingResolver(), nil, nil, nil, nil, false)
	return svc
}

func TestAddTimeout(t *testing.T) {
	tests := []struct {
		name             string
		initialTimeouts  []time.Duration // Initial timeouts to add sequentially
		newTimeout       time.Duration   // The timeout to add in the test
		expectedMinDur   time.Duration   // Expected minimum duration
		expectedMaxDur   time.Duration   // Expected maximum duration
		expectedDurExact time.Duration   // Expected exact duration (if > 0, overrides min/max)
	}{
		{
			name:           "Best Case - New Timeout",
			newTimeout:     5 * time.Second,
			expectedMinDur: 4 * time.Second, // Account for minor timing differences
			expectedMaxDur: 5 * time.Second,
		},
		{
			name:            "Edge Case - Accumulate Timeout",
			initialTimeouts: []time.Duration{2 * time.Second},
			newTimeout:      3 * time.Second,
			expectedMinDur:  4 * time.Second,
			expectedMaxDur:  5 * time.Second,
		},
		{
			name:             "Invalid Case - Zero Duration",
			newTimeout:       0,
			expectedDurExact: 0,
		},
		{
			name:             "Boundary Case - Negative Duration",
			newTimeout:       -5 * time.Second,
			expectedDurExact: 0,
		},
		{
			name:           "Boundary Case - Large Duration",
			newTimeout:     24 * time.Hour,
			expectedMinDur: 23*time.Hour + 59*time.Minute,
			expectedMaxDur: 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := setupTimeoutService()
			ctx := context.Background()
			username := "test_user"

			// Setup initial timeouts
			for i, dur := range tt.initialTimeouts {
				err := svc.AddTimeout(ctx, domain.PlatformTwitch, username, dur, "Initial setup")
				require.NoError(t, err, "Failed to setup initial timeout %d", i)
			}

			// Add the new timeout
			err := svc.AddTimeout(ctx, domain.PlatformTwitch, username, tt.newTimeout, "Test reason")
			require.NoError(t, err)

			timeout, err := svc.GetTimeoutPlatform(ctx, domain.PlatformTwitch, username)
			require.NoError(t, err)

			if tt.expectedDurExact == 0 && tt.expectedMaxDur > 0 {
				assert.GreaterOrEqual(t, timeout, tt.expectedMinDur)
				assert.LessOrEqual(t, timeout, tt.expectedMaxDur)
			} else {
				assert.Equal(t, tt.expectedDurExact, timeout)
			}
		})
	}
}

func TestAddTimeout_Concurrency(t *testing.T) {
	svc := setupTimeoutService()
	ctx := context.Background()

	// Try to add many timeouts concurrently to check for map panics
	var wg sync.WaitGroup
	concurrency := 50
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			_ = svc.AddTimeout(ctx, domain.PlatformTwitch, "concurrent_user", 100*time.Millisecond, "concurrent")
			_, _ = svc.GetTimeoutPlatform(ctx, domain.PlatformTwitch, "concurrent_user")
			_ = svc.ReduceTimeoutPlatform(ctx, domain.PlatformTwitch, "concurrent_user", 50*time.Millisecond)
		}()
	}

	wg.Wait()

	// Simply surviving without panic proves thread-safety
	timeout, err := svc.GetTimeoutPlatform(ctx, domain.PlatformTwitch, "concurrent_user")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, timeout, time.Duration(0))
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
	tests := []struct {
		name           string
		initialTimeout time.Duration
		userToClear    string
	}{
		{
			name:           "Best Case - Clear Existing Timeout",
			initialTimeout: 10 * time.Second,
			userToClear:    "eve",
		},
		{
			name:           "Invalid Case - Clear Non-existent Timeout",
			initialTimeout: 0,
			userToClear:    "frank",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := setupTimeoutService()
			ctx := context.Background()

			if tt.initialTimeout > 0 {
				err := svc.AddTimeout(ctx, domain.PlatformTwitch, tt.userToClear, tt.initialTimeout, "To be cleared")
				require.NoError(t, err)
			}

			err := svc.ClearTimeout(ctx, domain.PlatformTwitch, tt.userToClear)
			require.NoError(t, err)

			timeout, err := svc.GetTimeoutPlatform(ctx, domain.PlatformTwitch, tt.userToClear)
			require.NoError(t, err)
			assert.Equal(t, time.Duration(0), timeout)
		})
	}
}

func TestClearTimeout_Concurrency(t *testing.T) {
	svc := setupTimeoutService()
	ctx := context.Background()
	username := "clear_concurrent_user"

	err := svc.AddTimeout(ctx, domain.PlatformTwitch, username, 100*time.Hour, "Initial")
	require.NoError(t, err)

	var wg sync.WaitGroup
	concurrency := 50
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			_ = svc.ClearTimeout(ctx, domain.PlatformTwitch, username)
		}()
	}

	wg.Wait()

	// Verify the final timeout is correctly cleared and didn't panic
	timeout, err := svc.GetTimeoutPlatform(ctx, domain.PlatformTwitch, username)
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), timeout)
}

func TestGetTimeout(t *testing.T) {
	tests := []struct {
		name             string
		initialPlatform  string
		initialTimeout   time.Duration
		queryPlatform    string
		queryUser        string
		expectedMinDur   time.Duration
		expectedMaxDur   time.Duration
		expectedDurExact time.Duration
	}{
		{
			name:            "Best Case - Get Active Timeout",
			initialPlatform: domain.PlatformDiscord,
			initialTimeout:  10 * time.Second,
			queryPlatform:   domain.PlatformDiscord,
			queryUser:       "grace",
			expectedMinDur:  9 * time.Second,
			expectedMaxDur:  10 * time.Second,
		},
		{
			name:             "Edge Case - Cross Platform Isolation",
			initialPlatform:  domain.PlatformTwitch,
			initialTimeout:   10 * time.Second,
			queryPlatform:    domain.PlatformDiscord,
			queryUser:        "heidi",
			expectedDurExact: 0,
		},
		{
			name:             "Boundary Case - No Timeout",
			queryPlatform:    domain.PlatformTwitch,
			queryUser:        "ivan",
			expectedDurExact: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := setupTimeoutService()
			ctx := context.Background()

			if tt.initialTimeout > 0 {
				err := svc.AddTimeout(ctx, tt.initialPlatform, tt.queryUser, tt.initialTimeout, "Test timeout")
				require.NoError(t, err)
			}

			timeout, err := svc.GetTimeoutPlatform(ctx, tt.queryPlatform, tt.queryUser)
			require.NoError(t, err)

			if tt.expectedDurExact == 0 && tt.expectedMaxDur > 0 {
				assert.GreaterOrEqual(t, timeout, tt.expectedMinDur)
				assert.LessOrEqual(t, timeout, tt.expectedMaxDur)
			} else {
				assert.Equal(t, tt.expectedDurExact, timeout)
			}
		})
	}
}

func TestReduceTimeout(t *testing.T) {
	tests := []struct {
		name             string
		initialTimeout   time.Duration
		reduction        time.Duration
		expectedMinDur   time.Duration
		expectedMaxDur   time.Duration
		expectedDurExact time.Duration
	}{
		{
			name:           "Best Case - Partial Reduction",
			initialTimeout: 10 * time.Second,
			reduction:      4 * time.Second,
			expectedMinDur: 5 * time.Second, // Account for minor timing differences
			expectedMaxDur: 6 * time.Second,
		},
		{
			name:             "Boundary Case - Full Reduction",
			initialTimeout:   5 * time.Second,
			reduction:        10 * time.Second,
			expectedDurExact: 0,
		},
		{
			name:             "Boundary Case - Exact Reduction",
			initialTimeout:   5 * time.Second,
			reduction:        5 * time.Second,
			expectedDurExact: 0, // Reduces it entirely
		},
		{
			name:           "Boundary Case - Negative Reduction",
			initialTimeout: 5 * time.Second,
			reduction:      -2 * time.Second, // Essentially increases timeout
			expectedMinDur: 6 * time.Second,
			expectedMaxDur: 7 * time.Second,
		},
		{
			name:             "Invalid Case - Reduce Non-existent",
			initialTimeout:   0,
			reduction:        5 * time.Second,
			expectedDurExact: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := setupTimeoutService()
			ctx := context.Background()
			username := "test_user"

			if tt.initialTimeout > 0 {
				err := svc.AddTimeout(ctx, domain.PlatformTwitch, username, tt.initialTimeout, "Initial")
				require.NoError(t, err)
			}

			err := svc.ReduceTimeoutPlatform(ctx, domain.PlatformTwitch, username, tt.reduction)
			require.NoError(t, err)

			timeout, err := svc.GetTimeoutPlatform(ctx, domain.PlatformTwitch, username)
			require.NoError(t, err)

			if tt.expectedDurExact == 0 && tt.expectedMaxDur > 0 {
				assert.GreaterOrEqual(t, timeout, tt.expectedMinDur)
				assert.LessOrEqual(t, timeout, tt.expectedMaxDur)
			} else {
				assert.Equal(t, tt.expectedDurExact, timeout)
			}
		})
	}

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

func TestReduceTimeout_Concurrency(t *testing.T) {
	svc := setupTimeoutService()
	ctx := context.Background()
	username := "reduce_concurrent_user"

	// Give the user a massive timeout so reducing it won't drop it to 0 before we finish
	err := svc.AddTimeout(ctx, domain.PlatformTwitch, username, 100*time.Hour, "Initial")
	require.NoError(t, err)

	var wg sync.WaitGroup
	concurrency := 50
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			_ = svc.ReduceTimeoutPlatform(ctx, domain.PlatformTwitch, username, 1*time.Second)
		}()
	}

	wg.Wait()

	// Verify the final timeout is correctly decremented and didn't panic
	timeout, err := svc.GetTimeoutPlatform(ctx, domain.PlatformTwitch, username)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, timeout, 99*time.Hour)
}

func TestHandleBlaster_Timeout(t *testing.T) {
	t.Run("Integration Case - Blaster Applies Timeout", func(t *testing.T) {
		svc := setupTimeoutService()
		ctx := context.Background()
		item := domain.ItemMissile

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
