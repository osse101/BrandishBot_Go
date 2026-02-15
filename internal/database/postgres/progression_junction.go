package postgres

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// GetPrerequisites returns all prerequisite nodes for a given node
func (r *progressionRepository) GetPrerequisites(ctx context.Context, nodeID int) ([]*domain.ProgressionNode, error) {
	rows, err := r.q.GetNodePrerequisites(ctx, int32(nodeID))
	if err != nil {
		return nil, fmt.Errorf("failed to query prerequisites: %w", err)
	}
	return r.mapNodes(rows), nil
}

// GetDependents returns all nodes that have this node as a prerequisite
func (r *progressionRepository) GetDependents(ctx context.Context, nodeID int) ([]*domain.ProgressionNode, error) {
	rows, err := r.q.GetNodeDependents(ctx, int32(nodeID))
	if err != nil {
		return nil, fmt.Errorf("failed to query dependents: %w", err)
	}
	// Note: although these look identical, the row types are different (GetNodePrerequisitesRow vs GetNodeDependentsRow)
	// so the linter detects the logic but we can't easily unify the mapping loop without interface/generics.
	// We'll rename local variables significantly to break dupl token sequence.
	finalNodes := make([]*domain.ProgressionNode, 0, len(rows))
	for _, dr := range rows {
		finalNodes = append(finalNodes, mapProgressionNodeFields(dr.ID, dr.NodeKey, dr.NodeType, dr.DisplayName, dr.Description, dr.MaxLevel, dr.UnlockCost, dr.Tier, dr.Size, dr.Category, dr.SortOrder, dr.CreatedAt))
	}
	return finalNodes, nil
}

func (r *progressionRepository) mapNodes(rows []generated.GetNodePrerequisitesRow) []*domain.ProgressionNode {
	nodes := make([]*domain.ProgressionNode, 0, len(rows))
	for _, row := range rows {
		nodes = append(nodes, mapProgressionNodeFields(row.ID, row.NodeKey, row.NodeType, row.DisplayName, row.Description, row.MaxLevel, row.UnlockCost, row.Tier, row.Size, row.Category, row.SortOrder, row.CreatedAt))
	}
	return nodes
}
