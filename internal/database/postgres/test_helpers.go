package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	testDBConnString   string
	testPool           *pgxpool.Pool
	migrationsApplied  bool
	migrationsMux      sync.Mutex
)

// ensureMigrations applies migrations once for all tests in the package
func ensureMigrations(t *testing.T) {
	migrationsMux.Lock()
	defer migrationsMux.Unlock()

	if migrationsApplied {
		return
	}

	ctx := context.Background()
	if err := applyMigrations(ctx, t, testPool, "../../../migrations"); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}

	migrationsApplied = true
}

// applyMigrations runs all migration files in order
func applyMigrations(ctx context.Context, t *testing.T, pool *pgxpool.Pool, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations dir: %w", err)
	}

	var migrationFiles []string
	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			// Accept both .up.sql and .sql files (exclude .down.sql and archive dir)
			if (strings.HasSuffix(name, ".up.sql") || strings.HasSuffix(name, ".sql")) && !strings.HasSuffix(name, ".down.sql") {
				migrationFiles = append(migrationFiles, filepath.Join(migrationsDir, name))
			}
		}
	}
	sort.Strings(migrationFiles)

	t.Logf("Applying %d migrations in order:", len(migrationFiles))
	for i, file := range migrationFiles {
		t.Logf("  %d. %s", i+1, filepath.Base(file))
	}

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

		t.Logf("Executing: %s", filepath.Base(file))
		_, err = pool.Exec(ctx, contentStr)
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}
	}
	return nil
}
