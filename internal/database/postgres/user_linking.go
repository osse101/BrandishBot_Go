package postgres

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// GetUserByID retrieves a user by internal ID with all linked platform IDs
func (r *UserRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	query := `
		SELECT u.user_id, u.username, u.created_at, u.updated_at,
		       COALESCE(MAX(CASE WHEN p.name = 'twitch' THEN upl.platform_user_id END), '') as twitch_id,
		       COALESCE(MAX(CASE WHEN p.name = 'youtube' THEN upl.platform_user_id END), '') as youtube_id,
		       COALESCE(MAX(CASE WHEN p.name = 'discord' THEN upl.platform_user_id END), '') as discord_id
		FROM users u
		LEFT JOIN user_platform_links upl ON u.user_id = upl.user_id
		LEFT JOIN platforms p ON upl.platform_id = p.platform_id
		WHERE u.user_id = $1
		GROUP BY u.user_id, u.username, u.created_at, u.updated_at
	`
	var user domain.User
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID, &user.Username,
		&user.CreatedAt, &user.UpdatedAt,
		&user.TwitchID, &user.YoutubeID, &user.DiscordID,
	)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &user, nil
}

// UpdateUser updates a user's platform IDs via the user_platform_links junction table
func (r *UserRepository) UpdateUser(ctx context.Context, user domain.User) error {
	// Start transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Update user timestamp
	_, err = tx.Exec(ctx, `UPDATE users SET updated_at = NOW() WHERE user_id = $1`, user.ID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Helper function to update platform link
	updatePlatformLink := func(platformName, platformUserID string) error {
		if platformUserID == "" {
			// Remove platform link if ID is empty
			_, err := tx.Exec(ctx, `
				DELETE FROM user_platform_links 
				WHERE user_id = $1 
				AND platform_id = (SELECT platform_id FROM platforms WHERE name = $2)
			`, user.ID, platformName)
			return err
		}
		
		// Insert or update platform link
		_, err := tx.Exec(ctx, `
			INSERT INTO user_platform_links (user_id, platform_id, platform_user_id)
			VALUES ($1, (SELECT platform_id FROM platforms WHERE name = $2), $3)
			ON CONFLICT (user_id, platform_id) 
			DO UPDATE SET platform_user_id = $3
		`, user.ID, platformName, platformUserID)
		return err
	}

	// Update each platform
	if err := updatePlatformLink("twitch", user.TwitchID); err != nil {
		return fmt.Errorf("failed to update twitch link: %w", err)
	}
	if err := updatePlatformLink("youtube", user.YoutubeID); err != nil {
		return fmt.Errorf("failed to update youtube link: %w", err)
	}
	if err := updatePlatformLink("discord", user.DiscordID); err != nil {
		return fmt.Errorf("failed to update discord link: %w", err)
	}

	return tx.Commit(ctx)
}

// MergeUsersInTransaction merges secondary user into primary user atomically
// All operations succeed or all are rolled back (no data loss)
func (r *UserRepository) MergeUsersInTransaction(ctx context.Context, primaryUserID, secondaryUserID string, mergedUser domain.User, mergedInventory domain.Inventory) error {
	// Begin transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Delete secondary user's inventory
	_, err = tx.Exec(ctx, `DELETE FROM user_inventory WHERE user_id = $1`, secondaryUserID)
	if err != nil {
		return fmt.Errorf("failed to delete secondary inventory: %w", err)
	}

	// 2. Delete secondary user (CASCADE removes platform links)
	_, err = tx.Exec(ctx, `DELETE FROM users WHERE user_id = $1`, secondaryUserID)
	if err != nil {
		return fmt.Errorf("failed to delete secondary user: %w", err)
	}

	// 3. Update primary user timestamp
	_, err = tx.Exec(ctx, `UPDATE users SET updated_at = NOW() WHERE user_id = $1`, primaryUserID)
	if err != nil {
		return fmt.Errorf("failed to update primary user: %w", err)
	}

	// 4. Update primary user's platform links with merged data
	updatePlatformLink := func(platformName, platformUserID string) error {
		if platformUserID == "" {
			return nil // Skip empty platformsIDs
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO user_platform_links (user_id, platform_id, platform_user_id)
			VALUES ($1, (SELECT platform_id FROM platforms WHERE name = $2), $3)
			ON CONFLICT (user_id, platform_id) 
			DO UPDATE SET platform_user_id = $3
		`, primaryUserID, platformName, platformUserID)
		return err
	}

	if err := updatePlatformLink("twitch", mergedUser.TwitchID); err != nil {
		return fmt.Errorf("failed to update twitch link: %w", err)
	}
	if err := updatePlatformLink("youtube", mergedUser.YoutubeID); err != nil {
		return fmt.Errorf("failed to update youtube link: %w", err)
	}
	if err := updatePlatformLink("discord", mergedUser.DiscordID); err != nil {
		return fmt.Errorf("failed to update discord link: %w", err)
	}

	// 5. Update primary user's inventory with merged data
	_, err = tx.Exec(ctx, `
		INSERT INTO user_inventory (user_id, inventory_data)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE
		SET inventory_data = EXCLUDED.inventory_data
	`, primaryUserID, mergedInventory)
	if err != nil {
		return fmt.Errorf("failed to update primary inventory: %w", err)
	}

	// Commit transaction - all or nothing
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit merge transaction: %w", err)
	}

	return nil
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
