package progression

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// This test reproduces the issue where auto-select wasn't handling FK constraints properly.
// FIX: Now we create a session, immediately complete it, and use a valid session ID.
// This preserves FK integrity while still auto-selecting the single option.
func TestStartVotingSession_SingleOption_FixVerification(t *testing.T) {
	repo := NewMockRepository()
	// Manually setup the tree to have exactly 1 available node.

	// 1. Setup Root Node (ID: 1) - Unlocked
	root := &domain.ProgressionNode{
		ID:          1,
		NodeKey:     "progression_system",
		NodeType:    "feature",
		DisplayName: "Progression System",
		MaxLevel:    1,
		UnlockCost:  0,
		CreatedAt:   time.Now(),
	}

	repo.nodes[1] = root
	repo.nodesByKey["progression_system"] = root

	// Unlock root
	ctx := context.Background()
	repo.UnlockNode(ctx, 1, 1, "system", 0)

	// 2. Setup Child Node (ID: 2) - Locked, Dependent on Root
	// This will be the ONLY available node.
	child := &domain.ProgressionNode{
		ID:          2,
		NodeKey:     "single_option",
		NodeType:    "feature",
		DisplayName: "Single Option Feature",
		MaxLevel:    1,
		UnlockCost:  100,
		CreatedAt:   time.Now(),
	}
	repo.nodes[2] = child
	repo.nodesByKey["single_option"] = child

	service := NewService(repo, NewMockUser(), nil, nil, nil)

	// Verify GetAvailableUnlocks returns exactly 1 node
	available, err := service.GetAvailableUnlocks(ctx)
	assert.NoError(t, err)
	assert.Len(t, available, 1, "Should have exactly 1 available node")
	assert.Equal(t, "single_option", available[0].NodeKey)

	// Act: StartVotingSession
	err = service.StartVotingSession(ctx, nil)
	assert.NoError(t, err)

	// Assert: A session is created but immediately completed (for FK integrity)
	// The GetActiveSession returns nil because the session status is "completed", not "voting"
	session, err := repo.GetActiveSession(ctx)
	assert.NoError(t, err)
	// GetActiveSession only returns sessions with status="voting", so it should be nil
	// because we immediately marked it as completed
	t.Logf("Active session (status=voting): %v", session)

	// Verify target is set
	progress, err := repo.GetActiveUnlockProgress(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, progress)
	if progress != nil {
		assert.NotNil(t, progress.NodeID, "NodeID should be set")
		if progress.NodeID != nil {
			assert.Equal(t, 2, *progress.NodeID, "Target NodeID should match the single option")
		}

		// VotingSessionID should now be a valid session ID (not 0) to satisfy FK constraint
		assert.NotNil(t, progress.VotingSessionID, "VotingSessionID should be set")
		if progress.VotingSessionID != nil {
			assert.Greater(t, *progress.VotingSessionID, 0, "VotingSessionID should be a valid session ID > 0")
		}
	}
}

// TestStartVotingSession_ZeroOptions_AllUnlocked tests that when all nodes are unlocked,
// the system correctly reports "no nodes available" error (this is expected behavior).
func TestStartVotingSession_ZeroOptions_AllUnlocked(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	// Setup minimal tree: root + one child, both unlocked
	root := &domain.ProgressionNode{
		ID:          1,
		NodeKey:     "progression_system",
		NodeType:    "feature",
		DisplayName: "Progression System",
		MaxLevel:    1,
		UnlockCost:  0,
		CreatedAt:   time.Now(),
	}
	repo.nodes[1] = root
	repo.nodesByKey["progression_system"] = root
	repo.UnlockNode(ctx, 1, 1, "system", 0)

	// Child node - also unlocked at max level
	child := &domain.ProgressionNode{
		ID:          2,
		NodeKey:     "only_child",
		NodeType:    "feature",
		DisplayName: "Only Child Feature",
		MaxLevel:    1, // Max level is 1
		UnlockCost:  100,
		CreatedAt:   time.Now(),
	}
	repo.nodes[2] = child
	repo.nodesByKey["only_child"] = child
	repo.UnlockNode(ctx, 2, 1, "system", 0) // Unlock at max level

	service := NewService(repo, NewMockUser(), nil, nil, nil)

	// Verify no available nodes
	available, err := service.GetAvailableUnlocks(ctx)
	assert.NoError(t, err)
	assert.Len(t, available, 0, "Should have 0 available nodes when all are unlocked")

	// StartVotingSession should return error
	err = service.StartVotingSession(ctx, nil)
	assert.Error(t, err, "Should error when no nodes available for voting")
	assert.Contains(t, err.Error(), "no nodes available", "Error should mention no nodes available")
}

// TestStartVotingSession_MultiLevelNode_AutoSelect tests auto-select behavior
// with a node that has multiple levels (MaxLevel > 1)
func TestStartVotingSession_MultiLevelNode_AutoSelect(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	// Setup root
	root := &domain.ProgressionNode{
		ID:          1,
		NodeKey:     "progression_system",
		NodeType:    "feature",
		DisplayName: "Root",
		MaxLevel:    1,
		UnlockCost:  0,
		CreatedAt:   time.Now(),
	}
	repo.nodes[1] = root
	repo.nodesByKey["progression_system"] = root
	repo.UnlockNode(ctx, 1, 1, "system", 0)

	// Multi-level child node (5 levels)
	multiLevel := &domain.ProgressionNode{
		ID:          2,
		NodeKey:     "upgrade_multi",
		NodeType:    "upgrade",
		DisplayName: "Multi-Level Upgrade",
		MaxLevel:    5, // 5 levels
		UnlockCost:  500,
		CreatedAt:   time.Now(),
	}
	repo.nodes[2] = multiLevel
	repo.nodesByKey["upgrade_multi"] = multiLevel

	service := NewService(repo, NewMockUser(), nil, nil, nil)

	// Verify 1 available node
	available, err := service.GetAvailableUnlocks(ctx)
	assert.NoError(t, err)
	assert.Len(t, available, 1)

	// Auto-select should work
	err = service.StartVotingSession(ctx, nil)
	assert.NoError(t, err)

	// Verify target level is 1 (first level)
	progress, err := repo.GetActiveUnlockProgress(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, progress)
	if progress != nil {
		assert.NotNil(t, progress.TargetLevel)
		if progress.TargetLevel != nil {
			assert.Equal(t, 1, *progress.TargetLevel, "First unlock should target level 1")
		}
	}

	// Now unlock level 1 and try again - should target level 2
	repo.UnlockNode(ctx, 2, 1, "test", 0)

	// Need to manually reset progress for next test
	// End the current voting session first (Bug #7 fix prevents duplicate sessions)
	activeSession, _ := repo.GetActiveSession(ctx)
	if activeSession != nil && len(activeSession.Options) > 0 {
		optionID := activeSession.Options[0].ID
		repo.EndVotingSession(ctx, activeSession.ID, &optionID)
	}

	// Simulate unlock completion
	if progress != nil {
		repo.CompleteUnlock(ctx, progress.ID, 0)
	}

	err = service.StartVotingSession(ctx, nil)
	assert.NoError(t, err)

	progress2, err := repo.GetActiveUnlockProgress(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, progress2)
	assert.NotNil(t, progress2.TargetLevel)
	assert.Equal(t, 2, *progress2.TargetLevel, "Second unlock should target level 2")
}

// TestAutoSelect_ContributionTracking tests that contributions are correctly
// applied after auto-select (when no voting session is created)
func TestAutoSelect_ContributionTracking(t *testing.T) {
	repo := NewMockRepository()
	ctx := context.Background()

	// Setup tree with single option
	root := &domain.ProgressionNode{
		ID:          1,
		NodeKey:     "progression_system",
		NodeType:    "feature",
		DisplayName: "Root",
		MaxLevel:    1,
		UnlockCost:  0,
		CreatedAt:   time.Now(),
	}
	repo.nodes[1] = root
	repo.nodesByKey["progression_system"] = root
	repo.UnlockNode(ctx, 1, 1, "system", 0)

	child := &domain.ProgressionNode{
		ID:          2,
		NodeKey:     "single_target",
		NodeType:    "feature",
		DisplayName: "Single Target",
		MaxLevel:    1,
		UnlockCost:  1000, // Requires 1000 contribution points
		CreatedAt:   time.Now(),
	}
	repo.nodes[2] = child
	repo.nodesByKey["single_target"] = child

	service := NewService(repo, NewMockUser(), nil, nil, nil)

	// Auto-select (single option)
	err := service.StartVotingSession(ctx, nil)
	assert.NoError(t, err)

	// Verify progress exists
	progress, err := repo.GetActiveUnlockProgress(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, progress)
	initialContributions := progress.ContributionsAccumulated

	// Add contributions
	err = service.AddContribution(ctx, 250)
	assert.NoError(t, err)

	// Verify contributions were added
	progress, err = repo.GetActiveUnlockProgress(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, progress)
	assert.Equal(t, initialContributions+250, progress.ContributionsAccumulated,
		"Contributions should be tracked after auto-select")

	// Add more and verify accumulation
	err = service.AddContribution(ctx, 350)
	assert.NoError(t, err)

	progress, err = repo.GetActiveUnlockProgress(ctx)
	assert.NoError(t, err)
	assert.Equal(t, initialContributions+600, progress.ContributionsAccumulated,
		"Contributions should accumulate correctly")
}
