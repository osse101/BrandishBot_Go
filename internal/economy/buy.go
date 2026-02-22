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

	// 1. Validate request
	if err := validateQuantity(quantity); err != nil {
		return 0, err
	}

	// 2. Get user and item
	user, item, err := s.getBuyEntities(ctx, platform, platformID, itemName)
	if err != nil {
		return 0, err
	}

	// 3. Begin transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, fmt.Errorf(ErrMsgBeginTransactionFailed, err)
	}
	defer repository.SafeRollback(ctx, tx)

	// 4. Check eligibility (buyable + progression)
	if err := s.checkBuyEligibility(ctx, item); err != nil {
		return 0, err
	}

	// 5. Check funds and inventory
	moneySlotIndex, moneyBalance, err := s.getMoneyBalance(ctx, tx, user.ID)
	if err != nil {
		return 0, err
	}

	// 6. Calculate price and quantity
	actualQuantity, cost := s.calculatePurchaseDetails(ctx, item, quantity, moneyBalance)
	if actualQuantity == 0 {
		return 0, fmt.Errorf(ErrMsgInsufficientFundsToBuyOneFmt, item.InternalName, item.BaseValue, moneyBalance, domain.ErrInsufficientFunds)
	}

	// 7. Process inventory updates
	inventory, _ := tx.GetInventory(ctx, user.ID) // already fetched in getMoneyBalance
	processBuyTransaction(inventory, item, moneySlotIndex, actualQuantity, cost)

	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return 0, fmt.Errorf(ErrMsgUpdateInventoryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf(ErrMsgCommitTransactionFailed, err)
	}

	// 8. Finalize (XP, quests)
	s.finalizePurchase(ctx, user.ID, item, actualQuantity, cost)

	log.Info(LogMsgItemPurchased, "username", username, "item", itemName, "quantity", actualQuantity)
	return actualQuantity, nil
}

// GetBuyablePrices retrieves all buyable items with prices
func (s *service) GetBuyablePrices(ctx context.Context) ([]domain.Item, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgGetBuyablePricesCalled)

	allItems, err := s.repo.GetBuyablePrices(ctx)
	if err != nil {
		return nil, err
	}

	// Return all items if no progression service
	if s.progressionService == nil {
		return allItems, nil
	}

	// Extract item names for batch checking
	itemNames := make([]string, len(allItems))
	for i, item := range allItems {
		itemNames[i] = item.InternalName
	}

	// Batch check unlock status
	unlockStatus, err := s.progressionService.AreItemsUnlocked(ctx, itemNames)
	if err != nil {
		return nil, fmt.Errorf("failed to check item unlock status: %w", err)
	}

	// Filter to only unlocked items
	filtered := make([]domain.Item, 0, len(allItems))
	for _, item := range allItems {
		if unlockStatus[item.InternalName] {
			filtered = append(filtered, item)
		}
	}

	log.Info("Buyable prices filtered", "total", len(allItems), "unlocked", len(filtered))
	return filtered, nil
}

func (s *service) calculatePurchaseDetails(ctx context.Context, item *domain.Item, requestedQuantity, moneyBalance int) (int, int) {
	log := logger.FromContext(ctx)
	itemCategory := getItemCategory(item)
	discountedPrice := s.applyWeeklySaleDiscount(ctx, item.BaseValue, itemCategory)

	if discountedPrice < item.BaseValue {
		log.Info("Weekly sale discount applied", "item", item.InternalName, "category", itemCategory, "original_price", item.BaseValue, "discounted_price", discountedPrice)
	}

	actualQuantity, cost := calculateAffordableQuantity(requestedQuantity, discountedPrice, moneyBalance)

	if requestedQuantity > actualQuantity && actualQuantity > 0 {
		log.Info(LogMsgAdjustedPurchaseQty, "requested", requestedQuantity, "actual", actualQuantity)
	}

	return actualQuantity, cost
}

func (s *service) finalizePurchase(ctx context.Context, userID string, item *domain.Item, quantity, cost int) {
	// Publish item.bought event (job handler awards Merchant XP, quest handler tracks progress)
	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventTypeItemBought),
			Payload: domain.ItemBoughtPayload{
				UserID:       userID,
				ItemName:     item.InternalName,
				ItemCategory: getItemCategory(item),
				Quantity:     quantity,
				TotalValue:   cost,
				Timestamp:    s.now().Unix(),
			},
		})
	}
}
