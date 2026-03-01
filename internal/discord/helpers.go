package discord

import "github.com/osse101/BrandishBot_Go/internal/user"

// SimpleInventoryItem represents a stripped-down inventory item for display purposes
type SimpleInventoryItem struct {
	Name     string
	Quantity int
}

// ConvertToSimpleInventory converts a slice of user.InventoryItem (with full metadata)
// to a simplified format containing only name and quantity for Discord display.
// This helper eliminates duplication when converting inventory API response types.
func ConvertToSimpleInventory(inventoryItems []user.InventoryItem) []SimpleInventoryItem {
	itemsMap := make(map[string]int)
	itemOrder := make([]string, 0)

	for _, item := range inventoryItems {
		if _, exists := itemsMap[item.PublicName]; !exists {
			itemOrder = append(itemOrder, item.PublicName)
		}
		itemsMap[item.PublicName] += item.Quantity
	}

	items := make([]SimpleInventoryItem, 0, len(itemOrder))
	for _, name := range itemOrder {
		items = append(items, SimpleInventoryItem{
			Name:     name,
			Quantity: itemsMap[name],
		})
	}
	return items
}
