package economy

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/stats"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// SellResult contains the result of a sell operation
type SellResult struct {
	MoneyGained   int  `json:"money_gained"`
	ItemsSold     int  `json:"items_sold"`
	IsMarketSpike bool `json:"is_market_spike"`
	BonusMoney    int  `json:"bonus_money"`
}

// Service defines the interface for economy operations
type Service interface {
	GetSellablePrices(ctx context.Context) ([]domain.Item, error)
	GetBuyablePrices(ctx context.Context) ([]domain.Item, error)
	SellItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (*SellResult, error)
	BuyItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (int, error)
	Shutdown(ctx context.Context) error
}

// JobService defines the interface for job operations
type JobService interface {
	AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error)
}

// MarketSpikeChance is the probability of a market spike occurring (5%)
const MarketSpikeChance = 0.05

// MarketSpikeMultiplier is the multiplier applied when a market spike occurs (1.5x)
const MarketSpikeMultiplier = 1.5

type service struct {
	repo       repository.Economy
	jobService JobService
	statsSvc   stats.Service
	rnd        func() float64 // For rolling RNG
	wg         sync.WaitGroup
}

// NewService creates a new economy service
func NewService(repo repository.Economy, jobService JobService, statsSvc stats.Service) Service {
	return &service{
		repo:       repo,
		jobService: jobService,
		statsSvc:   statsSvc,
		rnd:        utils.RandomFloat,
	}
}

func (s *service) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	log := logger.FromContext(ctx)
	log.Info("GetSellablePrices called")
	return s.repo.GetSellablePrices(ctx)
}

// getSellEntities retrieves and validates all required entities for a sell transaction
func (s *service) getSellEntities(ctx context.Context, platform, platformID, itemName string) (*domain.User, *domain.Item, *domain.Item, error) {
	log := logger.FromContext(ctx)

	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		log.Error("Failed to get user", "error", err)
		return nil, nil, nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, nil, nil, domain.ErrUserNotFound
	}

	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		log.Error("Failed to get item", "error", err)
		return nil, nil, nil, fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return nil, nil, nil, fmt.Errorf("%w: %s", domain.ErrItemNotFound, itemName)
	}

	moneyItem, err := s.repo.GetItemByName(ctx, domain.ItemMoney)
	if err != nil {
		log.Error("Failed to get money item", "error", err)
		return nil, nil, nil, fmt.Errorf("failed to get money item: %w", err)
	}
	if moneyItem == nil {
		return nil, nil, nil, fmt.Errorf("%w: %s", domain.ErrItemNotFound, domain.ItemMoney)
	}

	return user, item, moneyItem, nil
}

// processSellTransaction handles the inventory updates for selling an item
func processSellTransaction(inventory *domain.Inventory, item, moneyItem *domain.Item, itemSlotIndex, actualSellQuantity, totalMoneyGained int) {
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
			inventory.Slots[i].Quantity += totalMoneyGained
			moneyFound = true
			break
		}
	}
	if !moneyFound {
		inventory.Slots = append(inventory.Slots, domain.InventorySlot{ItemID: moneyItem.ID, Quantity: totalMoneyGained})
	}
}

func (s *service) SellItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (*SellResult, error) {
	log := logger.FromContext(ctx)
	log.Info("SellItem called", "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", quantity)

	// Validate request
	if err := validateBuyRequest(quantity); err != nil { // Reuse same validation
		return nil, err
	}

	// Get all required entities
	user, item, moneyItem, err := s.getSellEntities(ctx, platform, platformID, itemName)
	if err != nil {
		return nil, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "error", err)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Get inventory and check if item exists
	inventory, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		log.Error("Failed to get inventory", "error", err)
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
	if itemSlotIndex == -1 {
		return nil, fmt.Errorf("item %s not in inventory", itemName)
	}

	// Determine actual sell quantity
	actualSellQuantity := quantity
	if slotQuantity < quantity {
		actualSellQuantity = slotQuantity
	}

	// Calculate base money gained
	baseMoneyGained := actualSellQuantity * item.BaseValue
	totalMoneyGained := baseMoneyGained
	isMarketSpike := false
	bonusMoney := 0

	// Check for Market Spike
	if s.rnd() < MarketSpikeChance {
		isMarketSpike = true
		totalMoneyGained = int(math.Ceil(float64(baseMoneyGained) * MarketSpikeMultiplier))
		bonusMoney = totalMoneyGained - baseMoneyGained

		log.Info("Market Spike triggered!", "user_id", user.ID, "item", itemName, "base_value", baseMoneyGained, "bonus", bonusMoney)

		if s.statsSvc != nil {
			_ = s.statsSvc.RecordUserEvent(ctx, user.ID, domain.EventEconomyMarketSpike, map[string]interface{}{
				"item_name":       itemName,
				"quantity":        actualSellQuantity,
				"base_value":      baseMoneyGained,
				"total_value":     totalMoneyGained,
				"bonus_value":     bonusMoney,
				"multiplier":      MarketSpikeMultiplier,
			})
		}
	}

	// Process the sell transaction
	processSellTransaction(inventory, item, moneyItem, itemSlotIndex, actualSellQuantity, totalMoneyGained)

	// Save updated inventory
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		log.Error("Failed to update inventory", "error", err)
		return nil, fmt.Errorf("failed to update inventory: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("Failed to commit transaction", "error", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Award Merchant XP based on transaction value (async)
	xp := calculateMerchantXP(totalMoneyGained)
	s.wg.Add(1)
	go s.awardMerchantXP(context.Background(), user.ID, xp, "sell", itemName, totalMoneyGained)

	log.Info("Item sold", "username", username, "item", itemName, "quantity", actualSellQuantity, "moneyGained", totalMoneyGained, "marketSpike", isMarketSpike)

	return &SellResult{
		MoneyGained:   totalMoneyGained,
		ItemsSold:     actualSellQuantity,
		IsMarketSpike: isMarketSpike,
		BonusMoney:    bonusMoney,
	}, nil
}

// validateBuyRequest validates the buy request parameters
func validateBuyRequest(quantity int) error {
	if quantity <= 0 {
		return fmt.Errorf("invalid %w: %d", domain.ErrInvalidInput, quantity)
	}
	if quantity > domain.MaxTransactionQuantity {
		return fmt.Errorf("quantity %d exceeds maximum allowed (%d)", quantity, domain.MaxTransactionQuantity)
	}
	return nil
}

// getBuyEntities retrieves and validates user and item for a buy transaction
func (s *service) getBuyEntities(ctx context.Context, platform, platformID, itemName string) (*domain.User, *domain.Item, error) {
	log := logger.FromContext(ctx)

	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		log.Error("Failed to get user", "error", err)
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, nil, fmt.Errorf("user not found")
	}

	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		log.Error("Failed to get item", "error", err)
		return nil, nil, fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return nil, nil, fmt.Errorf("item not found: %s", itemName)
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
	log.Info("BuyItem called", "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", quantity)

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
		log.Error("Failed to begin transaction", "error", err)
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Check if item is buyable
	isBuyable, err := s.repo.IsItemBuyable(ctx, itemName)
	if err != nil {
		log.Error("Failed to check buyable", "error", err)
		return 0, fmt.Errorf("failed to check if item is buyable: %w", err)
	}
	if !isBuyable {
		return 0, fmt.Errorf("item %s is not buyable", itemName)
	}

	// Get money item after buyable check
	moneyItem, err := s.repo.GetItemByName(ctx, domain.ItemMoney)
	if err != nil {
		log.Error("Failed to get money item", "error", err)
		return 0, fmt.Errorf("Failed to get money item: %w", err)
	}
	if moneyItem == nil {
		log.Error("Money item not found")
		return 0, fmt.Errorf("money item not found")
	}

	// Get inventory and check funds
	inventory, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		log.Error("Failed to get inventory", "error", err)
		return 0, fmt.Errorf("failed to get inventory: %w", err)
	}

	moneySlotIndex, moneyBalance := utils.FindSlot(inventory, moneyItem.ID)
	if moneyBalance <= 0 {
		return 0, fmt.Errorf("insufficient funds")
	}

	// Calculate affordable quantity
	actualQuantity, cost := calculateAffordableQuantity(quantity, item.BaseValue, moneyBalance)
	if actualQuantity == 0 {
		return 0, fmt.Errorf("insufficient funds to buy even one %s (cost: %d, balance: %d)", itemName, item.BaseValue, moneyBalance)
	}

	if quantity > actualQuantity {
		log.Info("Adjusted purchase quantity due to funds", "requested", quantity, "actual", actualQuantity)
	}

	// Process the transaction
	processBuyTransaction(inventory, item, moneySlotIndex, actualQuantity, cost)

	// Save updated inventory
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		log.Error("Failed to update inventory", "error", err)
		return 0, fmt.Errorf("failed to update inventory: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("Failed to commit transaction", "error", err)
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Award Merchant XP based on transaction value (async)
	xp := calculateMerchantXP(cost)
	s.wg.Add(1)
	go s.awardMerchantXP(context.Background(), user.ID, xp, "buy", itemName, cost)

	log.Info("Item purchased", "username", username, "item", itemName, "quantity", actualQuantity)
	return actualQuantity, nil
}

// calculateAffordableQuantity determines how many items can be purchased with available money
func calculateAffordableQuantity(desired, unitPrice, balance int) (quantity, cost int) {
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
		"action":    action,
		"item_name": itemName,
		"value":     value,
	}

	result, err := s.jobService.AwardXP(ctx, userID, job.JobKeyMerchant, xp, action, metadata)
	if err != nil {
		logger.FromContext(ctx).Warn("Failed to award Merchant XP", "error", err, "user_id", userID)
	} else if result != nil && result.LeveledUp {
		logger.FromContext(ctx).Info("Merchant leveled up!", "user_id", userID, "new_level", result.NewLevel)
	}
}

func (s *service) Shutdown(ctx context.Context) error {
	logger.FromContext(ctx).Info("Economy service shutting down, waiting for background tasks...")
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timed out: %w", ctx.Err())
	}
}
