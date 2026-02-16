package economy

import (
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// getItemCategory extracts the category from an item's types
// Uses the first type if available, otherwise returns generic "Item"
func getItemCategory(item *domain.Item) string {
	if item != nil && len(item.Types) > 0 {
		return item.Types[0]
	}
	return "Item"
}
