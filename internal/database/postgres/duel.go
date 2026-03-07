package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

type duelRepository struct {
	db *pgxpool.Pool
	q  *generated.Queries
}

func NewDuelRepository(db *pgxpool.Pool) repository.Duel {
	return &duelRepository{
		db: db,
		q:  generated.New(db),
	}
}

// Map generated.Duel to domain.Duel
func mapDuel(row generated.Duel) (*domain.Duel, error) {
	stakes, err := domain.UnmarshalStakes(row.Stakes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal stakes: %w", err)
	}

	var resultData *domain.DuelResult
	if len(row.ResultData) > 0 {
		resultData, err = domain.UnmarshalDuelResult(row.ResultData)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal result data: %w", err)
		}
	}

	var opponentID *uuid.UUID
	if row.OpponentID.Valid {
		id := uuid.UUID(row.OpponentID.Bytes)
		opponentID = &id
	}

	var winnerID *uuid.UUID
	if row.WinnerID.Valid {
		id := uuid.UUID(row.WinnerID.Bytes)
		winnerID = &id
	}

	return &domain.Duel{
		ID:           row.ID,
		ChallengerID: row.ChallengerID,
		OpponentID:   opponentID,
		State:        domain.DuelState(row.State),
		Stakes:       *stakes,
		CreatedAt:    row.CreatedAt.Time,
		ExpiresAt:    row.ExpiresAt.Time,
		StartedAt:    ptrTimestamptz(row.StartedAt),
		CompletedAt:  ptrTimestamptz(row.CompletedAt),
		WinnerID:     winnerID,
		ResultData:   resultData,
	}, nil
}

func ptrTimestamptz(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func (r *duelRepository) CreateDuel(ctx context.Context, duel *domain.Duel) error {
	stakes, err := domain.MarshalStakes(duel.Stakes)
	if err != nil {
		return fmt.Errorf("failed to marshal stakes: %w", err)
	}

	var opponentID pgtype.UUID
	if duel.OpponentID != nil {
		opponentID = pgtype.UUID{Bytes: *duel.OpponentID, Valid: true}
	}

	err = r.q.CreateDuel(ctx, generated.CreateDuelParams{
		ID:           duel.ID,
		ChallengerID: duel.ChallengerID,
		OpponentID:   opponentID,
		State:        string(duel.State),
		Stakes:       stakes,
		CreatedAt:    pgtype.Timestamptz{Time: duel.CreatedAt, Valid: true},
		ExpiresAt:    pgtype.Timestamptz{Time: duel.ExpiresAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to create duel: %w", err)
	}
	return nil
}

func (r *duelRepository) GetDuel(ctx context.Context, id uuid.UUID) (*domain.Duel, error) {
	row, err := r.q.GetDuel(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get duel: %w", err)
	}
	return mapDuel(row)
}

func (r *duelRepository) UpdateDuelState(ctx context.Context, id uuid.UUID, state domain.DuelState) error {
	err := r.q.UpdateDuelState(ctx, generated.UpdateDuelStateParams{
		ID:    id,
		State: string(state),
	})
	if err != nil {
		return fmt.Errorf("failed to update duel state: %w", err)
	}
	return nil
}

func (r *duelRepository) GetPendingDuelsForUser(ctx context.Context, userID uuid.UUID) ([]domain.Duel, error) {
	rows, err := r.q.GetPendingDuelsForUser(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get pending duels: %w", err)
	}

	duels := make([]domain.Duel, 0, len(rows))
	for _, row := range rows {
		d, err := mapDuel(row)
		if err != nil {
			return nil, err
		}
		duels = append(duels, *d)
	}
	return duels, nil
}

func (r *duelRepository) AcceptDuel(ctx context.Context, id uuid.UUID, result *domain.DuelResult) error {
	resultData, err := domain.MarshalDuelResult(*result)
	if err != nil {
		return fmt.Errorf("failed to marshal duel result: %w", err)
	}

	err = r.q.AcceptDuel(ctx, generated.AcceptDuelParams{
		ID:         id,
		WinnerID:   pgtype.UUID{Bytes: result.WinnerID, Valid: true},
		ResultData: resultData,
	})
	if err != nil {
		return fmt.Errorf("failed to accept duel: %w", err)
	}
	return nil
}

func (r *duelRepository) DeclineDuel(ctx context.Context, id uuid.UUID) error {
	err := r.q.DeclineDuel(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to decline duel: %w", err)
	}
	return nil
}

func (r *duelRepository) ExpireDuels(ctx context.Context) error {
	err := r.q.ExpireDuels(ctx)
	if err != nil {
		return fmt.Errorf("failed to expire duels: %w", err)
	}
	return nil
}

func (r *duelRepository) BeginTx(ctx context.Context) (repository.Tx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

type duelTx struct {
	tx pgx.Tx
	q  *generated.Queries
}

func (t *duelTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *duelTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

func (t *duelTx) GetDuel(ctx context.Context, id uuid.UUID) (*domain.Duel, error) {
	row, err := t.q.GetDuelForUpdate(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get duel for update: %w", err)
	}
	return mapDuel(row)
}

func (t *duelTx) UpdateDuelState(ctx context.Context, id uuid.UUID, state domain.DuelState) error {
	err := t.q.UpdateDuelState(ctx, generated.UpdateDuelStateParams{
		ID:    id,
		State: string(state),
	})
	if err != nil {
		return fmt.Errorf("failed to update duel state in tx: %w", err)
	}
	return nil
}

func (t *duelTx) AcceptDuel(ctx context.Context, id uuid.UUID, result *domain.DuelResult) error {
	resultData, err := domain.MarshalDuelResult(*result)
	if err != nil {
		return fmt.Errorf("failed to marshal duel result in tx: %w", err)
	}

	err = t.q.AcceptDuel(ctx, generated.AcceptDuelParams{
		ID:         id,
		WinnerID:   pgtype.UUID{Bytes: result.WinnerID, Valid: true},
		ResultData: resultData,
	})
	if err != nil {
		return fmt.Errorf("failed to accept duel in tx: %w", err)
	}
	return nil
}

func (r *duelRepository) BeginDuelTx(ctx context.Context) (repository.DuelTx, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin duel tx: %w", err)
	}
	return &duelTx{
		tx: tx,
		q:  r.q.WithTx(tx),
	}, nil
}

func (r *duelRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	user, err := r.q.GetUserByPlatformID(ctx, generated.GetUserByPlatformIDParams{
		Name:           platform,
		PlatformUserID: platformID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user by platform id: %w", err)
	}

	uid, err := uuid.Parse(user.UserID.String())
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	return mapUserAndLinks(ctx, r.q, uid, user.Username)
}
