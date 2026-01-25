package progression

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/osse101/BrandishBot_Go/internal/database"
	dbpostgres "github.com/osse101/BrandishBot_Go/internal/database/postgres"
	"github.com/osse101/BrandishBot_Go/internal/event"
)

// TestAutoSelectFKConstraintBug specifically tests the bug from the plan:
// "insert or update on table progression_unlock_progress violates
// foreign key constraint progression_unlock_progress_voting_session_id_fkey"
//
// This bug occurs when:
// 1. Auto-select creates a voting session
// 2. Session is immediately marked as "ended"
// 3. SetUnlockTarget tries to reference it via FK
func TestAutoSelectFKConstraintBug(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start Postgres container
	var pgContainer *postgres.PostgresContainer
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Skipf("Skipping integration test due to panic (likely Docker issue): %v", r)
			}
		}()
		pgContainer, err = postgres.Run(ctx,
			"postgres:15-alpine",
			postgres.WithDatabase("testdb"),
			postgres.WithUsername("testuser"),
			postgres.WithPassword("testpass"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(5*time.Second)),
		)
	}()

	if pgContainer == nil {
		if err != nil {
			t.Fatalf("failed to start postgres container: %v", err)
		}
		return
	}
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %v", err)
		}
	}()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := database.NewPool(connStr, 10, 30*time.Minute, time.Hour)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := applyMigrations(ctx, t, pool, "../../migrations"); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}

	bus := event.NewMemoryBus()
	repo := dbpostgres.NewProgressionRepository(pool, bus)
	userRepo := dbpostgres.NewUserRepository(pool)
	svc := NewService(repo, userRepo, bus)

	time.Sleep(100 * time.Millisecond)

	// Test 1: Auto-select with single option should NOT violate FK constraint
	t.Run("AutoSelectSingleOption", func(t *testing.T) {
		// Start voting - if only one option available, it auto-selects
		err := svc.StartVotingSession(ctx, nil)
		if err != nil {
			// Check if it's an FK constraint violation
			if strings.Contains(err.Error(), "foreign key") || strings.Contains(err.Error(), "violates") {
				t.Fatalf("FK constraint violation during auto-select: %v", err)
			}
			// Other errors might be expected (e.g., no nodes available)
			t.Logf("StartVotingSession returned: %v", err)
		}

		// Verify session was created and linked to progress
		session, err := repo.GetActiveSession(ctx)
		if err != nil {
			t.Fatalf("Failed to get active session: %v", err)
		}

		if session != nil {
			// Verify progress references this session
			progress, err := repo.GetActiveUnlockProgress(ctx)
			if err != nil {
				t.Fatalf("Failed to get unlock progress: %v", err)
			}

			if progress != nil && progress.VotingSessionID != nil {
				if *progress.VotingSessionID != session.ID {
					t.Errorf("Progress references session %d but active session is %d",
						*progress.VotingSessionID, session.ID)
				}

				// Verify the session actually exists (FK constraint would prevent this if violated)
				sessionByID, err := repo.GetSessionByID(ctx, *progress.VotingSessionID)
				if err != nil {
					t.Errorf("FK constraint violation: progress references non-existent session: %v", err)
				}
				if sessionByID == nil {
					t.Error("FK constraint violation: session referenced by progress does not exist")
				}
			}
		}
	})

	// Test 2: Zero-cost auto-unlock should not violate FK constraints
	t.Run("ZeroCostAutoUnlock", func(t *testing.T) {
		cleanupProgressionState(t, ctx, pool)

		// This tests the scenario where:
		// 1. Auto-select creates session
		// 2. Zero-cost node triggers immediate unlock
		// 3. Unlock ends the session
		// 4. New session is started
		// All of this should happen without FK violations

		err := svc.StartVotingSession(ctx, nil)
		if err != nil && !strings.Contains(err.Error(), "no nodes available") {
			if strings.Contains(err.Error(), "foreign key") || strings.Contains(err.Error(), "violates") {
				t.Fatalf("FK constraint violation: %v", err)
			}
		}

		// Wait for async operations
		time.Sleep(1 * time.Second)

		// Query for any FK constraint violations in the database
		var violationCount int
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM progression_unlock_progress
			WHERE voting_session_id IS NOT NULL
			AND NOT EXISTS (
				SELECT 1 FROM progression_voting_sessions
				WHERE id = progression_unlock_progress.voting_session_id
			)
		`).Scan(&violationCount)

		if err != nil {
			t.Fatalf("Failed to check for FK violations: %v", err)
		}

		if violationCount > 0 {
			t.Errorf("Found %d FK constraint violations: unlock_progress references non-existent sessions",
				violationCount)
		}
	})

	// Shutdown service
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	svc.Shutdown(shutdownCtx)
}
