package economy

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// GetBuyablePrices retrieves all buyable items with prices
func (s *service) GetBuyablePrices(ctx context.Context) ([]domain.Item, error) {
	log := logger.FromContext(ctx)
	log.Info("GetBuyablePrices called")
	return s.repo.GetBuyablePrices(ctx)
}
