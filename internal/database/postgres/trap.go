package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// TrapRepository implements the trap repository for PostgreSQL
type TrapRepository struct {
	db *pgxpool.Pool
	q  *generated.Queries
}

// NewTrapRepository creates a new TrapRepository
func NewTrapRepository(db *pgxpool.Pool) *TrapRepository {
	return &TrapRepository{
		db: db,
		q:  generated.New(db),
	}
}

// CreateTrap creates a new trap record
func (r *TrapRepository) CreateTrap(ctx context.Context, trap *domain.Trap) error {
	params := generated.CreateTrapParams{
		ID:             trap.ID,
		SetterID:       trap.SetterID,
		TargetID:       trap.TargetID,
		ShineLevel:     string(trap.ShineLevel),
		TimeoutSeconds: int32(trap.TimeoutSeconds),
		PlacedAt:       pgtype.Timestamptz{Time: trap.PlacedAt, Valid: true},
	}

	row, err := r.q.CreateTrap(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to create trap: %w", err)
	}

	// Update the trap with the returned values
	trap.ID = row.ID
	trap.PlacedAt = row.PlacedAt.Time
	return nil
}

// GetActiveTrap returns the active trap for a target user (nil if none)
func (r *TrapRepository) GetActiveTrap(ctx context.Context, targetID uuid.UUID) (*domain.Trap, error) {
	row, err := r.q.GetActiveTrap(ctx, targetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active trap: %w", err)
	}

	return r.mapTrap(row), nil
}

// GetActiveTrapForUpdate returns the active trap with SELECT FOR UPDATE lock
func (r *TrapRepository) GetActiveTrapForUpdate(ctx context.Context, targetID uuid.UUID) (*domain.Trap, error) {
	row, err := r.q.GetActiveTrapForUpdate(ctx, targetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active trap with lock: %w", err)
	}

	return r.mapTrap(row), nil
}

// TriggerTrap marks a trap as triggered
func (r *TrapRepository) TriggerTrap(ctx context.Context, trapID uuid.UUID) error {
	err := r.q.TriggerTrap(ctx, trapID)
	if err != nil {
		return fmt.Errorf("failed to trigger trap: %w", err)
	}
	return nil
}

// GetTrapsByUser returns traps placed by a user
func (r *TrapRepository) GetTrapsByUser(ctx context.Context, setterID uuid.UUID, limit int) ([]*domain.Trap, error) {
	rows, err := r.q.GetTrapsByUser(ctx, generated.GetTrapsByUserParams{
		SetterID: setterID,
		Limit:    int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get traps by user: %w", err)
	}

	traps := make([]*domain.Trap, 0, len(rows))
	for _, row := range rows {
		traps = append(traps, r.mapTrap(row))
	}

	return traps, nil
}

// GetTriggeredTrapsForTarget returns trap history for a target
func (r *TrapRepository) GetTriggeredTrapsForTarget(ctx context.Context, targetID uuid.UUID, limit int) ([]*domain.Trap, error) {
	rows, err := r.q.GetTriggeredTrapsForTarget(ctx, generated.GetTriggeredTrapsForTargetParams{
		TargetID: targetID,
		Limit:    int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get triggered traps for target: %w", err)
	}

	traps := make([]*domain.Trap, 0, len(rows))
	for _, row := range rows {
		traps = append(traps, r.mapTrap(row))
	}

	return traps, nil
}

// CleanupStaleTraps removes untriggered traps older than daysOld
func (r *TrapRepository) CleanupStaleTraps(ctx context.Context, daysOld int) (int, error) {
	err := r.q.CleanupStaleTraps(ctx, int32(daysOld))
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup stale traps: %w", err)
	}

	// CleanupStaleTraps doesn't return row count, so we return 0
	// In production, you'd want to modify the query to return the count
	return 0, nil
}

// mapTrap converts a generated UserTrap row to a domain Trap
func (r *TrapRepository) mapTrap(row generated.UserTrap) *domain.Trap {
	trap := &domain.Trap{
		ID:             row.ID,
		SetterID:       row.SetterID,
		TargetID:       row.TargetID,
		ShineLevel:     domain.ShineLevel(row.ShineLevel),
		TimeoutSeconds: int(row.TimeoutSeconds),
		PlacedAt:       row.PlacedAt.Time,
	}

	if row.TriggeredAt.Valid {
		trap.TriggeredAt = &row.TriggeredAt.Time
	}

	return trap
}
