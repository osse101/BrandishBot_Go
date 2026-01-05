package progression

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// PrerequisiteSyncer is an optional interface for syncing prerequisites to junction table
type PrerequisiteSyncer interface {
	SyncPrerequisites(ctx context.Context, nodeID int, prerequisiteIDs []int) error
}

// syncPrerequisites syncs a node's prerequisites to the junction table
// It clears any existing prerequisites and inserts the new ones
func syncPrerequisites(ctx context.Context, repo Repository, nodeID int, prerequisites []string, existingByKey map[string]*domain.ProgressionNode, insertedNodeIDs map[string]int) error {
	// Type assert repository to access junction table methods
	prereqSyncer, ok := repo.(PrerequisiteSyncer)
	if !ok {
		// If repository doesn't support junction table sync, skip silently
		// This allows gradual migration of repository implementations
		return nil
	}

	// Resolve prerequisite keys to IDs
	prereqIDs := make([]int, 0, len(prerequisites))
	for _, prereqKey := range prerequisites {
		var prereqID int

		// Try existing nodes first
		if existing, ok := existingByKey[prereqKey]; ok {
			prereqID = existing.ID
		} else if id, ok := insertedNodeIDs[prereqKey]; ok {
			// Try newly inserted nodes
			prereqID = id
		} else {
			// Prerequisites should have been validated earlier
			return fmt.Errorf("prerequisite '%s' not found", prereqKey)
		}

		prereqIDs = append(prereqIDs, prereqID)
	}

	// Sync to database (clear old, insert new)
	return prereqSyncer.SyncPrerequisites(ctx, nodeID, prereqIDs)
}
