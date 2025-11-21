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

// UpsertUser inserts a new user or updates existing user and their platform links
func (r *UserRepository) UpsertUser(ctx context.Context, user *domain.User) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Upsert User Core Data
	var userID string
	if user.ID == "" {
		// Insert new user
		query := `
			INSERT INTO users (username, created_at, updated_at)
			VALUES ($1, NOW(), NOW())
			RETURNING user_id
		`
		err := tx.QueryRow(ctx, query, user.Username).Scan(&userID)
		if err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
		}
		user.ID = userID
	} else {
		// Update existing user
		query := `
			UPDATE users 
			SET username = $1, updated_at = NOW()
			WHERE user_id = $2
		`
		_, err := tx.Exec(ctx, query, user.Username, user.ID)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
		userID = user.ID
	}

	// 2. Upsert Platform Links
	platforms := map[string]string{
		"twitch":  user.TwitchID,
		"youtube": user.YoutubeID,
		"discord": user.DiscordID,
	}

	for platformName, externalID := range platforms {
		if externalID == "" {
			continue
		}

		// Get Platform ID
		var platformID int
		err := tx.QueryRow(ctx, "SELECT platform_id FROM platforms WHERE platform_name = $1", platformName).Scan(&platformID)
		if err != nil {
			return fmt.Errorf("failed to get platform id for %s: %w", platformName, err)
		}

		// Upsert Link
		linkQuery := `
			INSERT INTO user_platform_links (user_id, platform_id, external_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, platform_id) DO UPDATE
			SET external_id = EXCLUDED.external_id
		`
		_, err = tx.Exec(ctx, linkQuery, userID, platformID, externalID)
		if err != nil {
			return fmt.Errorf("failed to upsert link for %s: %w", platformName, err)
		}
	}

	return tx.Commit(ctx)
}

// GetUserByPlatformID finds a user by their platform-specific ID
// GetUserByPlatformID finds a user by their platform-specific ID
func (r *UserRepository) GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	// 1. Find User ID
	query := `
		SELECT u.user_id, u.username
		FROM users u
		JOIN user_platform_links upl ON u.user_id = upl.user_id
		JOIN platforms p ON upl.platform_id = p.platform_id
		WHERE p.platform_name = $1 AND upl.external_id = $2
	`
	var user domain.User
	err := r.db.QueryRow(ctx, query, platform, platformID).Scan(&user.ID, &user.Username)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user core data: %w", err)
	}

	// 2. Fetch all platform links for this user
	linksQuery := `
		SELECT p.platform_name, upl.external_id
		FROM user_platform_links upl
		JOIN platforms p ON upl.platform_id = p.platform_id
		WHERE upl.user_id = $1
	`
	rows, err := r.db.Query(ctx, linksQuery, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user links: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var pName, extID string
		if err := rows.Scan(&pName, &extID); err != nil {
			return nil, fmt.Errorf("failed to scan link: %w", err)
		}
		switch pName {
		case "twitch":
			user.TwitchID = extID
		case "youtube":
			user.YoutubeID = extID
		case "discord":
			user.DiscordID = extID
		}
	}

	return &user, nil
}
