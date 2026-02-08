package utils

import (
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// qualityToValue maps QualityLevel to a numeric value for averaging
var qualityToValue = map[domain.QualityLevel]int{
	domain.QualityCursed:    0,
	domain.QualityJunk:      1,
	domain.QualityPoor:      2,
	domain.QualityCommon:    3,
	domain.QualityUncommon:  4,
	domain.QualityRare:      5,
	domain.QualityEpic:      6,
	domain.QualityLegendary: 7,
}

// valueToQuality maps numeric values back to QualityLevel
var valueToQuality = []domain.QualityLevel{
	domain.QualityCursed,    // 0
	domain.QualityJunk,      // 1
	domain.QualityPoor,      // 2
	domain.QualityCommon,    // 3
	domain.QualityUncommon,  // 4
	domain.QualityRare,      // 5
	domain.QualityEpic,      // 6
	domain.QualityLegendary, // 7
}

// CalculateAverageQuality calculates the weighted average quality level from consumed materials.
// Each material contributes to the average based on its quantity.
// Returns COMMON if no materials provided or if calculation fails.
//
// Example:
//   - 5x COMMON (value 3) + 3x LEGENDARY (value 7) = (5*3 + 3*7) / 8 = 36 / 8 = 4.5 → RARE (value 5)
//   - 10x COMMON + 1x LEGENDARY = (10*3 + 1*7) / 11 = 37 / 11 = 3.36 → COMMON (value 3)
func CalculateAverageQuality(materials []domain.InventorySlot) domain.QualityLevel {
	if len(materials) == 0 {
		return domain.QualityCommon
	}

	totalValue := 0
	totalQuantity := 0

	for _, material := range materials {
		qualityValue, ok := qualityToValue[material.QualityLevel]
		if !ok {
			// Unknown quality level, treat as COMMON
			qualityValue = qualityToValue[domain.QualityCommon]
		}
		totalValue += qualityValue * material.Quantity
		totalQuantity += material.Quantity
	}

	if totalQuantity == 0 {
		return domain.QualityCommon
	}

	// Calculate average and round to nearest integer
	averageValue := (totalValue + totalQuantity/2) / totalQuantity // Integer division with rounding

	// Clamp to valid range
	if averageValue < 0 {
		averageValue = 0
	}
	if averageValue >= len(valueToQuality) {
		averageValue = len(valueToQuality) - 1
	}

	return valueToQuality[averageValue]
}
