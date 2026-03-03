package economy

import (
	"context"
	"fmt"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
)

func (s *service) GetBuyablePrices(ctx context.Context) ([]domain.Item, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgGetBuyablePricesCalled)

	allItems, err := s.repo.GetBuyablePrices(ctx)
	if err != nil {
		return nil, err
	}

	if s.progressionService == nil {
		return allItems, nil
	}

	itemNames := make([]string, len(allItems))
	for i, item := range allItems {
		itemNames[i] = item.InternalName
	}

	unlockStatus, err := s.progressionService.AreItemsUnlocked(ctx, itemNames)
	if err != nil {
		return nil, fmt.Errorf("failed to check item unlock status: %w", err)
	}

	filtered := make([]domain.Item, 0, len(allItems))
	for _, item := range allItems {
		if unlockStatus[item.InternalName] {
			filtered = append(filtered, item)
		}
	}

	log.Info("Buyable prices filtered", "total", len(allItems), "unlocked", len(filtered))
	return filtered, nil
}

func (s *service) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgGetSellablePricesCalled)

	allItems, err := s.repo.GetSellablePrices(ctx)
	if err != nil {
		return nil, err
	}

	if s.progressionService == nil {
		for i := range allItems {
			sellPrice := s.calculateSellPriceWithModifier(ctx, allItems[i].BaseValue)
			allItems[i].SellPrice = &sellPrice
		}
		return allItems, nil
	}

	itemNames := make([]string, len(allItems))
	for i, item := range allItems {
		itemNames[i] = item.InternalName
	}

	unlockStatus, err := s.progressionService.AreItemsUnlocked(ctx, itemNames)
	if err != nil {
		return nil, fmt.Errorf("failed to check item unlock status: %w", err)
	}

	filtered := make([]domain.Item, 0, len(allItems))
	for _, item := range allItems {
		if unlockStatus[item.InternalName] {
			sellPrice := s.calculateSellPriceWithModifier(ctx, item.BaseValue)
			item.SellPrice = &sellPrice
			filtered = append(filtered, item)
		}
	}

	log.Info("Sellable prices filtered", "total", len(allItems), "unlocked", len(filtered))
	return filtered, nil
}

func calculateSellPrice(baseValue int) int {
	return int(float64(baseValue) * SellPriceRatio)
}

func (s *service) calculateSellPriceWithModifier(ctx context.Context, baseValue int) int {
	basePrice := calculateSellPrice(baseValue)

	if s.progressionService == nil {
		return basePrice
	}

	modified, err := s.progressionService.GetModifiedValue(ctx, "", "economy_bonus", float64(basePrice))
	if err != nil {
		logger.FromContext(ctx).Warn("Failed to apply economy_bonus modifier, using base price", "error", err)
		return basePrice
	}

	return int(modified)
}

func (s *service) applyWeeklySaleDiscount(ctx context.Context, basePrice int, itemCategory string) int {
	if s.progressionService != nil {
		unlocked, err := s.progressionService.IsFeatureUnlocked(ctx, progression.FeatureWeeklyDiscount)
		if err != nil {
			logger.FromContext(ctx).Warn("Failed to check if weekly discount is unlocked", "error", err)
			return basePrice
		}
		if !unlocked {
			return basePrice
		}
	}

	sale := s.getCurrentWeeklySale()
	if sale == nil {
		return basePrice
	}

	if sale.TargetCategory != nil && !strings.EqualFold(*sale.TargetCategory, itemCategory) {
		return basePrice
	}

	discount := float64(basePrice) * (sale.DiscountPercent / 100.0)
	return basePrice - int(discount)
}

func calculateAffordableQuantity(desired, unitPrice, balance int) (quantity, cost int) {
	if unitPrice == 0 {
		return desired, 0
	}
	if balance < unitPrice {
		return 0, 0
	}
	maxAffordable := balance / unitPrice
	if desired <= maxAffordable {
		return desired, desired * unitPrice
	}
	return maxAffordable, maxAffordable * unitPrice
}
