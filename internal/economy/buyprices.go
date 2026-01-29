package economy

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// GetBuyablePrices retrieves all buyable items with prices
func (s *service) GetBuyablePrices(ctx context.Context) ([]domain.Item, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgGetBuyablePricesCalled)

	allItems, err := s.repo.GetBuyablePrices(ctx)
	if err != nil {
		return nil, err
	}

	// Return all items if no progression service
	if s.progressionService == nil {
		return allItems, nil
	}

	// Extract item names for batch checking
	itemNames := make([]string, len(allItems))
	for i, item := range allItems {
		itemNames[i] = item.InternalName
	}

	// Batch check unlock status
	unlockStatus, err := s.progressionService.AreItemsUnlocked(ctx, itemNames)
	if err != nil {
		return nil, fmt.Errorf("failed to check item unlock status: %w", err)
	}

	// Filter to only unlocked items
	filtered := make([]domain.Item, 0, len(allItems))
	for _, item := range allItems {
		if unlockStatus[item.InternalName] {
			filtered = append(filtered, item)
		}
	}

	log.Info("Buyable prices filtered", "total", len(allItems), "unlocked", len(filtered))
	return filtered, nil
}
