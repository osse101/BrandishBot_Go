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

	shouldUseMap := len(items) >= InventoryLookupLinearScanThreshold

	if shouldUseMap && slotMap == nil {
		slotMap = BuildSlotMap(inventory)
	}

	for _, item := range items {
		if shouldUseMap {
			key := SlotKey{ItemID: item.ItemID, QualityLevel: item.QualityLevel}
			if existingSlotIndex, exists := slotMap[key]; exists {
				inventory.Slots[existingSlotIndex].Quantity += item.Quantity
			} else {
				inventory.Slots = append(inventory.Slots, domain.InventorySlot{
					ItemID:       item.ItemID,
					Quantity:     item.Quantity,
					QualityLevel: item.QualityLevel,
				})
				slotMap[key] = len(inventory.Slots) - 1
			}
		} else {
			isItemFound := false
			for slotIndex := range inventory.Slots {
				if inventory.Slots[slotIndex].ItemID == item.ItemID && inventory.Slots[slotIndex].QualityLevel == item.QualityLevel {
					inventory.Slots[slotIndex].Quantity += item.Quantity
					isItemFound = true
					break
				}
			}
			if !isItemFound {
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

func getShuffledIndices(inventory *domain.Inventory, itemID int, randomFloatGen func() float64) []int {
	itemSlotIndices := make([]int, 0)
	for i, slot := range inventory.Slots {
		if slot.ItemID == itemID {
			itemSlotIndices = append(itemSlotIndices, i)
		}
	}

	if len(itemSlotIndices) > 1 {
		for i := len(itemSlotIndices) - 1; i > 0; i-- {
			j := int(randomFloatGen() * float64(i+1))
			if j > i {
				j = i
			}
			itemSlotIndices[i], itemSlotIndices[j] = itemSlotIndices[j], itemSlotIndices[i]
		}
	}
	return itemSlotIndices
}

func ConsumeItemsWithTracking(inventory *domain.Inventory, itemID int, quantity int, randomFloatGen func() float64) ([]domain.InventorySlot, error) {
	totalAvailable := GetTotalQuantity(inventory, itemID)
	if totalAvailable < quantity {
		return nil, fmt.Errorf("insufficient items: have %d, need %d", totalAvailable, quantity)
	}

	itemSlotIndices := getShuffledIndices(inventory, itemID, randomFloatGen)

	remaining := quantity
	quantityReductions := make(map[int]int)
	consumed := make([]domain.InventorySlot, 0)

	for _, idx := range itemSlotIndices {
		if remaining == 0 {
			break
		}
		quantityInSlot := inventory.Slots[idx].Quantity
		quantityToConsume := quantityInSlot
		if quantityToConsume > remaining {
			quantityToConsume = remaining
		}
		quantityReductions[idx] = quantityToConsume
		remaining -= quantityToConsume

		consumed = append(consumed, domain.InventorySlot{
			ItemID:       inventory.Slots[idx].ItemID,
			Quantity:     quantityToConsume,
			QualityLevel: inventory.Slots[idx].QualityLevel,
		})
	}

	if remaining > 0 {
		return nil, fmt.Errorf("unexpected insufficient items after calculation")
	}

	updatedSlots := make([]domain.InventorySlot, 0, len(inventory.Slots))
	for i, slot := range inventory.Slots {
		if reductionAmount, ok := quantityReductions[i]; ok {
			if slot.Quantity > reductionAmount {
				slot.Quantity -= reductionAmount
				updatedSlots = append(updatedSlots, slot)
			}
		} else {
			updatedSlots = append(updatedSlots, slot)
		}
	}
	inventory.Slots = updatedSlots
	return consumed, nil
}

func ConsumeItems(inventory *domain.Inventory, itemID int, quantity int, randomFloatGen func() float64) error {
	_, err := ConsumeItemsWithTracking(inventory, itemID, quantity, randomFloatGen)
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

func FindRandomSlot(inventory *domain.Inventory, itemID int, randomFloatGen func() float64) (int, int) {
	itemSlotIndices := make([]int, 0)
	for i, slot := range inventory.Slots {
		if slot.ItemID == itemID {
			itemSlotIndices = append(itemSlotIndices, i)
		}
	}

	if len(itemSlotIndices) == 0 {
		return -1, 0
	}

	if len(itemSlotIndices) == 1 {
		selectedSlotIndex := itemSlotIndices[0]
		return selectedSlotIndex, inventory.Slots[selectedSlotIndex].Quantity
	}

	randomSlotIndex := int(randomFloatGen() * float64(len(itemSlotIndices)))
	if randomSlotIndex >= len(itemSlotIndices) {
		randomSlotIndex = len(itemSlotIndices) - 1
	}
	selectedSlotIndex := itemSlotIndices[randomSlotIndex]
	return selectedSlotIndex, inventory.Slots[selectedSlotIndex].Quantity
}

func GetTotalQuantity(inventory *domain.Inventory, itemID int) int {
	totalQuantity := 0
	for _, slot := range inventory.Slots {
		if slot.ItemID == itemID {
			totalQuantity += slot.Quantity
		}
	}
	return totalQuantity
}
