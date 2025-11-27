package economy

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/concurrency"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// Repository defines the interface for data access required by the economy service
type Repository interface {
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
	GetItemByName(ctx context.Context, itemName string) (*domain.Item, error)
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
	GetSellablePrices(ctx context.Context) ([]domain.Item, error)
	IsItemBuyable(ctx context.Context, itemName string) (bool, error)
}

// Service defines the interface for economy operations
type Service interface {
	GetSellablePrices(ctx context.Context) ([]domain.Item, error)
	SellItem(ctx context.Context, username, platform, itemName string, quantity int) (int, int, error)
	BuyItem(ctx context.Context, username, platform, itemName string, quantity int) (int, error)
}

type service struct {
	repo        Repository
	lockManager *concurrency.LockManager
}

// NewService creates a new economy service
func NewService(repo Repository, lockManager *concurrency.LockManager) Service {
	return &service{
		repo:        repo,
		lockManager: lockManager,
	}
}

func (s *service) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	log := logger.FromContext(ctx)
	log.Info("GetSellablePrices called")
	return s.repo.GetSellablePrices(ctx)
}

func (s *service) SellItem(ctx context.Context, username, platform, itemName string, quantity int) (int, int, error) {
	log := logger.FromContext(ctx)
	log.Info("SellItem called", "username", username, "item", itemName, "quantity", quantity)
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		log.Error("Failed to get user", "error", err, "username", username)
		return 0, 0, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		log.Warn("User not found", "username", username)
		return 0, 0, fmt.Errorf("user not found: %s", username)
	}

	lock := s.lockManager.GetLock(user.ID)
	lock.Lock()
	defer lock.Unlock()

	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		log.Error("Failed to get item", "error", err, "itemName", itemName)
		return 0, 0, fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		log.Warn("Item not found", "itemName", itemName)
		return 0, 0, fmt.Errorf("item not found: %s", itemName)
	}
	moneyItem, err := s.repo.GetItemByName(ctx, domain.ItemMoney)
	if err != nil {
		log.Error("Failed to get money item", "error", err)
		return 0, 0, fmt.Errorf("failed to get money item: %w", err)
	}
	if moneyItem == nil {
		log.Error("Money item not found")
		return 0, 0, fmt.Errorf("money item not found")
	}
	inventory, err := s.repo.GetInventory(ctx, user.ID)
	if err != nil {
		log.Error("Failed to get inventory", "error", err, "userID", user.ID)
		return 0, 0, fmt.Errorf("failed to get inventory: %w", err)
	}
	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
	if itemSlotIndex == -1 {
		log.Warn("Item not in inventory", "itemName", itemName)
		return 0, 0, fmt.Errorf("item %s not in inventory", itemName)
	}
	actualSellQuantity := quantity
	if slotQuantity < quantity {
		actualSellQuantity = slotQuantity
	}
	moneyGained := actualSellQuantity * item.BaseValue
	if inventory.Slots[itemSlotIndex].Quantity <= actualSellQuantity {
		inventory.Slots = append(inventory.Slots[:itemSlotIndex], inventory.Slots[itemSlotIndex+1:]...)
	} else {
		inventory.Slots[itemSlotIndex].Quantity -= actualSellQuantity
	}
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
	if err := s.repo.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		log.Error("Failed to update inventory", "error", err, "userID", user.ID)
		return 0, 0, fmt.Errorf("failed to update inventory: %w", err)
	}
	log.Info("Item sold", "username", username, "item", itemName, "quantity", actualSellQuantity, "moneyGained", moneyGained)
	return moneyGained, actualSellQuantity, nil
}

func (s *service) BuyItem(ctx context.Context, username, platform, itemName string, quantity int) (int, error) {
	log := logger.FromContext(ctx)
	log.Info("BuyItem called", "username", username, "item", itemName, "quantity", quantity)

	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		log.Error("Failed to get user", "error", err, "username", username)
		return 0, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		log.Warn("User not found", "username", username)
		return 0, fmt.Errorf("user not found: %s", username)
	}

	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		log.Error("Failed to get item", "error", err, "itemName", itemName)
		return 0, fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		log.Warn("Item not found", "itemName", itemName)
		return 0, fmt.Errorf("item not found: %s", itemName)
	}

	lock := s.lockManager.GetLock(user.ID)
	lock.Lock()
	defer lock.Unlock()

	isBuyable, err := s.repo.IsItemBuyable(ctx, itemName)
	if err != nil {
		log.Error("Failed to check buyable", "error", err, "itemName", itemName)
		return 0, fmt.Errorf("failed to check if item is buyable: %w", err)
	}
	if !isBuyable {
		log.Warn("Item not buyable", "itemName", itemName)
		return 0, fmt.Errorf("item %s is not buyable", itemName)
	}

	moneyItem, err := s.repo.GetItemByName(ctx, domain.ItemMoney)
	if err != nil {
		log.Error("Failed to get money item", "error", err)
		return 0, fmt.Errorf("failed to get money item: %w", err)
	}
	if moneyItem == nil {
		log.Error("Money item not found")
		return 0, fmt.Errorf("money item not found")
	}

	inventory, err := s.repo.GetInventory(ctx, user.ID)
	if err != nil {
		log.Error("Failed to get inventory", "error", err, "userID", user.ID)
		return 0, fmt.Errorf("failed to get inventory: %w", err)
	}

	moneySlotIndex, moneyBalance := utils.FindSlot(inventory, moneyItem.ID)
	if moneyBalance <= 0 {
		log.Warn("Insufficient funds", "username", username)
		return 0, fmt.Errorf("insufficient funds")
	}

	actualQuantity, cost := calculateAffordableQuantity(quantity, item.BaseValue, moneyBalance)
	if actualQuantity == 0 {
		log.Warn("Insufficient funds for any quantity", "username", username, "item", itemName)
		return 0, fmt.Errorf("insufficient funds to buy even one %s (cost: %d, balance: %d)", itemName, item.BaseValue, moneyBalance)
	}
	
	if quantity > actualQuantity {
		log.Info("Adjusted purchase quantity due to funds", "requested", quantity, "actual", actualQuantity)
	}

	if inventory.Slots[moneySlotIndex].Quantity == cost {
		inventory.Slots = append(inventory.Slots[:moneySlotIndex], inventory.Slots[moneySlotIndex+1:]...)
	} else {
		inventory.Slots[moneySlotIndex].Quantity -= cost
	}

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

	if err := s.repo.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		log.Error("Failed to update inventory", "error", err, "userID", user.ID)
		return 0, fmt.Errorf("failed to update inventory: %w", err)
	}

	log.Info("Item purchased", "username", username, "item", itemName, "quantity", actualQuantity)
	return actualQuantity, nil
}
