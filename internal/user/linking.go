package user

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// MergeUsers merges secondary user into primary user
// - Combines inventories (sum quantities, cap at max)
// - Combines statistics
// - Deletes secondary user
func (s *service) MergeUsers(ctx context.Context, primaryUserID, secondaryUserID string) error {
	log := logger.FromContext(ctx)
	log.Info("Merging users", "primary", primaryUserID, "secondary", secondaryUserID)

	// Get both inventories
	primaryInv, err := s.repo.GetInventory(ctx, primaryUserID)
	if err != nil {
		return fmt.Errorf("failed to get primary inventory: %w", err)
	}

	secondaryInv, err := s.repo.GetInventory(ctx, secondaryUserID)
	if err != nil {
		return fmt.Errorf("failed to get secondary inventory: %w", err)
	}

	// Max stack size for items
	const maxStackSize = 999999

	// Merge inventories
	if primaryInv == nil {
		primaryInv = &domain.Inventory{Slots: []domain.InventorySlot{}}
	}
	if secondaryInv != nil {
		for _, slot := range secondaryInv.Slots {
			found := false
			for i, pSlot := range primaryInv.Slots {
				if pSlot.ItemID == slot.ItemID {
					// Sum quantities with cap
					newQty := pSlot.Quantity + slot.Quantity
					if newQty > maxStackSize {
						newQty = maxStackSize
					}
					primaryInv.Slots[i].Quantity = newQty
					found = true
					break
				}
			}
			if !found {
				primaryInv.Slots = append(primaryInv.Slots, slot)
			}
		}
	}

	// Save merged inventory
	if err := s.repo.UpdateInventory(ctx, primaryUserID, *primaryInv); err != nil {
		return fmt.Errorf("failed to update primary inventory: %w", err)
	}

	// TODO: Merge statistics if stats service is available

	// Delete secondary user's inventory
	if err := s.repo.DeleteInventory(ctx, secondaryUserID); err != nil {
		log.Warn("Failed to delete secondary inventory", "error", err)
	}

	// Copy platform IDs from secondary to primary
	secondary, err := s.repo.GetUserByID(ctx, secondaryUserID)
	if err == nil && secondary != nil {
		primary, err := s.repo.GetUserByID(ctx, primaryUserID)
		if err == nil && primary != nil {
			if secondary.DiscordID != "" && primary.DiscordID == "" {
				primary.DiscordID = secondary.DiscordID
			}
			if secondary.TwitchID != "" && primary.TwitchID == "" {
				primary.TwitchID = secondary.TwitchID
			}
			if secondary.YoutubeID != "" && primary.YoutubeID == "" {
				primary.YoutubeID = secondary.YoutubeID
			}
			s.repo.UpdateUser(ctx, *primary)
		}
	}

	// Delete secondary user
	if err := s.repo.DeleteUser(ctx, secondaryUserID); err != nil {
		log.Warn("Failed to delete secondary user", "error", err)
	}

	log.Info("Users merged successfully", "primary", primaryUserID)
	return nil
}

// UnlinkPlatform removes a platform from a user account
func (s *service) UnlinkPlatform(ctx context.Context, userID, platform string) error {
	log := logger.FromContext(ctx)
	log.Info("Unlinking platform", "user_id", userID, "platform", platform)

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	switch platform {
	case domain.PlatformDiscord:
		user.DiscordID = ""
	case domain.PlatformTwitch:
		user.TwitchID = ""
	case domain.PlatformYoutube:
		user.YoutubeID = ""
	default:
		return fmt.Errorf("unknown platform: %s", platform)
	}

	if err := s.repo.UpdateUser(ctx, *user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	log.Info("Platform unlinked", "user_id", userID, "platform", platform)
	return nil
}

// GetLinkedPlatforms returns all platforms linked to a user
func (s *service) GetLinkedPlatforms(ctx context.Context, platform, platformID string) ([]string, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	var platforms []string
	if user.DiscordID != "" {
		platforms = append(platforms, domain.PlatformDiscord)
	}
	if user.TwitchID != "" {
		platforms = append(platforms, domain.PlatformTwitch)
	}
	if user.YoutubeID != "" {
		platforms = append(platforms, domain.PlatformYoutube)
	}

	return platforms, nil
}
