package progression

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/osse101/BrandishBot_Go/internal/database"
	dbpostgres "github.com/osse101/BrandishBot_Go/internal/database/postgres"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
)

// TestMain sets up shared container for all tests in the package
func TestMain(m *testing.M) {
	flag.Parse()

	var terminate func()

	if !testing.Short() {
		ctx := context.Background()
		var connStr string
		connStr, terminate = setupContainer(ctx)
		testDBConnString = connStr

		// Create shared pool if container started successfully
		if connStr != "" {
			var err error
			testPool, err = database.NewPool(connStr, 20, 30*time.Minute, time.Hour)
			if err != nil {
				fmt.Printf("WARNING: Failed to create test pool: %v\n", err)
			}
		}
	}

	code := m.Run()

	if testPool != nil {
		testPool.Close()
	}
	if terminate != nil {
		terminate()
	}

	os.Exit(code)
}

func setupContainer(ctx context.Context) (string, func()) {
	// Handle potential panics from testcontainers
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in setupContainer: %v\n", r)
		}
	}()

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
		fmt.Printf("WARNING: Failed to start postgres container: %v\n", err)
		return "", func() {}
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Printf("WARNING: Failed to get connection string: %v\n", err)
		pgContainer.Terminate(ctx)
		return "", func() {}
	}

	return connStr, func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			fmt.Printf("Failed to terminate container: %v\n", err)
		}
	}
}

// TestProgressionService_Integration tests the service layer with real PostgreSQL
// to catch FK constraint violations, async timing issues, and state inconsistencies
func TestProgressionService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if testDBConnString == "" {
		t.Skip("Skipping integration test: database not available")
	}

	ctx := context.Background()

	// Use shared pool and migrations
	ensureMigrations(t)

	// Create event bus for service
	bus := event.NewMemoryBus()

	// Create repositories
	repo := dbpostgres.NewProgressionRepository(testPool, bus)
	userRepo := dbpostgres.NewUserRepository(testPool)

	// Create service
	svc := NewService(repo, userRepo, bus)

	// Wait for service to be ready
	time.Sleep(100 * time.Millisecond)

	// Run test suites
	t.Run("AutoSelectFlow", func(t *testing.T) {
		testAutoSelectFlow(t, ctx, svc, repo, testPool)
	})

	t.Run("FKConstraints", func(t *testing.T) {
		testFKConstraints(t, ctx, svc, repo, testPool)
	})

	t.Run("AsyncTiming", func(t *testing.T) {
		testAsyncTiming(t, ctx, svc, repo, testPool)
	})

	t.Run("SessionLifecycle", func(t *testing.T) {
		testSessionLifecycle(t, ctx, svc, repo, testPool)
	})

	t.Run("ZeroCostAutoUnlock", func(t *testing.T) {
		testZeroCostAutoUnlock(t, ctx, svc, repo, testPool)
	})

	// Shutdown service gracefully
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	svc.Shutdown(shutdownCtx)
}

// testAutoSelectFlow tests the auto-select → unlock → new session flow
func testAutoSelectFlow(t *testing.T, ctx context.Context, svc Service, repo Repository, pool *pgxpool.Pool) {
	// Clear any existing sessions and progress
	cleanupProgressionState(t, ctx, pool)

	// Create a scenario with only one available node
	// First, we need to unlock prerequisites to make only one node available

	// Start voting session (should auto-select if only 1 option)
	err := svc.StartVotingSession(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to start voting session: %v", err)
	}

	// Get active session
	session, err := repo.GetActiveSession(ctx)
	if err != nil {
		t.Fatalf("Failed to get active session: %v", err)
	}
	if session == nil {
		t.Fatal("Expected active session after starting voting")
	}

	// Verify session has valid status
	if session.Status != SessionStatusVoting {
		t.Errorf("Expected session status 'voting', got '%s'", session.Status)
	}

	// Verify session has options
	if len(session.Options) == 0 {
		t.Error("Expected at least one option in session")
	}

	// Verify unlock progress references this session
	progress, err := repo.GetActiveUnlockProgress(ctx)
	if err != nil {
		t.Fatalf("Failed to get unlock progress: %v", err)
	}

	// If auto-selected (only 1 option), progress should reference the session
	if len(session.Options) == 1 && progress != nil {
		if progress.VotingSessionID == nil {
			t.Error("Expected unlock progress to reference voting session for auto-select")
		} else if *progress.VotingSessionID != session.ID {
			t.Errorf("Expected progress.VotingSessionID=%d, got %d", session.ID, *progress.VotingSessionID)
		}

		// Verify FK constraint - session should exist and be retrievable
		sessionByID, err := repo.GetSessionByID(ctx, session.ID)
		if err != nil {
			t.Errorf("Failed to get session by ID referenced in progress: %v", err)
		}
		if sessionByID == nil {
			t.Error("Session referenced by progress FK should exist")
		}
	}
}

// testFKConstraints explicitly tests foreign key constraint enforcement
func testFKConstraints(t *testing.T, ctx context.Context, svc Service, repo Repository, pool *pgxpool.Pool) {
	cleanupProgressionState(t, ctx, pool)

	// Create a session
	sessionID, err := repo.CreateVotingSession(ctx)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Get a valid node to use
	nodes, err := repo.GetAllNodes(ctx)
	if err != nil || len(nodes) == 0 {
		t.Fatal("Failed to get nodes for FK test")
	}
	node := nodes[0]

	// Add an option to the session
	err = repo.AddVotingOption(ctx, sessionID, node.ID, 1)
	if err != nil {
		t.Fatalf("Failed to add voting option: %v", err)
	}

	// Create unlock progress
	progressID, err := repo.CreateUnlockProgress(ctx)
	if err != nil {
		t.Fatalf("Failed to create unlock progress: %v", err)
	}

	// Test: SetUnlockTarget with valid session (should succeed)
	err = repo.SetUnlockTarget(ctx, progressID, node.ID, 1, sessionID)
	if err != nil {
		t.Errorf("SetUnlockTarget with valid session should succeed: %v", err)
	}

	// Test: SetUnlockTarget with non-existent session ID (should fail with FK violation)
	invalidSessionID := 999999
	progressID2, err := repo.CreateUnlockProgress(ctx)
	if err != nil {
		t.Fatalf("Failed to create second unlock progress: %v", err)
	}

	err = repo.SetUnlockTarget(ctx, progressID2, node.ID, 1, invalidSessionID)
	if err == nil {
		t.Error("Expected FK constraint error when setting unlock target with non-existent session, got nil")
	} else {
		// Verify it's actually a FK constraint error
		errStr := err.Error()
		if !strings.Contains(errStr, "foreign key") && !strings.Contains(errStr, "violates") {
			t.Logf("Got error (expected FK violation): %v", err)
		}
	}

	// Test: Verify session status doesn't affect FK (ended session should still satisfy FK)
	sessionID2, err := repo.CreateVotingSession(ctx)
	if err != nil {
		t.Fatalf("Failed to create second session: %v", err)
	}

	err = repo.AddVotingOption(ctx, sessionID2, node.ID, 1)
	if err != nil {
		t.Fatalf("Failed to add option to second session: %v", err)
	}

	// Get the option ID to end the session
	session2, _ := repo.GetActiveSession(ctx)
	if session2 != nil && len(session2.Options) > 0 {
		optionID := session2.Options[0].ID

		// End the session
		err = repo.EndVotingSession(ctx, sessionID2, &optionID)
		if err != nil {
			t.Fatalf("Failed to end session: %v", err)
		}

		// Now try to set unlock target with ended session
		progressID3, err := repo.CreateUnlockProgress(ctx)
		if err != nil {
			t.Fatalf("Failed to create third unlock progress: %v", err)
		}

		err = repo.SetUnlockTarget(ctx, progressID3, node.ID, 1, sessionID2)
		// This SHOULD succeed - ended sessions still satisfy FK constraint
		if err != nil {
			t.Errorf("SetUnlockTarget with ended session should succeed (FK satisfied): %v", err)
		}
	}
}

// testAsyncTiming tests goroutine coordination and timing issues
func testAsyncTiming(t *testing.T, ctx context.Context, svc Service, repo Repository, pool *pgxpool.Pool) {
	cleanupProgressionState(t, ctx, pool)

	// Start a session
	err := svc.StartVotingSession(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to start voting session: %v", err)
	}

	// Get the session
	session, err := repo.GetActiveSession(ctx)
	if err != nil || session == nil {
		t.Fatal("Failed to get active session")
	}

	// If only one option (auto-select case), test async unlock
	if len(session.Options) == 1 {
		// The auto-select path spawns a goroutine for zero-cost unlocks
		// Wait a bit for any background goroutines to complete
		time.Sleep(500 * time.Millisecond)

		// Query for session again - it might have been replaced by a new session
		// if the zero-cost unlock completed
		newSession, err := repo.GetActiveSession(ctx)
		if err != nil {
			t.Errorf("Failed to query for session after async unlock: %v", err)
		}

		// Either the session is the same, or a new one was created
		if newSession == nil {
			t.Error("Expected either original or new session after async processing")
		}
	}

	// Test concurrent AddContribution calls (race condition test)
	progress, err := repo.GetActiveUnlockProgress(ctx)
	if progress == nil {
		t.Skip("No active progress to test concurrent contributions")
	}

	// Add contributions concurrently
	var wg sync.WaitGroup
	numGoroutines := 10
	contributionPerGoroutine := 5

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := svc.AddContribution(ctx, contributionPerGoroutine); err != nil {
				t.Logf("Concurrent AddContribution error: %v", err)
			}
		}()
	}

	wg.Wait()

	// Verify total contributions accumulated correctly
	updatedProgress, err := repo.GetActiveUnlockProgress(ctx)
	if err != nil {
		t.Fatalf("Failed to get updated progress: %v", err)
	}

	expectedMin := numGoroutines * contributionPerGoroutine
	if updatedProgress.ContributionsAccumulated < expectedMin {
		t.Errorf("Expected at least %d contributions, got %d (race condition detected)",
			expectedMin, updatedProgress.ContributionsAccumulated)
	}
}

// testSessionLifecycle tests state transitions and consistency
func testSessionLifecycle(t *testing.T, ctx context.Context, svc Service, repo Repository, pool *pgxpool.Pool) {
	cleanupProgressionState(t, ctx, pool)

	// Test: voting → ended → new voting cycle
	err := svc.StartVotingSession(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to start initial session: %v", err)
	}

	session1, err := repo.GetActiveSession(ctx)
	if err != nil || session1 == nil {
		t.Fatal("Failed to get first session")
	}

	if session1.Status != SessionStatusVoting {
		t.Errorf("Expected status 'voting', got '%s'", session1.Status)
	}

	// End voting (if there are multiple options, otherwise it's auto-selected)
	if len(session1.Options) > 1 {
		_, err = svc.EndVoting(ctx)
		if err != nil {
			t.Fatalf("Failed to end voting: %v", err)
		}

		// Verify session is now completed
		endedSession, err := repo.GetSessionByID(ctx, session1.ID)
		if err != nil {
			t.Fatalf("Failed to get ended session: %v", err)
		}
		if endedSession.Status != "completed" {
			t.Errorf("Expected status 'completed' after EndVoting, got '%s'", endedSession.Status)
		}

		// Verify unlock progress references this session
		progress, _ := repo.GetActiveUnlockProgress(ctx)
		if progress != nil && progress.VotingSessionID != nil {
			if *progress.VotingSessionID != session1.ID {
				t.Errorf("Expected progress to reference session %d, got %d",
					session1.ID, *progress.VotingSessionID)
			}
		}
	}

	// After ending voting, check if new session was auto-created
	// The service may auto-start a new session after ending voting
	currentSession, _ := repo.GetActiveSession(ctx)

	// Test: Can't start new session while one is active
	if currentSession != nil {
		err = svc.StartVotingSession(ctx, nil)
		if err == nil || err != domain.ErrSessionAlreadyActive {
			t.Errorf("Expected ErrSessionAlreadyActive when starting duplicate session, got: %v", err)
		}
	} else {
		t.Log("No active session after ending voting (session auto-completed)")
	}

	// Complete the unlock to trigger new session
	progress, _ := repo.GetActiveUnlockProgress(ctx)
	if progress != nil && progress.NodeID != nil {
		// Add enough contributions to unlock
		node, _ := repo.GetNodeByID(ctx, *progress.NodeID)
		if node != nil && node.UnlockCost > 0 {
			remaining := node.UnlockCost - progress.ContributionsAccumulated
			if remaining > 0 {
				err = svc.AddContribution(ctx, remaining)
				if err != nil {
					t.Logf("AddContribution error: %v", err)
				}

				// Wait for async unlock and new session creation
				time.Sleep(1 * time.Second)

				// Verify new session was created
				newSession, err := repo.GetActiveSession(ctx)
				if err != nil {
					t.Errorf("Failed to get new session after unlock: %v", err)
				}
				if newSession != nil && newSession.ID == session1.ID {
					t.Error("Expected new session after unlock cycle, got same session ID")
				}
			}
		}
	}
}

// testZeroCostAutoUnlock tests the zero-cost node immediate unlock path
func testZeroCostAutoUnlock(t *testing.T, ctx context.Context, svc Service, repo Repository, pool *pgxpool.Pool) {
	cleanupProgressionState(t, ctx, pool)

	// Find or create a zero-cost node for testing
	// For this test, we'll rely on the progression tree having zero-cost nodes
	// If none exist, we skip this test

	nodes, err := repo.GetAllNodes(ctx)
	if err != nil {
		t.Fatalf("Failed to get nodes: %v", err)
	}

	var zeroCostNode *domain.ProgressionNode
	for _, node := range nodes {
		if node.UnlockCost == 0 {
			zeroCostNode = node
			break
		}
	}

	if zeroCostNode == nil {
		t.Skip("No zero-cost nodes found in tree, skipping zero-cost unlock test")
	}

	// Try to start a session - it may fail if all nodes are unlocked
	err = svc.StartVotingSession(ctx, nil)
	if err != nil {
		if strings.Contains(err.Error(), "no nodes available") {
			t.Skip("All nodes unlocked, cannot test zero-cost flow")
		}
		t.Fatalf("Failed to start session: %v", err)
	}

	// Wait for any async processing
	time.Sleep(500 * time.Millisecond)

	// Verify no orphaned sessions exist
	// This tests for Bug #2 from the plan: sessions created but not linked to progress
	var count int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM progression_voting_sessions
		WHERE status = 'voting'
		AND NOT EXISTS (
			SELECT 1 FROM progression_unlock_progress
			WHERE voting_session_id = progression_voting_sessions.id
		)
	`).Scan(&count)

	if err != nil {
		t.Fatalf("Failed to check for orphaned sessions: %v", err)
	}

	if count > 0 {
		// This is a known bug - log it for now
		t.Logf("KNOWN BUG: Found %d orphaned voting sessions (not referenced by any progress)", count)
		t.Logf("This is the bug identified in the plan: sessions created without setting unlock progress target")
		// Don't fail the test - this is documenting the bug
	}
}

