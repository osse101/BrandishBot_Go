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
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// CompostRepository implements repository.CompostRepository
type CompostRepository struct {
	*UserRepository
	db *pgxpool.Pool
	q  *generated.Queries
}

// NewCompostRepository creates a new CompostRepository
func NewCompostRepository(db *pgxpool.Pool) *CompostRepository {
	return &CompostRepository{
		UserRepository: NewUserRepository(db),
		db:             db,
		q:              generated.New(db),
	}
}

// GetBin retrieves the compost bin for a user (returns nil, nil if not found)
func (r *CompostRepository) GetBin(ctx context.Context, userID string) (*domain.CompostBin, error) {
	uid, err := parseUserUUID(userID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetCompostBin(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get compost bin: %w", err)
	}
	return mapCompostBin(row)
}

// CreateBin creates a new compost bin for the user
func (r *CompostRepository) CreateBin(ctx context.Context, userID string) (*domain.CompostBin, error) {
	uid, err := parseUserUUID(userID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateCompostBin(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to create compost bin: %w", err)
	}
	return mapCompostBin(row)
}

// GetAllItems returns all items with their content types (delegates to embedded UserRepository)
func (r *CompostRepository) GetAllItems(ctx context.Context) ([]domain.Item, error) {
	return r.UserRepository.GetAllItems(ctx)
}

// BeginTx starts a transaction and returns a CompostTx
func (r *CompostRepository) BeginTx(ctx context.Context) (repository.CompostTx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin compost transaction: %w", err)
	}
	return &compostTx{
		tx: tx,
		q:  r.q.WithTx(tx),
	}, nil
}

// compostTx implements repository.CompostTx
type compostTx struct {
	tx pgx.Tx
	q  *generated.Queries
}

func (t *compostTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *compostTx) Rollback(ctx context.Context) error {
	err := t.tx.Rollback(ctx)
	if errors.Is(err, pgx.ErrTxClosed) {
		return fmt.Errorf("%w: %w", repository.ErrTxClosed, err)
	}
	return err
}

func (t *compostTx) GetBinForUpdate(ctx context.Context, userID string) (*domain.CompostBin, error) {
	uid, err := parseUserUUID(userID)
	if err != nil {
		return nil, err
	}
	row, err := t.q.GetCompostBinForUpdate(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get compost bin for update: %w", err)
	}
	return mapCompostBin(row)
}

func (t *compostTx) UpdateBin(ctx context.Context, bin *domain.CompostBin) error {
	uid, err := parseUserUUID(bin.UserID)
	if err != nil {
		return err
	}
	itemsJSON, err := json.Marshal(bin.Items)
	if err != nil {
		return fmt.Errorf("failed to marshal bin items: %w", err)
	}
	return t.q.UpdateCompostBin(ctx, generated.UpdateCompostBinParams{
		UserID:       uid,
		Status:       string(bin.Status),
		Items:        itemsJSON,
		ItemCount:    int32(bin.ItemCount),
		StartedAt:    timeToPgtimetz(bin.StartedAt),
		ReadyAt:      timeToPgtimetz(bin.ReadyAt),
		SludgeAt:     timeToPgtimetz(bin.SludgeAt),
		InputValue:   int32(bin.InputValue),
		DominantType: bin.DominantType,
	})
}

func (t *compostTx) ResetBin(ctx context.Context, userID string) error {
	uid, err := parseUserUUID(userID)
	if err != nil {
		return err
	}
	return t.q.ResetCompostBin(ctx, uid)
}

func (t *compostTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return getInventoryForUpdate(ctx, t.q, userID)
}

func (t *compostTx) UpdateInventory(ctx context.Context, userID string, inv domain.Inventory) error {
	return updateInventory(ctx, t.q, userID, inv)
}

// mapCompostBin converts a generated.CompostBin to domain.CompostBin
func mapCompostBin(row generated.CompostBin) (*domain.CompostBin, error) {
	var items []domain.CompostBinItem
	if err := json.Unmarshal(row.Items, &items); err != nil {
		items = []domain.CompostBinItem{}
	}
	if items == nil {
		items = []domain.CompostBinItem{}
	}
	return &domain.CompostBin{
		ID:           row.ID.String(),
		UserID:       row.UserID.String(),
		Status:       domain.CompostBinStatus(row.Status),
		Capacity:     int(row.Capacity),
		Items:        items,
		ItemCount:    int(row.ItemCount),
		StartedAt:    pgtimetzToPtr(row.StartedAt),
		ReadyAt:      pgtimetzToPtr(row.ReadyAt),
		SludgeAt:     pgtimetzToPtr(row.SludgeAt),
		InputValue:   int(row.InputValue),
		DominantType: row.DominantType,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}, nil
}

// timeToPgtimetz converts *time.Time to pgtype.Timestamptz
func timeToPgtimetz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// pgtimetzToPtr converts pgtype.Timestamptz to *time.Time
func pgtimetzToPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
