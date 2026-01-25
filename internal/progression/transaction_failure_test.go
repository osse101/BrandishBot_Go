package progression

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/osse101/BrandishBot_Go/internal/database"
	dbpostgres "github.com/osse101/BrandishBot_Go/internal/database/postgres"
	"github.com/osse101/BrandishBot_Go/internal/event"
)

// TestTransactionFailures tests partial failure scenarios and rollback behavior
// This catches bugs where:
// - CreateSession succeeds, AddOption fails → session orphaned
// - CreateSession succeeds, SetUnlockTarget fails → inconsistent state
func TestTransactionFailures(t *testing.T) {
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

	// Run test scenarios
	t.Run("SetUnlockTargetWithInvalidNode", func(t *testing.T) {
		testSetUnlockTargetInvalidNode(t, ctx, repo, pool)
	})

	t.Run("SetUnlockTargetWithInvalidSession", func(t *testing.T) {
		testSetUnlockTargetInvalidSession(t, ctx, repo, pool)
	})

	t.Run("AddVotingOptionWithInvalidNode", func(t *testing.T) {
		testAddVotingOptionInvalidNode(t, ctx, repo, pool)
	})

	t.Run("ConcurrentSessionCreation", func(t *testing.T) {
		testConcurrentSessionCreation(t, ctx, repo, pool)
	})
}

// testSetUnlockTargetInvalidNode tests what happens when SetUnlockTarget is called
// with a node ID that doesn't exist
func testSetUnlockTargetInvalidNode(t *testing.T, ctx context.Context, repo Repository, pool *pgxpool.Pool) {
	cleanupProgressionState(t, ctx, pool)

	// Create session and progress
	sessionID, err := repo.CreateVotingSession(ctx)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	progressID, err := repo.CreateUnlockProgress(ctx)
	if err != nil {
		t.Fatalf("Failed to create progress: %v", err)
	}

	// Try to set unlock target with invalid node ID
	invalidNodeID := 999999
	err = repo.SetUnlockTarget(ctx, progressID, invalidNodeID, 1, sessionID)

	// Should fail due to FK constraint or validation
	if err == nil {
		t.Error("Expected error when setting unlock target with invalid node ID, got nil")
	}

	// Verify progress was not modified (rollback occurred)
	progress, err := repo.GetActiveUnlockProgress(ctx)
	if err != nil {
		t.Fatalf("Failed to get progress: %v", err)
	}

	if progress != nil && progress.NodeID != nil {
		t.Errorf("Expected progress.NodeID to be nil after failed SetUnlockTarget, got %d", *progress.NodeID)
	}
}

// testSetUnlockTargetInvalidSession tests what happens when SetUnlockTarget is called
// with a session ID that doesn't exist
func testSetUnlockTargetInvalidSession(t *testing.T, ctx context.Context, repo Repository, pool *pgxpool.Pool) {
	cleanupProgressionState(t, ctx, pool)

	// Get a valid node
	nodes, err := repo.GetAllNodes(ctx)
	if err != nil || len(nodes) == 0 {
		t.Fatal("Failed to get nodes")
	}
	node := nodes[0]

	// Create progress
	progressID, err := repo.CreateUnlockProgress(ctx)
	if err != nil {
		t.Fatalf("Failed to create progress: %v", err)
	}

	// Try to set unlock target with invalid session ID
	invalidSessionID := 999999
	err = repo.SetUnlockTarget(ctx, progressID, node.ID, 1, invalidSessionID)

	// Should fail due to FK constraint
	if err == nil {
		t.Error("Expected FK constraint error when setting unlock target with invalid session ID, got nil")
	}

	// Verify progress was not modified (rollback occurred)
	progress, err := repo.GetActiveUnlockProgress(ctx)
	if err != nil {
		t.Fatalf("Failed to get progress: %v", err)
	}

	if progress != nil && progress.NodeID != nil {
		t.Errorf("Expected progress.NodeID to be nil after failed SetUnlockTarget, got %d", *progress.NodeID)
	}
}

// testAddVotingOptionInvalidNode tests what happens when AddVotingOption is called
// with an invalid node ID
func testAddVotingOptionInvalidNode(t *testing.T, ctx context.Context, repo Repository, pool *pgxpool.Pool) {
	cleanupProgressionState(t, ctx, pool)

	// Create session
	sessionID, err := repo.CreateVotingSession(ctx)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Try to add option with invalid node ID
	invalidNodeID := 999999
	err = repo.AddVotingOption(ctx, sessionID, invalidNodeID, 1)

	// Should fail due to FK constraint
	if err == nil {
		t.Error("Expected FK constraint error when adding option with invalid node ID, got nil")
	}

	// Verify session has no options (rollback occurred or FK prevented insertion)
	session, err := repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if session != nil && len(session.Options) > 0 {
		t.Errorf("Expected session to have 0 options after failed AddVotingOption, got %d", len(session.Options))
	}
}

// testConcurrentSessionCreation tests what happens when multiple goroutines
// try to create sessions concurrently
func testConcurrentSessionCreation(t *testing.T, ctx context.Context, repo Repository, pool *pgxpool.Pool) {
	cleanupProgressionState(t, ctx, pool)

	// This tests for race conditions in session creation
	// Multiple goroutines should be able to create sessions without corruption

	sessionIDs := make(chan int, 10)
	errors := make(chan error, 10)

	// Start 10 concurrent session creations
	for i := 0; i < 10; i++ {
		go func() {
			sessionID, err := repo.CreateVotingSession(ctx)
			if err != nil {
				errors <- err
			} else {
				sessionIDs <- sessionID
			}
		}()
	}

	// Collect results
	var createdSessions []int
	var createErrors []error

	for i := 0; i < 10; i++ {
		select {
		case id := <-sessionIDs:
			createdSessions = append(createdSessions, id)
		case err := <-errors:
			createErrors = append(createErrors, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent session creation")
		}
	}

	// All should succeed
	if len(createErrors) > 0 {
		t.Errorf("Expected no errors, got %d errors: %v", len(createErrors), createErrors)
	}

	if len(createdSessions) != 10 {
		t.Errorf("Expected 10 sessions created, got %d", len(createdSessions))
	}

	// All IDs should be unique
	uniqueIDs := make(map[int]bool)
	for _, id := range createdSessions {
		if uniqueIDs[id] {
			t.Errorf("Duplicate session ID detected: %d", id)
		}
		uniqueIDs[id] = true
	}

	// Verify all sessions exist in database
	var count int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM progression_voting_sessions").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count sessions: %v", err)
	}

	if count != 10 {
		t.Errorf("Expected 10 sessions in database, got %d", count)
	}
}
