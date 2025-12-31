package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

type progressionRepository struct {
	pool *pgxpool.Pool
}

// NewProgressionRepository creates a new Postgres-backed progression repository
func NewProgressionRepository(pool *pgxpool.Pool) progression.Repository {
	return &progressionRepository{pool: pool}
}

// Node operations

func (r *progressionRepository) GetNodeByKey(ctx context.Context, nodeKey string) (*domain.ProgressionNode, error) {
	query := `
		SELECT id, node_key, node_type, display_name, description, parent_node_id, 
		       max_level, unlock_cost, sort_order, created_at
		FROM progression_nodes
		WHERE node_key = $1
		LIMIT 1`

	var node domain.ProgressionNode
	var parentID *int

	err := r.pool.QueryRow(ctx, query, nodeKey).Scan(
		&node.ID, &node.NodeKey, &node.NodeType, &node.DisplayName,
		&node.Description, &parentID, &node.MaxLevel, &node.UnlockCost,
		&node.SortOrder, &node.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get node by key: %w", err)
	}

	node.ParentNodeID = parentID
	return &node, nil
}

func (r *progressionRepository) GetNodeByID(ctx context.Context, id int) (*domain.ProgressionNode, error) {
	query := `
		SELECT id, node_key, node_type, display_name, description, parent_node_id,
		       max_level, unlock_cost, sort_order, created_at
		FROM progression_nodes
		WHERE id = $1`

	var node domain.ProgressionNode
	var parentID *int

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&node.ID, &node.NodeKey, &node.NodeType, &node.DisplayName,
		&node.Description, &parentID, &node.MaxLevel, &node.UnlockCost,
		&node.SortOrder, &node.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get node by ID: %w", err)
	}

	node.ParentNodeID = parentID
	return &node, nil
}

func (r *progressionRepository) GetAllNodes(ctx context.Context) ([]*domain.ProgressionNode, error) {
	query := `
		SELECT id, node_key, node_type, display_name, description, parent_node_id,
		       max_level, unlock_cost, sort_order, created_at
		FROM progression_nodes
		ORDER BY sort_order, id`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	defer rows.Close()

	var nodes []*domain.ProgressionNode
	for rows.Next() {
		var node domain.ProgressionNode
		var parentID *int

		err := rows.Scan(
			&node.ID, &node.NodeKey, &node.NodeType, &node.DisplayName,
			&node.Description, &parentID, &node.MaxLevel, &node.UnlockCost,
			&node.SortOrder, &node.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}

		node.ParentNodeID = parentID
		nodes = append(nodes, &node)
	}

	return nodes, rows.Err()
}

func (r *progressionRepository) GetChildNodes(ctx context.Context, parentID int) ([]*domain.ProgressionNode, error) {
	query := `
		SELECT id, node_key, node_type, display_name, description, parent_node_id,
		       max_level, unlock_cost, sort_order, created_at
		FROM progression_nodes
		WHERE parent_node_id = $1
		ORDER BY sort_order, id`

	rows, err := r.pool.Query(ctx, query, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query child nodes: %w", err)
	}
	defer rows.Close()

	var nodes []*domain.ProgressionNode
	for rows.Next() {
		var node domain.ProgressionNode
		var pID *int

		err := rows.Scan(
			&node.ID, &node.NodeKey, &node.NodeType, &node.DisplayName,
			&node.Description, &pID, &node.MaxLevel, &node.UnlockCost,
			&node.SortOrder, &node.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan child node: %w", err)
		}

		node.ParentNodeID = pID
		nodes = append(nodes, &node)
	}

	return nodes, rows.Err()
}

// InsertNode inserts a new progression node and returns its ID
// Implements progression.NodeInserter interface
func (r *progressionRepository) InsertNode(ctx context.Context, node *domain.ProgressionNode) (int, error) {
	query := `
		INSERT INTO progression_nodes (node_key, node_type, display_name, description, parent_node_id, max_level, unlock_cost, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	var id int
	err := r.pool.QueryRow(ctx, query,
		node.NodeKey,
		node.NodeType,
		node.DisplayName,
		node.Description,
		node.ParentNodeID,
		node.MaxLevel,
		node.UnlockCost,
		node.SortOrder,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to insert node: %w", err)
	}

	return id, nil
}

// UpdateNode updates an existing progression node
// Implements progression.NodeUpdater interface
func (r *progressionRepository) UpdateNode(ctx context.Context, nodeID int, node *domain.ProgressionNode) error {
	query := `
		UPDATE progression_nodes 
		SET node_type = $2, display_name = $3, description = $4, parent_node_id = $5, 
		    max_level = $6, unlock_cost = $7, sort_order = $8
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query,
		nodeID,
		node.NodeType,
		node.DisplayName,
		node.Description,
		node.ParentNodeID,
		node.MaxLevel,
		node.UnlockCost,
		node.SortOrder,
	)

	if err != nil {
		return fmt.Errorf("failed to update node: %w", err)
	}

	return nil
}

// Unlock operations

func (r *progressionRepository) GetUnlock(ctx context.Context, nodeID int, level int) (*domain.ProgressionUnlock, error) {
	query := `
		SELECT id, node_id, current_level, unlocked_at, unlocked_by, engagement_score
		FROM progression_unlocks
		WHERE node_id = $1 AND current_level = $2`

	var unlock domain.ProgressionUnlock
	err := r.pool.QueryRow(ctx, query, nodeID, level).Scan(
		&unlock.ID, &unlock.NodeID, &unlock.CurrentLevel,
		&unlock.UnlockedAt, &unlock.UnlockedBy, &unlock.EngagementScore,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get unlock: %w", err)
	}

	return &unlock, nil
}

func (r *progressionRepository) GetAllUnlocks(ctx context.Context) ([]*domain.ProgressionUnlock, error) {
	query := `
		SELECT id, node_id, current_level, unlocked_at, unlocked_by, engagement_score
		FROM progression_unlocks
		ORDER BY unlocked_at`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query unlocks: %w", err)
	}
	defer rows.Close()

	var unlocks []*domain.ProgressionUnlock
	for rows.Next() {
		var unlock domain.ProgressionUnlock
		err := rows.Scan(
			&unlock.ID, &unlock.NodeID, &unlock.CurrentLevel,
			&unlock.UnlockedAt, &unlock.UnlockedBy, &unlock.EngagementScore,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan unlock: %w", err)
		}
		unlocks = append(unlocks, &unlock)
	}

	return unlocks, rows.Err()
}

func (r *progressionRepository) IsNodeUnlocked(ctx context.Context, nodeKey string, level int) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM progression_unlocks pu
			JOIN progression_nodes pn ON pu.node_id = pn.id
			WHERE pn.node_key = $1 AND pu.current_level >= $2
		)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, nodeKey, level).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if node unlocked: %w", err)
	}

	return exists, nil
}

func (r *progressionRepository) UnlockNode(ctx context.Context, nodeID int, level int, unlockedBy string, engagementScore int) error {
	query := `
		INSERT INTO progression_unlocks (node_id, current_level, unlocked_by, engagement_score)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (node_id, current_level) DO NOTHING`

	_, err := r.pool.Exec(ctx, query, nodeID, level, unlockedBy, engagementScore)
	if err != nil {
		return fmt.Errorf("failed to unlock node: %w", err)
	}

	return nil
}

func (r *progressionRepository) RelockNode(ctx context.Context, nodeID int, level int) error {
	query := `DELETE FROM progression_unlocks WHERE node_id = $1 AND current_level = $2`

	_, err := r.pool.Exec(ctx, query, nodeID, level)
	if err != nil {
		return fmt.Errorf("failed to relock node: %w", err)
	}

	return nil
}

// Voting operations

func (r *progressionRepository) GetActiveVoting(ctx context.Context) (*domain.ProgressionVoting, error) {
	query := `
		SELECT id, node_id, target_level, vote_count, voting_started_at, voting_ends_at, is_active
		FROM progression_voting
		WHERE is_active = true
		ORDER BY voting_started_at DESC
		LIMIT 1`

	var voting domain.ProgressionVoting
	var endsAt *time.Time

	err := r.pool.QueryRow(ctx, query).Scan(
		&voting.ID, &voting.NodeID, &voting.TargetLevel, &voting.VoteCount,
		&voting.VotingStartedAt, &endsAt, &voting.IsActive,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active voting: %w", err)
	}

	voting.VotingEndsAt = endsAt
	return &voting, nil
}

func (r *progressionRepository) StartVoting(ctx context.Context, nodeID int, level int, endsAt *time.Time) error {
	query := `
		INSERT INTO progression_voting (node_id, target_level, vote_count, voting_ends_at, is_active)
		VALUES ($1, $2, 0, $3, true)
		ON CONFLICT (node_id, target_level) DO UPDATE
		SET voting_started_at = CURRENT_TIMESTAMP, voting_ends_at = $3, is_active = true, vote_count = 0`

	_, err := r.pool.Exec(ctx, query, nodeID, level, endsAt)
	if err != nil {
		return fmt.Errorf("failed to start voting: %w", err)
	}

	return nil
}

func (r *progressionRepository) GetVoting(ctx context.Context, nodeID int, level int) (*domain.ProgressionVoting, error) {
	query := `
		SELECT id, node_id, target_level, vote_count, voting_started_at, voting_ends_at, is_active
		FROM progression_voting
		WHERE node_id = $1 AND target_level = $2`

	var voting domain.ProgressionVoting
	var endsAt *time.Time

	err := r.pool.QueryRow(ctx, query, nodeID, level).Scan(
		&voting.ID, &voting.NodeID, &voting.TargetLevel, &voting.VoteCount,
		&voting.VotingStartedAt, &endsAt, &voting.IsActive,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get voting: %w", err)
	}

	voting.VotingEndsAt = endsAt
	return &voting, nil
}

func (r *progressionRepository) IncrementVote(ctx context.Context, nodeID int, level int) error {
	query := `
		UPDATE progression_voting
		SET vote_count = vote_count + 1
		WHERE node_id = $1 AND target_level = $2`

	_, err := r.pool.Exec(ctx, query, nodeID, level)
	if err != nil {
		return fmt.Errorf("failed to increment vote: %w", err)
	}

	return nil
}

func (r *progressionRepository) EndVoting(ctx context.Context, nodeID int, level int) error {
	query := `
		UPDATE progression_voting
		SET is_active = false
		WHERE node_id = $1 AND target_level = $2`

	_, err := r.pool.Exec(ctx, query, nodeID, level)
	if err != nil {
		return fmt.Errorf("failed to end voting: %w", err)
	}

	return nil
}

// User vote tracking

func (r *progressionRepository) HasUserVoted(ctx context.Context, userID string, nodeID int, level int) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_votes
			WHERE user_id = $1 AND node_id = $2 AND target_level = $3
		)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, userID, nodeID, level).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if user voted: %w", err)
	}

	return exists, nil
}

func (r *progressionRepository) RecordUserVote(ctx context.Context, userID string, nodeID int, level int) error {
	query := `
		INSERT INTO user_votes (user_id, node_id, target_level)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, node_id, target_level) DO NOTHING`

	_, err := r.pool.Exec(ctx, query, userID, nodeID, level)
	if err != nil {
		return fmt.Errorf("failed to record user vote: %w", err)
	}

	return nil
}

// User progression

func (r *progressionRepository) UnlockUserProgression(ctx context.Context, userID string, progressionType string, key string, metadata map[string]interface{}) error {
	var metadataJSON []byte
	var err error

	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO user_progression (user_id, progression_type, progression_key, metadata)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, progression_type, progression_key) DO NOTHING`

	_, err = r.pool.Exec(ctx, query, userID, progressionType, key, metadataJSON)
	if err != nil {
		return fmt.Errorf("failed to unlock user progression: %w", err)
	}

	return nil
}

func (r *progressionRepository) IsUserProgressionUnlocked(ctx context.Context, userID string, progressionType string, key string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_progression
			WHERE user_id = $1 AND progression_type = $2 AND progression_key = $3
		)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, userID, progressionType, key).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user progression: %w", err)
	}

	return exists, nil
}

func (r *progressionRepository) GetUserProgressions(ctx context.Context, userID string, progressionType string) ([]*domain.UserProgression, error) {
	query := `
		SELECT user_id, progression_type, progression_key, unlocked_at, metadata
		FROM user_progression
		WHERE user_id = $1 AND progression_type = $2
		ORDER BY unlocked_at`

	rows, err := r.pool.Query(ctx, query, userID, progressionType)
	if err != nil {
		return nil, fmt.Errorf("failed to query user progressions: %w", err)
	}
	defer rows.Close()

	var progressions []*domain.UserProgression
	for rows.Next() {
		var prog domain.UserProgression
		var metadataJSON []byte

		err := rows.Scan(
			&prog.UserID, &prog.ProgressionType, &prog.ProgressionKey,
			&prog.UnlockedAt, &metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user progression: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &prog.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		progressions = append(progressions, &prog)
	}

	return progressions, rows.Err()
}

// Engagement tracking

func (r *progressionRepository) RecordEngagement(ctx context.Context, metric *domain.EngagementMetric) error {
	var metadataJSON []byte
	var err error

	if metric.Metadata != nil {
		metadataJSON, err = json.Marshal(metric.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO engagement_metrics (user_id, metric_type, metric_value, metadata)
		VALUES ($1, $2, $3, $4)`

	_, err = r.pool.Exec(ctx, query, metric.UserID, metric.MetricType, metric.MetricValue, metadataJSON)
	if err != nil {
		return fmt.Errorf("failed to record engagement: %w", err)
	}

	return nil
}

func (r *progressionRepository) GetEngagementScore(ctx context.Context, since *time.Time) (int, error) {
	// Get weights
	weights, err := r.GetEngagementWeights(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get weights: %w", err)
	}

	// Build query
	var query string
	var args []interface{}

	if since != nil {
		query = `
			SELECT metric_type, SUM(metric_value) as total
			FROM engagement_metrics
			WHERE recorded_at >= $1
			GROUP BY metric_type`
		args = append(args, since)
	} else {
		query = `
			SELECT metric_type, SUM(metric_value) as total
			FROM engagement_metrics
			GROUP BY metric_type`
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to query engagement metrics: %w", err)
	}
	defer rows.Close()

	totalScore := 0
	for rows.Next() {
		var metricType string
		var total int

		if err := rows.Scan(&metricType, &total); err != nil {
			return 0, fmt.Errorf("failed to scan metric: %w", err)
		}

		weight := weights[metricType]
		totalScore += int(float64(total) * weight)
	}

	return totalScore, rows.Err()
}

func (r *progressionRepository) GetUserEngagement(ctx context.Context, userID string) (*domain.ContributionBreakdown, error) {
	query := `
		SELECT metric_type, SUM(metric_value) as total
		FROM engagement_metrics
		WHERE user_id = $1
		GROUP BY metric_type`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user engagement: %w", err)
	}
	defer rows.Close()

	breakdown := &domain.ContributionBreakdown{}
	weights, _ := r.GetEngagementWeights(ctx)

	for rows.Next() {
		var metricType string
		var total int

		if err := rows.Scan(&metricType, &total); err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}

		switch metricType {
		case "message":
			breakdown.MessagesSent = total
		case "command":
			breakdown.CommandsUsed = total
		case "item_crafted":
			breakdown.ItemsCrafted = total
		case "item_used":
			breakdown.ItemsUsed = total
		}

		weight := weights[metricType]
		breakdown.TotalScore += int(float64(total) * weight)
	}

	return breakdown, rows.Err()
}

func (r *progressionRepository) GetEngagementWeights(ctx context.Context) (map[string]float64, error) {
	query := `SELECT metric_type, weight FROM engagement_weights`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query engagement weights: %w", err)
	}
	defer rows.Close()

	weights := make(map[string]float64)
	for rows.Next() {
		var metricType string
		var weight float64

		if err := rows.Scan(&metricType, &weight); err != nil {
			return nil, fmt.Errorf("failed to scan weight: %w", err)
		}

		weights[metricType] = weight
	}

	// Default weights if none found
	if len(weights) == 0 {
		weights = map[string]float64{
			"message":      1.0,
			"command":      2.0,
			"item_crafted": 3.0,
			"item_used":    1.5,
		}
	}

	return weights, rows.Err()
}

// Reset operations

func (r *progressionRepository) ResetTree(ctx context.Context, resetBy string, reason string, preserveUserData bool) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer SafeRollback(ctx, tx)

	// Count nodes for reset record
	var nodeCount int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM progression_unlocks").Scan(&nodeCount)
	if err != nil {
		return fmt.Errorf("failed to count unlocks: %w", err)
	}

	// Get engagement score
	engagementScore := 0
	err = tx.QueryRow(ctx, "SELECT COALESCE(SUM(metric_value), 0) FROM engagement_metrics").Scan(&engagementScore)
	if err != nil {
		return fmt.Errorf("failed to get engagement score: %w", err)
	}

	// Record reset
	_, err = tx.Exec(ctx, `
		INSERT INTO progression_resets (reset_by, reason, nodes_reset_count, engagement_score_at_reset)
		VALUES ($1, $2, $3, $4)`,
		resetBy, reason, nodeCount, engagementScore)
	if err != nil {
		return fmt.Errorf("failed to record reset: %w", err)
	}

	// Clear unlocks (except root)
	_, err = tx.Exec(ctx, `
		DELETE FROM progression_unlocks
		WHERE node_id != (SELECT id FROM progression_nodes WHERE node_key = 'progression_system')`)
	if err != nil {
		return fmt.Errorf("failed to clear unlocks: %w", err)
	}

	// Clear voting
	_, err = tx.Exec(ctx, "DELETE FROM progression_voting")
	if err != nil {
		return fmt.Errorf("failed to clear voting: %w", err)
	}

	// Clear user votes
	_, err = tx.Exec(ctx, "DELETE FROM user_votes")
	if err != nil {
		return fmt.Errorf("failed to clear user votes: %w", err)
	}

	// Optionally preserve user progression
	if !preserveUserData {
		_, err = tx.Exec(ctx, "DELETE FROM user_progression")
		if err != nil {
			return fmt.Errorf("failed to clear user progression: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *progressionRepository) RecordReset(ctx context.Context, reset *domain.ProgressionReset) error {
	query := `
		INSERT INTO progression_resets (reset_by, reason, nodes_reset_count, engagement_score_at_reset)
		VALUES ($1, $2, $3, $4)`

	_, err := r.pool.Exec(ctx, query, reset.ResetBy, reset.Reason, reset.NodesResetCount, reset.EngagementScoreAtReset)
	if err != nil {
		return fmt.Errorf("failed to record reset: %w", err)
	}

	return nil
}

// Transaction support

func (r *progressionRepository) BeginTx(ctx context.Context) (repository.Tx, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &progressionTx{tx: tx}, nil
}

// progressionTx implements repository.Tx for progression operations
type progressionTx struct {
	tx pgx.Tx
}

func (t *progressionTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *progressionTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

// GetInventory is required by repository.Tx but not used in progression
func (t *progressionTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return nil, fmt.Errorf("inventory operations not supported in progression transactions")
}

// UpdateInventory is required by repository.Tx but not used in progression
func (t *progressionTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return fmt.Errorf("inventory operations not supported in progression transactions")
}
