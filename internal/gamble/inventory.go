package gamble

import (
	"context"
	"fmt"
	"sort"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// consumeItem consumes an item from the inventory and returns its quality level
func consumeItem(inventory *domain.Inventory, itemID, quantity int) (domain.QualityLevel, error) {
	for i := range inventory.Slots {
		if inventory.Slots[i].ItemID == itemID {
			if inventory.Slots[i].Quantity < quantity {
				return "", domain.ErrInsufficientQuantity
			}
			qualityLevel := inventory.Slots[i].QualityLevel
			if inventory.Slots[i].Quantity == quantity {
				// Remove slot
				inventory.Slots = append(inventory.Slots[:i], inventory.Slots[i+1:]...)
			} else {
				// Reduce quantity
				inventory.Slots[i].Quantity -= quantity
			}
			return qualityLevel, nil
		}
	}
	return domain.QualityLevel(""), domain.ErrItemNotFound
}

func (s *service) awardItemsToWinner(ctx context.Context, tx repository.GambleTx, winnerID string, allOpenedItems []domain.GambleOpenedItem) error {
	inv, err := tx.GetInventory(ctx, winnerID)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToGetWinnerInv, err)
	}

	itemsToAdd := make(map[int]int)
	for _, item := range allOpenedItems {
		itemsToAdd[item.ItemID] += item.Quantity
	}

	for i, slot := range inv.Slots {
		if qty, ok := itemsToAdd[slot.ItemID]; ok {
			inv.Slots[i].Quantity += qty
			delete(itemsToAdd, slot.ItemID)
		}
	}

	var newItemIDs []int
	if len(itemsToAdd) > 0 {
		newItemIDs = make([]int, 0, len(itemsToAdd))
		for itemID := range itemsToAdd {
			newItemIDs = append(newItemIDs, itemID)
		}
		sort.Ints(newItemIDs)
	}

	for _, itemID := range newItemIDs {
		inv.Slots = append(inv.Slots, domain.InventorySlot{ItemID: itemID, Quantity: itemsToAdd[itemID]})
	}

	if err := tx.UpdateInventory(ctx, winnerID, *inv); err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToUpdateWinnerInv, err)
	}
	return nil
}
