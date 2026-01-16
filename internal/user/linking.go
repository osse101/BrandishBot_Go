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

	primaryInv, secondaryInv, err := s.getInventoriesForMerge(ctx, primaryUserID, secondaryUserID)
	if err != nil {
		return err
	}

	mergedInv := s.mergeInventories(primaryInv, secondaryInv)

	if err := s.repo.UpdateInventory(ctx, primaryUserID, *mergedInv); err != nil {
		return fmt.Errorf("failed to update primary inventory: %w", err)
	}

	if err := s.repo.DeleteInventory(ctx, secondaryUserID); err != nil {
		log.Warn("Failed to delete secondary inventory", "error", err)
	}

	primary, secondary, err := s.getUsersForMerge(ctx, primaryUserID, secondaryUserID)
	if err != nil {
		return err
	}

	mergedUser := s.mergeUserProfiles(primary, secondary)

	if err := s.repo.MergeUsersInTransaction(ctx, primaryUserID, secondaryUserID, *mergedUser, *mergedInv); err != nil {
		return fmt.Errorf("failed to merge users in transaction: %w", err)
	}

	s.invalidateUserCaches(primary, secondary, mergedUser)

	log.Info("Users merged successfully", "primary", primaryUserID)
	return nil
}

func (s *service) getInventoriesForMerge(ctx context.Context, primaryUserID, secondaryUserID string) (*domain.Inventory, *domain.Inventory, error) {
	primaryInv, err := s.repo.GetInventory(ctx, primaryUserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get primary inventory: %w", err)
	}
	secondaryInv, err := s.repo.GetInventory(ctx, secondaryUserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get secondary inventory: %w", err)
	}
	return primaryInv, secondaryInv, nil
}

func (s *service) mergeInventories(primary, secondary *domain.Inventory) *domain.Inventory {
	const maxStackSize = MaxStackSize
	if primary == nil {
		primary = &domain.Inventory{Slots: []domain.InventorySlot{}}
	}
	if secondary == nil {
		return primary
	}

	for _, sSlot := range secondary.Slots {
		found := false
		for i, pSlot := range primary.Slots {
			if pSlot.ItemID == sSlot.ItemID {
				newQty := pSlot.Quantity + sSlot.Quantity
				if newQty > maxStackSize {
					newQty = maxStackSize
				}
				primary.Slots[i].Quantity = newQty
				found = true
				break
			}
		}
		if !found {
			primary.Slots = append(primary.Slots, sSlot)
		}
	}
	return primary
}

func (s *service) getUsersForMerge(ctx context.Context, primaryUserID, secondaryUserID string) (*domain.User, *domain.User, error) {
	primary, err := s.repo.GetUserByID(ctx, primaryUserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get primary user: %w", err)
	}
	if primary == nil {
		return nil, nil, fmt.Errorf("primary user not found")
	}

	secondary, err := s.repo.GetUserByID(ctx, secondaryUserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get secondary user: %w", err)
	}
	if secondary == nil {
		return nil, nil, fmt.Errorf("secondary user not found")
	}
	return primary, secondary, nil
}

func (s *service) mergeUserProfiles(primary, secondary *domain.User) *domain.User {
	merged := *primary
	if secondary.DiscordID != "" && merged.DiscordID == "" {
		merged.DiscordID = secondary.DiscordID
	}
	if secondary.TwitchID != "" && merged.TwitchID == "" {
		merged.TwitchID = secondary.TwitchID
	}
	if secondary.YoutubeID != "" && merged.YoutubeID == "" {
		merged.YoutubeID = secondary.YoutubeID
	}
	return &merged
}

func (s *service) invalidateUserCaches(users ...*domain.User) {
	for _, u := range users {
		if u == nil {
			continue
		}
		for platform, platformID := range getPlatformKeysFromUser(*u) {
			s.userCache.Invalidate(platform, platformID)
		}
	}
}

// UnlinkPlatform removes a platform from a user account
func (s *service) UnlinkPlatform(ctx context.Context, userID, platform string) error {
	log := logger.FromContext(ctx)
	log.Info("Unlinking platform", "user_id", userID, "platform", platform)

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	var platformID string
	switch platform {
	case domain.PlatformDiscord:
		platformID = user.DiscordID
		user.DiscordID = ""
	case domain.PlatformTwitch:
		platformID = user.TwitchID
		user.TwitchID = ""
	case domain.PlatformYoutube:
		platformID = user.YoutubeID
		user.YoutubeID = ""
	default:
		return fmt.Errorf("unknown platform: %s", platform)
	}

	if err := s.repo.UpdateUser(ctx, *user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Invalidate cache
	if platformID != "" {
		s.userCache.Invalidate(platform, platformID)
	}

	// Also invalidate other keys as user object changed
	keys := getPlatformKeysFromUser(*user)
	for p, id := range keys {
		s.userCache.Invalidate(p, id)
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
