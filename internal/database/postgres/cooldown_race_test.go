package postgres

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/cooldown"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestCooldownService_RaceCondition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if testDBConnString == "" {
		t.Skip("Skipping integration test: database not available")
	}

	ctx := context.Background()

	// Apply migrations once
	ensureMigrations(t)

	// Create Test User
	userID := "550e8400-e29b-41d4-a716-446655441000"
	_, err := testPool.Exec(ctx, `
		INSERT INTO users (user_id, username, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`, userID, "cd-race-runner")
	require.NoError(t, err)

	// Initialize Service
	svc := cooldown.NewPostgresService(testPool, cooldown.Config{
		DevMode: false,
		Cooldowns: map[string]time.Duration{
			domain.ActionSearch: 5 * time.Minute,
		},
	}, nil) // No progression service needed for this test

	// 5. Run Concurrent Test
	concurrentCalls := 10
	var successfulCalls int32
	var cooldownHits int32
	var failures int32

	var wg sync.WaitGroup
	wg.Add(concurrentCalls)

	// Start gate to synchronize goroutines
	start := make(chan struct{})

	for i := 0; i < concurrentCalls; i++ {
		go func() {
			defer wg.Done()
			<-start // Wait for signal

			err := svc.EnforceCooldown(ctx, userID, domain.ActionSearch, func() error {
				// Simulate some work
				time.Sleep(10 * time.Millisecond)
				return nil
			})

			if err == nil {
				atomic.AddInt32(&successfulCalls, 1)
				t.Log("Call succeeded")
			} else {
				var cdErr cooldown.ErrOnCooldown
				if assert.ErrorAs(t, err, &cdErr) {
					atomic.AddInt32(&cooldownHits, 1)
				} else {
					atomic.AddInt32(&failures, 1)
					t.Logf("Unexpected error: %v", err)
				}
			}
		}()
	}

	// Release the hounds
	close(start)
	wg.Wait()

	// 6. Assertions
	t.Logf("Results: Success=%d, CooldownHits=%d, Failures=%d", successfulCalls, cooldownHits, failures)

	assert.Equal(t, int32(1), successfulCalls, "Exactly one call should succeed")
	assert.Equal(t, int32(concurrentCalls-1), cooldownHits, "All other calls should hit cooldown")
	assert.Equal(t, int32(0), failures, "No unexpected failures should occur")
}
