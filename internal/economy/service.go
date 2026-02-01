package economy

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// Service defines the interface for economy operations
type Service interface {
	GetSellablePrices(ctx context.Context) ([]domain.Item, error)
	GetBuyablePrices(ctx context.Context) ([]domain.Item, error)
	SellItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (int, int, error)
	BuyItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (int, error)
	Shutdown(ctx context.Context) error
}

// JobService defines the interface for job operations
type JobService interface {
	AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error)
}

// ProgressionService defines the interface for progression operations
type ProgressionService interface {
	IsItemUnlocked(ctx context.Context, itemName string) (bool, error)
	AreItemsUnlocked(ctx context.Context, itemNames []string) (map[string]bool, error)
}

type service struct {
	repo               repository.Economy
	jobService         JobService
	namingResolver     naming.Resolver
	progressionService ProgressionService
	wg                 sync.WaitGroup
}

// NewService creates a new economy service
func NewService(repo repository.Economy, jobService JobService, namingResolver naming.Resolver, progressionService ProgressionService) Service {
	return &service{
		repo:               repo,
		jobService:         jobService,
		namingResolver:     namingResolver,
		progressionService: progressionService,
	}
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
			sellPrice := calculateSellPrice(allItems[i].BaseValue)
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
			sellPrice := calculateSellPrice(item.BaseValue)
			item.SellPrice = &sellPrice
			filtered = append(filtered, item)
		}
	}

	log.Info("Sellable prices filtered", "total", len(allItems), "unlocked", len(filtered))
	return filtered, nil
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

// resolveItemName attempts to resolve a user-provided item name to its internal name.
// It first tries the naming resolver, then falls back to using the input as-is.
// This allows users to use either public names ("junkbox") or internal names ("lootbox_tier0").
func (s *service) resolveItemName(ctx context.Context, itemName string) (string, error) {
	// Try naming resolver first (handles public names)
	if s.namingResolver != nil {
		if internalName, ok := s.namingResolver.ResolvePublicName(itemName); ok {
			return internalName, nil
		}
	}

	// Fall back - assume it's already an internal name
	// Validate by checking if item exists
	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		return "", fmt.Errorf(ErrMsgResolveItemFailedFmt, itemName, err)
	}
	if item == nil {
		return "", fmt.Errorf(ErrMsgItemNotFoundPublicFmt, itemName, domain.ErrItemNotFound)
	}

	return itemName, nil
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

// calculateSellPrice calculates the sell price for an item based on its base value.
// Uses SellPriceRatio to determine the percentage of base_value returned when selling.
// Returns integer price (rounded down to prevent fractional currency).
func calculateSellPrice(baseValue int) int {
	return int(float64(baseValue) * SellPriceRatio)
}

// processSellTransaction handles the inventory updates for selling an item
func processSellTransaction(inventory *domain.Inventory, item, moneyItem *domain.Item, itemSlotIndex, actualSellQuantity int) int {
	sellPrice := calculateSellPrice(item.BaseValue)
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
		if slot.ItemID == moneyItem.ID {
			inventory.Slots[i].Quantity += moneyGained
			moneyFound = true
			break
		}
	}
	if !moneyFound {
		inventory.Slots = append(inventory.Slots, domain.InventorySlot{ItemID: moneyItem.ID, Quantity: moneyGained})
	}

	return moneyGained
}

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

	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
	if itemSlotIndex == -1 {
		return 0, 0, fmt.Errorf(ErrMsgItemNotInInventoryFmt, itemName, domain.ErrNotInInventory)
	}

	// Determine actual sell quantity
	actualSellQuantity := quantity
	if slotQuantity < quantity {
		actualSellQuantity = slotQuantity
	}

	// Process the sell transaction
	moneyGained := processSellTransaction(inventory, item, moneyItem, itemSlotIndex, actualSellQuantity)

	// Save updated inventory
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return 0, 0, fmt.Errorf(ErrMsgUpdateInventoryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf(ErrMsgCommitTransactionFailed, err)
	}

	// Award Merchant XP based on transaction value (async)
	xp := calculateMerchantXP(moneyGained)
	s.wg.Add(1)
	go s.awardMerchantXP(context.Background(), user.ID, xp, ActionTypeSell, itemName, moneyGained)

	log.Info(LogMsgItemSold, "username", username, "item", itemName, "quantity", actualSellQuantity, "moneyGained", moneyGained)
	return moneyGained, actualSellQuantity, nil
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

func (s *service) BuyItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (int, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgBuyItemCalled, "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", quantity)

	// Validate request
	if err := validateBuyRequest(quantity); err != nil {
		return 0, err
	}

	// Get user and item
	user, item, err := s.getBuyEntities(ctx, platform, platformID, itemName)
	if err != nil {
		return 0, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, fmt.Errorf(ErrMsgBeginTransactionFailed, err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Check if item is buyable (use internal name from resolved item)
	isBuyable, err := s.repo.IsItemBuyable(ctx, item.InternalName)
	if err != nil {
		return 0, fmt.Errorf(ErrMsgCheckBuyableFailed, err)
	}
	if !isBuyable {
		return 0, fmt.Errorf(ErrMsgItemNotBuyableFmt, item.InternalName, domain.ErrNotBuyable)
	}

	// Check if item is unlocked (progression)
	if s.progressionService != nil {
		unlocked, err := s.progressionService.IsItemUnlocked(ctx, item.InternalName)
		if err != nil {
			return 0, fmt.Errorf("failed to check unlock status: %w", err)
		}
		if !unlocked {
			return 0, domain.ErrItemLocked
		}
	}

	// Get money item after buyable check
	moneyItem, err := s.repo.GetItemByName(ctx, domain.ItemMoney)
	if err != nil {
		return 0, fmt.Errorf(ErrMsgGetMoneyItemFailed, err)
	}
	if moneyItem == nil {
		return 0, fmt.Errorf(ErrMsgItemNotFoundFmt, domain.ItemMoney, domain.ErrItemNotFound)
	}

	// Get inventory and check funds
	inventory, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		return 0, fmt.Errorf(ErrMsgGetInventoryFailed, err)
	}

	moneySlotIndex, moneyBalance := utils.FindSlot(inventory, moneyItem.ID)
	if moneyBalance <= 0 {
		return 0, domain.ErrInsufficientFunds
	}

	// Calculate affordable quantity
	actualQuantity, cost := calculateAffordableQuantity(quantity, item.BaseValue, moneyBalance)
	if actualQuantity == 0 {
		return 0, fmt.Errorf(ErrMsgInsufficientFundsToBuyOneFmt, itemName, item.BaseValue, moneyBalance, domain.ErrInsufficientFunds)
	}

	if quantity > actualQuantity {
		log.Info(LogMsgAdjustedPurchaseQty, "requested", quantity, "actual", actualQuantity)
	}

	// Process the transaction
	processBuyTransaction(inventory, item, moneySlotIndex, actualQuantity, cost)

	// Save updated inventory
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return 0, fmt.Errorf(ErrMsgUpdateInventoryFailed, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf(ErrMsgCommitTransactionFailed, err)
	}

	// Award Merchant XP based on transaction value (async)
	xp := calculateMerchantXP(cost)
	s.wg.Add(1)
	go s.awardMerchantXP(context.Background(), user.ID, xp, ActionTypeBuy, itemName, cost)

	log.Info(LogMsgItemPurchased, "username", username, "item", itemName, "quantity", actualQuantity)
	return actualQuantity, nil
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

// calculateMerchantXP calculates XP based on transaction value
// Formula: XP = ceil(transactionValue / 10)
func calculateMerchantXP(transactionValue int) int {
	return int(math.Ceil(float64(transactionValue) / job.MerchantXPValueDivisor))
}

// awardMerchantXP awards Merchant job XP for buy/sell transactions
// NOTE: Caller must call s.wg.Add(1) before launching this in a goroutine
func (s *service) awardMerchantXP(ctx context.Context, userID string, xp int, action, itemName string, value int) {
	defer s.wg.Done()

	if s.jobService == nil || xp <= 0 {
		return
	}

	metadata := map[string]interface{}{
		MetadataKeyAction:   action,
		MetadataKeyItemName: itemName,
		MetadataKeyValue:    value,
	}

	result, err := s.jobService.AwardXP(ctx, userID, job.JobKeyMerchant, xp, action, metadata)
	if err != nil {
		logger.FromContext(ctx).Error(ErrMsgAwardMerchantXPFailed, userID, err)
	} else if result != nil && result.LeveledUp {
		logger.FromContext(ctx).Info(LogMsgMerchantLeveledUp, "user_id", userID, "new_level", result.NewLevel)
	}
}

func (s *service) Shutdown(ctx context.Context) error {
	logger.FromContext(ctx).Info(LogMsgEconomyShuttingDown)
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf(ErrMsgShutdownTimedOut, ctx.Err())
	}
}
