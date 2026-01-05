package postgres

import (
	"context"
	"fmt"
)

// SyncPrerequisites synchronizes a node's prerequisites in the junction table
// Implements progression.PrerequisiteSyncer interface
func (r *progressionRepository) SyncPrerequisites(ctx context.Context, nodeID int, prerequisiteIDs []int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer SafeRollback(ctx, tx)
	
	// Clear existing prerequisites for this node
	_, err = tx.Exec(ctx, `DELETE FROM progression_prerequisites WHERE node_id = $1`, nodeID)
	if err != nil {
		return fmt.Errorf("failed to clear prerequisites: %w", err)
	}
	
	// Insert new prerequisites
	for _, prereqID := range prerequisiteIDs {
		_, err = tx.Exec(ctx, `
			INSERT INTO progression_prerequisites (node_id, prerequisite_node_id)
			VALUES ($1, $2)
			ON CONFLICT (node_id, prerequisite_node_id) DO NOTHING
		`, nodeID, prereqID)
		if err != nil {
			return fmt.Errorf("failed to insert prerequisite: %w", err)
		}
	}
	
	return tx.Commit(ctx)
}
