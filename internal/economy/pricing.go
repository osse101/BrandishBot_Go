package economy

import (
	"context"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
)

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

// applyWeeklySaleDiscount applies the current weekly sale discount to a buy price
// Returns the discounted price. Requires feature_weekly_discount to be unlocked.
func (s *service) applyWeeklySaleDiscount(ctx context.Context, basePrice int, itemCategory string) int {
	// Check if weekly discount feature is unlocked
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

	// Check if item category matches the sale
	if sale.TargetCategory != nil && !strings.EqualFold(*sale.TargetCategory, itemCategory) {
		return basePrice
	}

	// Apply discount
	discount := float64(basePrice) * (sale.DiscountPercent / 100.0)
	return basePrice - int(discount)
}

// calculateAffordableQuantity determines how many items can be purchased with available money
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
