package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// HarvestRepository implements the harvest repository for PostgreSQL
type HarvestRepository struct {
	*UserRepository
	db *pgxpool.Pool
	q  *generated.Queries
}

// NewHarvestRepository creates a new harvest repository
func NewHarvestRepository(db *pgxpool.Pool) *HarvestRepository {
	return &HarvestRepository{
		UserRepository: NewUserRepository(db),
		db:             db,
		q:              generated.New(db),
	}
}

// GetHarvestState retrieves the harvest state for a user
func (r *HarvestRepository) GetHarvestState(ctx context.Context, userID string) (*domain.HarvestState, error) {
	state, err := fetchHarvestState(ctx, userID, r.q.GetHarvestState)
	if err != nil {
		if errors.Is(err, domain.ErrHarvestStateNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get harvest state: %w", err)
	}
	return state, nil
}

// CreateHarvestState initializes harvest state for a new user
func (r *HarvestRepository) CreateHarvestState(ctx context.Context, userID string) (*domain.HarvestState, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	state, err := r.q.CreateHarvestState(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to create harvest state: %w", err)
	}

	return &domain.HarvestState{
		UserID:          state.UserID.String(),
		LastHarvestedAt: state.LastHarvestedAt.Time,
		CreatedAt:       state.CreatedAt.Time,
		UpdatedAt:       state.UpdatedAt.Time,
	}, nil
}

// BeginTx starts a transaction and returns a HarvestTx
func (r *HarvestRepository) BeginTx(ctx context.Context) (repository.HarvestTx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin harvest transaction: %w", err)
	}
	return &harvestTx{
		tx:       tx,
		userRepo: r.UserRepository,
		q:        r.q.WithTx(tx),
	}, nil
}

// harvestTx implements repository.HarvestTx
type harvestTx struct {
	tx       pgx.Tx
	userRepo *UserRepository
	q        *generated.Queries
}

// Commit commits the transaction
func (t *harvestTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// Rollback rolls back the transaction
func (t *harvestTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

// GetHarvestStateWithLock retrieves the harvest state with FOR UPDATE lock
func (t *harvestTx) GetHarvestStateWithLock(ctx context.Context, userID string) (*domain.HarvestState, error) {
	state, err := fetchHarvestState(ctx, userID, t.q.GetHarvestStateWithLock)
	if err != nil {
		if errors.Is(err, domain.ErrHarvestStateNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get harvest state with lock: %w", err)
	}
	return state, nil
}

// UpdateHarvestState updates the last harvested timestamp
func (t *harvestTx) UpdateHarvestState(ctx context.Context, userID string, lastHarvestedAt time.Time) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	return t.q.UpdateHarvestState(ctx, generated.UpdateHarvestStateParams{
		Column1:         userUUID,
		LastHarvestedAt: pgtype.Timestamptz{Time: lastHarvestedAt, Valid: true},
	})
}

// GetInventory retrieves a user's inventory within transaction
func (t *harvestTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return getInventoryForUpdate(ctx, t.q, userID)
}

// UpdateInventory updates a user's inventory within transaction
func (t *harvestTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return updateInventory(ctx, t.q, userID, inventory)
}

// fetchHarvestState is a helper to fetch and map harvest state with common logic
func fetchHarvestState(ctx context.Context, userID string, fetcher func(context.Context, uuid.UUID) (generated.HarvestState, error)) (*domain.HarvestState, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	state, err := fetcher(ctx, userUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrHarvestStateNotFound
		}
		return nil, err
	}

	return &domain.HarvestState{
		UserID:          state.UserID.String(),
		LastHarvestedAt: state.LastHarvestedAt.Time,
		CreatedAt:       state.CreatedAt.Time,
		UpdatedAt:       state.UpdatedAt.Time,
	}, nil
}
