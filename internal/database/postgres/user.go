package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// UserRepository implements the user repository for PostgreSQL
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// UpsertUser inserts a new user or updates existing platform IDs if the user exists
func (r *UserRepository) UpsertUser(ctx context.Context, user domain.User) error {
	query := `
		INSERT INTO users (id, username, twitch_id, youtube_id, discord_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE
		SET
			username = EXCLUDED.username,
			twitch_id = COALESCE(NULLIF(EXCLUDED.twitch_id, ''), users.twitch_id),
			youtube_id = COALESCE(NULLIF(EXCLUDED.youtube_id, ''), users.youtube_id),
			discord_id = COALESCE(NULLIF(EXCLUDED.discord_id, ''), users.discord_id),
			updated_at = NOW()
	`

	_, err := r.db.Exec(ctx, query, user.ID, user.Username, user.TwitchID, user.YoutubeID, user.DiscordID)
	if err != nil {
		return fmt.Errorf("failed to upsert user: %w", err)
	}

	return nil
}

// GetUserByPlatformID finds a user by their platform-specific ID
func (r *UserRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	var query string
	switch platform {
	case "twitch":
		query = `SELECT id, username, twitch_id, youtube_id, discord_id FROM users WHERE twitch_id = $1`
	case "youtube":
		query = `SELECT id, username, twitch_id, youtube_id, discord_id FROM users WHERE youtube_id = $1`
	case "discord":
		query = `SELECT id, username, twitch_id, youtube_id, discord_id FROM users WHERE discord_id = $1`
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	var user domain.User
	var twitchID, youtubeID, discordID *string

	err := r.db.QueryRow(ctx, query, platformID).Scan(&user.ID, &user.Username, &twitchID, &youtubeID, &discordID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if twitchID != nil {
		user.TwitchID = *twitchID
	}
	if youtubeID != nil {
		user.YoutubeID = *youtubeID
	}
	if discordID != nil {
		user.DiscordID = *discordID
	}

	return &user, nil
}
