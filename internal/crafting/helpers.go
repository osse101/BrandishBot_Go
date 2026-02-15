package crafting

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func (s *service) validateUser(ctx context.Context, platform, platformID string) (*domain.User, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

func (s *service) validateItem(ctx context.Context, itemName string) (*domain.Item, error) {
	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return nil, fmt.Errorf("item not found: %s | %w", itemName, domain.ErrItemNotFound)
	}
	return item, nil
}

func (s *service) validateQuantity(quantity int) error {
	if quantity <= 0 {
		return fmt.Errorf("quantity must be positive (got %d): %w", quantity, domain.ErrInvalidQuantity)
	}
	return nil
}

func (s *service) validatePlatformInput(platform, platformID string) error {
	if platform == "" || platformID == "" {
		return fmt.Errorf("platform and platformID cannot be empty: %w", domain.ErrInvalidInput)
	}
	validPlatforms := []string{domain.PlatformTwitch, domain.PlatformDiscord, domain.PlatformYoutube}
	for _, p := range validPlatforms {
		if platform == p {
			return nil
		}
	}
	return fmt.Errorf("invalid platform '%s': %w", platform, domain.ErrInvalidPlatform)
}

func (s *service) validateItemName(itemName string) error {
	if itemName == "" {
		return fmt.Errorf("item name cannot be empty: %w", domain.ErrInvalidInput)
	}
	return nil
}

// resolveItemName attempts to resolve a user-provided item name to its internal name.
// It first tries the naming resolver, then falls back to using the input as-is.
// This allows users to use either public names ("junkbox") or internal names ("lootbox_tier0").
func (s *service) resolveItemName(ctx context.Context, itemName string) (string, error) {
	// Try naming resolver first (handles public names)
	if s.namingResolver != nil {
		if internalName, ok := s.namingResolver.ResolvePublicName(itemName); ok {
			return internalName, nil
		}
	}

	// Fall back - assume it's already an internal name
	// Validate by checking if item exists
	item, err := s.validateItem(ctx, itemName)
	if err != nil {
		return "", fmt.Errorf("failed to resolve item name '%s': %w", itemName, err)
	}
	if item == nil {
		return "", fmt.Errorf("item not found: %s (not found as public or internal name) | %w", itemName, domain.ErrItemNotFound)
	}

	return itemName, nil
}
