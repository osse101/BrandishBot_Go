package postgres

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// GetPrerequisites returns all prerequisite nodes for a given node
func (r *progressionRepository) GetPrerequisites(ctx context.Context, nodeID int) ([]*domain.ProgressionNode, error) {
	rows, err := r.q.GetNodePrerequisites(ctx, int32(nodeID))
	if err != nil {
		return nil, fmt.Errorf("failed to query prerequisites: %w", err)
	}

	nodes := make([]*domain.ProgressionNode, 0, len(rows))
	for _, row := range rows {
		node := &domain.ProgressionNode{
			ID:          int(row.ID),
			NodeKey:     row.NodeKey,
			NodeType:    row.NodeType,
			DisplayName: row.DisplayName,
			Description: row.Description.String,
			MaxLevel:    int(row.MaxLevel.Int32),
			UnlockCost:  int(row.UnlockCost.Int32),
			Tier:        int(row.Tier),
			Size:        row.Size,
			Category:    row.Category,
			SortOrder:   int(row.SortOrder.Int32),
			CreatedAt:   row.CreatedAt.Time,
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetDependents returns all nodes that have this node as a prerequisite
func (r *progressionRepository) GetDependents(ctx context.Context, nodeID int) ([]*domain.ProgressionNode, error) {
	rows, err := r.q.GetNodeDependents(ctx, int32(nodeID))
	if err != nil {
		return nil, fmt.Errorf("failed to query dependents: %w", err)
	}

	nodes := make([]*domain.ProgressionNode, 0, len(rows))
	for _, row := range rows {
		node := &domain.ProgressionNode{
			ID:          int(row.ID),
			NodeKey:     row.NodeKey,
			NodeType:    row.NodeType,
			DisplayName: row.DisplayName,
			Description: row.Description.String,
			MaxLevel:    int(row.MaxLevel.Int32),
			UnlockCost:  int(row.UnlockCost.Int32),
			Tier:        int(row.Tier),
			Size:        row.Size,
			Category:    row.Category,
			SortOrder:   int(row.SortOrder.Int32),
			CreatedAt:   row.CreatedAt.Time,
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}
