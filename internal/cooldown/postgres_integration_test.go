//go:build integration

package cooldown

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/osse101/BrandishBot_Go/internal/database"
)

// TestMain sets up shared container for all tests in the package
func TestMain(m *testing.M) {
	flag.Parse()

	var terminate func()

	if !testing.Short() {
		if conn := os.Getenv("TEST_DB_CONN"); conn != "" {
			testDBConnString = conn
			var err error
			testPool, err = database.NewPool(conn, 20, 30*time.Minute, time.Hour)
			if err != nil {
				fmt.Printf("WARNING: Failed to create test pool from env: %v\n", err)
			}
		} else {
			ctx := context.Background()
			var connStr string
			connStr, terminate = setupContainer(ctx)
			testDBConnString = connStr

			// Create shared pool if container started successfully
			if connStr != "" {
				var err error
				testPool, err = database.NewPool(connStr, 20, 30*time.Minute, time.Hour)
				if err != nil {
					fmt.Printf("WARNING: Failed to create test pool: %v\n", err)
				}
			}
		}
	}

	code := m.Run()

	if testPool != nil {
		testPool.Close()
	}
	if terminate != nil {
		terminate()
	}

	os.Exit(code)
}

func setupContainer(ctx context.Context) (string, func()) {
	// Handle potential panics from testcontainers
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in setupContainer: %v\n", r)
		}
	}()

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(15*time.Second)),
	)
	if err != nil {
		fmt.Printf("WARNING: Failed to start postgres container: %v\n", err)
		return "", func() {}
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Printf("WARNING: Failed to get connection string: %v\n", err)
		pgContainer.Terminate(ctx)
		return "", func() {}
	}

	return connStr, func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			fmt.Printf("Failed to terminate container: %v\n", err)
		}
	}
}

type mockProgressionServiceIntegration struct{}

func (m *mockProgressionServiceIntegration) GetModifiedValue(ctx context.Context, userID string, modifier string, baseValue float64) (float64, error) {
	return baseValue, nil
}
func (m *mockProgressionServiceIntegration) IsFeatureUnlocked(ctx context.Context, featureName string) (bool, error) {
	return true, nil
}
func (m *mockProgressionServiceIntegration) IsItemUnlocked(ctx context.Context, itemName string) (bool, error) {
	return true, nil
}
func (m *mockProgressionServiceIntegration) RecordEngagement(ctx context.Context, userID, metricType string, value int) error {
	return nil
}
func (m *mockProgressionServiceIntegration) GetNodeStatus(ctx context.Context, nodeKey string) (string, error) {
	return "unlocked", nil
}
func (m *mockProgressionServiceIntegration) IsNodeUnlocked(ctx context.Context, nodeKey string, level int) (bool, error) {
	return true, nil
}
func (m *mockProgressionServiceIntegration) ResetProgressionTree(ctx context.Context, resetBy string, reason string, preserveUserData bool) error {
	return nil
}
func (m *mockProgressionServiceIntegration) StartVotingSession(ctx context.Context, unlockedNodeID *int) error {
	return nil
}
func (m *mockProgressionServiceIntegration) VoteForUnlock(ctx context.Context, platform string, platformID string, username string, optionIndex int) error {
	return nil
}
func (m *mockProgressionServiceIntegration) Shutdown(ctx context.Context) error {
	return nil
}

func TestPostgresBackend_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if testDBConnString == "" {
		t.Skip("Skipping integration test: database not available")
	}

	ctx := context.Background()
	ensureMigrations(t)

	// Create a test progression mock
	mockProgression := &mockProgressionServiceIntegration{}

	config := Config{
		DevMode: false,
		Cooldowns: map[string]time.Duration{
			"test_action": 100 * time.Millisecond,
			"slow_action": 1 * time.Second,
		},
	}

	svc := NewPostgresService(testPool, config, mockProgression)

	t.Run("EnforceCooldown CheckThenLock", func(t *testing.T) {
		userID := fmt.Sprintf("test_user_enforce_%d", time.Now().UnixNano())
		action := "test_action"

		// Should succeed first time
		err := svc.EnforceCooldown(ctx, userID, action, func() error {
			return nil
		})
		require.NoError(t, err)

		// Should fail immediately after
		err = svc.EnforceCooldown(ctx, userID, action, func() error {
			return nil
		})
		require.Error(t, err)
		var cooldownErr ErrOnCooldown
		assert.True(t, errors.As(err, &cooldownErr))

		// Wait for cooldown to expire
		time.Sleep(150 * time.Millisecond)

		// Should succeed again
		err = svc.EnforceCooldown(ctx, userID, action, func() error {
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("EnforceCooldown Concurrency", func(t *testing.T) {
		userID := fmt.Sprintf("test_user_concurrent_%d", time.Now().UnixNano())
		action := "slow_action"

		var successCount int32
		var wg sync.WaitGroup
		concurrency := 10

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := svc.EnforceCooldown(ctx, userID, action, func() error {
					// Simulate some work
					time.Sleep(50 * time.Millisecond)
					return nil
				})
				if err == nil {
					atomic.AddInt32(&successCount, 1)
				}
			}()
		}

		wg.Wait()

		// Only one should have succeeded due to check-then-lock with advisory lock
		assert.Equal(t, int32(1), successCount)
	})

	t.Run("ResetCooldown", func(t *testing.T) {
		userID := fmt.Sprintf("test_user_reset_%d", time.Now().UnixNano())
		action := "slow_action"

		// Use the action
		err := svc.EnforceCooldown(ctx, userID, action, func() error { return nil })
		require.NoError(t, err)

		// Verify on cooldown
		onCooldown, _, err := svc.CheckCooldown(ctx, userID, action)
		require.NoError(t, err)
		assert.True(t, onCooldown)

		// Reset it
		err = svc.ResetCooldown(ctx, userID, action)
		require.NoError(t, err)

		// Verify no longer on cooldown
		onCooldown, _, err = svc.CheckCooldown(ctx, userID, action)
		require.NoError(t, err)
		assert.False(t, onCooldown)
	})

	t.Run("Function Failure Does Not Apply Cooldown", func(t *testing.T) {
		userID := fmt.Sprintf("test_user_fail_%d", time.Now().UnixNano())
		action := "slow_action"
		expectedErr := errors.New("simulated failure")

		// Fail the action
		err := svc.EnforceCooldown(ctx, userID, action, func() error {
			return expectedErr
		})
		require.ErrorIs(t, err, expectedErr)

		// Check if on cooldown (should not be)
		onCooldown, _, err := svc.CheckCooldown(ctx, userID, action)
		require.NoError(t, err)
		assert.False(t, onCooldown)
	})

	t.Run("GetLastUsed", func(t *testing.T) {
		userID := fmt.Sprintf("test_user_lastused_%d", time.Now().UnixNano())
		action := "test_action"

		// Never used
		lastUsed, err := svc.GetLastUsed(ctx, userID, action)
		require.NoError(t, err)
		assert.Nil(t, lastUsed)

		// Use it
		err = svc.EnforceCooldown(ctx, userID, action, func() error { return nil })
		require.NoError(t, err)

		// Get it
		lastUsed, err = svc.GetLastUsed(ctx, userID, action)
		require.NoError(t, err)
		require.NotNil(t, lastUsed)
		assert.WithinDuration(t, time.Now(), *lastUsed, 2*time.Second)
	})
}
