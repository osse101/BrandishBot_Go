package postgres

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/cooldown"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// mockProgressionService implements cooldown.ProgressionService for testing modifiers
type mockProgressionService struct {
	modifiers map[string]float64
}

func (m *mockProgressionService) GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error) {
	if mod, ok := m.modifiers[featureKey]; ok {
		// Modifier logic: usually subtracts from baseValue
		// e.g. base 300s, mod 30s -> return 270s
		return baseValue - mod, nil
	}
	return baseValue, nil
}

func TestCooldownService_StandardLifecycle_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if testDBConnString == "" {
		t.Skip("Skipping integration test: database not available")
	}

	ctx := context.Background()

	// Use shared pool and migrations
	ensureMigrations(t)

	// Create test user
	userID := fmt.Sprintf("550e8400-e29b-41d4-a716-44665544%04d", rand.Intn(9999))
	_, err := testPool.Exec(ctx, `
		INSERT INTO users (user_id, username, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`, userID, "cooldown-lifecycle-user")
	require.NoError(t, err)

	t.Run("Basic Lifecycle", func(t *testing.T) {
		svc := cooldown.NewPostgresService(testPool, cooldown.Config{
			DevMode: false,
			Cooldowns: map[string]time.Duration{
				domain.ActionSearch: 5 * time.Minute,
			},
		}, nil)

		action := domain.ActionSearch

		// 1. Check - should not be on cooldown
		onCooldown, remaining, err := svc.CheckCooldown(ctx, userID, action)
		require.NoError(t, err)
		assert.False(t, onCooldown)
		assert.Zero(t, remaining)

		// 2. Enforce - should succeed
		err = svc.EnforceCooldown(ctx, userID, action, func() error {
			return nil
		})
		require.NoError(t, err)

		// 3. Check again - should be on cooldown now
		onCooldown, remaining, err = svc.CheckCooldown(ctx, userID, action)
		require.NoError(t, err)
		assert.True(t, onCooldown)
		assert.Greater(t, remaining, 4*time.Minute) // Should be close to 5m

		// 4. Enforce again - should fail
		err = svc.EnforceCooldown(ctx, userID, action, func() error {
			return nil
		})
		require.Error(t, err)
		var cooldownErr cooldown.ErrOnCooldown
		assert.ErrorAs(t, err, &cooldownErr)

		// 5. GetLastUsed
		lastUsed, err := svc.GetLastUsed(ctx, userID, action)
		require.NoError(t, err)
		require.NotNil(t, lastUsed)
		assert.WithinDuration(t, time.Now(), *lastUsed, 5*time.Second)

		// 6. Reset
		err = svc.ResetCooldown(ctx, userID, action)
		require.NoError(t, err)

		// 7. Check after reset - should be clear
		onCooldown, _, err = svc.CheckCooldown(ctx, userID, action)
		require.NoError(t, err)
		assert.False(t, onCooldown)
	})

	t.Run("Progression Modifiers", func(t *testing.T) {
		// Mock progression to reduce cooldown by 60 seconds (1 minute)
		mockProg := &mockProgressionService{
			modifiers: map[string]float64{
				"search_cooldown_reduction": 60 * 1e9, // 60 seconds in nanoseconds (float64 duration)
			},
		}

		svc := cooldown.NewPostgresService(testPool, cooldown.Config{
			DevMode: false,
			Cooldowns: map[string]time.Duration{
				domain.ActionSearch: 5 * time.Minute,
			},
		}, mockProg)

		action := domain.ActionSearch
		// Use a different user or action to avoid collision with previous test, or just reset
		// Reuse same user, reset already happened

		// Enforce cooldown
		err = svc.EnforceCooldown(ctx, userID, action, func() error {
			return nil
		})
		require.NoError(t, err)

		// Check cooldown - should utilize modifier
		// Base: 5m, Modifier: -1m => Effective: 4m
		onCooldown, remaining, err := svc.CheckCooldown(ctx, userID, action)
		require.NoError(t, err)
		assert.True(t, onCooldown)

		// Remaining should be <= 4m (not 5m)
		assert.LessOrEqual(t, remaining, 4*time.Minute)
		assert.Greater(t, remaining, 3*time.Minute)
	})
}
