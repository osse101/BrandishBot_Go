package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

type progressionRepository struct {
	pool *pgxpool.Pool
	q    *generated.Queries
	bus  event.Bus
}

// NewProgressionRepository creates a new Postgres-backed progression repository
func NewProgressionRepository(pool *pgxpool.Pool, bus event.Bus) repository.Progression {
	return &progressionRepository{
		pool: pool,
		q:    generated.New(pool),
		bus:  bus,
	}
}

// Node operations

func (r *progressionRepository) GetNodeByKey(ctx context.Context, nodeKey string) (*domain.ProgressionNode, error) {
	node, err := r.q.GetNodeByKey(ctx, nodeKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get node by key: %w", err)
	}

	return mapProgressionNodeFields(node.ID, node.NodeKey, node.NodeType, node.DisplayName, node.Description, node.MaxLevel, node.UnlockCost, node.Tier, node.Size, node.Category, node.SortOrder, node.CreatedAt), nil
}

func (r *progressionRepository) GetNodeByID(ctx context.Context, id int) (*domain.ProgressionNode, error) {
	node, err := r.q.GetNodeByID(ctx, int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get node by ID: %w", err)
	}

	return mapProgressionNodeFields(node.ID, node.NodeKey, node.NodeType, node.DisplayName, node.Description, node.MaxLevel, node.UnlockCost, node.Tier, node.Size, node.Category, node.SortOrder, node.CreatedAt), nil
}

func (r *progressionRepository) GetAllNodes(ctx context.Context) ([]*domain.ProgressionNode, error) {
	rows, err := r.q.GetAllNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}

	nodes := make([]*domain.ProgressionNode, 0, len(rows))
	for _, node := range rows {
		nodes = append(nodes, mapProgressionNodeFields(node.ID, node.NodeKey, node.NodeType, node.DisplayName, node.Description, node.MaxLevel, node.UnlockCost, node.Tier, node.Size, node.Category, node.SortOrder, node.CreatedAt))
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
		if errors.Is(err, pgx.ErrNoRows) {
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

	unlocks := make([]*domain.ProgressionUnlock, 0, len(rows))
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

	// Publish event at data layer - ensures ALL unlock paths trigger cache invalidation
	if r.bus != nil {
		// Get node key for event payload
		node, _ := r.GetNodeByID(ctx, nodeID)
		nodeKey := ""
		if node != nil {
			nodeKey = node.NodeKey
		}

		if err := r.bus.Publish(ctx, event.Event{
			Type:    "progression.node_unlocked",
			Version: "1.0",
			Payload: map[string]interface{}{
				"node_id":  nodeID,
				"node_key": nodeKey,
				"level":    level,
				"source":   unlockedBy,
			},
		}); err != nil {
			// Log but don't fail - event publishing errors shouldn't block node unlocks
			logger.FromContext(ctx).Error("failed to publish node unlocked event", "error", err, "node_id", nodeID, "level", level)
		}
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

	// Clear any unlock progress records targeting this node to prevent stale state
	if err := r.q.ClearUnlockProgressForNode(ctx, pgtype.Int4{Int32: int32(nodeID), Valid: true}); err != nil {
		logger.FromContext(ctx).Warn("failed to clear unlock progress for relocked node", "error", err, "node_id", nodeID)
		// Don't fail the relock operation, just log
	}

	// Publish event at data layer
	if r.bus != nil {
		// Get node key for event payload
		node, _ := r.GetNodeByID(ctx, nodeID)
		nodeKey := ""
		if node != nil {
			nodeKey = node.NodeKey
		}

		if err := r.bus.Publish(ctx, event.Event{
			Type:    "progression.node_relocked",
			Version: "1.0",
			Payload: map[string]interface{}{
				"node_id":  nodeID,
				"node_key": nodeKey,
				"level":    level,
			},
		}); err != nil {
			// Log but don't fail - event publishing errors shouldn't block node relocks
			logger.FromContext(ctx).Error("failed to publish node relocked event", "error", err, "node_id", nodeID, "level", level)
		}
	}

	return nil
}

// Voting operations

func (r *progressionRepository) GetActiveVoting(ctx context.Context) (*domain.ProgressionVoting, error) {
	row, err := r.q.GetActiveVoting(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
		VotingEndsAt:    ptrTime(row.VotingEndsAt),
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
		if errors.Is(err, pgx.ErrNoRows) {
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
		VotingEndsAt:    ptrTime(row.VotingEndsAt),
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

	progressions := make([]*domain.UserProgression, 0, len(rows))
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
		RecordedAt:  pgtype.Timestamp{Time: metric.RecordedAt, Valid: !metric.RecordedAt.IsZero()},
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
		f, err := numericToFloat64(row.Weight)
		if err != nil {
			// Log warning but use default weight of 1.0 for this metric
			logger.FromContext(ctx).Warn("failed to convert weight for metric, using default", "metric", row.MetricType, "error", err)
			f = 1.0
		}
		weights[row.MetricType] = f
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
	h, err := beginTx(ctx, r.pool, r.q)
	if err != nil {
		return err
	}
	defer SafeRollback(ctx, h.Tx())

	q := h.Queries()

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

	// Clear user votes first (has FK to voting sessions)
	if err := q.ClearAllUserVotes(ctx); err != nil {
		return fmt.Errorf("failed to clear user votes: %w", err)
	}

	// Clear unlock progress (has FK to voting sessions)
	if err := q.ClearAllUnlockProgress(ctx); err != nil {
		return fmt.Errorf("failed to clear unlock progress: %w", err)
	}

	// Clear voting sessions (has FK to voting options via winning_option_id)
	if err := q.ClearAllVotingSessions(ctx); err != nil {
		return fmt.Errorf("failed to clear voting sessions: %w", err)
	}

	// Clear voting options (now safe after sessions cleared)
	if err := q.ClearAllVotingOptions(ctx); err != nil {
		return fmt.Errorf("failed to clear voting options: %w", err)
	}

	// Clear unlocks (except root)
	if err := q.ClearUnlocksExceptRoot(ctx); err != nil {
		return fmt.Errorf("failed to clear unlocks: %w", err)
	}

	// Clear voting (legacy table)
	if err := q.ClearAllVoting(ctx); err != nil {
		return fmt.Errorf("failed to clear voting: %w", err)
	}

	// Optionally preserve user progression
	if !preserveUserData {
		if err := q.ClearAllUserProgression(ctx); err != nil {
			return fmt.Errorf("failed to clear user progression: %w", err)
		}
	}

	return h.Commit(ctx)
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
	return nil, errors.New(ErrMsgInventoryOpsNotSupportedInProgression)
}

// UpdateInventory is required by repository.Tx but not used in progression
func (t *progressionTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return errors.New(ErrMsgInventoryOpsNotSupportedInProgression)
}

// GetNodeByFeatureKey retrieves a node by its modifier feature_key and returns the current unlock level
func (r *progressionRepository) GetNodeByFeatureKey(ctx context.Context, featureKey string) (*domain.ProgressionNode, int, error) {
	row, err := r.q.GetNodeByFeatureKey(ctx, []byte(featureKey))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get node by feature key: %w", err)
	}

	// Parse ModifierConfig
	var modifierConfig *domain.ModifierConfig
	if len(row.ModifierConfig) > 0 {
		modifierConfig = &domain.ModifierConfig{}
		if err := json.Unmarshal(row.ModifierConfig, modifierConfig); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal modifier config: %w", err)
		}
	}

	node := &domain.ProgressionNode{
		ID:             int(row.ID),
		NodeKey:        row.NodeKey,
		NodeType:       row.NodeType,
		DisplayName:    row.DisplayName,
		Description:    row.Description.String,
		MaxLevel:       int(row.MaxLevel.Int32),
		UnlockCost:     int(row.UnlockCost.Int32),
		Tier:           int(row.Tier),
		Size:           row.Size,
		Category:       row.Category,
		SortOrder:      int(row.SortOrder.Int32),
		CreatedAt:      row.CreatedAt.Time,
		ModifierConfig: modifierConfig,
	}

	return node, int(row.UnlockLevel), nil
}

func (r *progressionRepository) GetDailyEngagementTotals(ctx context.Context, since time.Time) (map[time.Time]int, error) {
	rows, err := r.q.GetDailyEngagementTotals(ctx, pgtype.Timestamp{Time: since, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to query daily totals: %w", err)
	}

	totals := make(map[time.Time]int)
	for _, row := range rows {
		if row.Day.Valid {
			totals[row.Day.Time] = int(row.TotalPoints)
		}
	}

	return totals, nil
}

func (r *progressionRepository) GetSyncMetadata(ctx context.Context, configName string) (*domain.SyncMetadata, error) {
	row, err := r.q.GetSyncMetadata(ctx, configName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New(ErrMsgSyncMetadataNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sync metadata: %w", err)
	}

	return &domain.SyncMetadata{
		ConfigName:   row.ConfigName,
		LastSyncTime: row.LastSyncTime.Time,
		FileHash:     row.FileHash,
		FileModTime:  row.FileModTime.Time,
	}, nil
}

func (r *progressionRepository) UpsertSyncMetadata(ctx context.Context, metadata *domain.SyncMetadata) error {
	params := generated.UpsertSyncMetadataParams{
		ConfigName:   metadata.ConfigName,
		LastSyncTime: pgtype.Timestamptz{Time: metadata.LastSyncTime, Valid: true},
		FileHash:     metadata.FileHash,
		FileModTime:  pgtype.Timestamptz{Time: metadata.FileModTime, Valid: true},
	}

	if err := r.q.UpsertSyncMetadata(ctx, params); err != nil {
		return fmt.Errorf("failed to upsert sync metadata: %w", err)
	}

	return nil
}

// Dynamic prerequisite operations

func (r *progressionRepository) CountUnlockedNodesBelowTier(ctx context.Context, tier int) (int, error) {
	count, err := r.q.CountUnlockedNodesBelowTier(ctx, int32(tier))
	if err != nil {
		return 0, fmt.Errorf("failed to count unlocked nodes below tier: %w", err)
	}
	return int(count), nil
}

func (r *progressionRepository) CountTotalUnlockedNodes(ctx context.Context) (int, error) {
	count, err := r.q.CountTotalUnlockedNodes(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count total unlocked nodes: %w", err)
	}
	return int(count), nil
}

func (r *progressionRepository) GetNodeDynamicPrerequisites(ctx context.Context, nodeID int) ([]byte, error) {
	data, err := r.q.GetNodeDynamicPrerequisites(ctx, int32(nodeID))
	if err != nil {
		return nil, fmt.Errorf("failed to get node dynamic prerequisites: %w", err)
	}
	return data, nil
}

func (r *progressionRepository) UpdateNodeDynamicPrerequisites(ctx context.Context, nodeID int, jsonData []byte) error {
	if err := r.q.UpdateNodeDynamicPrerequisites(ctx, generated.UpdateNodeDynamicPrerequisitesParams{
		ID:                   int32(nodeID),
		DynamicPrerequisites: jsonData,
	}); err != nil {
		return fmt.Errorf("failed to update node dynamic prerequisites: %w", err)
	}
	return nil
}
