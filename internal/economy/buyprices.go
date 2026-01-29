package economy

import (
	"context"

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

	// Filter out locked items if progression service is available
	if s.progressionService == nil {
		return allItems, nil
	}

	filtered := make([]domain.Item, 0, len(allItems))
	for _, item := range allItems {
		unlocked, err := s.progressionService.IsItemUnlocked(ctx, item.InternalName)
		if err != nil {
			// Log error but don't fail the request - include item if check fails
			log.Warn("Failed to check unlock status", "item", item.InternalName, "error", err)
			filtered = append(filtered, item)
			continue
		}
		if unlocked {
			filtered = append(filtered, item)
		}
	}

	return filtered, nil
}
