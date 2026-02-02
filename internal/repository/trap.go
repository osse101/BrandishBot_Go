package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// TrapRepository handles trap persistence
type TrapRepository interface {
	// CreateTrap creates a new trap record
	CreateTrap(ctx context.Context, trap *domain.Trap) error

	// GetActiveTrap returns the active trap for a target user (nil if none)
	GetActiveTrap(ctx context.Context, targetID uuid.UUID) (*domain.Trap, error)

	// GetActiveTrapForUpdate returns the active trap with SELECT FOR UPDATE lock
	GetActiveTrapForUpdate(ctx context.Context, targetID uuid.UUID) (*domain.Trap, error)

	// TriggerTrap marks a trap as triggered
	TriggerTrap(ctx context.Context, trapID uuid.UUID) error

	// GetTrapsByUser returns traps placed by a user (for stats)
	GetTrapsByUser(ctx context.Context, setterID uuid.UUID, limit int) ([]*domain.Trap, error)

	// GetTriggeredTrapsForTarget returns trap history for a target (for stats)
	GetTriggeredTrapsForTarget(ctx context.Context, targetID uuid.UUID, limit int) ([]*domain.Trap, error)

	// CleanupStaleTraps removes untriggered traps older than daysOld
	CleanupStaleTraps(ctx context.Context, daysOld int) (int, error)
}
