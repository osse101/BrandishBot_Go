package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Duel defines the interface for duel data access
type Duel interface {
	CreateDuel(ctx context.Context, duel *domain.Duel) error
	GetDuel(ctx context.Context, id uuid.UUID) (*domain.Duel, error)
	UpdateDuelState(ctx context.Context, id uuid.UUID, state domain.DuelState) error
	GetPendingDuelsForUser(ctx context.Context, userID uuid.UUID) ([]domain.Duel, error)
	AcceptDuel(ctx context.Context, id uuid.UUID, result *domain.DuelResult) error
	DeclineDuel(ctx context.Context, id uuid.UUID) error
	ExpireDuels(ctx context.Context) error

	// Transaction support
	BeginTx(ctx context.Context) (Tx, error)
	BeginDuelTx(ctx context.Context) (DuelTx, error)

	// User operations
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
}

// DuelTx extends Tx with duel-specific transactional operations
type DuelTx interface {
	Tx // Commit, Rollback

	// Duel operations within transaction
	GetDuel(ctx context.Context, id uuid.UUID) (*domain.Duel, error)
	UpdateDuelState(ctx context.Context, id uuid.UUID, state domain.DuelState) error
	AcceptDuel(ctx context.Context, id uuid.UUID, result *domain.DuelResult) error
}
