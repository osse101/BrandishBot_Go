package repository

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Item defines the interface for item configuration persistence
type Item interface {
	// Item operations
	GetAllItems(ctx context.Context) ([]domain.Item, error)
	GetItemByID(ctx context.Context, id int) (*domain.Item, error)
	GetItemByInternalName(ctx context.Context, internalName string) (*domain.Item, error)
	InsertItem(ctx context.Context, item *domain.Item) (int, error)
	UpdateItem(ctx context.Context, itemID int, item *domain.Item) error

	// Item type operations
	GetAllItemTypes(ctx context.Context) ([]domain.ItemType, error)
	InsertItemType(ctx context.Context, typeName string) (int, error)
	ClearItemTags(ctx context.Context, itemID int) error
	AssignItemTag(ctx context.Context, itemID, typeID int) error

	// Sync metadata operations
	GetSyncMetadata(ctx context.Context, configName string) (*domain.SyncMetadata, error)
	UpsertSyncMetadata(ctx context.Context, metadata *domain.SyncMetadata) error
}
