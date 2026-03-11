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

	unlockedItems, err := s.filterUnlockedItems(ctx, allItems, false)
	if err == nil {
		log.Info("Buyable prices filtered", "total", len(allItems), "unlocked", len(unlockedItems))
	}
	return unlockedItems, err
}

func (s *service) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgGetSellablePricesCalled)

	allItems, err := s.repo.GetSellablePrices(ctx)
	if err != nil {
		return nil, err
	}

	unlockedItems, err := s.filterUnlockedItems(ctx, allItems, true)
	if err == nil {
		log.Info("Sellable prices filtered", "total", len(allItems), "unlocked", len(unlockedItems))
	}
	return unlockedItems, err
}

func (s *service) filterUnlockedItems(ctx context.Context, items []domain.Item, calculateSellPrice bool) ([]domain.Item, error) {
	if s.progressionService == nil {
		if calculateSellPrice {
			for i := range items {
				sellPrice := s.calculateSellPriceWithModifier(ctx, items[i].BaseValue)
				items[i].SellPrice = &sellPrice
			}
		}
		return items, nil
	}

	itemNames := make([]string, len(items))
	for i, item := range items {
		itemNames[i] = item.InternalName
	}

	unlockStatus, err := s.progressionService.AreItemsUnlocked(ctx, itemNames)
	if err != nil {
		return nil, fmt.Errorf("failed to check item unlock status: %w", err)
	}

	unlockedItems := make([]domain.Item, 0, len(items))
	for _, item := range items {
		if unlockStatus[item.InternalName] {
			if calculateSellPrice {
				sellPrice := s.calculateSellPriceWithModifier(ctx, item.BaseValue)
				item.SellPrice = &sellPrice
			}
			unlockedItems = append(unlockedItems, item)
		}
	}

	return unlockedItems, nil
}

func calculateSellPrice(baseValue int) int {
	return int(float64(baseValue) * SellPriceRatio)
}

func (s *service) calculateSellPriceWithModifier(ctx context.Context, baseValue int) int {
	basePrice := calculateSellPrice(baseValue)

	if s.progressionService == nil {
		return basePrice
	}

	modifiedPrice, err := s.progressionService.GetModifiedValue(ctx, "", "economy_bonus", float64(basePrice))
	if err != nil {
		logger.FromContext(ctx).Warn("Failed to apply economy_bonus modifier, using base price", "error", err)
		return basePrice
	}

	return int(modifiedPrice)
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
