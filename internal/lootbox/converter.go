package lootbox

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

func (s *service) convertToDroppedItems(ctx context.Context, dropCounts map[string]dropInfo, boxQuality domain.QualityLevel) ([]DroppedItem, error) {
	log := logger.FromContext(ctx)

	itemNames := make([]string, 0, len(dropCounts))
	for itemName := range dropCounts {
		itemNames = append(itemNames, itemName)
	}

	items, err := s.repo.GetItemsByNames(ctx, itemNames)
	if err != nil {
		log.Error(ErrContextFailedToGetDroppedItems, LogFieldError, err)
		return nil, err
	}

	itemMap := make(map[string]*domain.Item, len(items))
	for i := range items {
		itemMap[items[i].InternalName] = &items[i]
	}

	// Check if lucky upgrade is unlocked via progression
	canUpgrade := false
	if s.progressionSvc != nil {
		// "feature_gamble" is the key for the gamble feature which unlocks lucky upgrades
		unlocked, err := s.progressionSvc.IsNodeUnlocked(ctx, "feature_gamble", 1)
		if err == nil {
			canUpgrade = unlocked
		}
	}

	drops := make([]DroppedItem, 0, len(dropCounts))
	for itemName, info := range dropCounts {
		item, found := itemMap[itemName]
		if !found {
			log.Warn(LogMsgDroppedItemNotInDB, LogFieldItem, itemName)
			continue
		}

		// Use a random roll for quality.
		quality, mult := s.calculateQuality(s.rnd(), boxQuality, canUpgrade)

		droppedItem := s.constructDroppedItem(item, info, quality, mult)
		drops = append(drops, droppedItem)
	}

	return drops, nil
}

func (s *service) constructDroppedItem(item *domain.Item, info dropInfo, quality domain.QualityLevel, mult float64) DroppedItem {
	quantity := info.Qty
	boostedValue := int(float64(item.BaseValue) * mult)

	// Currency special logic: convert quality to quantity, force COMMON quality
	if item.IsCurrency() {
		quantity = int(float64(info.Qty) * mult)
		if info.Qty > 0 && quantity == 0 {
			quantity = 1
		}
		boostedValue = item.BaseValue  // Keep base value (usually 1)
		quality = domain.QualityCommon // Force COMMON for all currency
	} else {
		// Normal item truncation protection
		if item.BaseValue > 0 && boostedValue == 0 {
			boostedValue = 1
		}
	}

	return DroppedItem{
		ItemID:       item.ID,
		ItemName:     item.InternalName,
		Quantity:     quantity,
		Value:        boostedValue,
		QualityLevel: quality,
	}
}
