package progression

import (
	"context"
	"testing"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/stretchr/testify/assert"
)

// This test reproduces the issue where a voting session is created even if there is only one available option.
// UPDATE: Now it verifies the FIX (no session created).
func TestStartVotingSession_SingleOption_FixVerification(t *testing.T) {
	repo := NewMockRepository()
	// Manually setup the tree to have exactly 1 available node.

	// 1. Setup Root Node (ID: 1) - Unlocked
	root := &domain.ProgressionNode{
		ID:           1,
		NodeKey:      "progression_system",
		NodeType:     "feature",
		DisplayName:  "Progression System",
		MaxLevel:     1,
		UnlockCost:   0,
		CreatedAt:    time.Now(),
	}

	repo.nodes[1] = root
	repo.nodesByKey["progression_system"] = root

	// Unlock root
	ctx := context.Background()
	repo.UnlockNode(ctx, 1, 1, "system", 0)

	// 2. Setup Child Node (ID: 2) - Locked, Dependent on Root
	// This will be the ONLY available node.
	parentID := 1
	child := &domain.ProgressionNode{
		ID:           2,
		NodeKey:      "single_option",
		NodeType:     "feature",
		DisplayName:  "Single Option Feature",
		ParentNodeID: &parentID,
		MaxLevel:     1,
		UnlockCost:   100,
		CreatedAt:    time.Now(),
	}
	repo.nodes[2] = child
	repo.nodesByKey["single_option"] = child

	service := NewService(repo, nil)

	// Verify GetAvailableUnlocks returns exactly 1 node
	available, err := service.GetAvailableUnlocks(ctx)
	assert.NoError(t, err)
	assert.Len(t, available, 1, "Should have exactly 1 available node")
	assert.Equal(t, "single_option", available[0].NodeKey)

	// Act: StartVotingSession
	err = service.StartVotingSession(ctx, nil)
	assert.NoError(t, err)

	// Assert: NEW BEHAVIOR (Fixed) -> No voting session created, target set immediately.

	session, err := repo.GetActiveSession(ctx)
	assert.NoError(t, err)

	if session == nil {
		t.Log("FIX VERIFIED: No session created")
	} else {
		t.Errorf("FIX FAILED: Session was created but should have been skipped. Status: %s", session.Status)
	}

	// Verify target is set
	progress, err := repo.GetActiveUnlockProgress(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, progress)
	if progress != nil {
		assert.NotNil(t, progress.NodeID, "NodeID should be set")
		if progress.NodeID != nil {
			assert.Equal(t, 2, *progress.NodeID, "Target NodeID should match the single option")
		}

		// In the mock SetUnlockTarget sets VotingSessionID.
		// We passed 0 for sessionID in the fix.
		// Let's verify that.
		if progress.VotingSessionID != nil {
			assert.Equal(t, 0, *progress.VotingSessionID, "VotingSessionID should be 0 (or nil depending on impl)")
		}
	}
}
