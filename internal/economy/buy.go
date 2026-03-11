package economy

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

func (s *service) BuyItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (int, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgBuyItemCalled, "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", quantity)

	if err := validateQuantity(quantity); err != nil {
		return 0, err
	}

	user, item, err := s.getBuyEntities(ctx, platform, platformID, itemName)
	if err != nil {
		return 0, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, fmt.Errorf(ErrMsgBeginTransactionFailed, err)
	}
	defer repository.SafeRollback(ctx, tx)

	if err := s.checkBuyEligibility(ctx, item); err != nil {
		return 0, err
	}

	moneySlotIndex, availableMoney, err := s.getMoneyBalance(ctx, tx, user.ID)
	if err != nil {
		return 0, err
	}

	actualQuantity, totalCost := s.calculatePurchaseDetails(ctx, item, quantity, availableMoney)
	if actualQuantity == 0 {
		return 0, fmt.Errorf(ErrMsgInsufficientFundsToBuyOneFmt, item.InternalName, item.BaseValue, availableMoney, domain.ErrInsufficientFunds)
	}

	inventory, _ := tx.GetInventory(ctx, user.ID)
	processBuyTransaction(inventory, item.ID, moneySlotIndex, actualQuantity, totalCost)

	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return 0, fmt.Errorf(ErrMsgUpdateInventoryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf(ErrMsgCommitTransactionFailed, err)
	}

	s.finalizePurchase(ctx, user.ID, item, actualQuantity, totalCost)

	log.Info(LogMsgItemPurchased, "username", username, "item", itemName, "quantity", actualQuantity)
	return actualQuantity, nil
}

func (s *service) calculatePurchaseDetails(ctx context.Context, item *domain.Item, requestedQuantity, availableMoney int) (int, int) {
	log := logger.FromContext(ctx)
	itemCategory := getItemCategory(item)
	discountedPrice := s.applyWeeklySaleDiscount(ctx, item.BaseValue, itemCategory)

	if discountedPrice < item.BaseValue {
		log.Info("Weekly sale discount applied", "item", item.InternalName, "category", itemCategory, "original_price", item.BaseValue, "discounted_price", discountedPrice)
	}

	actualQuantity, totalCost := calculateAffordableQuantity(requestedQuantity, discountedPrice, availableMoney)

	if requestedQuantity > actualQuantity && actualQuantity > 0 {
		log.Info(LogMsgAdjustedPurchaseQty, "requested", requestedQuantity, "actual", actualQuantity)
	}

	return actualQuantity, totalCost
}

func (s *service) finalizePurchase(ctx context.Context, userID string, item *domain.Item, quantity, totalCost int) {
	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventTypeItemBought),
			Payload: domain.ItemBoughtPayload{
				UserID:       userID,
				ItemName:     item.InternalName,
				ItemCategory: getItemCategory(item),
				Quantity:     quantity,
				TotalValue:   totalCost,
				Timestamp:    s.now().Unix(),
			},
		})
	}
}
