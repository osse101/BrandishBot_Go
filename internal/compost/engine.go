package compost

import (
	"math"
	"sort"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// Engine provides pure compost logic (no DB dependencies)
type Engine struct{}

// NewEngine creates a new compost engine
func NewEngine() *Engine {
	return &Engine{}
}

// CalculateReadyAt computes when composting will finish
func (e *Engine) CalculateReadyAt(startedAt time.Time, totalItemCount int) time.Time {
	return startedAt.Add(WarmupDuration + time.Duration(totalItemCount)*PerItemDuration)
}

// CalculateSludgeAt computes when ready compost turns to sludge
func (e *Engine) CalculateSludgeAt(readyAt time.Time) time.Time {
	return readyAt.Add(SludgeTimeout)
}

// CalculateInputValue sums the weighted values of all bin items
func (e *Engine) CalculateInputValue(items []domain.CompostBinItem) int {
	total := 0
	for _, item := range items {
		mult := utils.GetQualityMultiplier(item.QualityLevel)
		total += int(math.Round(float64(item.BaseValue) * mult * float64(item.Quantity)))
	}
	return total
}

// DetermineDominantType finds the content type with the highest total weighted value.
// Falls back to "material" if items is empty or has no typed content.
func (e *Engine) DetermineDominantType(items []domain.CompostBinItem) string {
	typeValues := make(map[string]int)
	for _, item := range items {
		mult := utils.GetQualityMultiplier(item.QualityLevel)
		itemValue := int(math.Round(float64(item.BaseValue) * mult * float64(item.Quantity)))
		for _, ct := range item.ContentTypes {
			typeValues[ct] += itemValue
		}
	}

	if len(typeValues) == 0 {
		return domain.ContentTypeMaterial
	}

	dominant := ""
	maxVal := 0
	for t, v := range typeValues {
		if v > maxVal || (v == maxVal && t < dominant) {
			maxVal = v
			dominant = t
		}
	}
	return dominant
}

// CalculateOutput determines what the compost produces
func (e *Engine) CalculateOutput(inputValue int, dominantType string, isSludge bool, allItems []domain.Item, multiplier float64) *domain.CompostOutput {
	if isSludge {
		sludgeQty := inputValue / 10
		if sludgeQty < 1 {
			sludgeQty = 1
		}
		return &domain.CompostOutput{
			Items:      map[string]int{"compost_sludge": sludgeQty},
			IsSludge:   true,
			TotalValue: sludgeQty,
			Message:    MsgHarvestSludge,
		}
	}

	outputValue := int(math.Round(float64(inputValue) * multiplier))
	if outputValue < 1 {
		outputValue = 1
	}

	// Filter items by dominant type, sort by base_value descending
	var candidates []domain.Item
	for _, item := range allItems {
		if domain.HasType(item.ContentType, dominantType) {
			candidates = append(candidates, item)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].BaseValue > candidates[j].BaseValue
	})

	// Pick the highest-value item whose base_value <= outputValue
	for _, item := range candidates {
		if item.BaseValue <= outputValue && item.BaseValue > 0 {
			qty := outputValue / item.BaseValue
			if qty < 1 {
				qty = 1
			}
			return &domain.CompostOutput{
				Items:      map[string]int{item.InternalName: qty},
				IsSludge:   false,
				TotalValue: outputValue,
				Message:    MsgHarvestComplete,
			}
		}
	}

	// Fallback: give money
	return &domain.CompostOutput{
		Items:      map[string]int{"money": outputValue},
		IsSludge:   false,
		TotalValue: outputValue,
		Message:    MsgHarvestFallback,
	}
}

// TotalItemCount returns the sum of all item quantities in the bin
func (e *Engine) TotalItemCount(items []domain.CompostBinItem) int {
	total := 0
	for _, item := range items {
		total += item.Quantity
	}
	return total
}
