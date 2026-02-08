package utils

import (
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// shineToValue maps ShineLevel to a numeric value for averaging
var shineToValue = map[domain.ShineLevel]int{
	domain.ShineCursed:    0,
	domain.ShineJunk:      1,
	domain.ShinePoor:      2,
	domain.ShineCommon:    3,
	domain.ShineUncommon:  4,
	domain.ShineRare:      5,
	domain.ShineEpic:      6,
	domain.ShineLegendary: 7,
}

// valueToShine maps numeric values back to ShineLevel
var valueToShine = []domain.ShineLevel{
	domain.ShineCursed,    // 0
	domain.ShineJunk,      // 1
	domain.ShinePoor,      // 2
	domain.ShineCommon,    // 3
	domain.ShineUncommon,  // 4
	domain.ShineRare,      // 5
	domain.ShineEpic,      // 6
	domain.ShineLegendary, // 7
}

// CalculateAverageShine calculates the weighted average shine level from consumed materials.
// Each material contributes to the average based on its quantity.
// Returns COMMON if no materials provided or if calculation fails.
//
// Example:
//   - 5x COMMON (value 3) + 3x LEGENDARY (value 7) = (5*3 + 3*7) / 8 = 36 / 8 = 4.5 → RARE (value 5)
//   - 10x COMMON + 1x LEGENDARY = (10*3 + 1*7) / 11 = 37 / 11 = 3.36 → COMMON (value 3)
func CalculateAverageShine(materials []domain.InventorySlot) domain.ShineLevel {
	if len(materials) == 0 {
		return domain.ShineCommon
	}

	totalValue := 0
	totalQuantity := 0

	for _, material := range materials {
		shineValue, ok := shineToValue[material.ShineLevel]
		if !ok {
			// Unknown shine level, treat as COMMON
			shineValue = shineToValue[domain.ShineCommon]
		}
		totalValue += shineValue * material.Quantity
		totalQuantity += material.Quantity
	}

	if totalQuantity == 0 {
		return domain.ShineCommon
	}

	// Calculate average and round to nearest integer
	averageValue := (totalValue + totalQuantity/2) / totalQuantity // Integer division with rounding

	// Clamp to valid range
	if averageValue < 0 {
		averageValue = 0
	}
	if averageValue >= len(valueToShine) {
		averageValue = len(valueToShine) - 1
	}

	return valueToShine[averageValue]
}
