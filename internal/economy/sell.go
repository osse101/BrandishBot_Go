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

func (s *service) SellItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (int, int, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgSellItemCalled, "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", quantity)

	if err := validateQuantity(quantity); err != nil {
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

	inventory, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		return 0, 0, fmt.Errorf(ErrMsgGetInventoryFailed, err)
	}

	itemSlotIndex, slotQuantity := utils.FindRandomSlot(inventory, item.ID, s.rnd)
	if itemSlotIndex == -1 {
		return 0, 0, fmt.Errorf(ErrMsgItemNotInInventoryFmt, itemName, domain.ErrNotInInventory)
	}

	finalQty := quantity
	if slotQuantity < quantity {
		finalQty = slotQuantity
	}

	sellPrice := s.calculateSellPriceWithModifier(ctx, item.BaseValue)
	moneyGained := finalQty * sellPrice

	processSellTransaction(inventory, moneyItem.ID, itemSlotIndex, finalQty, moneyGained)

	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
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

func (s *service) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgGetSellablePricesCalled)

	allItems, err := s.repo.GetSellablePrices(ctx)
	if err != nil {
		return nil, err
	}

	if s.progressionService == nil {
		for i := range allItems {
			sellPrice := s.calculateSellPriceWithModifier(ctx, allItems[i].BaseValue)
			allItems[i].SellPrice = &sellPrice
		}
		return allItems, nil
	}

	itemNames := make([]string, len(allItems))
	for i, item := range allItems {
		itemNames[i] = item.InternalName
	}

	unlockStatus, err := s.progressionService.AreItemsUnlocked(ctx, itemNames)
	if err != nil {
		return nil, fmt.Errorf("failed to check item unlock status: %w", err)
	}

	filtered := make([]domain.Item, 0, len(allItems))
	for _, item := range allItems {
		if unlockStatus[item.InternalName] {
			sellPrice := s.calculateSellPriceWithModifier(ctx, item.BaseValue)
			item.SellPrice = &sellPrice
			filtered = append(filtered, item)
		}
	}

	log.Info("Sellable prices filtered", "total", len(allItems), "unlocked", len(filtered))
	return filtered, nil
}
