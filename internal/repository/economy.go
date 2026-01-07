package repository

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Economy defines the interface for economy persistence
type Economy interface {
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	GetItemByName(ctx context.Context, itemName string) (*domain.Item, error)
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
	GetSellablePrices(ctx context.Context) ([]domain.Item, error)
	IsItemBuyable(ctx context.Context, itemName string) (bool, error)
	GetBuyablePrices(ctx context.Context) ([]domain.Item, error)
	BeginTx(ctx context.Context) (EconomyTx, error)
}

// EconomyTx defines the interface for economy transactions
type EconomyTx interface {
	Tx
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
}
