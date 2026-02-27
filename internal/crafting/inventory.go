package crafting

import (
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// calculateMaxPossibleCrafts calculates the maximum number of crafts possible given available materials
func calculateMaxPossibleCrafts(inventory *domain.Inventory, recipe *domain.Recipe, requestedQuantity int) int {
	maxPossible := requestedQuantity
	for _, cost := range recipe.BaseCost {
		userQuantity := utils.GetTotalQuantity(inventory, cost.ItemID)
		if cost.Quantity > 0 {
			affordableWithThis := userQuantity / cost.Quantity
			if affordableWithThis < maxPossible {
				maxPossible = affordableWithThis
			}
		}
	}
	return maxPossible
}

// consumeRecipeMaterials removes the required materials from inventory for crafting.
// Returns the consumed materials with their quality levels for calculating output quality.
func consumeRecipeMaterials(inventory *domain.Inventory, recipe *domain.Recipe, actualQuantity int, rnd func() float64) ([]domain.InventorySlot, error) {
	allConsumed := make([]domain.InventorySlot, 0)

	for _, cost := range recipe.BaseCost {
		totalNeeded := cost.Quantity * actualQuantity
		consumed, err := utils.ConsumeItemsWithTracking(inventory, cost.ItemID, totalNeeded, rnd)
		if err != nil {
			return nil, fmt.Errorf("insufficient material (itemID: %d) | %w", cost.ItemID, domain.ErrInsufficientQuantity)
		}
		allConsumed = append(allConsumed, consumed...)
	}

	return allConsumed, nil
}

// addItemToInventory adds items to the inventory with specified quality level.
// Only stacks with slots that have matching ItemID AND QualityLevel.
func addItemToInventory(inventory *domain.Inventory, itemID, quantity int, qualityLevel domain.QualityLevel) {
	// Find slot with matching ItemID and QualityLevel
	for i, slot := range inventory.Slots {
		if slot.ItemID == itemID && slot.QualityLevel == qualityLevel {
			inventory.Slots[i].Quantity += quantity
			return
		}
	}
	// Item not found with matching quality, add new slot
	inventory.Slots = append(inventory.Slots, domain.InventorySlot{
		ItemID:       itemID,
		Quantity:     quantity,
		QualityLevel: qualityLevel,
	})
}
