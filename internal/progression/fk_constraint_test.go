package progression

import (
	"context"
	"strings"
	"testing"
	"time"

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
	if testDBConnString == "" {
		t.Skip("Skipping integration test: database not available")
	}

	ctx := context.Background()

	// Use shared pool and migrations
	ensureMigrations(t)

	bus := event.NewMemoryBus()
	repo := dbpostgres.NewProgressionRepository(testPool, bus)
	userRepo := dbpostgres.NewUserRepository(testPool)
	svc := NewService(repo, userRepo, bus, nil, nil)

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
		cleanupProgressionState(ctx, t, testPool)

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
		err = testPool.QueryRow(ctx, `
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
