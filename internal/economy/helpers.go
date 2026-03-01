package economy

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// getItemCategory extracts the category from an item's types
// Uses the first type if available, otherwise returns generic "Item"
func getItemCategory(item *domain.Item) string {
	if item != nil && len(item.Types) > 0 {
		return item.Types[0]
	}
	return "Item"
}

// getBuyEntities retrieves and validates user and item for a buy transaction
func (s *service) getBuyEntities(ctx context.Context, platform, platformID, itemName string) (*domain.User, *domain.Item, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, nil, fmt.Errorf(ErrMsgGetUserFailed, err)
	}
	if user == nil {
		return nil, nil, domain.ErrUserNotFound
	}

	// Resolve public name to internal name
	resolvedName, err := s.resolveItemName(ctx, itemName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve item %q: %w", itemName, err)
	}

	item, err := s.repo.GetItemByName(ctx, resolvedName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get item %q: %w", resolvedName, err)
	}
	if item == nil {
		return nil, nil, fmt.Errorf("item not found: %q: %w", resolvedName, domain.ErrItemNotFound)
	}

	return user, item, nil
}

// getSellEntities retrieves and validates all required entities for a sell transaction
func (s *service) getSellEntities(ctx context.Context, platform, platformID, itemName string) (*domain.User, *domain.Item, *domain.Item, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf(ErrMsgGetUserFailed, err)
	}
	if user == nil {
		return nil, nil, nil, domain.ErrUserNotFound
	}

	// Resolve public name to internal name
	resolvedName, err := s.resolveItemName(ctx, itemName)
	if err != nil {
		return nil, nil, nil, err
	}

	item, err := s.repo.GetItemByName(ctx, resolvedName)
	if err != nil {
		return nil, nil, nil, fmt.Errorf(ErrMsgGetItemFailed, err)
	}
	if item == nil {
		return nil, nil, nil, fmt.Errorf(ErrMsgItemNotFoundFmt, resolvedName, domain.ErrItemNotFound)
	}

	moneyItem, err := s.repo.GetItemByName(ctx, domain.ItemMoney)
	if err != nil {
		return nil, nil, nil, fmt.Errorf(ErrMsgGetMoneyItemFailed, err)
	}
	if moneyItem == nil {
		return nil, nil, nil, fmt.Errorf(ErrMsgItemNotFoundFmt, domain.ItemMoney, domain.ErrItemNotFound)
	}

	return user, item, moneyItem, nil
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
	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		return "", fmt.Errorf(ErrMsgResolveItemFailedFmt, itemName, err)
	}
	if item == nil {
		return "", fmt.Errorf(ErrMsgItemNotFoundPublicFmt, itemName, domain.ErrItemNotFound)
	}

	return itemName, nil
}
