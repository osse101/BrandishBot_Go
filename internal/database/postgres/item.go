package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// ItemRepository implements repository.Item for PostgreSQL using sqlc
type ItemRepository struct {
	pool *pgxpool.Pool
	q    *generated.Queries
}

// NewItemRepository creates a new ItemRepository
func NewItemRepository(pool *pgxpool.Pool) repository.Item {
	return &ItemRepository{
		pool: pool,
		q:    generated.New(pool),
	}
}

// GetAllItems retrieves all items from the database
func (r *ItemRepository) GetAllItems(ctx context.Context) ([]domain.Item, error) {
	rows, err := r.q.GetAllItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all items: %w", err)
	}

	items := make([]domain.Item, len(rows))
	for i, row := range rows {
		items[i] = domain.Item{
			ID:             int(row.ItemID),
			InternalName:   row.InternalName,
			PublicName:     row.PublicName.String,
			DefaultDisplay: row.DefaultDisplay.String,
			Description:    row.ItemDescription.String,
			BaseValue:      int(row.BaseValue.Int32),
			Handler:        textToPtr(row.Handler),
			Types:          row.Types,
			ContentType:    row.ContentType,
		}
	}

	return items, nil
}

// GetItemByID retrieves an item by ID
func (r *ItemRepository) GetItemByID(ctx context.Context, id int) (*domain.Item, error) {
	row, err := r.q.GetItemByID(ctx, int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get item: %w", err)
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
		ContentType:    row.ContentType,
	}, nil
}

// GetItemByInternalName retrieves an item by internal name
func (r *ItemRepository) GetItemByInternalName(ctx context.Context, internalName string) (*domain.Item, error) {
	row, err := r.q.GetItemByInternalName(ctx, internalName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get item: %w", err)
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
		ContentType:    row.ContentType,
	}, nil
}

// InsertItem inserts a new item into the database
func (r *ItemRepository) InsertItem(ctx context.Context, item *domain.Item) (int, error) {
	params := generated.InsertItemParams{
		InternalName:    item.InternalName,
		PublicName:      strToText(item.PublicName),
		DefaultDisplay:  strToText(item.DefaultDisplay),
		ItemDescription: strToText(item.Description),
		BaseValue:       intToInt4(item.BaseValue),
		Handler:         ptrToText(item.Handler),
		ContentType:     item.ContentType,
	}

	itemID, err := r.q.InsertItem(ctx, params)
	if err != nil {
		return 0, fmt.Errorf("failed to insert item: %w", err)
	}

	return int(itemID), nil
}

// UpdateItem updates an existing item in the database
func (r *ItemRepository) UpdateItem(ctx context.Context, itemID int, item *domain.Item) error {
	params := generated.UpdateItemParams{
		PublicName:      strToText(item.PublicName),
		DefaultDisplay:  strToText(item.DefaultDisplay),
		ItemDescription: strToText(item.Description),
		BaseValue:       intToInt4(item.BaseValue),
		Handler:         ptrToText(item.Handler),
		ContentType:     item.ContentType,
		ItemID:          int32(itemID),
	}

	if err := r.q.UpdateItem(ctx, params); err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	return nil
}

// GetAllItemTypes retrieves all item types from the database
func (r *ItemRepository) GetAllItemTypes(ctx context.Context) ([]domain.ItemType, error) {
	rows, err := r.q.GetAllItemTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all item types: %w", err)
	}

	types := make([]domain.ItemType, len(rows))
	for i, row := range rows {
		types[i] = domain.ItemType{
			ID:   int(row.ItemTypeID),
			Name: row.TypeName,
		}
	}

	return types, nil
}

// InsertItemType inserts a new item type and returns its ID
func (r *ItemRepository) InsertItemType(ctx context.Context, typeName string) (int, error) {
	typeID, err := r.q.InsertItemType(ctx, typeName)
	if err != nil {
		return 0, fmt.Errorf("failed to insert item type: %w", err)
	}

	return int(typeID), nil
}

// ClearItemTags removes all tags for an item
func (r *ItemRepository) ClearItemTags(ctx context.Context, itemID int) error {
	if err := r.q.ClearItemTags(ctx, int32(itemID)); err != nil {
		return fmt.Errorf("failed to clear item tags: %w", err)
	}

	return nil
}

// AssignItemTag assigns a tag to an item
func (r *ItemRepository) AssignItemTag(ctx context.Context, itemID, typeID int) error {
	params := generated.AssignItemTagParams{
		ItemID:     int32(itemID),
		ItemTypeID: int32(typeID),
	}

	if err := r.q.AssignItemTag(ctx, params); err != nil {
		return fmt.Errorf("failed to assign item tag: %w", err)
	}

	return nil
}

// GetSyncMetadata retrieves sync metadata for a config file
func (r *ItemRepository) GetSyncMetadata(ctx context.Context, configName string) (*domain.SyncMetadata, error) {
	row, err := r.q.GetSyncMetadata(ctx, configName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New(ErrMsgSyncMetadataNotFound)
		}
		return nil, fmt.Errorf("failed to get sync metadata: %w", err)
	}

	return &domain.SyncMetadata{
		ConfigName:   row.ConfigName,
		LastSyncTime: row.LastSyncTime.Time,
		FileHash:     row.FileHash,
		FileModTime:  row.FileModTime.Time,
	}, nil
}

// UpsertSyncMetadata inserts or updates sync metadata for a config file
func (r *ItemRepository) UpsertSyncMetadata(ctx context.Context, metadata *domain.SyncMetadata) error {
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
