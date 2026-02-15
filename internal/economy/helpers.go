package economy

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

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

// calculateSellPrice calculates the sell price for an item based on its base value.
// Uses SellPriceRatio to determine the percentage of base_value returned when selling.
// Returns integer price (rounded down to prevent fractional currency).
func calculateSellPrice(baseValue int) int {
	return int(float64(baseValue) * SellPriceRatio)
}

// calculateSellPriceWithModifier applies economy_bonus modifier to sell price
func (s *service) calculateSellPriceWithModifier(ctx context.Context, baseValue int) int {
	basePrice := calculateSellPrice(baseValue)

	if s.progressionService == nil {
		return basePrice
	}

	modified, err := s.progressionService.GetModifiedValue(ctx, "economy_bonus", float64(basePrice))
	if err != nil {
		logger.FromContext(ctx).Warn("Failed to apply economy_bonus modifier, using base price", "error", err)
		return basePrice
	}

	return int(modified)
}

// getItemCategory extracts the category from an item's types
// Uses the first type if available, otherwise returns generic "Item"
func getItemCategory(item *domain.Item) string {
	if item != nil && len(item.Types) > 0 {
		return item.Types[0]
	}
	return "Item"
}
