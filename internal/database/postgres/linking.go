package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/linking"
)

// LinkingRepository implements linking.Repository
type LinkingRepository struct {
	db *pgxpool.Pool
}

// NewLinkingRepository creates a new linking repository
func NewLinkingRepository(db *pgxpool.Pool) *LinkingRepository {
	return &LinkingRepository{db: db}
}

// CreateToken creates a new link token
func (r *LinkingRepository) CreateToken(ctx context.Context, token *linking.LinkToken) error {
	query := `
		INSERT INTO link_tokens (token, source_platform, source_platform_id, state, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, query,
		token.Token,
		token.SourcePlatform,
		token.SourcePlatformID,
		token.State,
		token.CreatedAt,
		token.ExpiresAt,
	)
	return err
}

// GetToken retrieves a link token by its string value
func (r *LinkingRepository) GetToken(ctx context.Context, tokenStr string) (*linking.LinkToken, error) {
	query := `
		SELECT token, source_platform, source_platform_id, 
		       COALESCE(target_platform, ''), COALESCE(target_platform_id, ''),
		       state, created_at, expires_at
		FROM link_tokens
		WHERE token = $1
	`
	var token linking.LinkToken
	err := r.db.QueryRow(ctx, query, tokenStr).Scan(
		&token.Token,
		&token.SourcePlatform,
		&token.SourcePlatformID,
		&token.TargetPlatform,
		&token.TargetPlatformID,
		&token.State,
		&token.CreatedAt,
		&token.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}
	return &token, nil
}

// UpdateToken updates a link token
func (r *LinkingRepository) UpdateToken(ctx context.Context, token *linking.LinkToken) error {
	query := `
		UPDATE link_tokens
		SET target_platform = $2, target_platform_id = $3, state = $4
		WHERE token = $1
	`
	_, err := r.db.Exec(ctx, query,
		token.Token,
		token.TargetPlatform,
		token.TargetPlatformID,
		token.State,
	)
	return err
}

// InvalidateTokensForSource marks all pending/claimed tokens for a source as expired
func (r *LinkingRepository) InvalidateTokensForSource(ctx context.Context, platform, platformID string) error {
	query := `
		UPDATE link_tokens
		SET state = 'expired'
		WHERE source_platform = $1 AND source_platform_id = $2 AND state IN ('pending', 'claimed')
	`
	_, err := r.db.Exec(ctx, query, platform, platformID)
	return err
}

// CleanupExpired removes expired tokens older than 1 hour
func (r *LinkingRepository) CleanupExpired(ctx context.Context) error {
	query := `
		DELETE FROM link_tokens
		WHERE expires_at < NOW() - INTERVAL '1 hour'
	`
	_, err := r.db.Exec(ctx, query)
	return err
}

// GetClaimedTokenForSource finds a claimed token for confirmation
func (r *LinkingRepository) GetClaimedTokenForSource(ctx context.Context, platform, platformID string) (*linking.LinkToken, error) {
	query := `
		SELECT token, source_platform, source_platform_id, 
		       COALESCE(target_platform, ''), COALESCE(target_platform_id, ''),
		       state, created_at, expires_at
		FROM link_tokens
		WHERE source_platform = $1 AND source_platform_id = $2 AND state = 'claimed'
		ORDER BY created_at DESC
		LIMIT 1
	`
	var token linking.LinkToken
	err := r.db.QueryRow(ctx, query, platform, platformID).Scan(
		&token.Token,
		&token.SourcePlatform,
		&token.SourcePlatformID,
		&token.TargetPlatform,
		&token.TargetPlatformID,
		&token.State,
		&token.CreatedAt,
		&token.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("no claimed token found: %w", err)
	}
	return &token, nil
}
