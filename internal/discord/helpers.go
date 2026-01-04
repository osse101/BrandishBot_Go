package discord

import "github.com/osse101/BrandishBot_Go/internal/user"

// SimpleInventoryItem represents a stripped-down inventory item for display purposes
type SimpleInventoryItem struct {
	Name     string
	Quantity int
}

// ConvertToSimpleInventory converts a slice of user.UserInventoryItem (with full metadata)
// to a simplified format containing only name and quantity for Discord display.
// This helper eliminates duplication when converting inventory API response types.
func ConvertToSimpleInventory(inventoryItems []user.UserInventoryItem) []SimpleInventoryItem {
	items := make([]SimpleInventoryItem, 0, len(inventoryItems))
	for _, item := range inventoryItems {
		items = append(items, SimpleInventoryItem{
			Name:     item.Name,
			Quantity: item.Quantity,
		})
	}
	return items
}
