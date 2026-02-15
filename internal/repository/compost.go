package repository

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// CompostRepository defines the interface for compost data access
type CompostRepository interface {
	GetBin(ctx context.Context, userID string) (*domain.CompostBin, error)
	CreateBin(ctx context.Context, userID string) (*domain.CompostBin, error)
	GetAllItems(ctx context.Context) ([]domain.Item, error)
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	BeginTx(ctx context.Context) (CompostTx, error)
}

// CompostTx defines transactional compost operations
type CompostTx interface {
	Tx // Commit, Rollback
	GetBinForUpdate(ctx context.Context, userID string) (*domain.CompostBin, error)
	UpdateBin(ctx context.Context, bin *domain.CompostBin) error
	ResetBin(ctx context.Context, userID string) error
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inv domain.Inventory) error
}
