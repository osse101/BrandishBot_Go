package progression

import (
	"context"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/testing/leaktest"
)

// =============================================================================
// Memory Leak Tests
// =============================================================================
// NOTE: Mocks are defined in service_test.go and test_helper.go

func TestStartVotingSession_NoGoroutineLeak(t *testing.T) {
	repo := NewMockRepository()
	setupTestNodes(repo)
	svc := NewService(repo, NewMockUser(), nil, nil, nil)

	checker := leaktest.NewGoroutineChecker(t)

	// Execute multiple voting session starts
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_ = svc.StartVotingSession(ctx, nil)
		// Some sessions may fail - that's expected
	}

	// Check for leaks
	checker.Check(0)
}

func TestVoteForUnlock_NoGoroutineLeak(t *testing.T) {
	repo := NewMockRepository()
	setupTestNodes(repo)
	svc := NewService(repo, NewMockUser(), nil, nil, nil)

	// Start a voting session first
	ctx := context.Background()
	_ = svc.StartVotingSession(ctx, nil)

	checker := leaktest.NewGoroutineChecker(t)

	// Cast multiple votes
	users := []string{"user1", "user2", "user3"}
	for _, userID := range users {
		// Vote for first available node
		session, _ := svc.GetActiveVotingSession(ctx)
		if session != nil && len(session.Options) > 0 && session.Options[0].NodeDetails != nil {
			_ = svc.VoteForUnlock(ctx, domain.PlatformDiscord, userID, userID, 1)
		}
	}

	// Check for leaks
	checker.Check(0)
}

func TestEndVoting_NoGoroutineLeak(t *testing.T) {
	repo := NewMockRepository()
	setupTestNodes(repo)
	svc := NewService(repo, NewMockUser(), nil, nil, nil)

	ctx := context.Background()

	checker := leaktest.NewGoroutineChecker(t)

	// Start and end multiple voting cycles
	for i := 0; i < 3; i++ {
		_ = svc.StartVotingSession(ctx, nil)

		// Cast some votes
		session, _ := svc.GetActiveVotingSession(ctx)
		if session != nil && len(session.Options) > 0 && session.Options[0].NodeDetails != nil {
			_ = svc.VoteForUnlock(ctx, domain.PlatformDiscord, "voter1", "voter1", 1)
		}

		// End voting
		_, _ = svc.EndVoting(ctx)
	}

	// Check for leaks
	checker.Check(0)
}

func TestAddContribution_NoGoroutineLeak(t *testing.T) {
	repo := NewMockRepository()
	setupTestNodes(repo)
	svc := NewService(repo, NewMockUser(), nil, nil, nil)

	// Start progress tracking
	ctx := context.Background()
	_ = svc.StartVotingSession(ctx, nil)

	checker := leaktest.NewGoroutineChecker(t)

	// Add contributions multiple times
	for i := 0; i < 10; i++ {
		_ = svc.AddContribution(ctx, 100)
	}

	// Check for leaks
	checker.Check(0)
}

func TestCheckAndUnlockNode_NoGoroutineLeak(t *testing.T) {
	repo := NewMockRepository()
	setupTestNodes(repo)
	svc := NewService(repo, NewMockUser(), nil, nil, nil)

	ctx := context.Background()

	// Setup: Create progress with enough points
	_ = svc.StartVotingSession(ctx, nil)
	session, _ := svc.GetActiveVotingSession(ctx)
	if session != nil && len(session.Options) > 0 && session.Options[0].NodeDetails != nil {
		_ = svc.VoteForUnlock(ctx, domain.PlatformDiscord, "user1", "user1", 1)
		_, _ = svc.EndVoting(ctx)

		// Add contributions to meet threshold
		_ = svc.AddContribution(ctx, 10000)
	}

	checker := leaktest.NewGoroutineChecker(t)

	// Attempt unlock
	_, _ = svc.CheckAndUnlockNode(ctx)

	// Check for leaks
	checker.Check(0)
}

func TestRecordEngagement_NoGoroutineLeak(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo, NewMockUser(), nil, nil, nil)

	checker := leaktest.NewGoroutineChecker(t)

	ctx := context.Background()

	// Record multiple engagements
	metrics := []struct {
		userID string
		metric string
		value  int
	}{
		{"user1", "message", 1},
		{"user2", "command", 1},
		{"user3", "vote_cast", 1},
		{"user1", "item_used", 2},
		{"user2", "item_crafted", 1},
	}

	for _, m := range metrics {
		_ = svc.RecordEngagement(ctx, m.userID, m.metric, m.value)
	}

	// Check for leaks
	checker.Check(0)
}

func TestGetProgressionStatus_NoGoroutineLeak(t *testing.T) {
	repo := NewMockRepository()
	setupTestNodes(repo)
	svc := NewService(repo, NewMockUser(), nil, nil, nil)

	ctx := context.Background()
	_ = svc.StartVotingSession(ctx, nil)

	checker := leaktest.NewGoroutineChecker(t)

	// Get status multiple times (tests caching logic)
	for i := 0; i < 5; i++ {
		_, _ = svc.GetProgressionStatus(ctx)
	}

	// Check for leaks
	checker.Check(0)
}

// Helper function to setup test nodes (reused from existing tests)
func setupTestNodes(repo *MockRepository) {
	// Root nodes
	repo.nodes[1] = &domain.ProgressionNode{
		ID:          1,
		NodeKey:     "root_1",
		DisplayName: "Root Feature 1",
		Description: "Root node",
		MaxLevel:    1,
	}
	repo.nodesByKey["root_1"] = repo.nodes[1]

	repo.nodes[2] = &domain.ProgressionNode{
		ID:          2,
		NodeKey:     "root_2",
		DisplayName: "Root Feature 2",
		Description: "Another root",
		MaxLevel:    1,
	}
	repo.nodesByKey["root_2"] = repo.nodes[2]

	// Child nodes
	repo.nodes[3] = &domain.ProgressionNode{
		ID:          3,
		NodeKey:     "child_1",
		DisplayName: "Child Feature",
		Description: "Dependent node",
		MaxLevel:    2,
	}
	repo.nodesByKey["child_1"] = repo.nodes[3]
}
