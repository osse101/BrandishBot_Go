package utils

import (
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

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

func GetTotalQuantity(inventory *domain.Inventory, itemID int) int {
	total := 0
	for _, slot := range inventory.Slots {
		if slot.ItemID == itemID {
			total += slot.Quantity
		}
	}
	return total
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
	reductions := make(map[int]int)
	consumed := make([]domain.InventorySlot, 0)

	for _, idx := range matchingIndices {
		if remaining == 0 {
			break
		}
		slotQty := inventory.Slots[idx].Quantity
		take := slotQty
		if take > remaining {
			take = remaining
		}
		reductions[idx] = take
		remaining -= take

		consumed = append(consumed, domain.InventorySlot{
			ItemID:       inventory.Slots[idx].ItemID,
			Quantity:     take,
			QualityLevel: inventory.Slots[idx].QualityLevel,
		})
	}

	if remaining > 0 {
		return nil, fmt.Errorf("unexpected insufficient items after calculation")
	}

	newSlots := make([]domain.InventorySlot, 0, len(inventory.Slots))
	for i, slot := range inventory.Slots {
		if reduce, ok := reductions[i]; ok {
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
