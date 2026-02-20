package lootbox

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// convertToDroppedItems applies quality rolls to pool drops and appends consolation money.
// Items are pre-fetched in dropInfo.Item, so no database call is needed here.
func (s *service) convertToDroppedItems(ctx context.Context, dropCounts map[string]*dropInfo, consolationMoney int, moneyItem *domain.Item, boxQuality domain.QualityLevel) ([]DroppedItem, error) {
	log := logger.FromContext(ctx)

	// Check if lucky upgrade is unlocked via progression.
	canUpgrade := false
	if s.progressionSvc != nil {
		unlocked, err := s.progressionSvc.IsNodeUnlocked(ctx, "feature_gamble", 1)
		if err == nil {
			canUpgrade = unlocked
		}
	}

	drops := make([]DroppedItem, 0, len(dropCounts)+1)

	for itemName, info := range dropCounts {
		if info.Item == nil {
			log.Warn(LogMsgDroppedItemNotInDB, LogFieldItem, itemName)
			continue
		}
		quality, mult := s.calculateQuality(s.rnd(), boxQuality, canUpgrade)
		drops = append(drops, s.constructDroppedItem(info.Item, info.Qty, quality, mult))
	}

	// Consolation money bypasses the quality roll and is forced to COMMON.
	if consolationMoney > 0 {
		if moneyItem != nil {
			drops = append(drops, DroppedItem{
				ItemID:       moneyItem.ID,
				ItemName:     moneyItem.InternalName,
				Quantity:     consolationMoney,
				Value:        moneyItem.BaseValue,
				QualityLevel: domain.QualityCommon,
			})
		} else {
			log.Warn("Consolation money cannot be awarded: money item not found in database")
		}
	}

	return drops, nil
}

func (s *service) constructDroppedItem(item *domain.Item, qty int, quality domain.QualityLevel, mult float64) DroppedItem {
	quantity := qty
	boostedValue := int(float64(item.BaseValue) * mult)

	// Currency special logic: convert quality to quantity, force COMMON quality.
	if item.IsCurrency() {
		quantity = int(float64(qty) * mult)
		if qty > 0 && quantity == 0 {
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
