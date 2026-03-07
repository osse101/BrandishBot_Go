package utils

import (
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

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

var valueToQuality = []domain.QualityLevel{
	domain.QualityCursed,
	domain.QualityJunk,
	domain.QualityPoor,
	domain.QualityCommon,
	domain.QualityUncommon,
	domain.QualityRare,
	domain.QualityEpic,
	domain.QualityLegendary,
}

func GetQualityValue(q domain.QualityLevel) int {
	if v, ok := qualityToValue[q]; ok {
		return v
	}
	return qualityToValue[domain.QualityCommon]
}

func CompareQuality(q1, q2 domain.QualityLevel) int {
	return GetQualityValue(q1) - GetQualityValue(q2)
}

func GetQualityMultiplier(q domain.QualityLevel) float64 {
	switch q {
	case domain.QualityLegendary:
		return domain.MultLegendary
	case domain.QualityEpic:
		return domain.MultEpic
	case domain.QualityRare:
		return domain.MultRare
	case domain.QualityUncommon:
		return domain.MultUncommon
	case domain.QualityPoor:
		return domain.MultPoor
	case domain.QualityJunk:
		return domain.MultJunk
	case domain.QualityCursed:
		return domain.MultCursed
	default:
		return domain.MultCommon
	}
}

func CalculateAverageQuality(materials []domain.InventorySlot) domain.QualityLevel {
	if len(materials) == 0 {
		return domain.QualityCommon
	}

	totalValue := 0
	totalQuantity := 0

	for _, material := range materials {
		qualityValue, ok := qualityToValue[material.QualityLevel]
		if !ok {
			qualityValue = qualityToValue[domain.QualityCommon]
		}
		totalValue += qualityValue * material.Quantity
		totalQuantity += material.Quantity
	}

	if totalQuantity == 0 {
		return domain.QualityCommon
	}

	averageValue := (totalValue + totalQuantity/2) / totalQuantity

	if averageValue < 0 {
		averageValue = 0
	}
	if averageValue >= len(valueToQuality) {
		averageValue = len(valueToQuality) - 1
	}

	return valueToQuality[averageValue]
}
