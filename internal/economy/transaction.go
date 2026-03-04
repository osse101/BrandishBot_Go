package economy

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

func processExchangeTransaction(inv *domain.Inventory, removeSlotIdx, removeQty, addQty, addItemID int) {
	if inv.Slots[removeSlotIdx].Quantity <= removeQty {
		inv.Slots = append(inv.Slots[:removeSlotIdx], inv.Slots[removeSlotIdx+1:]...)
	} else {
		inv.Slots[removeSlotIdx].Quantity -= removeQty
	}

	itemFound := false
	for i, slot := range inv.Slots {
		if slot.ItemID == addItemID && slot.QualityLevel == domain.QualityCommon {
			inv.Slots[i].Quantity += addQty
			itemFound = true
			break
		}
	}
	if !itemFound {
		inv.Slots = append(inv.Slots, domain.InventorySlot{
			ItemID:       addItemID,
			Quantity:     addQty,
			QualityLevel: domain.QualityCommon,
		})
	}
}

func processBuyTransaction(inv *domain.Inventory, itemID, moneySlotIdx, actualQty, cost int) {
	processExchangeTransaction(inv, moneySlotIdx, cost, actualQty, itemID)
}

func processSellTransaction(inv *domain.Inventory, moneyItemID, itemSlotIdx, actualSellQty, moneyGained int) {
	processExchangeTransaction(inv, itemSlotIdx, actualSellQty, moneyGained, moneyItemID)
}

func (s *service) getMoneyBalance(ctx context.Context, tx repository.EconomyTx, userID string) (int, int, error) {
	moneyItem, err := s.repo.GetItemByName(ctx, domain.ItemMoney)
	if err != nil {
		return 0, 0, fmt.Errorf(ErrMsgGetMoneyItemFailed, err)
	}
	if moneyItem == nil {
		return 0, 0, fmt.Errorf(ErrMsgItemNotFoundFmt, domain.ItemMoney, domain.ErrItemNotFound)
	}

	inv, err := tx.GetInventory(ctx, userID)
	if err != nil {
		return 0, 0, fmt.Errorf(ErrMsgGetInventoryFailed, err)
	}

	moneySlotIdx, moneyBalance := utils.FindRandomSlot(inv, moneyItem.ID, s.rnd)
	if moneyBalance <= 0 {
		return 0, 0, domain.ErrInsufficientFunds
	}

	return moneySlotIdx, moneyBalance, nil
}
