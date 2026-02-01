package postgres

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/cooldown"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func TestCooldownService_ConcurrentRequests_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if testDBConnString == "" {
		t.Skip("Skipping integration test: database not available")
	}

	ctx := context.Background()

	// Use shared pool and apply migrations once
	ensureMigrations(t)

	// Create a test user first (cooldowns table has FK to users)
	userID := "550e8400-e29b-41d4-a716-446655440000" // Valid UUID format
	_, err := testPool.Exec(ctx, `
		INSERT INTO users (user_id, username, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`, userID, "test-cooldown-user")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Initialize cooldown service
	cooldownSvc := cooldown.NewPostgresService(testPool, cooldown.Config{
		DevMode: false,
		Cooldowns: map[string]time.Duration{
			domain.ActionSearch: 5 * time.Minute,
		},
	}, nil) // No progression service for this test

	action := domain.ActionSearch

	// Track how many requests successfully execute
	var successCount atomic.Int32
	var wg sync.WaitGroup

	// Fire 10 concurrent requests
	numRequests := 10
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			err := cooldownSvc.EnforceCooldown(ctx, userID, action, func() error {
				// Simulate some work
				time.Sleep(10 * time.Millisecond)
				successCount.Add(1)
				return nil
			})

			// Some will succeed, most will hit cooldown
			if err == nil {
				t.Logf("Request %d: SUCCESS", id)
			} else {
				t.Logf("Request %d: On cooldown (%v)", id, err)
			}
		}(i)
	}

	wg.Wait()

	// CRITICAL: Only ONE request should have succeeded
	// This proves the race condition is fixed
	assert.Equal(t, int32(1), successCount.Load(),
		"Expected exactly 1 successful request, got %d. Race condition not fixed!", successCount.Load())
}
