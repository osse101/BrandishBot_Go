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

func (s *service) BuyItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (int, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgBuyItemCalled, "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", quantity)

	// 1. Validate request
	if err := validateBuyRequest(quantity); err != nil {
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

// validateBuyRequest validates the buy request parameters
func validateBuyRequest(quantity int) error {
	if quantity <= 0 {
		return fmt.Errorf(ErrMsgInvalidQuantityFmt, quantity, domain.ErrInvalidInput)
	}
	if quantity > domain.MaxTransactionQuantity {
		return fmt.Errorf(ErrMsgQuantityExceedsMaxFmt, quantity, domain.MaxTransactionQuantity, domain.ErrInvalidInput)
	}
	return nil
}

// getBuyEntities retrieves and validates user and item for a buy transaction
func (s *service) getBuyEntities(ctx context.Context, platform, platformID, itemName string) (*domain.User, *domain.Item, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, nil, fmt.Errorf(ErrMsgGetUserFailed, err)
	}
	if user == nil {
		return nil, nil, domain.ErrUserNotFound
	}

	// Resolve public name to internal name
	resolvedName, err := s.resolveItemName(ctx, itemName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve item %q: %w", itemName, err)
	}

	item, err := s.repo.GetItemByName(ctx, resolvedName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get item %q: %w", resolvedName, err)
	}
	if item == nil {
		return nil, nil, fmt.Errorf("item not found: %q: %w", resolvedName, domain.ErrItemNotFound)
	}

	return user, item, nil
}

func (s *service) checkBuyEligibility(ctx context.Context, item *domain.Item) error {
	// Check if item is buyable
	isBuyable, err := s.repo.IsItemBuyable(ctx, item.InternalName)
	if err != nil {
		return fmt.Errorf(ErrMsgCheckBuyableFailed, err)
	}
	if !isBuyable {
		return fmt.Errorf(ErrMsgItemNotBuyableFmt, item.InternalName, domain.ErrNotBuyable)
	}

	// Check if item is unlocked (progression)
	if s.progressionService != nil {
		unlocked, err := s.progressionService.IsItemUnlocked(ctx, item.InternalName)
		if err != nil {
			return fmt.Errorf("failed to check unlock status: %w", err)
		}
		if !unlocked {
			return domain.ErrItemLocked
		}
	}
	return nil
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

// processBuyTransaction handles the inventory updates for buying an item
func processBuyTransaction(inventory *domain.Inventory, item *domain.Item, moneySlotIndex, actualQuantity, cost int) {
	// Deduct money
	if inventory.Slots[moneySlotIndex].Quantity == cost {
		inventory.Slots = append(inventory.Slots[:moneySlotIndex], inventory.Slots[moneySlotIndex+1:]...)
	} else {
		inventory.Slots[moneySlotIndex].Quantity -= cost
	}

	// Add purchased item
	itemFound := false
	for i, slot := range inventory.Slots {
		if slot.ItemID == item.ID {
			inventory.Slots[i].Quantity += actualQuantity
			itemFound = true
			break
		}
	}
	if !itemFound {
		inventory.Slots = append(inventory.Slots, domain.InventorySlot{ItemID: item.ID, Quantity: actualQuantity})
	}
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

// calculateAffordableQuantity determines how many items can be purchased with available money
func calculateAffordableQuantity(desired, unitPrice, balance int) (quantity, cost int) {
	if unitPrice == 0 {
		return desired, 0
	}
	if balance < unitPrice {
		return 0, 0
	}
	maxAffordable := balance / unitPrice
	if desired <= maxAffordable {
		return desired, desired * unitPrice
	}
	return maxAffordable, maxAffordable * unitPrice
}
