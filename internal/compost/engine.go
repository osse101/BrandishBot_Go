package compost

import (
	"math"
	"sort"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) CalculateReadyAt(startedAt time.Time, totalItemCount int, speedMultiplier float64) time.Time {
	baseDuration := float64(WarmupDuration + time.Duration(totalItemCount)*PerItemDuration)
	reducedDuration := time.Duration(baseDuration * (1.0 - speedMultiplier))
	return startedAt.Add(reducedDuration)
}

func (e *Engine) CalculateSludgeAt(readyAt time.Time, sludgeExtHours float64) time.Time {
	extDuration := time.Duration(sludgeExtHours * float64(time.Hour))
	return readyAt.Add(SludgeTimeout + extDuration)
}

func (e *Engine) CalculateInputValue(items []domain.CompostBinItem) int {
	total := 0
	for _, item := range items {
		qualityMultiplier := utils.GetQualityMultiplier(item.QualityLevel)
		total += int(math.Round(float64(item.BaseValue) * qualityMultiplier * float64(item.Quantity)))
	}
	return total
}

func (e *Engine) DetermineDominantType(items []domain.CompostBinItem) string {
	typeValues := make(map[string]int)
	for _, item := range items {
		qualityMultiplier := utils.GetQualityMultiplier(item.QualityLevel)
		itemValue := int(math.Round(float64(item.BaseValue) * qualityMultiplier * float64(item.Quantity)))
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

func (e *Engine) CalculateOutput(inputValue int, dominantType string, isSludge bool, allItems []domain.Item, multiplier float64) *domain.CompostOutput {
	if isSludge {
		sludgeQuantity := inputValue / 10
		if sludgeQuantity < 1 {
			sludgeQuantity = 1
		}
		return &domain.CompostOutput{
			Items:      map[string]int{domain.ItemSludge: sludgeQuantity},
			IsSludge:   true,
			TotalValue: sludgeQuantity,
			Message:    MsgHarvestSludge,
		}
	}

	outputValue := int(math.Round(float64(inputValue) * multiplier))
	if outputValue < 1 {
		outputValue = 1
	}

	var candidates []domain.Item
	for _, item := range allItems {
		if domain.HasType(item.ContentType, dominantType) {
			candidates = append(candidates, item)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].BaseValue > candidates[j].BaseValue
	})

	for _, item := range candidates {
		if item.BaseValue <= outputValue && item.BaseValue > 0 {
			quantity := outputValue / item.BaseValue
			if quantity < 1 {
				quantity = 1
			}
			return &domain.CompostOutput{
				Items:      map[string]int{item.InternalName: quantity},
				IsSludge:   false,
				TotalValue: outputValue,
				Message:    MsgHarvestComplete,
			}
		}
	}

	return &domain.CompostOutput{
		Items:      map[string]int{domain.ItemMoney: outputValue},
		IsSludge:   false,
		TotalValue: outputValue,
		Message:    MsgHarvestFallback,
	}
}

func (e *Engine) TotalItemCount(items []domain.CompostBinItem) int {
	total := 0
	for _, item := range items {
		total += item.Quantity
	}
	return total
}
