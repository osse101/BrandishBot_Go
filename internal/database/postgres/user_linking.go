package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/osse101/BrandishBot_Go/internal/database/generated"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// GetUserByID retrieves a user by internal ID with all linked platform IDs
func (r *UserRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	row, err := r.q.GetUserByID(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	user := domain.User{
		ID:        row.UserID.String(),
		Username:  row.Username,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}

	links, err := r.q.GetUserPlatformLinks(ctx, row.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user links: %w", err)
	}

	for _, link := range links {
		switch link.Name {
		case "twitch":
			user.TwitchID = link.PlatformUserID
		case "youtube":
			user.YoutubeID = link.PlatformUserID
		case "discord":
			user.DiscordID = link.PlatformUserID
		}
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
	defer SafeRollback(ctx, tx)

	q := r.q.WithTx(tx)
	userUUID, err := uuid.Parse(user.ID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	// Update user timestamp
	err = q.UpdateUserTimestamp(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("failed to update user timestamp: %w", err)
	}

	// Helper function to update platform link
	updatePlatformLink := func(platformName, platformUserID string) error {
		if platformUserID == "" {
			return q.DeleteUserPlatformLink(ctx, generated.DeleteUserPlatformLinkParams{
				UserID: userUUID,
				Name:   platformName,
			})
		}

		platformID, err := q.GetPlatformID(ctx, platformName)
		if err != nil {
			return fmt.Errorf("failed to get platform id: %w", err)
		}

		return q.UpsertUserPlatformLink(ctx, generated.UpsertUserPlatformLinkParams{
			UserID:         userUUID,
			PlatformID:     platformID,
			PlatformUserID: platformUserID,
		})
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
func (r *UserRepository) MergeUsersInTransaction(ctx context.Context, primaryUserID, secondaryUserID string, mergedUser domain.User, mergedInventory domain.Inventory) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer SafeRollback(ctx, tx)

	q := r.q.WithTx(tx)
	primUUID, err := uuid.Parse(primaryUserID)
	if err != nil {
		return fmt.Errorf("invalid primary user id: %w", err)
	}
	secUUID, err := uuid.Parse(secondaryUserID)
	if err != nil {
		return fmt.Errorf("invalid secondary user id: %w", err)
	}

	// 1. Delete secondary user's inventory
	if err := q.DeleteInventory(ctx, secUUID); err != nil {
		return fmt.Errorf("failed to delete secondary inventory: %w", err)
	}

	// 2. Delete secondary user (CASCADE removes platform links)
	if err := q.DeleteUser(ctx, secUUID); err != nil {
		return fmt.Errorf("failed to delete secondary user: %w", err)
	}

	// 3. Update primary user timestamp
	if err := q.UpdateUserTimestamp(ctx, primUUID); err != nil {
		return fmt.Errorf("failed to update primary user: %w", err)
	}

	// 4. Update primary user's platform links with merged data
	updatePlatformLink := func(platformName, platformUserID string) error {
		if platformUserID == "" {
			return nil
		}
		platformID, err := q.GetPlatformID(ctx, platformName)
		if err != nil {
			return fmt.Errorf("failed to get platform id: %w", err)
		}
		return q.UpsertUserPlatformLink(ctx, generated.UpsertUserPlatformLinkParams{
			UserID:         primUUID,
			PlatformID:     platformID,
			PlatformUserID: platformUserID,
		})
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
	inventoryJSON, err := json.Marshal(mergedInventory)
	if err != nil {
		return fmt.Errorf("failed to marshal inventory: %w", err)
	}

	err = q.UpdateInventory(ctx, generated.UpdateInventoryParams{
		UserID:        primUUID,
		InventoryData: inventoryJSON,
	})
	if err != nil {
		return fmt.Errorf("failed to update primary inventory: %w", err)
	}

	return tx.Commit(ctx)
}

// DeleteUser deletes a user by ID
func (r *UserRepository) DeleteUser(ctx context.Context, userID string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}
	return r.q.DeleteUser(ctx, userUUID)
}

// DeleteInventory deletes a user's inventory
func (r *UserRepository) DeleteInventory(ctx context.Context, userID string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}
	return r.q.DeleteInventory(ctx, userUUID)
}
