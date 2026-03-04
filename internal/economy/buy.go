package economy

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

func (s *service) BuyItem(ctx context.Context, platform, platformID, username, itemName string, qty int) (int, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgBuyItemCalled, "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", qty)

	if err := validateQuantity(qty); err != nil {
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

	moneySlotIdx, moneyBalance, err := s.getMoneyBalance(ctx, tx, user.ID)
	if err != nil {
		return 0, err
	}

	finalQty, cost := s.calculatePurchaseDetails(ctx, item, qty, moneyBalance)
	if finalQty == 0 {
		return 0, fmt.Errorf(ErrMsgInsufficientFundsToBuyOneFmt, item.InternalName, item.BaseValue, moneyBalance, domain.ErrInsufficientFunds)
	}

	inv, _ := tx.GetInventory(ctx, user.ID)
	processBuyTransaction(inv, item.ID, moneySlotIdx, finalQty, cost)

	if err := tx.UpdateInventory(ctx, user.ID, *inv); err != nil {
		return 0, fmt.Errorf(ErrMsgUpdateInventoryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf(ErrMsgCommitTransactionFailed, err)
	}

	s.finalizePurchase(ctx, user.ID, item, finalQty, cost)

	log.Info(LogMsgItemPurchased, "username", username, "item", itemName, "quantity", finalQty)
	return finalQty, nil
}

func (s *service) calculatePurchaseDetails(ctx context.Context, item *domain.Item, requestedQty, moneyBalance int) (int, int) {
	log := logger.FromContext(ctx)
	category := getItemCategory(item)
	discountedPrice := s.applyWeeklySaleDiscount(ctx, item.BaseValue, category)

	if discountedPrice < item.BaseValue {
		log.Info("Weekly sale discount applied", "item", item.InternalName, "category", category, "original_price", item.BaseValue, "discounted_price", discountedPrice)
	}

	finalQty, cost := calculateAffordableQuantity(requestedQty, discountedPrice, moneyBalance)

	if requestedQty > finalQty && finalQty > 0 {
		log.Info(LogMsgAdjustedPurchaseQty, "requested", requestedQty, "actual", finalQty)
	}

	return finalQty, cost
}

func (s *service) finalizePurchase(ctx context.Context, userID string, item *domain.Item, qty, cost int) {
	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventTypeItemBought),
			Payload: domain.ItemBoughtPayload{
				UserID:       userID,
				ItemName:     item.InternalName,
				ItemCategory: getItemCategory(item),
				Quantity:     qty,
				TotalValue:   cost,
				Timestamp:    s.now().Unix(),
			},
		})
	}
}
