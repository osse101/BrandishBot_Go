package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// LinkingRepository implements repository.Linking
type LinkingRepository struct {
	pool *pgxpool.Pool
	q    *generated.Queries
}

// NewLinkingRepository creates a new linking repository
func NewLinkingRepository(pool *pgxpool.Pool) *LinkingRepository {
	return &LinkingRepository{
		pool: pool,
		q:    generated.New(pool),
	}
}

// CreateToken creates a new link token
func (r *LinkingRepository) CreateToken(ctx context.Context, token *repository.LinkToken) error {
	err := r.q.CreateToken(ctx, generated.CreateTokenParams{
		Token:            token.Token,
		SourcePlatform:   token.SourcePlatform,
		SourcePlatformID: token.SourcePlatformID,
		State:            pgtype.Text{String: token.State, Valid: token.State != ""},
		CreatedAt:        pgtype.Timestamptz{Time: token.CreatedAt, Valid: true},
		ExpiresAt:        pgtype.Timestamptz{Time: token.ExpiresAt, Valid: true},
	})
	return err
}

// GetToken retrieves a link token by its string value
func (r *LinkingRepository) GetToken(ctx context.Context, tokenStr string) (*repository.LinkToken, error) {
	row, err := r.q.GetToken(ctx, tokenStr)
	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}

	return &repository.LinkToken{
		Token:            row.Token,
		SourcePlatform:   row.SourcePlatform,
		SourcePlatformID: row.SourcePlatformID,
		TargetPlatform:   row.TargetPlatform,
		TargetPlatformID: row.TargetPlatformID,
		State:            row.State.String,
		CreatedAt:        row.CreatedAt.Time,
		ExpiresAt:        row.ExpiresAt.Time,
	}, nil
}

// UpdateToken updates a link token
func (r *LinkingRepository) UpdateToken(ctx context.Context, token *repository.LinkToken) error {
	err := r.q.UpdateToken(ctx, generated.UpdateTokenParams{
		Token:            token.Token,
		TargetPlatform:   pgtype.Text{String: token.TargetPlatform, Valid: token.TargetPlatform != ""},
		TargetPlatformID: pgtype.Text{String: token.TargetPlatformID, Valid: token.TargetPlatformID != ""},
		State:            pgtype.Text{String: token.State, Valid: token.State != ""},
	})
	return err
}

// InvalidateTokensForSource marks all pending/claimed tokens for a source as expired
func (r *LinkingRepository) InvalidateTokensForSource(ctx context.Context, platform, platformID string) error {
	return r.q.InvalidateTokensForSource(ctx, generated.InvalidateTokensForSourceParams{
		SourcePlatform:   platform,
		SourcePlatformID: platformID,
	})
}

// CleanupExpired removes expired tokens older than 1 hour
func (r *LinkingRepository) CleanupExpired(ctx context.Context) error {
	return r.q.CleanupExpiredTokens(ctx)
}

// GetClaimedTokenForSource finds a claimed token for confirmation
func (r *LinkingRepository) GetClaimedTokenForSource(ctx context.Context, platform, platformID string) (*repository.LinkToken, error) {
	row, err := r.q.GetClaimedTokenForSource(ctx, generated.GetClaimedTokenForSourceParams{
		SourcePlatform:   platform,
		SourcePlatformID: platformID,
	})
	if err != nil {
		return nil, fmt.Errorf("no claimed token found: %w", err)
	}

	return &repository.LinkToken{
		Token:            row.Token,
		SourcePlatform:   row.SourcePlatform,
		SourcePlatformID: row.SourcePlatformID,
		TargetPlatform:   row.TargetPlatform,
		TargetPlatformID: row.TargetPlatformID,
		State:            row.State.String,
		CreatedAt:        row.CreatedAt.Time,
		ExpiresAt:        row.ExpiresAt.Time,
	}, nil
}
