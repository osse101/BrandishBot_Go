package postgres

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/database/generated"
)

// SyncPrerequisites synchronizes a node's prerequisites in the junction table
// Implements progression.PrerequisiteSyncer interface
func (r *progressionRepository) SyncPrerequisites(ctx context.Context, nodeID int, prerequisiteIDs []int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer SafeRollback(ctx, tx)

	q := r.q.WithTx(tx)
	
	// Clear existing prerequisites for this node
	err = q.ClearNodePrerequisites(ctx, int32(nodeID))
	if err != nil {
		return fmt.Errorf("failed to clear prerequisites: %w", err)
	}
	
	// Insert new prerequisites
	for _, prereqID := range prerequisiteIDs {
		err = q.InsertNodePrerequisite(ctx, generated.InsertNodePrerequisiteParams{
			NodeID:             int32(nodeID),
			PrerequisiteNodeID: int32(prereqID),
		})
		if err != nil {
			return fmt.Errorf("failed to insert prerequisite: %w", err)
		}
	}
	
	return tx.Commit(ctx)
}
