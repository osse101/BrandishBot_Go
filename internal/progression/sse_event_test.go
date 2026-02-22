package progression

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbpostgres "github.com/osse101/BrandishBot_Go/internal/database/postgres"
	"github.com/osse101/BrandishBot_Go/internal/event"
)

// TestSSEEventConsistency_Integration verifies that SSE-related events (like ProgressionTargetSet)
// are correctly emitted during all lifecycle transitions, ensuring consistent UI updates.
func TestSSEEventConsistency_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if testDBConnString == "" {
		t.Skip("Skipping integration test: database not available")
	}

	ctx := context.Background()
	ensureMigrations(t)

	// Create event bus for service
	bus := event.NewMemoryBus()

	// Capture events in a channel for verification
	targetSetChan := make(chan event.Event, 10)
	bus.Subscribe(event.ProgressionTargetSet, func(ctx context.Context, evt event.Event) error {
		targetSetChan <- evt
		return nil
	})

	// Create repositories
	repo := dbpostgres.NewProgressionRepository(testPool, bus)
	userRepo := dbpostgres.NewUserRepository(testPool)

	// Create service
	svc := NewService(repo, userRepo, bus, nil, nil)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		svc.Shutdown(shutdownCtx)
	}()

	cleanupProgressionState(ctx, t, testPool)

	// Ensure root is unlocked (progression_system)
	root, err := repo.GetNodeByKey(ctx, "progression_system")
	require.NoError(t, err)
	err = repo.UnlockNode(ctx, root.ID, 1, "setup", 0)
	require.NoError(t, err)

	// Step 1: Initial Start
	err = svc.StartVotingSession(ctx, nil)
	require.NoError(t, err)

	// Check if it was auto-select or multiple options
	session, err := repo.GetActiveSession(ctx)
	require.NoError(t, err)
	require.NotNil(t, session)

	if len(session.Options) == 1 {
		// Auto-select path: Verify event emitted immediately
		select {
		case evt := <-targetSetChan:
			assert.Equal(t, event.ProgressionTargetSet, evt.Type)
			payload, err := event.DecodePayload[event.ProgressionTargetSetPayloadV1](evt.Payload)
			require.NoError(t, err)
			assert.True(t, payload.AutoSelected)
			t.Logf("Verified initial auto-select event for node: %s", payload.NodeKey)
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for ProgressionTargetSet event on initial start (auto-select)")
		}
	} else {
		// Manual voting path: Verify event emitted after EndVoting
		t.Logf("Manual voting started with %d options, ending it to verify event", len(session.Options))
		winner, err := svc.EndVoting(ctx)
		require.NoError(t, err)

		select {
		case evt := <-targetSetChan:
			assert.Equal(t, event.ProgressionTargetSet, evt.Type)
			payload, err := event.DecodePayload[event.ProgressionTargetSetPayloadV1](evt.Payload)
			require.NoError(t, err)
			assert.Equal(t, winner.NodeDetails.NodeKey, payload.NodeKey)
			t.Logf("Verified event after manual voting for node: %s", payload.NodeKey)
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for ProgressionTargetSet event after EndVoting")
		}
	}

	// Step 2: Transition Path (Unlock current node -> Transition to next)
	progress, err := repo.GetActiveUnlockProgress(ctx)
	require.NoError(t, err)
	require.NotNil(t, progress.NodeID)

	node, err := repo.GetNodeByID(ctx, *progress.NodeID)
	require.NoError(t, err)

	err = svc.AddContribution(ctx, node.UnlockCost)
	require.NoError(t, err)

	// Wait for the NEXT ProgressionTargetSet event from transition
	select {
	case evt := <-targetSetChan:
		assert.Equal(t, event.ProgressionTargetSet, evt.Type)
		payload, err := event.DecodePayload[event.ProgressionTargetSetPayloadV1](evt.Payload)
		require.NoError(t, err)
		t.Logf("Verified transition event for node: %s", payload.NodeKey)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for ProgressionTargetSet event during post-unlock transition")
	}
}
