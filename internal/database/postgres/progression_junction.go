package postgres

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// GetPrerequisites returns all prerequisite nodes for a given node
func (r *progressionRepository) GetPrerequisites(ctx context.Context, nodeID int) ([]*domain.ProgressionNode, error) {
	query := `
		SELECT n.id, n.node_key, n.node_type, n.display_name, n.description,
		       n.max_level, n.unlock_cost, n.tier, n.size, n.category, n.sort_order, n.created_at
		FROM progression_nodes n
		INNER JOIN progression_prerequisites p ON n.id = p.prerequisite_node_id
		WHERE p.node_id = $1
		ORDER BY n.sort_order, n.id`

	rows, err := r.pool.Query(ctx, query, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query prerequisites: %w", err)
	}
	defer rows.Close()

	var nodes []*domain.ProgressionNode
	for rows.Next() {
		var node domain.ProgressionNode

		err := rows.Scan(
			&node.ID, &node.NodeKey, &node.NodeType, &node.DisplayName,
			&node.Description, &node.MaxLevel, &node.UnlockCost,
			&node.Tier, &node.Size, &node.Category, &node.SortOrder, &node.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan prerequisite node: %w", err)
		}

		nodes = append(nodes, &node)
	}

	return nodes, rows.Err()
}

// GetDependents returns all nodes that have this node as a prerequisite
func (r *progressionRepository) GetDependents(ctx context.Context, nodeID int) ([]*domain.ProgressionNode, error) {
	query := `
		SELECT n.id, n.node_key, n.node_type, n.display_name, n.description,
		       n.max_level, n.unlock_cost, n.tier, n.size, n.category, n.sort_order, n.created_at
		FROM progression_nodes n
		INNER JOIN progression_prerequisites p ON n.id = p.node_id
		WHERE p.prerequisite_node_id = $1
		ORDER BY n.sort_order, n.id`

	rows, err := r.pool.Query(ctx, query, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query dependents: %w", err)
	}
	defer rows.Close()

	var nodes []*domain.ProgressionNode
	for rows.Next() {
		var node domain.ProgressionNode

		err := rows.Scan(
			&node.ID, &node.NodeKey, &node.NodeType, &node.DisplayName,
			&node.Description, &node.MaxLevel, &node.UnlockCost,
			&node.Tier, &node.Size, &node.Category, &node.SortOrder, &node.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dependent node: %w", err)
		}

		nodes = append(nodes, &node)
	}

	return nodes, rows.Err()
}
