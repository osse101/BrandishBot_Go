package economy

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// processExchangeTransaction is a helper handling the repeated pattern of removing a cost/item and adding a resulting item/money.
func processExchangeTransaction(inventory *domain.Inventory, removeSlotIndex, removeAmount, addAmount, addItemID int) {
	// Remove item cost/source
	if inventory.Slots[removeSlotIndex].Quantity <= removeAmount {
		inventory.Slots = append(inventory.Slots[:removeSlotIndex], inventory.Slots[removeSlotIndex+1:]...)
	} else {
		inventory.Slots[removeSlotIndex].Quantity -= removeAmount
	}

	// Add purchased/earned item
	itemFound := false
	for i, slot := range inventory.Slots {
		if slot.ItemID == addItemID && slot.QualityLevel == domain.QualityCommon {
			inventory.Slots[i].Quantity += addAmount
			itemFound = true
			break
		}
	}
	if !itemFound {
		inventory.Slots = append(inventory.Slots, domain.InventorySlot{
			ItemID:       addItemID,
			Quantity:     addAmount,
			QualityLevel: domain.QualityCommon,
		})
	}
}

// processBuyTransaction handles the inventory updates for buying an item
func processBuyTransaction(inventory *domain.Inventory, itemID, moneySlotIndex, actualQuantity, cost int) {
	processExchangeTransaction(inventory, moneySlotIndex, cost, actualQuantity, itemID)
}

// processSellTransaction handles the inventory updates for selling an item
func processSellTransaction(inventory *domain.Inventory, moneyItemID, itemSlotIndex, actualSellQuantity, moneyGained int) {
	processExchangeTransaction(inventory, itemSlotIndex, actualSellQuantity, moneyGained, moneyItemID)
}

func (s *service) getMoneyBalance(ctx context.Context, tx repository.EconomyTx, userID string) (int, int, error) {
	moneyItem, err := s.repo.GetItemByName(ctx, domain.ItemMoney)
	if err != nil {
		return 0, 0, fmt.Errorf(ErrMsgGetMoneyItemFailed, err)
	}
	if moneyItem == nil {
		return 0, 0, fmt.Errorf(ErrMsgItemNotFoundFmt, domain.ItemMoney, domain.ErrItemNotFound)
	}

	inventory, err := tx.GetInventory(ctx, userID)
	if err != nil {
		return 0, 0, fmt.Errorf(ErrMsgGetInventoryFailed, err)
	}

	moneySlotIndex, moneyBalance := utils.FindRandomSlot(inventory, moneyItem.ID, s.rnd)
	if moneyBalance <= 0 {
		return 0, 0, domain.ErrInsufficientFunds
	}

	return moneySlotIndex, moneyBalance, nil
}
