package utils

import (
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

const InventoryLookupLinearScanThreshold = 50

type SlotKey struct {
	ItemID       int
	QualityLevel domain.QualityLevel
}

func GetTotalQuantity(inventory *domain.Inventory, itemID int) int {
	total := 0
	for _, slot := range inventory.Slots {
		if slot.ItemID == itemID {
			total += slot.Quantity
		}
	}
	return total
}
