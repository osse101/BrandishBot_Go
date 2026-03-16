package utils

import (
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

const InventoryLookupLinearScanThreshold = 50

type SlotKey struct {
	ItemID       int
	QualityLevel domain.QualityLevel
}

func BuildSlotMap(inventory *domain.Inventory) map[SlotKey]int {
	slotMap := make(map[SlotKey]int, len(inventory.Slots))
	for i, slot := range inventory.Slots {
		key := SlotKey{ItemID: slot.ItemID, QualityLevel: slot.QualityLevel}
		slotMap[key] = i
	}
	return slotMap
}
