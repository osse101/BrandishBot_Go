package progression

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
	testDBConnString  string
	testPool          *pgxpool.Pool
	migrationsApplied bool
	migrationsMux     sync.Mutex
)

// ensureMigrations applies migrations once for all tests in the package
func ensureMigrations(t *testing.T) {
	migrationsMux.Lock()
	defer migrationsMux.Unlock()

	if migrationsApplied {
		return
	}

	ctx := context.Background()
	if err := applyMigrations(ctx, t, testPool, "../../migrations"); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}

	migrationsApplied = true
}

// applyMigrations applies SQL migrations from the migrations directory
func applyMigrations(ctx context.Context, t *testing.T, pool *pgxpool.Pool, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations dir: %w", err)
	}

	var migrationFiles []string
	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			if (strings.HasSuffix(name, ".up.sql") || strings.HasSuffix(name, ".sql")) && !strings.HasSuffix(name, ".down.sql") {
				migrationFiles = append(migrationFiles, filepath.Join(migrationsDir, name))
			}
		}
	}
	sort.Strings(migrationFiles)

	t.Logf("Applying %d migrations", len(migrationFiles))

	for _, file := range migrationFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		contentStr := string(content)

		// Strip out goose markers
		contentStr = strings.Replace(contentStr, "-- +goose Up\n", "", 1)
		contentStr = strings.Replace(contentStr, "-- +goose Up", "", 1)

		if downIdx := strings.Index(contentStr, "-- +goose Down"); downIdx != -1 {
			contentStr = contentStr[:downIdx]
		}

		contentStr = strings.TrimSpace(contentStr)

		_, err = pool.Exec(ctx, contentStr)
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}
	}
	return nil
}

// cleanupProgressionState cleans up progression state between tests
func cleanupProgressionState(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	// Delete in order to respect FK constraints
	// First, clear FK references
	_, _ = pool.Exec(ctx, "UPDATE progression_voting_sessions SET winning_option_id = NULL WHERE winning_option_id IS NOT NULL")

	queries := []string{
		"DELETE FROM progression_unlock_progress",
		"DELETE FROM progression_voting_options",
		"DELETE FROM progression_voting_sessions",
		"DELETE FROM engagement_metrics",
		"DELETE FROM progression_unlocks",
	}

	for _, query := range queries {
		_, err := pool.Exec(ctx, query)
		if err != nil {
			// Ignore errors for tables that don't exist
			if !strings.Contains(err.Error(), "does not exist") {
				t.Logf("Warning: cleanup query failed: %v", err)
			}
		}
	}
}
