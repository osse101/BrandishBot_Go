package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// GambleRepository implements the gamble repository for PostgreSQL
type GambleRepository struct {
	*UserRepository
	db *pgxpool.Pool
	q  *generated.Queries
}

// NewGambleRepository creates a new GambleRepository
func NewGambleRepository(db *pgxpool.Pool) *GambleRepository {
	return &GambleRepository{
		UserRepository: NewUserRepository(db),
		db:             db,
		q:              generated.New(db),
	}
}

// CreateGamble inserts a new gamble record
func (r *GambleRepository) CreateGamble(ctx context.Context, gamble *domain.Gamble) error {
	initiatorID, err := uuid.Parse(gamble.InitiatorID)
	if err != nil {
		return fmt.Errorf("invalid initiator id: %w", err)
	}

	params := generated.CreateGambleParams{
		ID:           gamble.ID,
		InitiatorID:  initiatorID,
		State:        string(gamble.State),
		CreatedAt:    pgtype.Timestamptz{Time: gamble.CreatedAt, Valid: true},
		JoinDeadline: pgtype.Timestamptz{Time: gamble.JoinDeadline, Valid: true},
	}

	err = r.q.CreateGamble(ctx, params)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return domain.ErrGambleAlreadyActive
		}
		return fmt.Errorf("failed to create gamble: %w", err)
	}
	return nil
}

// GetGamble retrieves a gamble by ID, including participants
func (r *GambleRepository) GetGamble(ctx context.Context, id uuid.UUID) (*domain.Gamble, error) {
	// Get Gamble
	g, err := r.q.GetGamble(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get gamble: %w", err)
	}

	gamble := &domain.Gamble{
		ID:           g.ID,
		InitiatorID:  g.InitiatorID.String(),
		State:        domain.GambleState(g.State),
		CreatedAt:    g.CreatedAt.Time,
		JoinDeadline: g.JoinDeadline.Time,
	}

	// Get Participants
	participants, err := r.q.GetGambleParticipants(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	for _, p := range participants {
		var bets []domain.LootboxBet
		if err := json.Unmarshal(p.LootboxBets, &bets); err != nil {
			return nil, fmt.Errorf("failed to unmarshal bets: %w", err)
		}

		gamble.Participants = append(gamble.Participants, domain.Participant{
			GambleID:    p.GambleID,
			UserID:      p.UserID.String(),
			LootboxBets: bets,
			Username:    p.Username,
		})
	}

	return gamble, nil
}

// JoinGamble adds a participant to a gamble
func (r *GambleRepository) JoinGamble(ctx context.Context, participant *domain.Participant) error {
	userID, err := uuid.Parse(participant.UserID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	betsBytes, err := json.Marshal(participant.LootboxBets)
	if err != nil {
		return fmt.Errorf("failed to marshal bets: %w", err)
	}

	params := generated.JoinGambleParams{
		GambleID:    participant.GambleID,
		UserID:      userID,
		LootboxBets: betsBytes,
	}

	err = r.q.JoinGamble(ctx, params)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return domain.ErrUserAlreadyJoined
		}
		return fmt.Errorf("failed to join gamble: %w", err)
	}
	return nil
}

// UpdateGambleState updates the state of a gamble
func (r *GambleRepository) UpdateGambleState(ctx context.Context, id uuid.UUID, state domain.GambleState) error {
	params := generated.UpdateGambleStateParams{
		State: string(state),
		ID:    id,
	}
	err := r.q.UpdateGambleState(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to update gamble state: %w", err)
	}
	return nil
}

// UpdateGambleStateIfMatches performs a compare-and-swap operation on gamble state
// Returns the number of rows affected (0 if state didn't match, 1 if updated)
// this prevents Bug #4: duplicate execution of gambles
func (r *GambleRepository) UpdateGambleStateIfMatches(
	ctx context.Context,
	id uuid.UUID,
	expectedState, newState domain.GambleState,
) (int64, error) {
	params := generated.UpdateGambleStateIfMatchesParams{
		State:   string(newState),
		ID:      id,
		State_2: string(expectedState),
	}
	result, err := r.q.UpdateGambleStateIfMatches(ctx, params)
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

	// Transaction is preferred for atomicity
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx for saving items: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			// Just log locally since we can't easily get a logger here without dependency injection changes
			fmt.Printf("failed to rollback tx: %v\n", err)
		}
	}()

	q := r.q.WithTx(tx)

	for _, item := range items {
		userID, err := uuid.Parse(item.UserID)
		if err != nil {
			return fmt.Errorf("invalid user id: %w", err)
		}

		params := generated.SaveOpenedItemParams{
			GambleID: pgtype.UUID{Bytes: item.GambleID, Valid: true},
			UserID:   pgtype.UUID{Bytes: userID, Valid: true},
			ItemID:   pgtype.Int4{Int32: int32(item.ItemID), Valid: true},
			Value:    item.Value,
		}

		err = q.SaveOpenedItem(ctx, params)
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
	g, err := r.q.GetActiveGamble(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active gamble: %w", err)
	}

	return &domain.Gamble{
		ID:           g.ID,
		InitiatorID:  g.InitiatorID.String(),
		State:        domain.GambleState(g.State),
		CreatedAt:    g.CreatedAt.Time,
		JoinDeadline: g.JoinDeadline.Time,
	}, nil
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
		q:        r.q.WithTx(tx),
	}, nil
}

// gambleTx implements repository.GambleTx interface
type gambleTx struct {
	tx       pgx.Tx
	userRepo *UserRepository
	q        *generated.Queries
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
	params := generated.UpdateGambleStateIfMatchesParams{
		State:   string(newState),
		ID:      id,
		State_2: string(expectedState),
	}
	result, err := t.q.UpdateGambleStateIfMatches(ctx, params)
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

	for _, item := range items {
		userID, err := uuid.Parse(item.UserID)
		if err != nil {
			return fmt.Errorf("invalid user id: %w", err)
		}

		params := generated.SaveOpenedItemParams{
			GambleID: pgtype.UUID{Bytes: item.GambleID, Valid: true},
			UserID:   pgtype.UUID{Bytes: userID, Valid: true},
			ItemID:   pgtype.Int4{Int32: int32(item.ItemID), Valid: true},
			Value:    item.Value,
		}
		err = t.q.SaveOpenedItem(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to insert opened item: %w", err)
		}
	}

	return nil
}

// CompleteGamble marks gamble as completed within transaction
func (t *gambleTx) CompleteGamble(ctx context.Context, result *domain.GambleResult) error {
	params := generated.UpdateGambleStateParams{
		State: string(domain.GambleStateCompleted),
		ID:    result.GambleID,
	}
	err := t.q.UpdateGambleState(ctx, params)
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
