package postgres_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/cooldown"
	"github.com/osse101/BrandishBot_Go/internal/database"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestCooldownService_ConcurrentRequests_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	// Start Postgres container (inline setup like other integration tests)
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
		t.Fatalf("failed to start postgres container: %v", err)
	}
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := database.NewPool(connStr, 100, 30*time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Apply migrations
	if err := applyMigrations(ctx, pool, "../../../migrations"); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}

	// Create a test user first (cooldowns table has FK to users)
	userID := "550e8400-e29b-41d4-a716-446655440000" // Valid UUID format
	_, err = pool.Exec(ctx, `
		INSERT INTO users (user_id, username, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
	`, userID, "test-cooldown-user")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Initialize cooldown service
	cooldownSvc := cooldown.NewPostgresService(pool, cooldown.Config{
		DevMode: false,
		Cooldowns: map[string]time.Duration{
			domain.ActionSearch: 5 * time.Minute,
		},
	})

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

// applyMigrations runs all .sql migration files in the given directory
func applyMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations dir: %w", err)
	}

	var migrationFiles []string
	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			// Accept both .up.sql and .sql files (exclude .down.sql)
			if (strings.HasSuffix(name, ".up.sql") || strings.HasSuffix(name, ".sql")) && !strings.HasSuffix(name, ".down.sql") {
				migrationFiles = append(migrationFiles, filepath.Join(migrationsDir, name))
			}
		}
	}
	sort.Strings(migrationFiles)

	for _, file := range migrationFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		contentStr := string(content)

		// Strip out goose markers (for goose v3 compatibility)
		// Remove "-- +goose Up" from the beginning
		contentStr = strings.Replace(contentStr, "-- +goose Up\n", "", 1)
		// Remove "-- +goose Up" without newline
		contentStr = strings.Replace(contentStr, "-- +goose Up", "", 1)

		// Strip out the "Down" section if it exists (goose-style migrations)
		if downIdx := strings.Index(contentStr, "-- +goose Down"); downIdx != -1 {
			contentStr = contentStr[:downIdx]
		}

		// Trim any leading/trailing whitespace
		contentStr = strings.TrimSpace(contentStr)

		_, err = pool.Exec(ctx, contentStr)
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}
	}
	return nil
}
