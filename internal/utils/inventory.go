package utils

import (
	"fmt"

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
func RemoveFromSlot(inventory *domain.Inventory, slotIndex, quantity int) {
	if slotIndex < 0 || slotIndex >= len(inventory.Slots) {
		return
	}
	if inventory.Slots[slotIndex].Quantity == quantity {
		inventory.Slots = append(inventory.Slots[:slotIndex], inventory.Slots[slotIndex+1:]...)
	} else {
		inventory.Slots[slotIndex].Quantity -= quantity
	}
}

func getShuffledIndices(inventory *domain.Inventory, itemID int, rnd func() float64) []int {
	matchingIndices := make([]int, 0)
	for i, slot := range inventory.Slots {
		if slot.ItemID == itemID {
			matchingIndices = append(matchingIndices, i)
		}
	}

	if len(matchingIndices) > 1 {
		for i := len(matchingIndices) - 1; i > 0; i-- {
			j := int(rnd() * float64(i+1))
			if j > i {
				j = i
			}
			matchingIndices[i], matchingIndices[j] = matchingIndices[j], matchingIndices[i]
		}
	}
	return matchingIndices
}

func ConsumeItemsWithTracking(inventory *domain.Inventory, itemID int, quantity int, rnd func() float64) ([]domain.InventorySlot, error) {
	totalAvailable := GetTotalQuantity(inventory, itemID)
	if totalAvailable < quantity {
		return nil, fmt.Errorf("insufficient items: have %d, need %d", totalAvailable, quantity)
	}

	matchingIndices := getShuffledIndices(inventory, itemID, rnd)

	remaining := quantity
	qtyReductions := make(map[int]int)
	consumed := make([]domain.InventorySlot, 0)

	for _, idx := range matchingIndices {
		if remaining == 0 {
			break
		}
		slotQty := inventory.Slots[idx].Quantity
		qtyToTake := slotQty
		if qtyToTake > remaining {
			qtyToTake = remaining
		}
		qtyReductions[idx] = qtyToTake
		remaining -= qtyToTake

		consumed = append(consumed, domain.InventorySlot{
			ItemID:       inventory.Slots[idx].ItemID,
			Quantity:     qtyToTake,
			QualityLevel: inventory.Slots[idx].QualityLevel,
		})
	}

	if remaining > 0 {
		return nil, fmt.Errorf("unexpected insufficient items after calculation")
	}

	newSlots := make([]domain.InventorySlot, 0, len(inventory.Slots))
	for i, slot := range inventory.Slots {
		if reduce, ok := qtyReductions[i]; ok {
			if slot.Quantity > reduce {
				slot.Quantity -= reduce
				newSlots = append(newSlots, slot)
			}
		} else {
			newSlots = append(newSlots, slot)
		}
	}
	inventory.Slots = newSlots
	return consumed, nil
}

func ConsumeItems(inventory *domain.Inventory, itemID int, quantity int, rnd func() float64) error {
	_, err := ConsumeItemsWithTracking(inventory, itemID, quantity, rnd)
	return err
}
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

func GetTotalQuantity(inventory *domain.Inventory, itemID int) int {
	total := 0
	for _, slot := range inventory.Slots {
		if slot.ItemID == itemID {
			total += slot.Quantity
		}
	}
	return total
}
