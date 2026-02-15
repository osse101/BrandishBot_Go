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

	// Validate request
	if err := validateBuyRequest(quantity); err != nil { // Reuse same validation
		return 0, 0, err
	}

	// Get all required entities
	user, item, moneyItem, err := s.getSellEntities(ctx, platform, platformID, itemName)
	if err != nil {
		return 0, 0, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf(ErrMsgBeginTransactionFailed, err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Get inventory and check if item exists
	inventory, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		return 0, 0, fmt.Errorf(ErrMsgGetInventoryFailed, err)
	}

	// Use random selection in case multiple slots with different quality levels exist
	itemSlotIndex, slotQuantity := utils.FindRandomSlot(inventory, item.ID, s.rnd)
	if itemSlotIndex == -1 {
		return 0, 0, fmt.Errorf(ErrMsgItemNotInInventoryFmt, itemName, domain.ErrNotInInventory)
	}

	// Determine actual sell quantity
	actualSellQuantity := quantity
	if slotQuantity < quantity {
		actualSellQuantity = slotQuantity
	}

	// Process the sell transaction
	moneyGained := s.processSellTransaction(ctx, inventory, item, moneyItem, itemSlotIndex, actualSellQuantity)

	// Save updated inventory
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return 0, 0, fmt.Errorf(ErrMsgUpdateInventoryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf(ErrMsgCommitTransactionFailed, err)
	}

	// Publish item.sold event (job handler awards Merchant XP, quest handler tracks progress)
	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventTypeItemSold),
			Payload: domain.ItemSoldPayload{
				UserID:       user.ID,
				ItemName:     item.InternalName,
				ItemCategory: getItemCategory(item),
				Quantity:     actualSellQuantity,
				TotalValue:   moneyGained,
				Timestamp:    s.now().Unix(),
			},
		})
	}

	log.Info(LogMsgItemSold, "username", username, "item", itemName, "quantity", actualSellQuantity, "moneyGained", moneyGained)
	return moneyGained, actualSellQuantity, nil
}

// getSellEntities retrieves and validates all required entities for a sell transaction
func (s *service) getSellEntities(ctx context.Context, platform, platformID, itemName string) (*domain.User, *domain.Item, *domain.Item, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf(ErrMsgGetUserFailed, err)
	}
	if user == nil {
		return nil, nil, nil, domain.ErrUserNotFound
	}

	// Resolve public name to internal name
	resolvedName, err := s.resolveItemName(ctx, itemName)
	if err != nil {
		return nil, nil, nil, err
	}

	item, err := s.repo.GetItemByName(ctx, resolvedName)
	if err != nil {
		return nil, nil, nil, fmt.Errorf(ErrMsgGetItemFailed, err)
	}
	if item == nil {
		return nil, nil, nil, fmt.Errorf(ErrMsgItemNotFoundFmt, resolvedName, domain.ErrItemNotFound)
	}

	moneyItem, err := s.repo.GetItemByName(ctx, domain.ItemMoney)
	if err != nil {
		return nil, nil, nil, fmt.Errorf(ErrMsgGetMoneyItemFailed, err)
	}
	if moneyItem == nil {
		return nil, nil, nil, fmt.Errorf(ErrMsgItemNotFoundFmt, domain.ItemMoney, domain.ErrItemNotFound)
	}

	return user, item, moneyItem, nil
}

// processSellTransaction handles the inventory updates for selling an item
func (s *service) processSellTransaction(ctx context.Context, inventory *domain.Inventory, item, moneyItem *domain.Item, itemSlotIndex, actualSellQuantity int) int {
	sellPrice := s.calculateSellPriceWithModifier(ctx, item.BaseValue)
	moneyGained := actualSellQuantity * sellPrice

	// Remove sold items
	if inventory.Slots[itemSlotIndex].Quantity <= actualSellQuantity {
		inventory.Slots = append(inventory.Slots[:itemSlotIndex], inventory.Slots[itemSlotIndex+1:]...)
	} else {
		inventory.Slots[itemSlotIndex].Quantity -= actualSellQuantity
	}

	// Add money
	moneyFound := false
	for i, slot := range inventory.Slots {
		if slot.ItemID == moneyItem.ID && slot.QualityLevel == domain.QualityCommon {
			inventory.Slots[i].Quantity += moneyGained
			moneyFound = true
			break
		}
	}
	if !moneyFound {
		inventory.Slots = append(inventory.Slots, domain.InventorySlot{
			ItemID:       moneyItem.ID,
			Quantity:     moneyGained,
			QualityLevel: domain.QualityCommon,
		})
	}

	return moneyGained
}

func (s *service) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgGetSellablePricesCalled)

	allItems, err := s.repo.GetSellablePrices(ctx)
	if err != nil {
		return nil, err
	}

	// Return all items if no progression service
	if s.progressionService == nil {
		// Populate sell prices for all items
		for i := range allItems {
			sellPrice := s.calculateSellPriceWithModifier(ctx, allItems[i].BaseValue)
			allItems[i].SellPrice = &sellPrice
		}
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

	// Filter to only unlocked items and populate sell prices
	filtered := make([]domain.Item, 0, len(allItems))
	for _, item := range allItems {
		if unlockStatus[item.InternalName] {
			// Calculate and set sell price
			sellPrice := s.calculateSellPriceWithModifier(ctx, item.BaseValue)
			item.SellPrice = &sellPrice
			filtered = append(filtered, item)
		}
	}

	log.Info("Sellable prices filtered", "total", len(allItems), "unlocked", len(filtered))
	return filtered, nil
}
