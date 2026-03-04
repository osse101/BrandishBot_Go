package economy

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

func (s *service) SellItem(ctx context.Context, platform, platformID, username, itemName string, qty int) (int, int, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgSellItemCalled, "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", qty)

	if err := validateQuantity(qty); err != nil {
		return 0, 0, err
	}

	user, item, moneyItem, err := s.getSellEntities(ctx, platform, platformID, itemName)
	if err != nil {
		return 0, 0, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf(ErrMsgBeginTransactionFailed, err)
	}
	defer repository.SafeRollback(ctx, tx)

	inv, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		return 0, 0, fmt.Errorf(ErrMsgGetInventoryFailed, err)
	}

	itemSlotIdx, slotQty := utils.FindRandomSlot(inv, item.ID, s.rnd)
	if itemSlotIdx == -1 {
		return 0, 0, fmt.Errorf(ErrMsgItemNotInInventoryFmt, itemName, domain.ErrNotInInventory)
	}

	finalQty := qty
	if slotQty < qty {
		finalQty = slotQty
	}

	sellPrice := s.calculateSellPriceWithModifier(ctx, item.BaseValue)
	moneyGained := finalQty * sellPrice

	processSellTransaction(inv, moneyItem.ID, itemSlotIdx, finalQty, moneyGained)

	if err := tx.UpdateInventory(ctx, user.ID, *inv); err != nil {
		return 0, 0, fmt.Errorf(ErrMsgUpdateInventoryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf(ErrMsgCommitTransactionFailed, err)
	}

	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventTypeItemSold),
			Payload: domain.ItemSoldPayload{
				UserID:       user.ID,
				ItemName:     item.InternalName,
				ItemCategory: getItemCategory(item),
				Quantity:     finalQty,
				TotalValue:   moneyGained,
				Timestamp:    s.now().Unix(),
			},
		})
	}

	log.Info(LogMsgItemSold, "username", username, "item", itemName, "quantity", finalQty, "moneyGained", moneyGained)
	return moneyGained, finalQty, nil
}
