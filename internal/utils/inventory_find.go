package utils

import "github.com/osse101/BrandishBot_Go/internal/domain"

func FindSlot(inventory *domain.Inventory, itemID int) (int, int) {
	for i, slot := range inventory.Slots {
		if slot.ItemID == itemID {
			return i, slot.Quantity
		}
	}
	return -1, 0
}

func FindSlotWithQuality(inventory *domain.Inventory, itemID int, qualityLevel domain.QualityLevel) (int, int) {
	for i, slot := range inventory.Slots {
		if slot.ItemID == itemID && slot.QualityLevel == qualityLevel {
			return i, slot.Quantity
		}
	}
	return -1, 0
}

func FindRandomSlot(inventory *domain.Inventory, itemID int, rnd func() float64) (int, int) {
	matchingIndices := make([]int, 0)
	for i, slot := range inventory.Slots {
		if slot.ItemID == itemID {
			matchingIndices = append(matchingIndices, i)
		}
	}

	if len(matchingIndices) == 0 {
		return -1, 0
	}

	if len(matchingIndices) == 1 {
		slotIdx := matchingIndices[0]
		return slotIdx, inventory.Slots[slotIdx].Quantity
	}

	randomIdx := int(rnd() * float64(len(matchingIndices)))
	if randomIdx >= len(matchingIndices) {
		randomIdx = len(matchingIndices) - 1
	}
	slotIdx := matchingIndices[randomIdx]
	return slotIdx, inventory.Slots[slotIdx].Quantity
}

func BuildSlotMap(inventory *domain.Inventory) map[SlotKey]int {
	slotMap := make(map[SlotKey]int, len(inventory.Slots))
	for i, slot := range inventory.Slots {
		key := SlotKey{ItemID: slot.ItemID, QualityLevel: slot.QualityLevel}
		slotMap[key] = i
	}
	return slotMap
}
