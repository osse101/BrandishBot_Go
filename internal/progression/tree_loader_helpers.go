package progression

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// PrerequisiteSyncer is an optional interface for syncing prerequisites to junction table
type PrerequisiteSyncer interface {
	SyncPrerequisites(ctx context.Context, nodeID int, prerequisiteIDs []int) error
}

// DynamicPrerequisiteSyncer is an optional interface for syncing dynamic prerequisites
type DynamicPrerequisiteSyncer interface {
	UpdateNodeDynamicPrerequisites(ctx context.Context, nodeID int, jsonData []byte) error
}

// syncPrerequisites syncs a node's prerequisites to the junction table
// It clears any existing prerequisites and inserts the new ones
func syncPrerequisites(ctx context.Context, repo repository.Progression, nodeID int, prerequisites []string, existingByKey map[string]*domain.ProgressionNode, insertedNodeIDs map[string]int) error {
	// Type assert repository to access junction table methods
	prereqSyncer, ok := repo.(PrerequisiteSyncer)
	if !ok {
		// If repository doesn't support junction table sync, skip silently
		// This allows gradual migration of repository implementations
		return nil
	}

	// Resolve only static prerequisite keys to IDs
	prereqIDs := make([]int, 0, len(prerequisites))
	for _, prereqStr := range prerequisites {
		isDynamic, _, staticKey, err := ParsePrerequisite(prereqStr)
		if err != nil {
			return fmt.Errorf("failed to parse prerequisite: %w", err)
		}

		// Skip dynamic prerequisites - they're handled separately
		if isDynamic {
			continue
		}

		var prereqID int

		// Try existing nodes first
		if existing, ok := existingByKey[staticKey]; ok {
			prereqID = existing.ID
		} else if id, ok := insertedNodeIDs[staticKey]; ok {
			// Try newly inserted nodes
			prereqID = id
		} else {
			// Prerequisites should have been validated earlier
			return fmt.Errorf("prerequisite '%s' not found", staticKey)
		}

		prereqIDs = append(prereqIDs, prereqID)
	}

	// Sync to database (clear old, insert new)
	return prereqSyncer.SyncPrerequisites(ctx, nodeID, prereqIDs)
}

// syncDynamicPrerequisites parses and stores dynamic prerequisites in JSONB column
func syncDynamicPrerequisites(ctx context.Context, repo repository.Progression, nodeID int, prerequisites []string) error {
	// Type assert repository to access dynamic prerequisites methods
	dynamicSyncer, ok := repo.(DynamicPrerequisiteSyncer)
	if !ok {
		// If repository doesn't support dynamic prerequisites, skip silently
		return nil
	}

	dynamicPrereqs := []domain.DynamicPrerequisite{}

	for _, prereqStr := range prerequisites {
		isDynamic, dynamicPrereq, _, err := ParsePrerequisite(prereqStr)
		if err != nil {
			return fmt.Errorf("failed to parse prerequisite: %w", err)
		}

		if isDynamic {
			dynamicPrereqs = append(dynamicPrereqs, *dynamicPrereq)
		}
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(dynamicPrereqs)
	if err != nil {
		return fmt.Errorf("failed to marshal dynamic prerequisites: %w", err)
	}

	// Update database
	return dynamicSyncer.UpdateNodeDynamicPrerequisites(ctx, nodeID, jsonData)
}
