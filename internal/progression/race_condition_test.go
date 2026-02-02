package progression

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	dbpostgres "github.com/osse101/BrandishBot_Go/internal/database/postgres"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
)

// TestRaceConditions tests for race conditions in concurrent operations
// Run with: go test -race -run TestRaceConditions
func TestRaceConditions(t *testing.T) {
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

	// Run test scenarios
	t.Run("ConcurrentAddContribution", func(t *testing.T) {
		testConcurrentAddContribution(t, ctx, svc, repo, testPool)
	})

	t.Run("ConcurrentVoting", func(t *testing.T) {
		testConcurrentVoting(t, ctx, svc, repo, testPool)
	})

	t.Run("ConcurrentCheckAndUnlock", func(t *testing.T) {
		testConcurrentCheckAndUnlock(t, ctx, svc, repo, testPool)
	})

	t.Run("SessionEndingDuringVote", func(t *testing.T) {
		testSessionEndingDuringVote(t, ctx, svc, repo, testPool)
	})

	// Shutdown service
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	svc.Shutdown(shutdownCtx)
}

// testConcurrentAddContribution tests race conditions when multiple goroutines
// add contributions simultaneously
func testConcurrentAddContribution(t *testing.T, ctx context.Context, svc Service, repo Repository, pool *pgxpool.Pool) {
	cleanupProgressionState(t, ctx, pool)

	// Start a voting session to have an active progress
	if err := svc.StartVotingSession(ctx, nil); err != nil {
		t.Skipf("Cannot start session: %v", err)
	}

	// Get initial progress
	initialProgress, err := repo.GetActiveUnlockProgress(ctx)
	if err != nil || initialProgress == nil {
		t.Skip("No active progress available")
	}

	initialContributions := initialProgress.ContributionsAccumulated

	// Add contributions concurrently
	var wg sync.WaitGroup
	numGoroutines := 20
	contributionPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := svc.AddContribution(ctx, contributionPerGoroutine); err != nil {
				t.Logf("AddContribution error: %v", err)
			}
		}()
	}

	wg.Wait()

	// Verify total is correct (no lost updates due to race conditions)
	finalProgress, err := repo.GetActiveUnlockProgress(ctx)
	if err != nil {
		t.Fatalf("Failed to get final progress: %v", err)
	}

	expectedTotal := initialContributions + (numGoroutines * contributionPerGoroutine)
	if finalProgress.ContributionsAccumulated < expectedTotal {
		t.Errorf("Race condition detected: expected at least %d contributions, got %d",
			expectedTotal, finalProgress.ContributionsAccumulated)
	}
}

// testConcurrentVoting tests race conditions when multiple users vote simultaneously
func testConcurrentVoting(t *testing.T, ctx context.Context, svc Service, repo Repository, pool *pgxpool.Pool) {
	cleanupProgressionState(t, ctx, pool)

	// This test validates that concurrent voting doesn't lose votes or corrupt state
	// We test the voting option increment logic, not the full VoteForUnlock flow
	// (which requires user creation)

	// Start a voting session
	if err := svc.StartVotingSession(ctx, nil); err != nil {
		t.Skipf("Cannot start session: %v", err)
	}

	// Get session
	session, err := repo.GetActiveSession(ctx)
	if err != nil || session == nil {
		t.Skip("No active session")
	}

	if len(session.Options) < 2 {
		t.Skip("Need at least 2 options to test concurrent voting")
	}

	// Directly increment vote counts concurrently (bypasses user validation)
	var wg sync.WaitGroup
	numVotes := 100

	for i := 0; i < numVotes; i++ {
		wg.Add(1)
		optionIndex := i % len(session.Options)
		optionID := session.Options[optionIndex].ID

		go func(oid int) {
			defer wg.Done()
			if err := repo.IncrementOptionVote(ctx, oid); err != nil {
				t.Logf("IncrementOptionVote error: %v", err)
			}
		}(optionID)
	}

	wg.Wait()

	// Verify votes were recorded (no lost votes due to race conditions)
	updatedSession, err := repo.GetActiveSession(ctx)
	if err != nil {
		t.Fatalf("Failed to get updated session: %v", err)
	}

	totalVotes := 0
	for _, option := range updatedSession.Options {
		totalVotes += option.VoteCount
	}

	// Should have exactly numVotes (if no race conditions)
	if totalVotes != numVotes {
		t.Errorf("Race condition detected: expected %d votes, got %d (lost %d votes)",
			numVotes, totalVotes, numVotes-totalVotes)
	}
}

// testConcurrentCheckAndUnlock tests what happens when multiple goroutines
// call CheckAndUnlockNode simultaneously
func testConcurrentCheckAndUnlock(t *testing.T, ctx context.Context, svc Service, repo Repository, pool *pgxpool.Pool) {
	cleanupProgressionState(t, ctx, pool)

	// For this test, we need an auto-select scenario or to manually set the target
	// Let's manually create the scenario
	nodes, err := repo.GetAllNodes(ctx)
	if err != nil || len(nodes) == 0 {
		t.Skip("No nodes available")
	}

	// Create progress and session
	sessionID, err := repo.CreateVotingSession(ctx)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	progressID, err := repo.CreateUnlockProgress(ctx)
	if err != nil {
		t.Fatalf("Failed to create progress: %v", err)
	}

	// Pick a node with non-zero cost
	var node *domain.ProgressionNode
	for _, n := range nodes {
		if n.UnlockCost > 0 && n.UnlockCost < 100 {
			node = n
			break
		}
	}

	if node == nil {
		t.Skip("No suitable node found")
	}

	// Set unlock target
	err = repo.SetUnlockTarget(ctx, progressID, node.ID, 1, sessionID)
	if err != nil {
		t.Fatalf("Failed to set unlock target: %v", err)
	}

	// Add contributions to almost meet threshold
	remaining := node.UnlockCost - 5
	if remaining > 0 {
		repo.AddContribution(ctx, progressID, remaining)
	}

	// Now add final contributions and trigger multiple concurrent unlock checks
	var wg sync.WaitGroup
	unlockResults := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Add enough to trigger unlock
			svc.AddContribution(ctx, 2)

			// Try to unlock
			unlock, err := svc.CheckAndUnlockNode(ctx)
			if err != nil {
				t.Logf("CheckAndUnlockNode error: %v", err)
			}
			unlockResults <- (unlock != nil)
		}()
	}

	wg.Wait()
	close(unlockResults)

	// Only one goroutine should successfully unlock
	unlockCount := 0
	for result := range unlockResults {
		if result {
			unlockCount++
		}
	}

	// Should unlock at most once (semaphore prevents concurrent unlocks)
	if unlockCount > 1 {
		t.Errorf("Race condition: node unlocked %d times (expected at most 1)", unlockCount)
	}
}

// testSessionEndingDuringVote tests what happens when voting ends while
// votes are being cast
func testSessionEndingDuringVote(t *testing.T, ctx context.Context, svc Service, repo Repository, pool *pgxpool.Pool) {
	cleanupProgressionState(t, ctx, pool)

	// Start session
	if err := svc.StartVotingSession(ctx, nil); err != nil {
		t.Skipf("Cannot start session: %v", err)
	}

	session, err := repo.GetActiveSession(ctx)
	if err != nil || session == nil || len(session.Options) < 2 {
		t.Skip("Need active session with multiple options")
	}

	// Increment vote counts concurrently while ending session
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		optionIndex := i % len(session.Options)
		optionID := session.Options[optionIndex].ID

		go func(oid int) {
			defer wg.Done()
			// Increment vote (might fail or succeed depending on timing)
			_ = repo.IncrementOptionVote(ctx, oid)
		}(optionID)
	}

	// End voting while votes are being incremented
	time.Sleep(10 * time.Millisecond)
	_, err = svc.EndVoting(ctx)
	if err != nil {
		t.Logf("EndVoting error: %v", err)
	}

	wg.Wait()

	// Verify system is in consistent state (no corruption)
	// Session should be ended
	endedSession, err := repo.GetSessionByID(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if endedSession.Status != "completed" && endedSession.Status != "voting" {
		t.Errorf("Session in unexpected state: %s", endedSession.Status)
	}

	// No database corruption
	var sessionCount, optionCount int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM progression_voting_sessions").Scan(&sessionCount)
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM progression_voting_options").Scan(&optionCount)

	if sessionCount < 1 || optionCount < 1 {
		t.Error("Database corruption detected: missing expected records")
	}
}
