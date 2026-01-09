package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// EconomyRepository implements the economy repository for PostgreSQL
type EconomyRepository struct {
	db *pgxpool.Pool
	q  *generated.Queries
}

// NewEconomyRepository creates a new EconomyRepository
func NewEconomyRepository(db *pgxpool.Pool) *EconomyRepository {
	return &EconomyRepository{
		db: db,
		q:  generated.New(db),
	}
}

// EconomyTx implements repository.EconomyTx
type EconomyTx struct {
	tx pgx.Tx
	q  *generated.Queries
}

// BeginTx starts a new transaction
func (r *EconomyRepository) BeginTx(ctx context.Context) (repository.EconomyTx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &EconomyTx{
		tx: tx,
		q:  r.q.WithTx(tx),
	}, nil
}

// Commit commits the transaction
func (t *EconomyTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// Rollback rolls back the transaction
func (t *EconomyTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

// GetUserByPlatformID finds a user by their platform-specific ID
func (r *EconomyRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	row, err := r.q.GetUserByPlatformID(ctx, generated.GetUserByPlatformIDParams{
		Name:           platform,
		PlatformUserID: platformID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user core data: %w", err)
	}
	return mapUserAndLinks(ctx, r.q, row.UserID, row.Username)
}

// GetItemByName retrieves an item by its internal name
func (r *EconomyRepository) GetItemByName(ctx context.Context, itemName string) (*domain.Item, error) {
	row, err := r.q.GetItemByName(ctx, itemName)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Return nil if item not found
		}
		return nil, fmt.Errorf("failed to get item by name: %w", err)
	}

	return &domain.Item{
		ID:             int(row.ItemID),
		InternalName:   row.InternalName,
		PublicName:     row.PublicName.String,
		DefaultDisplay: row.DefaultDisplay.String,
		Description:    row.ItemDescription.String,
		BaseValue:      int(row.BaseValue.Int32),
		Handler:        textToPtr(row.Handler),
		Types:          row.Types,
	}, nil
}

// GetInventory retrieves the user's inventory
func (r *EconomyRepository) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return getInventory(ctx, r.q, userID)
}

// UpdateInventory updates the user's inventory
func (r *EconomyRepository) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return updateInventory(ctx, r.q, userID, inventory)
}

// GetInventory for Tx
func (t *EconomyTx) GetInventory(ctx context.Context, userID string) (*domain.Inventory, error) {
	return getInventory(ctx, t.q, userID)
}

// UpdateInventory for Tx
func (t *EconomyTx) UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error {
	return updateInventory(ctx, t.q, userID, inventory)
}

// GetSellablePrices retrieves all sellable items with their prices
func (r *EconomyRepository) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	rows, err := r.q.GetSellablePrices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query sellable items: %w", err)
	}

	var items []domain.Item
	for _, row := range rows {
		items = append(items, domain.Item{
			ID:           int(row.ItemID),
			InternalName: row.InternalName,
			Description:  row.ItemDescription.String,
			BaseValue:    int(row.BaseValue.Int32),
		})
	}

	return items, nil
}

// IsItemBuyable checks if an item has the 'buyable' type
func (r *EconomyRepository) IsItemBuyable(ctx context.Context, itemName string) (bool, error) {
	return r.q.IsItemBuyable(ctx, itemName)
}

// GetBuyablePrices retrieves all buyable items with their prices
func (r *EconomyRepository) GetBuyablePrices(ctx context.Context) ([]domain.Item, error) {
	rows, err := r.q.GetBuyablePrices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query buyable items: %w", err)
	}

	var items []domain.Item
	for _, row := range rows {
		items = append(items, domain.Item{
			ID:           int(row.ItemID),
			InternalName: row.InternalName,
			Description:  row.ItemDescription.String,
			BaseValue:    int(row.BaseValue.Int32),
		})
	}

	return items, nil
}
