package utils

import (
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func AddItemsToInventory(inventory *domain.Inventory, items []domain.InventorySlot, slotMap map[SlotKey]int) {
	if len(items) == 0 {
		return
	}

	useMap := len(items) >= InventoryLookupLinearScanThreshold

	if useMap && slotMap == nil {
		slotMap = BuildSlotMap(inventory)
	}

	for _, item := range items {
		if useMap {
			key := SlotKey{ItemID: item.ItemID, QualityLevel: item.QualityLevel}
			if idx, exists := slotMap[key]; exists {
				inventory.Slots[idx].Quantity += item.Quantity
			} else {
				inventory.Slots = append(inventory.Slots, domain.InventorySlot{
					ItemID:       item.ItemID,
					Quantity:     item.Quantity,
					QualityLevel: item.QualityLevel,
				})
				slotMap[key] = len(inventory.Slots) - 1
			}
		} else {
			found := false
			for i := range inventory.Slots {
				if inventory.Slots[i].ItemID == item.ItemID && inventory.Slots[i].QualityLevel == item.QualityLevel {
					inventory.Slots[i].Quantity += item.Quantity
					found = true
					break
				}
			}
			if !found {
				inventory.Slots = append(inventory.Slots, domain.InventorySlot{
					ItemID:       item.ItemID,
					Quantity:     item.Quantity,
					QualityLevel: item.QualityLevel,
				})
			}
		}
	}
}
