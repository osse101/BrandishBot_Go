package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

type progressionRepository struct {
	pool *pgxpool.Pool
	q    *generated.Queries
}

// NewProgressionRepository creates a new Postgres-backed progression repository
func NewProgressionRepository(pool *pgxpool.Pool) progression.Repository {
	return &progressionRepository{
		pool: pool,
		q:    generated.New(pool),
	}
}

// Node operations

func (r *progressionRepository) GetNodeByKey(ctx context.Context, nodeKey string) (*domain.ProgressionNode, error) {
	node, err := r.q.GetNodeByKey(ctx, nodeKey)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get node by key: %w", err)
	}

	return &domain.ProgressionNode{
		ID:          int(node.ID),
		NodeKey:     node.NodeKey,
		NodeType:    node.NodeType,
		DisplayName: node.DisplayName,
		Description: node.Description.String,
		MaxLevel:    int(node.MaxLevel.Int32),
		UnlockCost:  int(node.UnlockCost.Int32),
		Tier:        int(node.Tier),
		Size:        node.Size,
		Category:    node.Category,
		SortOrder:   int(node.SortOrder.Int32),
		CreatedAt:   node.CreatedAt.Time,
	}, nil
}

func (r *progressionRepository) GetNodeByID(ctx context.Context, id int) (*domain.ProgressionNode, error) {
	node, err := r.q.GetNodeByID(ctx, int32(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get node by ID: %w", err)
	}

	return &domain.ProgressionNode{
		ID:          int(node.ID),
		NodeKey:     node.NodeKey,
		NodeType:    node.NodeType,
		DisplayName: node.DisplayName,
		Description: node.Description.String,
		MaxLevel:    int(node.MaxLevel.Int32),
		UnlockCost:  int(node.UnlockCost.Int32),
		Tier:        int(node.Tier),
		Size:        node.Size,
		Category:    node.Category,
		SortOrder:   int(node.SortOrder.Int32),
		CreatedAt:   node.CreatedAt.Time,
	}, nil
}

func (r *progressionRepository) GetAllNodes(ctx context.Context) ([]*domain.ProgressionNode, error) {
	rows, err := r.q.GetAllNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}

	var nodes []*domain.ProgressionNode
	for _, node := range rows {
		nodes = append(nodes, &domain.ProgressionNode{
			ID:          int(node.ID),
			NodeKey:     node.NodeKey,
			NodeType:    node.NodeType,
			DisplayName: node.DisplayName,
			Description: node.Description.String,
			MaxLevel:    int(node.MaxLevel.Int32),
			UnlockCost:  int(node.UnlockCost.Int32),
			Tier:        int(node.Tier),
			Size:        node.Size,
			Category:    node.Category,
			SortOrder:   int(node.SortOrder.Int32),
			CreatedAt:   node.CreatedAt.Time,
		})
	}

	return nodes, nil
}

// GetChildNodes is deprecated - use junction table queries for prerequisites instead
// Kept for backwards compatibility but will return empty since parent_node_id is removed
func (r *progressionRepository) GetChildNodes(ctx context.Context, parentID int) ([]*domain.ProgressionNode, error) {
	// Return empty slice - parent_node_id field removed in v2.0
	return []*domain.ProgressionNode{}, nil
}

// InsertNode inserts a new progression node and returns its ID
// Implements progression.NodeInserter interface
func (r *progressionRepository) InsertNode(ctx context.Context, node *domain.ProgressionNode) (int, error) {
	id, err := r.q.InsertNode(ctx, generated.InsertNodeParams{
		NodeKey:     node.NodeKey,
		NodeType:    node.NodeType,
		DisplayName: node.DisplayName,
		Description: pgtype.Text{String: node.Description, Valid: node.Description != ""},
		MaxLevel:    pgtype.Int4{Int32: int32(node.MaxLevel), Valid: true},
		UnlockCost:  pgtype.Int4{Int32: int32(node.UnlockCost), Valid: true},
		Tier:        int32(node.Tier),
		Size:        node.Size,
		Category:    node.Category,
		SortOrder:   pgtype.Int4{Int32: int32(node.SortOrder), Valid: true},
	})

	if err != nil {
		return 0, fmt.Errorf("failed to insert node: %w", err)
	}

	return int(id), nil
}

// UpdateNode updates an existing progression node
// Implements progression.NodeUpdater interface
func (r *progressionRepository) UpdateNode(ctx context.Context, nodeID int, node *domain.ProgressionNode) error {
	err := r.q.UpdateNode(ctx, generated.UpdateNodeParams{
		ID:          int32(nodeID),
		NodeType:    node.NodeType,
		DisplayName: node.DisplayName,
		Description: pgtype.Text{String: node.Description, Valid: node.Description != ""},
		MaxLevel:    pgtype.Int4{Int32: int32(node.MaxLevel), Valid: true},
		UnlockCost:  pgtype.Int4{Int32: int32(node.UnlockCost), Valid: true},
		Tier:        int32(node.Tier),
		Size:        node.Size,
		Category:    node.Category,
		SortOrder:   pgtype.Int4{Int32: int32(node.SortOrder), Valid: true},
	})

	if err != nil {
		return fmt.Errorf("failed to update node: %w", err)
	}

	return nil
}

// Unlock operations

func (r *progressionRepository) GetUnlock(ctx context.Context, nodeID int, level int) (*domain.ProgressionUnlock, error) {
	row, err := r.q.GetUnlock(ctx, generated.GetUnlockParams{
		NodeID:       pgtype.Int4{Int32: int32(nodeID), Valid: true},
		CurrentLevel: pgtype.Int4{Int32: int32(level), Valid: true},
	})

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get unlock: %w", err)
	}

	return &domain.ProgressionUnlock{
		ID:              int(row.ID),
		NodeID:          int(row.NodeID.Int32),
		CurrentLevel:    int(row.CurrentLevel.Int32),
		UnlockedAt:      row.UnlockedAt.Time,
		UnlockedBy:      row.UnlockedBy.String,
		EngagementScore: int(row.EngagementScore.Int32),
	}, nil
}

func (r *progressionRepository) GetAllUnlocks(ctx context.Context) ([]*domain.ProgressionUnlock, error) {
	rows, err := r.q.GetAllUnlocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query unlocks: %w", err)
	}

	var unlocks []*domain.ProgressionUnlock
	for _, row := range rows {
		unlocks = append(unlocks, &domain.ProgressionUnlock{
			ID:              int(row.ID),
			NodeID:          int(row.NodeID.Int32),
			CurrentLevel:    int(row.CurrentLevel.Int32),
			UnlockedAt:      row.UnlockedAt.Time,
			UnlockedBy:      row.UnlockedBy.String,
			EngagementScore: int(row.EngagementScore.Int32),
		})
	}

	return unlocks, nil
}

func (r *progressionRepository) IsNodeUnlocked(ctx context.Context, nodeKey string, level int) (bool, error) {
	return r.q.IsNodeUnlocked(ctx, generated.IsNodeUnlockedParams{
		NodeKey:      nodeKey,
		CurrentLevel: pgtype.Int4{Int32: int32(level), Valid: true},
	})
}

func (r *progressionRepository) UnlockNode(ctx context.Context, nodeID int, level int, unlockedBy string, engagementScore int) error {
	err := r.q.UnlockNode(ctx, generated.UnlockNodeParams{
		NodeID:          pgtype.Int4{Int32: int32(nodeID), Valid: true},
		CurrentLevel:    pgtype.Int4{Int32: int32(level), Valid: true},
		UnlockedBy:      pgtype.Text{String: unlockedBy, Valid: unlockedBy != ""},
		EngagementScore: pgtype.Int4{Int32: int32(engagementScore), Valid: true},
	})

	if err != nil {
		return fmt.Errorf("failed to unlock node: %w", err)
	}

	return nil
}

func (r *progressionRepository) RelockNode(ctx context.Context, nodeID int, level int) error {
	err := r.q.RelockNode(ctx, generated.RelockNodeParams{
		NodeID:       pgtype.Int4{Int32: int32(nodeID), Valid: true},
		CurrentLevel: pgtype.Int4{Int32: int32(level), Valid: true},
	})

	if err != nil {
		return fmt.Errorf("failed to relock node: %w", err)
	}

	return nil
}

// Voting operations

func (r *progressionRepository) GetActiveVoting(ctx context.Context) (*domain.ProgressionVoting, error) {
	row, err := r.q.GetActiveVoting(ctx)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active voting: %w", err)
	}

	voting := &domain.ProgressionVoting{
		ID:              int(row.ID),
		NodeID:          int(row.NodeID.Int32),
		TargetLevel:     int(row.TargetLevel.Int32),
		VoteCount:       int(row.VoteCount.Int32),
		VotingStartedAt: row.VotingStartedAt.Time,
		IsActive:        row.IsActive.Bool,
	}
	if row.VotingEndsAt.Valid {
		t := row.VotingEndsAt.Time
		voting.VotingEndsAt = &t
	}

	return voting, nil
}

func (r *progressionRepository) StartVoting(ctx context.Context, nodeID int, level int, endsAt *time.Time) error {
	var endsAtParams pgtype.Timestamp
	if endsAt != nil {
		endsAtParams = pgtype.Timestamp{Time: *endsAt, Valid: true}
	} else {
		endsAtParams = pgtype.Timestamp{Valid: false}
	}

	err := r.q.StartVoting(ctx, generated.StartVotingParams{
		NodeID:       pgtype.Int4{Int32: int32(nodeID), Valid: true},
		TargetLevel:  pgtype.Int4{Int32: int32(level), Valid: true},
		VotingEndsAt: endsAtParams,
	})

	if err != nil {
		return fmt.Errorf("failed to start voting: %w", err)
	}

	return nil
}

func (r *progressionRepository) GetVoting(ctx context.Context, nodeID int, level int) (*domain.ProgressionVoting, error) {
	row, err := r.q.GetVoting(ctx, generated.GetVotingParams{
		NodeID:      pgtype.Int4{Int32: int32(nodeID), Valid: true},
		TargetLevel: pgtype.Int4{Int32: int32(level), Valid: true},
	})

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get voting: %w", err)
	}

	voting := &domain.ProgressionVoting{
		ID:              int(row.ID),
		NodeID:          int(row.NodeID.Int32),
		TargetLevel:     int(row.TargetLevel.Int32),
		VoteCount:       int(row.VoteCount.Int32),
		VotingStartedAt: row.VotingStartedAt.Time,
		IsActive:        row.IsActive.Bool,
	}
	if row.VotingEndsAt.Valid {
		t := row.VotingEndsAt.Time
		voting.VotingEndsAt = &t
	}

	return voting, nil
}

func (r *progressionRepository) IncrementVote(ctx context.Context, nodeID int, level int) error {
	err := r.q.IncrementVote(ctx, generated.IncrementVoteParams{
		NodeID:      pgtype.Int4{Int32: int32(nodeID), Valid: true},
		TargetLevel: pgtype.Int4{Int32: int32(level), Valid: true},
	})

	if err != nil {
		return fmt.Errorf("failed to increment vote: %w", err)
	}

	return nil
}

func (r *progressionRepository) EndVoting(ctx context.Context, nodeID int, level int) error {
	err := r.q.EndVoting(ctx, generated.EndVotingParams{
		NodeID:      pgtype.Int4{Int32: int32(nodeID), Valid: true},
		TargetLevel: pgtype.Int4{Int32: int32(level), Valid: true},
	})

	if err != nil {
		return fmt.Errorf("failed to end voting: %w", err)
	}

	return nil
}

// User vote tracking

func (r *progressionRepository) HasUserVoted(ctx context.Context, userID string, nodeID int, level int) (bool, error) {
	return r.q.HasUserVoted(ctx, generated.HasUserVotedParams{
		UserID:      userID,
		NodeID:      int32(nodeID),
		TargetLevel: int32(level),
	})
}

func (r *progressionRepository) RecordUserVote(ctx context.Context, userID string, nodeID int, level int) error {
	err := r.q.RecordUserVote(ctx, generated.RecordUserVoteParams{
		UserID:      userID,
		NodeID:      int32(nodeID),
		TargetLevel: int32(level),
	})

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

	err = r.q.UnlockUserProgression(ctx, generated.UnlockUserProgressionParams{
		UserID:          userID,
		ProgressionType: progressionType,
		ProgressionKey:  key,
		Metadata:        metadataJSON,
	})

	if err != nil {
		return fmt.Errorf("failed to unlock user progression: %w", err)
	}

	return nil
}

func (r *progressionRepository) IsUserProgressionUnlocked(ctx context.Context, userID string, progressionType string, key string) (bool, error) {
	return r.q.IsUserProgressionUnlocked(ctx, generated.IsUserProgressionUnlockedParams{
		UserID:          userID,
		ProgressionType: progressionType,
		ProgressionKey:  key,
	})
}

func (r *progressionRepository) GetUserProgressions(ctx context.Context, userID string, progressionType string) ([]*domain.UserProgression, error) {
	rows, err := r.q.GetUserProgressions(ctx, generated.GetUserProgressionsParams{
		UserID:          userID,
		ProgressionType: progressionType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query user progressions: %w", err)
	}

	var progressions []*domain.UserProgression
	for _, row := range rows {
		prog := &domain.UserProgression{
			UserID:          row.UserID,
			ProgressionType: row.ProgressionType,
			ProgressionKey:  row.ProgressionKey,
			UnlockedAt:      row.UnlockedAt.Time,
		}

		if len(row.Metadata) > 0 {
			if err := json.Unmarshal(row.Metadata, &prog.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		progressions = append(progressions, prog)
	}

	return progressions, nil
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

	err = r.q.RecordEngagement(ctx, generated.RecordEngagementParams{
		UserID:      metric.UserID,
		MetricType:  metric.MetricType,
		MetricValue: pgtype.Int4{Int32: int32(metric.MetricValue), Valid: true},
		Metadata:    metadataJSON,
	})

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

	type metricRow struct {
		MetricType string
		Total      int64
	}
	var metrics []metricRow

	if since != nil {
		rowsSince, err := r.q.GetEngagementMetricsAggregatedSince(ctx, pgtype.Timestamp{Time: *since, Valid: true})
		if err != nil {
			return 0, fmt.Errorf("failed to query engagement metrics: %w", err)
		}
		for _, r := range rowsSince {
			metrics = append(metrics, metricRow{MetricType: r.MetricType, Total: r.Total})
		}
	} else {
		rowsAll, err := r.q.GetEngagementMetricsAggregated(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to query engagement metrics: %w", err)
		}
		for _, r := range rowsAll {
			metrics = append(metrics, metricRow{MetricType: r.MetricType, Total: r.Total})
		}
	}

	totalScore := 0
	for _, m := range metrics {
		weight := weights[m.MetricType]
		totalScore += int(float64(m.Total) * weight)
	}

	return totalScore, nil
}

func (r *progressionRepository) GetUserEngagement(ctx context.Context, userID string) (*domain.ContributionBreakdown, error) {
	rows, err := r.q.GetUserEngagementAggregated(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user engagement: %w", err)
	}

	breakdown := &domain.ContributionBreakdown{}
	weights, _ := r.GetEngagementWeights(ctx)

	for _, row := range rows {
		total := int(row.Total)
		switch row.MetricType {
		case "message":
			breakdown.MessagesSent = total
		case "command":
			breakdown.CommandsUsed = total
		case "item_crafted":
			breakdown.ItemsCrafted = total
		case "item_used":
			breakdown.ItemsUsed = total
		}

		weight := weights[row.MetricType]
		breakdown.TotalScore += int(float64(total) * weight)
	}

	return breakdown, nil
}

func (r *progressionRepository) GetEngagementWeights(ctx context.Context) (map[string]float64, error) {
	rows, err := r.q.GetEngagementWeights(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query engagement weights: %w", err)
	}

	weights := make(map[string]float64)
	for _, row := range rows {
		f, _ := row.Weight.Float64Value()
		weights[row.MetricType] = f.Float64
	}

	if len(weights) == 0 {
		weights = map[string]float64{
			"message":      1.0,
			"command":      2.0,
			"item_crafted": 3.0,
			"item_used":    1.5,
		}
	}

	return weights, nil
}

// Reset operations

func (r *progressionRepository) ResetTree(ctx context.Context, resetBy string, reason string, preserveUserData bool) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer SafeRollback(ctx, tx)

	q := r.q.WithTx(tx)

	// Count unlocks
	nodeCount, err := q.CountUnlocks(ctx)
	if err != nil {
		return fmt.Errorf("failed to count unlocks: %w", err)
	}

	// Get engagement score
	engagementScore, err := q.GetTotalEngagementScore(ctx)
	if err != nil {
		return fmt.Errorf("failed to get engagement score: %w", err)
	}

	// Record reset
	err = q.RecordReset(ctx, generated.RecordResetParams{
		ResetBy:                pgtype.Text{String: resetBy, Valid: resetBy != ""},
		Reason:                 pgtype.Text{String: reason, Valid: reason != ""},
		NodesResetCount:        pgtype.Int4{Int32: int32(nodeCount), Valid: true},
		EngagementScoreAtReset: pgtype.Int4{Int32: int32(engagementScore), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to record reset: %w", err)
	}

	// Clear unlocks (except root)
	if err := q.ClearUnlocksExceptRoot(ctx); err != nil {
		return fmt.Errorf("failed to clear unlocks: %w", err)
	}

	// Clear voting
	if err := q.ClearAllVoting(ctx); err != nil {
		return fmt.Errorf("failed to clear voting: %w", err)
	}

	// Clear user votes
	if err := q.ClearAllUserVotes(ctx); err != nil {
		return fmt.Errorf("failed to clear user votes: %w", err)
	}

	// Optionally preserve user progression
	if !preserveUserData {
		if err := q.ClearAllUserProgression(ctx); err != nil {
			return fmt.Errorf("failed to clear user progression: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *progressionRepository) RecordReset(ctx context.Context, reset *domain.ProgressionReset) error {
	err := r.q.RecordReset(ctx, generated.RecordResetParams{
		ResetBy:                pgtype.Text{String: reset.ResetBy, Valid: reset.ResetBy != ""},
		Reason:                 pgtype.Text{String: reset.Reason, Valid: reset.Reason != ""},
		NodesResetCount:        pgtype.Int4{Int32: int32(reset.NodesResetCount), Valid: true},
		EngagementScoreAtReset: pgtype.Int4{Int32: int32(reset.EngagementScoreAtReset), Valid: true},
	})

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

	return &progressionTx{
		tx: tx,
		q:  r.q.WithTx(tx),
	}, nil
}

// progressionTx implements repository.Tx for progression operations
type progressionTx struct {
	tx pgx.Tx
	q  *generated.Queries
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
