package utils

import (
	"fmt"

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
// NOTE: This map assumes unique Item IDs. If multiple slots have the same ItemID
// (e.g. different shine levels), this map will only point to the last one visited.
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

// slotKey uniquely identifies an inventory slot for map lookups
type slotKey struct {
	ItemID     int
	ShineLevel domain.ShineLevel
}

// AddItemsToInventory adds multiple items to inventory using a hybrid lookup strategy.
// For small batches (< InventoryLookupLinearScanThreshold), uses linear scan to avoid map allocation overhead.
// For larger batches, uses map-based lookup for O(N+M) complexity.
//
// NOTE: This function respects ShineLevel. Items with different shine levels will NOT stack.
func AddItemsToInventory(inventory *domain.Inventory, items []domain.InventorySlot) {
	if len(items) == 0 {
		return
	}

	useMap := len(items) >= InventoryLookupLinearScanThreshold

	if useMap {
		// Build internal map keyed by (ID, Shine)
		slotMap := make(map[slotKey]int, len(inventory.Slots))
		for i, slot := range inventory.Slots {
			key := slotKey{ItemID: slot.ItemID, ShineLevel: slot.ShineLevel}
			slotMap[key] = i
		}

		for _, item := range items {
			key := slotKey{ItemID: item.ItemID, ShineLevel: item.ShineLevel}
			if idx, exists := slotMap[key]; exists {
				inventory.Slots[idx].Quantity += item.Quantity
			} else {
				inventory.Slots = append(inventory.Slots, item)
				slotMap[key] = len(inventory.Slots) - 1
			}
		}
	} else {
		// Linear scan
		for _, item := range items {
			found := false
			for i := range inventory.Slots {
				if inventory.Slots[i].ItemID == item.ItemID && inventory.Slots[i].ShineLevel == item.ShineLevel {
					inventory.Slots[i].Quantity += item.Quantity
					found = true
					break
				}
			}
			if !found {
				inventory.Slots = append(inventory.Slots, item)
			}
		}
	}
}

// GetTotalQuantity returns the total quantity of an item across all slots in the inventory.
func GetTotalQuantity(inventory *domain.Inventory, itemID int) int {
	total := 0
	for _, slot := range inventory.Slots {
		if slot.ItemID == itemID {
			total += slot.Quantity
		}
	}
	return total
}

// ConsumeItems removes a specific quantity of an item from the inventory.
// It searches for all slots containing the item, shuffles them using the provided RNG
// to maintain random selection behavior, and consumes items until the required quantity is met.
// Returns error if insufficient items are available.
func ConsumeItems(inventory *domain.Inventory, itemID int, quantity int, rnd func() float64) error {
	totalAvailable := GetTotalQuantity(inventory, itemID)
	if totalAvailable < quantity {
		return fmt.Errorf("insufficient items: have %d, need %d", totalAvailable, quantity)
	}

	// Find all matching indices
	matchingIndices := make([]int, 0)
	for i, slot := range inventory.Slots {
		if slot.ItemID == itemID {
			matchingIndices = append(matchingIndices, i)
		}
	}

	// Shuffle indices to simulate random selection
	if len(matchingIndices) > 1 {
		for i := len(matchingIndices) - 1; i > 0; i-- {
			j := int(rnd() * float64(i+1))
			// Ensure j is within bounds, though logic should guarantee it
			if j >= 0 && j <= i {
				matchingIndices[i], matchingIndices[j] = matchingIndices[j], matchingIndices[i]
			}
		}
	}

	remaining := quantity
	reductions := make(map[int]int)

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
	}

	if remaining > 0 {
		// Should be covered by initial check, but just in case
		return fmt.Errorf("unexpected insufficient items after calculation")
	}

	// Rebuild inventory slots
	newSlots := make([]domain.InventorySlot, 0, len(inventory.Slots))
	for i, slot := range inventory.Slots {
		if reduce, ok := reductions[i]; ok {
			if slot.Quantity > reduce {
				slot.Quantity -= reduce
				newSlots = append(newSlots, slot)
			}
			// If quantity == reduce, it's fully consumed and not appended (removed)
		} else {
			newSlots = append(newSlots, slot)
		}
	}
	inventory.Slots = newSlots
	return nil
}
