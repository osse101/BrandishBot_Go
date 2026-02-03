package utils

import (
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// InventoryLookupLinearScanThreshold defines when to switch from linear scan to map-based lookup.
// Benchmarks show linear scan is faster for small M (items to add) even with large N (inventory size).
// Map overhead ~30µs vs Linear ~2µs for M=5, N=1000
const InventoryLookupLinearScanThreshold = 10

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

// FindRandomSlot finds a random slot with the given item ID in an inventory.
// If multiple slots exist with the same item ID, one is selected randomly using the provided RNG function.
// Returns the index of the randomly selected slot and the quantity found.
// Returns -1, 0 if not found.
func FindRandomSlot(inventory *domain.Inventory, itemID int, rnd func() float64) (int, int) {
	// Find all matching slots
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

	// Randomly select one
	randomIdx := int(rnd() * float64(len(matchingIndices)))
	slotIdx := matchingIndices[randomIdx]
	return slotIdx, inventory.Slots[slotIdx].Quantity
}

// BuildSlotMap creates a map of item ID to slot index for O(1) lookups.
// This is useful when adding many items to an inventory to avoid repeated linear scans.
func BuildSlotMap(inventory *domain.Inventory) map[int]int {
	slotMap := make(map[int]int, len(inventory.Slots))
	for i, slot := range inventory.Slots {
		slotMap[slot.ItemID] = i
	}
	return slotMap
}

// RemoveFromSlot removes a quantity from an inventory slot at the given index.
// If the quantity equals the slot quantity, the slot is removed entirely.
// Assumes the caller has already validated that slotIndex is valid and quantity <= slot.Quantity.
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

// AddItemsToInventory adds multiple items to inventory using a hybrid lookup strategy.
// For small batches (< InventoryLookupLinearScanThreshold), uses linear scan to avoid map allocation overhead.
// For larger batches, uses map-based lookup for O(N+M) complexity.
// The slotMap parameter is optional and will be created if nil and needed.
func AddItemsToInventory(inventory *domain.Inventory, items []domain.InventorySlot, slotMap map[int]int) {
	if len(items) == 0 {
		return
	}

	useMap := len(items) >= InventoryLookupLinearScanThreshold

	// Build map if needed and not provided
	if useMap && slotMap == nil {
		slotMap = BuildSlotMap(inventory)
	}

	for _, item := range items {
		if useMap {
			// Map-based lookup
			if idx, exists := slotMap[item.ItemID]; exists {
				inventory.Slots[idx].Quantity += item.Quantity
			} else {
				inventory.Slots = append(inventory.Slots, domain.InventorySlot{ItemID: item.ItemID, Quantity: item.Quantity})
				slotMap[item.ItemID] = len(inventory.Slots) - 1
			}
		} else {
			// Linear scan
			found := false
			for i := range inventory.Slots {
				if inventory.Slots[i].ItemID == item.ItemID {
					inventory.Slots[i].Quantity += item.Quantity
					found = true
					break
				}
			}
			if !found {
				inventory.Slots = append(inventory.Slots, domain.InventorySlot{ItemID: item.ItemID, Quantity: item.Quantity})
			}
		}
	}
}
