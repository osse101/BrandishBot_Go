package user

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Repository defines the interface for user persistence
type Repository interface {
	UpsertUser(ctx context.Context, user *domain.User) error
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
	GetItemByName(ctx context.Context, itemName string) (*domain.Item, error)
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
	GetSellablePrices(ctx context.Context) ([]domain.Item, error)
	IsItemBuyable(ctx context.Context, itemName string) (bool, error)
}

// Service defines the interface for user operations
type Service interface {
	RegisterUser(ctx context.Context, user domain.User) (domain.User, error)
	FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	HandleIncomingMessage(ctx context.Context, platform, platformID, username string) (domain.User, error)
	AddItem(ctx context.Context, username, platform, itemName string, quantity int) error
	RemoveItem(ctx context.Context, username, platform, itemName string, quantity int) (int, error)
	GiveItem(ctx context.Context, ownerUsername, receiverUsername, platform, itemName string, quantity int) error
	GetSellablePrices(ctx context.Context) ([]domain.Item, error)
	SellItem(ctx context.Context, username, platform, itemName string, quantity int) (moneyGained int, itemsSold int, err error)
	BuyItem(ctx context.Context, username, platform, itemName string, quantity int) (bought int, err error)
	UseItem(ctx context.Context, username, platform, itemName string, quantity int, targetUsername string) (message string, err error)
}

// service implements the Service interface
type service struct {
	repo Repository
}

// NewService creates a new user service
func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

// RegisterUser registers a new user
func (s *service) RegisterUser(ctx context.Context, user domain.User) (domain.User, error) {
	if err := s.repo.UpsertUser(ctx, &user); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

// FindUserByPlatformID finds a user by their platform-specific ID
func (s *service) FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return s.repo.GetUserByPlatformID(ctx, platform, platformID)
}

// HandleIncomingMessage checks if a user exists for an incoming message and creates one if not.
func (s *service) HandleIncomingMessage(ctx context.Context, platform, platformID, username string) (domain.User, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return domain.User{}, fmt.Errorf("failed to get user: %w", err)
	}
	if user != nil {
		return *user, nil
	}

	// TODO: Check if error is actually "not found"

	newUser := domain.User{
		Username: username,
	}

	switch platform {
	case "twitch":
		newUser.TwitchID = platformID
	case "youtube":
		newUser.YoutubeID = platformID
	case "discord":
		newUser.DiscordID = platformID
	default:
		return domain.User{}, fmt.Errorf("unsupported platform: %s", platform)
	}

	if _, err := s.RegisterUser(ctx, newUser); err != nil {
		return domain.User{}, err
	}

	return newUser, nil
}

func (s *service) AddItem(ctx context.Context, username, platform, itemName string, quantity int) error {
	// 1. Get User
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found: %s", username)
	}

	// 2. Get Item
	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return fmt.Errorf("item not found: %s", itemName)
	}

	// 3. Get Inventory
	inventory, err := s.repo.GetInventory(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	// 4. Update Inventory
	found := false
	for i, slot := range inventory.Slots {
		if slot.ItemID == item.ID {
			inventory.Slots[i].Quantity += quantity
			found = true
			break
		}
	}

	if !found {
		inventory.Slots = append(inventory.Slots, domain.InventorySlot{
			ItemID:   item.ID,
			Quantity: quantity,
		})
	}

	// 5. Save Inventory
	if err := s.repo.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	return nil
}

func (s *service) RemoveItem(ctx context.Context, username, platform, itemName string, quantity int) (int, error) {
	// 1. Get User
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return 0, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return 0, fmt.Errorf("user not found: %s", username)
	}

	// 2. Get Item
	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		return 0, fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return 0, fmt.Errorf("item not found: %s", itemName)
	}

	// 3. Get Inventory
	inventory, err := s.repo.GetInventory(ctx, user.ID)
	if err != nil {
		return 0, fmt.Errorf("failed to get inventory: %w", err)
	}

	// 4. Find and remove from inventory
	var removed int
	found := false
	for i, slot := range inventory.Slots {
		if slot.ItemID == item.ID {
			found = true
			// Calculate how many we can actually remove
			removed = quantity
			if slot.Quantity < quantity {
				removed = slot.Quantity
			}
			
			// Update or remove the slot
			if slot.Quantity <= quantity {
				// Remove the slot entirely
				inventory.Slots = append(inventory.Slots[:i], inventory.Slots[i+1:]...)
			} else {
				// Just decrease the quantity
				inventory.Slots[i].Quantity -= removed
			}
			break
		}
	}

	if !found {
		return 0, fmt.Errorf("item %s not in inventory", itemName)
	}

	// 5. Save Inventory
	if err := s.repo.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return 0, fmt.Errorf("failed to update inventory: %w", err)
	}

	return removed, nil
}

func (s *service) GiveItem(ctx context.Context, ownerUsername, receiverUsername, platform, itemName string, quantity int) error {
	// 1. Get Owner
	owner, err := s.repo.GetUserByUsername(ctx, ownerUsername)
	if err != nil {
		return fmt.Errorf("failed to get owner: %w", err)
	}
	if owner == nil {
		return fmt.Errorf("owner not found: %s", ownerUsername)
	}

	// 2. Get Receiver
	receiver, err := s.repo.GetUserByUsername(ctx, receiverUsername)
	if err != nil {
		return fmt.Errorf("failed to get receiver: %w", err)
	}
	if receiver == nil {
		return fmt.Errorf("receiver not found: %s", receiverUsername)
	}

	// 3. Get Item
	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		return fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return fmt.Errorf("item not found: %s", itemName)
	}

	// 4. Get Owner's Inventory and Validate
	ownerInventory, err := s.repo.GetInventory(ctx, owner.ID)
	if err != nil {
		return fmt.Errorf("failed to get owner inventory: %w", err)
	}

	// Find item in owner's inventory and validate quantity
	var ownerSlotIndex int = -1
	var ownerHasEnough bool = false
	for i, slot := range ownerInventory.Slots {
		if slot.ItemID == item.ID {
			ownerSlotIndex = i
			if slot.Quantity >= quantity {
				ownerHasEnough = true
			}
			break
		}
	}

	if ownerSlotIndex == -1 {
		return fmt.Errorf("owner does not have item %s in inventory", itemName)
	}

	if !ownerHasEnough {
		return fmt.Errorf("owner does not have enough %s (has %d, needs %d)", 
			itemName, ownerInventory.Slots[ownerSlotIndex].Quantity, quantity)
	}

	// 5. Get Receiver's Inventory
	receiverInventory, err := s.repo.GetInventory(ctx, receiver.ID)
	if err != nil {
		return fmt.Errorf("failed to get receiver inventory: %w", err)
	}

	// 6. Remove from owner
	if ownerInventory.Slots[ownerSlotIndex].Quantity == quantity {
		// Remove slot entirely
		ownerInventory.Slots = append(ownerInventory.Slots[:ownerSlotIndex], ownerInventory.Slots[ownerSlotIndex+1:]...)
	} else {
		// Decrease quantity
		ownerInventory.Slots[ownerSlotIndex].Quantity -= quantity
	}

	// 7. Add to receiver
	found := false
	for i, slot := range receiverInventory.Slots {
		if slot.ItemID == item.ID {
			receiverInventory.Slots[i].Quantity += quantity
			found = true
			break
		}
	}
	if !found {
		receiverInventory.Slots = append(receiverInventory.Slots, domain.InventorySlot{
			ItemID:   item.ID,
			Quantity: quantity,
		})
	}

	// 8. Update both inventories
	if err := s.repo.UpdateInventory(ctx, owner.ID, *ownerInventory); err != nil {
		return fmt.Errorf("failed to update owner inventory: %w", err)
	}

	if err := s.repo.UpdateInventory(ctx, receiver.ID, *receiverInventory); err != nil {
		// Note: In a real application, this should be wrapped in a database transaction
		// to ensure atomicity. For now, we have a small risk of inconsistent state.
		return fmt.Errorf("failed to update receiver inventory: %w", err)
	}

	return nil
}

func (s *service) GetSellablePrices(ctx context.Context) ([]domain.Item, error) {
	return s.repo.GetSellablePrices(ctx)
}

func (s *service) SellItem(ctx context.Context, username, platform, itemName string, quantity int) (int, int, error) {
	// 1. Get User
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return 0, 0, fmt.Errorf("user not found: %s", username)
	}

	// 2. Get Item to Sell
	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return 0, 0, fmt.Errorf("item not found: %s", itemName)
	}

	// 3. Get Money Item
	moneyItem, err := s.repo.GetItemByName(ctx, "money")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get money item: %w", err)
	}
	if moneyItem == nil {
		return 0, 0, fmt.Errorf("money item not found")
	}

	// 4. Get Inventory
	inventory, err := s.repo.GetInventory(ctx, user.ID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get inventory: %w", err)
	}

	// 5. Find item in inventory and calculate sell quantity
	var itemSlotIndex int = -1
	var actualSellQuantity int = 0
	for i, slot := range inventory.Slots {
		if slot.ItemID == item.ID {
			itemSlotIndex = i
			// Sell min of requested or owned
			if slot.Quantity < quantity {
				actualSellQuantity = slot.Quantity
			} else {
				actualSellQuantity = quantity
			}
			break
		}
	}

	if itemSlotIndex == -1 {
		return 0, 0, fmt.Errorf("item %s not in inventory", itemName)
	}

	// 6. Calculate money gained
	moneyGained := actualSellQuantity * item.BaseValue

	// 7. Remove sold items from inventory
	if inventory.Slots[itemSlotIndex].Quantity <= actualSellQuantity {
		// Remove slot entirely
		inventory.Slots = append(inventory.Slots[:itemSlotIndex], inventory.Slots[itemSlotIndex+1:]...)
	} else {
		// Decrease quantity
		inventory.Slots[itemSlotIndex].Quantity -= actualSellQuantity
	}

	// 8. Add money to inventory
	moneyFound := false
	for i, slot := range inventory.Slots {
		if slot.ItemID == moneyItem.ID {
			inventory.Slots[i].Quantity += moneyGained
			moneyFound = true
			break
		}
	}
	if !moneyFound {
		inventory.Slots = append(inventory.Slots, domain.InventorySlot{
			ItemID:   moneyItem.ID,
			Quantity: moneyGained,
		})
	}

	// 9. Update inventory
	if err := s.repo.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return 0, 0, fmt.Errorf("failed to update inventory: %w", err)
	}

	return moneyGained, actualSellQuantity, nil
}

func (s *service) BuyItem(ctx context.Context, username, platform, itemName string, quantity int) (int, error) {
	// 1. Get User
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return 0, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return 0, fmt.Errorf("user not found: %s", username)
	}

	// 2. Get Item to Buy
	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		return 0, fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return 0, fmt.Errorf("item not found: %s", itemName)
	}

	// 3. Check if Item is Buyable
	isBuyable, err := s.repo.IsItemBuyable(ctx, itemName)
	if err != nil {
		return 0, fmt.Errorf("failed to check if item is buyable: %w", err)
	}
	if !isBuyable {
		return 0, fmt.Errorf("item %s is not buyable", itemName)
	}

	// 4. Get Money Item
	moneyItem, err := s.repo.GetItemByName(ctx, "money")
	if err != nil {
		return 0, fmt.Errorf("failed to get money item: %w", err)
	}
	if moneyItem == nil {
		return 0, fmt.Errorf("money item not found")
	}

	// 5. Get Inventory
	inventory, err := s.repo.GetInventory(ctx, user.ID)
	if err != nil {
		return 0, fmt.Errorf("failed to get inventory: %w", err)
	}

	// 6. Check Money Balance
	var moneyBalance int = 0
	var moneySlotIndex int = -1
	for i, slot := range inventory.Slots {
		if slot.ItemID == moneyItem.ID {
			moneyBalance = slot.Quantity
			moneySlotIndex = i
			break
		}
	}

	if moneyBalance <= 0 {
		return 0, fmt.Errorf("insufficient funds")
	}

	// 7. Calculate Affordable Quantity
	maxAffordable := moneyBalance / item.BaseValue
	if maxAffordable == 0 {
		return 0, fmt.Errorf("insufficient funds to buy even one %s (cost: %d, balance: %d)", itemName, item.BaseValue, moneyBalance)
	}

	actualQuantity := quantity
	if actualQuantity > maxAffordable {
		actualQuantity = maxAffordable
	}

	// 8. Deduct Money
	cost := actualQuantity * item.BaseValue
	if inventory.Slots[moneySlotIndex].Quantity == cost {
		// Remove money slot if balance becomes 0
		inventory.Slots = append(inventory.Slots[:moneySlotIndex], inventory.Slots[moneySlotIndex+1:]...)
	} else {
		inventory.Slots[moneySlotIndex].Quantity -= cost
	}

	// 9. Add Bought Items
	itemFound := false
	for i, slot := range inventory.Slots {
		if slot.ItemID == item.ID {
			inventory.Slots[i].Quantity += actualQuantity
			itemFound = true
			break
		}
	}
	if !itemFound {
		inventory.Slots = append(inventory.Slots, domain.InventorySlot{
			ItemID:   item.ID,
			Quantity: actualQuantity,
		})
	}

	// 10. Update Inventory
	if err := s.repo.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return 0, fmt.Errorf("failed to update inventory: %w", err)
	}

	return actualQuantity, nil
}

func (s *service) UseItem(ctx context.Context, username, platform, itemName string, quantity int, targetUsername string) (string, error) {
	// 1. Get User
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return "", fmt.Errorf("user not found: %s", username)
	}

	// 2. Get Inventory
	inventory, err := s.repo.GetInventory(ctx, user.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get inventory: %w", err)
	}

	// 3. Get Item to Use
	itemToUse, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		return "", fmt.Errorf("failed to get item: %w", err)
	}
	if itemToUse == nil {
		return "", fmt.Errorf("item not found: %s", itemName)
	}

	// 4. Check if user has enough of the item
	var itemSlotIndex int = -1
	for i, slot := range inventory.Slots {
		if slot.ItemID == itemToUse.ID {
			itemSlotIndex = i
			break
		}
	}

	if itemSlotIndex == -1 || inventory.Slots[itemSlotIndex].Quantity < quantity {
		return "", fmt.Errorf("insufficient quantity of %s", itemName)
	}

	// 5. Handle Item Effects
	var message string
	switch itemName {
	case "lootbox1":
		// Effect: Consume 1 lootbox1, Grant 1 lootbox0
		
		// Get lootbox0
		lootbox0, err := s.repo.GetItemByName(ctx, "lootbox0")
		if err != nil {
			return "", fmt.Errorf("failed to get lootbox0: %w", err)
		}
		if lootbox0 == nil {
			return "", fmt.Errorf("lootbox0 not found")
		}

		// Remove lootbox1
		if inventory.Slots[itemSlotIndex].Quantity == quantity {
			inventory.Slots = append(inventory.Slots[:itemSlotIndex], inventory.Slots[itemSlotIndex+1:]...)
		} else {
			inventory.Slots[itemSlotIndex].Quantity -= quantity
		}

		// Add lootbox0
		found := false
		for i, slot := range inventory.Slots {
			if slot.ItemID == lootbox0.ID {
				inventory.Slots[i].Quantity += quantity
				found = true
				break
			}
		}
		if !found {
			inventory.Slots = append(inventory.Slots, domain.InventorySlot{
				ItemID:   lootbox0.ID,
				Quantity: quantity,
			})
		}
		message = fmt.Sprintf("Used %d lootbox1", quantity)

	case "blaster":
		// Effect: Consume quantity blasters, return message
		if targetUsername == "" {
			return "", fmt.Errorf("target username is required for blaster")
		}

		// Remove blaster
		if inventory.Slots[itemSlotIndex].Quantity == quantity {
			inventory.Slots = append(inventory.Slots[:itemSlotIndex], inventory.Slots[itemSlotIndex+1:]...)
		} else {
			inventory.Slots[itemSlotIndex].Quantity -= quantity
		}

		message = fmt.Sprintf("%s has BLASTED %s %d times!", username, targetUsername, quantity)

	default:
		return "", fmt.Errorf("item %s cannot be used", itemName)
	}

	// 6. Update Inventory
	if err := s.repo.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return "", fmt.Errorf("failed to update inventory: %w", err)
	}

	return message, nil
}

