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

	// Initialize cooldown service
	cooldownSvc := cooldown.NewPostgresService(pool, cooldown.Config{
		DevMode: false,
		Cooldowns: map[string]time.Duration{
			domain.ActionSearch: 5 * time.Minute,
		},
	})

	userID := "test-concurrent-user"
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
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrationFiles []string
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}
		migrationFiles = append(migrationFiles, file.Name())
	}

	// CRITICAL: Sort migration files to ensure correct order
	sort.Strings(migrationFiles)

	for _, filename := range migrationFiles {
		content, err := os.ReadFile(filepath.Join(migrationsDir, filename))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", filename, err)
		}

		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}
	}

	return nil
}
