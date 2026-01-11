package postgres_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/cooldown"
	"github.com/osse101/BrandishBot_Go/internal/database"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestCooldownService_RaceCondition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	// 1. Setup Postgres Container
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
	require.NoError(t, err)
	defer func() {
		_ = pgContainer.Terminate(ctx)
	}()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := database.NewPool(connStr, 20, 30*time.Minute, time.Hour) // MaxConns=20 for concurrency
	require.NoError(t, err)
	defer pool.Close()

	// 2. Apply Migrations
	// Assuming migrations are in the standard location relative to this file
	require.NoError(t, applyMigrations(ctx, pool, "../../../migrations"))

	// 3. Create Test User
	userID := "550e8400-e29b-41d4-a716-446655440000"
	_, err = pool.Exec(ctx, `
		INSERT INTO users (user_id, username, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`, userID, "race-runner")
	require.NoError(t, err)

	// 4. Initialize Service
	svc := cooldown.NewPostgresService(pool, cooldown.Config{
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
