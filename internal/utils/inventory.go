package utils

import "github.com/osse101/BrandishBot_Go/internal/domain"

// FindSlot finds a slot with the given item ID in an inventory.
// Returns the index of the slot and the quantity found.
// Returns -1, 0 if not found.
func FindSlot(inventory *domain.Inventory, itemID int) (int, int) {
	for i, slot := range inventory.Slots {
		if slot.ItemID == itemID {
			return i, slot.Quantity
		}
	}
	return -1, 0
}
