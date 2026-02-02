package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Expedition defines the interface for expedition data access
type Expedition interface {
	CreateExpedition(ctx context.Context, expedition *domain.Expedition) error
	GetExpedition(ctx context.Context, id uuid.UUID) (*domain.ExpeditionDetails, error)
	AddParticipant(ctx context.Context, participant *domain.ExpeditionParticipant) error
	UpdateExpeditionState(ctx context.Context, id uuid.UUID, state domain.ExpeditionState) error
	GetActiveExpedition(ctx context.Context) (*domain.ExpeditionDetails, error)
	GetParticipants(ctx context.Context, expeditionID uuid.UUID) ([]domain.ExpeditionParticipant, error)
	SaveParticipantRewards(ctx context.Context, expeditionID uuid.UUID, userID uuid.UUID, rewards *domain.ExpeditionRewards) error
	CompleteExpedition(ctx context.Context, expeditionID uuid.UUID) error

	// Transaction support
	BeginTx(ctx context.Context) (Tx, error)
	BeginExpeditionTx(ctx context.Context) (ExpeditionTx, error)

	// User operations
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)

	// Inventory operations
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
}

// ExpeditionTx extends Tx with expedition-specific transactional operations
type ExpeditionTx interface {
	Tx // Commit, Rollback

	// Expedition operations within transaction
	GetExpedition(ctx context.Context, id uuid.UUID) (*domain.ExpeditionDetails, error)
	UpdateExpeditionState(ctx context.Context, id uuid.UUID, state domain.ExpeditionState) error
	GetParticipants(ctx context.Context, expeditionID uuid.UUID) ([]domain.ExpeditionParticipant, error)
	SaveParticipantRewards(ctx context.Context, expeditionID uuid.UUID, userID uuid.UUID, rewards *domain.ExpeditionRewards) error

	// Inventory operations within transaction
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
}
