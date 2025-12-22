package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// GambleRepository implements the gamble repository for PostgreSQL
type GambleRepository struct {
	*UserRepository
	db *pgxpool.Pool
}

// NewGambleRepository creates a new GambleRepository
func NewGambleRepository(db *pgxpool.Pool) *GambleRepository {
	return &GambleRepository{
		UserRepository: NewUserRepository(db),
		db:             db,
	}
}

// CreateGamble inserts a new gamble record
func (r *GambleRepository) CreateGamble(ctx context.Context, gamble *domain.Gamble) error {
	query := `
		INSERT INTO gambles (id, initiator_id, state, created_at, join_deadline)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(ctx, query, gamble.ID, gamble.InitiatorID, gamble.State, gamble.CreatedAt, gamble.JoinDeadline)
	if err != nil {
		return fmt.Errorf("failed to create gamble: %w", err)
	}
	return nil
}

// GetGamble retrieves a gamble by ID, including participants
func (r *GambleRepository) GetGamble(ctx context.Context, id uuid.UUID) (*domain.Gamble, error) {
	// Get Gamble
	query := `
		SELECT id, initiator_id, state, created_at, join_deadline
		FROM gambles
		WHERE id = $1
	`
	var gamble domain.Gamble
	err := r.db.QueryRow(ctx, query, id).Scan(
		&gamble.ID, &gamble.InitiatorID, &gamble.State, &gamble.CreatedAt, &gamble.JoinDeadline,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get gamble: %w", err)
	}

	// Get Participants
	partQuery := `
		SELECT p.gamble_id, p.user_id, p.lootbox_bets, u.username
		FROM gamble_participants p
		JOIN users u ON p.user_id = u.user_id
		WHERE p.gamble_id = $1
	`
	rows, err := r.db.Query(ctx, partQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p domain.Participant
		if err := rows.Scan(&p.GambleID, &p.UserID, &p.LootboxBets, &p.Username); err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}
		gamble.Participants = append(gamble.Participants, p)
	}

	return &gamble, nil
}

// JoinGamble adds a participant to a gamble
func (r *GambleRepository) JoinGamble(ctx context.Context, participant *domain.Participant) error {
	query := `
		INSERT INTO gamble_participants (gamble_id, user_id, lootbox_bets)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.Exec(ctx, query, participant.GambleID, participant.UserID, participant.LootboxBets)
	if err != nil {
		return fmt.Errorf("failed to join gamble: %w", err)
	}
	return nil
}

// UpdateGambleState updates the state of a gamble
func (r *GambleRepository) UpdateGambleState(ctx context.Context, id uuid.UUID, state domain.GambleState) error {
	query := `UPDATE gambles SET state = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, state, id)
	if err != nil {
		return fmt.Errorf("failed to update gamble state: %w", err)
	}
	return nil
}

// UpdateGambleStateIfMatches performs a compare-and-swap operation on gamble state
// Returns the number of rows affected (0 if state didn't match, 1 if updated)
// This prevents Bug #4: duplicate execution of gambles
func (r *GambleRepository) UpdateGambleStateIfMatches(
	ctx context.Context,
	id uuid.UUID,
	expectedState, newState domain.GambleState,
) (int64, error) {
	query := `
		UPDATE gambles 
		SET state = $1 
		WHERE id = $2 AND state = $3
	`
	result, err := r.db.Exec(ctx, query, newState, id, expectedState)
	if err != nil {
		return 0, fmt.Errorf("failed to update gamble state: %w", err)
	}
	return result.RowsAffected(), nil
}

// SaveOpenedItems saves the items opened during the gamble
func (r *GambleRepository) SaveOpenedItems(ctx context.Context, items []domain.GambleOpenedItem) error {
	if len(items) == 0 {
		return nil
	}

	// Bulk insert
	// Note: pgx CopyFrom is better for bulk, but for simplicity/consistency we'll use a loop or batched insert.
	// Given the expected scale (10 participants * 5 boxes = 50 items), a transaction with individual inserts is fine for v1.

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx for saving items: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO gamble_opened_items (gamble_id, user_id, item_id, value)
		VALUES ($1, $2, $3, $4)
	`

	for _, item := range items {
		_, err := tx.Exec(ctx, query, item.GambleID, item.UserID, item.ItemID, item.Value)
		if err != nil {
			return fmt.Errorf("failed to insert opened item: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// CompleteGamble marks the gamble as completed (state update is handled by UpdateGambleState, this might be redundant or for result storage?)
// The service calls UpdateGambleState separately.
// The interface has CompleteGamble(ctx, result).
// We should probably update state AND maybe store the result if we had a results table.
// For now, just update state to Completed.
func (r *GambleRepository) CompleteGamble(ctx context.Context, result *domain.GambleResult) error {
	return r.UpdateGambleState(ctx, result.GambleID, domain.GambleStateCompleted)
}

// GetActiveGamble retrieves the current active gamble (Joining or Opening)
func (r *GambleRepository) GetActiveGamble(ctx context.Context) (*domain.Gamble, error) {
	query := `
		SELECT id, initiator_id, state, created_at, join_deadline
		FROM gambles
		WHERE state IN ('Joining', 'Opening')
		LIMIT 1
	`
	var gamble domain.Gamble
	err := r.db.QueryRow(ctx, query).Scan(
		&gamble.ID, &gamble.InitiatorID, &gamble.State, &gamble.CreatedAt, &gamble.JoinDeadline,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active gamble: %w", err)
	}
	return &gamble, nil
}

// BeginGambleTx starts a transaction and returns a GambleTx for gamble operations
func (r *GambleRepository) BeginGambleTx(ctx context.Context) (repository.GambleTx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin gamble transaction: %w", err)
	}
	return &gambleTx{
		tx:       tx,
		userRepo: r.UserRepository,
	}, nil
}

// gambleTx implements repository.GambleTx interface
type gambleTx struct {
	tx       pgx.Tx
	userRepo *UserRepository
}

// Commit commits the transaction
func (t *gambleTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// Rollback rolls back the transaction
func (t *gambleTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

// UpdateGambleStateIfMatches performs CAS operation within transaction
func (t *gambleTx) UpdateGambleStateIfMatches(
	ctx context.Context,
	id uuid.UUID,
	expectedState, newState domain.GambleState,
) (int64, error) {
	query := `
		UPDATE gambles 
		SET state = $1 
		WHERE id = $2 AND state = $3
	`
	result, err := t.tx.Exec(ctx, query, newState, id, expectedState)
	if err != nil {
		return 0, fmt.Errorf("failed to update gamble state: %w", err)
	}
	return result.RowsAffected(), nil
}

// SaveOpenedItems saves opened items within transaction
func (t *gambleTx) SaveOpenedItems(ctx context.Context, items []domain.GambleOpenedItem) error {
	if len(items) == 0 {
		return nil
	}

	query := `
		INSERT INTO gamble_opened_items (gamble_id, user_id, item_id, value)
		VALUES ($1, $2, $3, $4)
	`

	for _, item := range items {
		_, err := t.tx.Exec(ctx, query, item.GambleID, item.UserID, item.ItemID, item.Value)
		if err != nil {
			return fmt.Errorf("failed to insert opened item: %w", err)
		}
	}

	return nil
}

// CompleteGamble marks gamble as completed within transaction
func (t *gambleTx) CompleteGamble(ctx context.Context, result *domain.GambleResult) error {
	query := `UPDATE gambles SET state = $1 WHERE id = $2`
	_, err := t.tx.Exec(ctx, query, domain.GambleStateCompleted, result.GambleID)
	if err != nil {
		return fmt.Errorf("failed to complete gamble: %w", err)
	}
	return nil
}

// GetInventory retrieves inventory within transaction
func (t *gambleTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	// Use UserTx wrapper for transactional inventory access with row locking
	userTx := &UserTx{tx: t.tx}
	return userTx.GetInventory(ctx, userID)
}

// UpdateInventory updates inventory within transaction
func (t *gambleTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	// Use UserTx wrapper for transactional inventory update
	userTx := &UserTx{tx: t.tx}
	return userTx.UpdateInventory(ctx, userID, inventory)
}
