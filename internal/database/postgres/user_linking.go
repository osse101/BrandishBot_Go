package postgres

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// GetUserByID retrieves a user by internal ID
func (r *UserRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	query := `
		SELECT user_id, username, 
		       COALESCE(twitch_id, ''), COALESCE(youtube_id, ''), COALESCE(discord_id, ''),
		       created_at, updated_at
		FROM users
		WHERE user_id = $1
	`
	var user domain.User
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID, &user.Username,
		&user.TwitchID, &user.YoutubeID, &user.DiscordID,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &user, nil
}

// UpdateUser updates a user's platform IDs
func (r *UserRepository) UpdateUser(ctx context.Context, user domain.User) error {
	query := `
		UPDATE users
		SET twitch_id = NULLIF($2, ''),
		    youtube_id = NULLIF($3, ''),
		    discord_id = NULLIF($4, ''),
		    updated_at = NOW()
		WHERE user_id = $1
	`
	_, err := r.db.Exec(ctx, query,
		user.ID,
		user.TwitchID,
		user.YoutubeID,
		user.DiscordID,
	)
	return err
}

// DeleteUser deletes a user by ID
func (r *UserRepository) DeleteUser(ctx context.Context, userID string) error {
	query := `DELETE FROM users WHERE user_id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

// DeleteInventory deletes a user's inventory
func (r *UserRepository) DeleteInventory(ctx context.Context, userID string) error {
	query := `DELETE FROM user_inventory WHERE user_id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}
