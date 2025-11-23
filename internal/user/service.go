package user

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Repository defines the interface for user persistence
type Repository interface {
    UpsertUser(ctx context.Context, user *domain.User) error
    GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
    GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
    UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
    GetItemByName(ctx context.Context, itemName string) (*domain.Item, error)
    GetItemByID(ctx context.Context, id int) (*domain.Item, error)
    GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
    GetSellablePrices(ctx context.Context) ([]domain.Item, error)
    IsItemBuyable(ctx context.Context, itemName string) (bool, error)
    BeginTx(ctx context.Context) (repository.Tx, error)
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
    SellItem(ctx context.Context, username, platform, itemName string, quantity int) (int, int, error)
    BuyItem(ctx context.Context, username, platform, itemName string, quantity int) (int, error)
    UseItem(ctx context.Context, username, platform, itemName string, quantity int, targetUsername string) (string, error)
    GetInventory(ctx context.Context, username string) ([]UserInventoryItem, error)
    TimeoutUser(ctx context.Context, username string, duration time.Duration, reason string) error
}

type UserInventoryItem struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Quantity    int    `json:"quantity"`
    Value       int    `json:"value"`
}

// ItemEffectHandler defines the function signature for item effects
type ItemEffectHandler func(ctx context.Context, s *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error)

// service implements the Service interface
type service struct {
    repo         Repository
    itemHandlers map[string]ItemEffectHandler
    timeoutMu    sync.Mutex
    timeouts     map[string]*time.Timer
}

// NewService creates a new user service
func NewService(repo Repository) Service {
    s := &service{
        repo:         repo,
        itemHandlers: make(map[string]ItemEffectHandler),
        timeouts:     make(map[string]*time.Timer),
    }
    s.registerHandlers()
    return s
}

func (s *service) registerHandlers() {
    s.itemHandlers[domain.ItemLootbox1] = s.handleLootbox1
    s.itemHandlers[domain.ItemBlaster] = s.handleBlaster
    s.itemHandlers[domain.ItemLootbox0] = s.handleLootbox0
    s.itemHandlers[domain.ItemLootbox2] = s.handleLootbox2
}

// RegisterUser registers a new user
func (s *service) RegisterUser(ctx context.Context, user domain.User) (domain.User, error) {
    log := logger.FromContext(ctx)
    log.Info("RegisterUser called", "username", user.Username)
    if err := s.repo.UpsertUser(ctx, &user); err != nil {
        log.Error("Failed to upsert user", "error", err, "username", user.Username)
        return domain.User{}, err
    }
    log.Info("User registered", "user_id", user.ID, "username", user.Username)
    return user, nil
}

// FindUserByPlatformID finds a user by their platform-specific ID
func (s *service) FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
    log := logger.FromContext(ctx)
    log.Info("FindUserByPlatformID called", "platform", platform, "platformID", platformID)
    user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
    if err != nil {
        log.Error("Failed to find user by platform ID", "error", err, "platform", platform, "platformID", platformID)
        return nil, err
    }
    if user != nil {
        log.Info("User found", "userID", user.ID, "username", user.Username)
    }
    return user, nil
}

// HandleIncomingMessage checks if a user exists for an incoming message and creates one if not.
func (s *service) HandleIncomingMessage(ctx context.Context, platform, platformID, username string) (domain.User, error) {
    log := logger.FromContext(ctx)
    log.Info("HandleIncomingMessage called", "platform", platform, "platformID", platformID, "username", username)
    user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
    if err != nil && err.Error() != "user not found" {
        log.Error("Failed to get user", "error", err, "platform", platform, "platformID", platformID)
        return domain.User{}, fmt.Errorf("failed to get user: %w", err)
    }
    if user != nil {
        log.Info("Existing user found", "userID", user.ID)
        return *user, nil
    }
    // TODO: Check if error is actually "not found"
    newUser := domain.User{Username: username}
    switch platform {
    case "twitch":
        newUser.TwitchID = platformID
    case "youtube":
        newUser.YoutubeID = platformID
    case "discord":
        newUser.DiscordID = platformID
    default:
        log.Error("Unsupported platform", "platform", platform)
        return domain.User{}, fmt.Errorf("unsupported platform: %s", platform)
    }
    if _, err := s.RegisterUser(ctx, newUser); err != nil {
        log.Error("Failed to register new user", "error", err, "username", username)
        return domain.User{}, err
    }
    log.Info("New user registered", "username", username)
    return newUser, nil
}

func (s *service) AddItem(ctx context.Context, username, platform, itemName string, quantity int) error {
    log := logger.FromContext(ctx)
    log.Info("AddItem called", "username", username, "item", itemName, "quantity", quantity)
    user, err := s.repo.GetUserByUsername(ctx, username)
    if err != nil {
        log.Error("Failed to get user", "error", err, "username", username)
        return fmt.Errorf("failed to get user: %w", err)
    }
    if user == nil {
        log.Warn("User not found", "username", username)
        return fmt.Errorf("user not found: %s", username)
    }
    item, err := s.repo.GetItemByName(ctx, itemName)
    if err != nil {
        log.Error("Failed to get item", "error", err, "itemName", itemName)
        return fmt.Errorf("failed to get item: %w", err)
    }
    if item == nil {
        log.Warn("Item not found", "itemName", itemName)
        return fmt.Errorf("item not found: %s", itemName)
    }
    inventory, err := s.repo.GetInventory(ctx, user.ID)
    if err != nil {
        log.Error("Failed to get inventory", "error", err, "userID", user.ID)
        return fmt.Errorf("failed to get inventory: %w", err)
    }
    found := false
    for i, slot := range inventory.Slots {
        if slot.ItemID == item.ID {
            inventory.Slots[i].Quantity += quantity
            found = true
            break
        }
    }
    if !found {
        inventory.Slots = append(inventory.Slots, domain.InventorySlot{ItemID: item.ID, Quantity: quantity})
    }
    if err := s.repo.UpdateInventory(ctx, user.ID, *inventory); err != nil {
        log.Error("Failed to update inventory", "error", err, "userID", user.ID)
        return fmt.Errorf("failed to update inventory: %w", err)
    }
    log.Info("Item added successfully", "username", username, "item", itemName, "quantity", quantity)
    return nil
}

func (s *service) RemoveItem(ctx context.Context, username, platform, itemName string, quantity int) (int, error) {
    log := logger.FromContext(ctx)
    log.Info("RemoveItem called", "username", username, "item", itemName, "quantity", quantity)
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
    inventory, err := s.repo.GetInventory(ctx, user.ID)
    if err != nil {
        log.Error("Failed to get inventory", "error", err, "userID", user.ID)
        return 0, fmt.Errorf("failed to get inventory: %w", err)
    }
    removed := 0
    found := false
    for i, slot := range inventory.Slots {
        if slot.ItemID == item.ID {
            found = true
            removed = quantity
            if slot.Quantity < quantity {
                removed = slot.Quantity
            }
            if slot.Quantity <= quantity {
                inventory.Slots = append(inventory.Slots[:i], inventory.Slots[i+1:]...)
            } else {
                inventory.Slots[i].Quantity -= removed
            }
            break
        }
    }
    if !found {
        log.Warn("Item not in inventory", "itemName", itemName)
        return 0, fmt.Errorf("item %s not in inventory", itemName)
    }
    if err := s.repo.UpdateInventory(ctx, user.ID, *inventory); err != nil {
        log.Error("Failed to update inventory", "error", err, "userID", user.ID)
        return 0, fmt.Errorf("failed to update inventory: %w", err)
    }
    log.Info("Item removed", "username", username, "item", itemName, "removed", removed)
    return removed, nil
}

func (s *service) GiveItem(ctx context.Context, ownerUsername, receiverUsername, platform, itemName string, quantity int) error {
    log := logger.FromContext(ctx)
    log.Info("GiveItem called", "owner", ownerUsername, "receiver", receiverUsername, "item", itemName, "quantity", quantity)

    owner, err := s.validateUser(ctx, ownerUsername)
    if err != nil {
        return err
    }

    receiver, err := s.validateUser(ctx, receiverUsername)
    if err != nil {
        return err
    }

    item, err := s.validateItem(ctx, itemName)
    if err != nil {
        return err
    }

    return s.executeGiveItemTx(ctx, owner, receiver, item, quantity)
}

func (s *service) executeGiveItemTx(ctx context.Context, owner, receiver *domain.User, item *domain.Item, quantity int) error {
    log := logger.FromContext(ctx)
    tx, err := s.repo.BeginTx(ctx)
    if err != nil {
        log.Error("Failed to begin transaction", "error", err)
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback(ctx)

    ownerInventory, err := tx.GetInventory(ctx, owner.ID)
    if err != nil {
        return fmt.Errorf("failed to get owner inventory: %w", err)
    }

    ownerSlotIndex := -1
    for i, slot := range ownerInventory.Slots {
        if slot.ItemID == item.ID {
            ownerSlotIndex = i
            break
        }
    }

    if ownerSlotIndex == -1 {
        return fmt.Errorf("owner does not have item %s in inventory", item.Name)
    }

    if ownerInventory.Slots[ownerSlotIndex].Quantity < quantity {
        return fmt.Errorf("owner does not have enough %s (has %d, needs %d)", item.Name, ownerInventory.Slots[ownerSlotIndex].Quantity, quantity)
    }

    receiverInventory, err := tx.GetInventory(ctx, receiver.ID)
    if err != nil {
        return fmt.Errorf("failed to get receiver inventory: %w", err)
    }

    // Remove from owner
    if ownerInventory.Slots[ownerSlotIndex].Quantity == quantity {
        ownerInventory.Slots = append(ownerInventory.Slots[:ownerSlotIndex], ownerInventory.Slots[ownerSlotIndex+1:]...)
    } else {
        ownerInventory.Slots[ownerSlotIndex].Quantity -= quantity
    }

    // Add to receiver
    found := false
    for i, slot := range receiverInventory.Slots {
        if slot.ItemID == item.ID {
            receiverInventory.Slots[i].Quantity += quantity
            found = true
            break
        }
    }
    if !found {
        receiverInventory.Slots = append(receiverInventory.Slots, domain.InventorySlot{ItemID: item.ID, Quantity: quantity})
    }

    if err := tx.UpdateInventory(ctx, owner.ID, *ownerInventory); err != nil {
        return fmt.Errorf("failed to update owner inventory: %w", err)
    }
    if err := tx.UpdateInventory(ctx, receiver.ID, *receiverInventory); err != nil {
        return fmt.Errorf("failed to update receiver inventory: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    log.Info("Item transferred", "owner", owner.Username, "receiver", receiver.Username, "item", item.Name, "quantity", quantity)
    return nil
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
    itemSlotIndex := -1
    actualSellQuantity := 0
    for i, slot := range inventory.Slots {
        if slot.ItemID == item.ID {
            itemSlotIndex = i
            if slot.Quantity < quantity {
                actualSellQuantity = slot.Quantity
            } else {
                actualSellQuantity = quantity
            }
            break
        }
    }
    if itemSlotIndex == -1 {
        log.Warn("Item not in inventory", "itemName", itemName)
        return 0, 0, fmt.Errorf("item %s not in inventory", itemName)
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

    user, err := s.validateUser(ctx, username)
    if err != nil {
        return 0, err
    }

    item, err := s.validateItem(ctx, itemName)
    if err != nil {
        return 0, err
    }

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

    moneyBalance := 0
    moneySlotIndex := -1
    for i, slot := range inventory.Slots {
        if slot.ItemID == moneyItem.ID {
            moneyBalance = slot.Quantity
            moneySlotIndex = i
            break
        }
    }

    if moneyBalance <= 0 {
        log.Warn("Insufficient funds", "username", username)
        return 0, fmt.Errorf("insufficient funds")
    }

    maxAffordable := moneyBalance / item.BaseValue
    if maxAffordable == 0 {
        log.Warn("Insufficient funds for any quantity", "username", username, "item", itemName)
        return 0, fmt.Errorf("insufficient funds to buy even one %s (cost: %d, balance: %d)", itemName, item.BaseValue, moneyBalance)
    }

    actualQuantity := quantity
    if actualQuantity > maxAffordable {
        actualQuantity = maxAffordable
        log.Info("Adjusted purchase quantity due to funds", "requested", quantity, "actual", actualQuantity)
    }

    cost := actualQuantity * item.BaseValue
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

func (s *service) UseItem(ctx context.Context, username, platform, itemName string, quantity int, targetUsername string) (string, error) {
    log := logger.FromContext(ctx)
    log.Info("UseItem called", "username", username, "item", itemName, "quantity", quantity, "target", targetUsername)
    user, err := s.repo.GetUserByUsername(ctx, username)
    if err != nil {
        log.Error("Failed to get user", "error", err, "username", username)
        return "", fmt.Errorf("failed to get user: %w", err)
    }
    if user == nil {
        log.Warn("User not found", "username", username)
        return "", fmt.Errorf("user not found: %s", username)
    }
    inventory, err := s.repo.GetInventory(ctx, user.ID)
    if err != nil {
        log.Error("Failed to get inventory", "error", err, "userID", user.ID)
        return "", fmt.Errorf("failed to get inventory: %w", err)
    }
    itemToUse, err := s.repo.GetItemByName(ctx, itemName)
    if err != nil {
        log.Error("Failed to get item", "error", err, "itemName", itemName)
        return "", fmt.Errorf("failed to get item: %w", err)
    }
    if itemToUse == nil {
        log.Warn("Item not found", "itemName", itemName)
        return "", fmt.Errorf("item not found: %s", itemName)
    }
    itemSlotIndex := -1
    for i, slot := range inventory.Slots {
        if slot.ItemID == itemToUse.ID {
            itemSlotIndex = i
            break
        }
    }
    if itemSlotIndex == -1 || inventory.Slots[itemSlotIndex].Quantity < quantity {
        log.Warn("Insufficient quantity", "itemName", itemName, "available", inventory.Slots[itemSlotIndex].Quantity, "required", quantity)
        return "", fmt.Errorf("insufficient quantity of %s", itemName)
    }
    handler, exists := s.itemHandlers[itemName]
    if !exists {
        log.Warn("No handler for item", "itemName", itemName)
        return "", fmt.Errorf("item %s has no effect", itemName)
    }
    args := map[string]interface{}{ "targetUsername": targetUsername, "username": username }
    message, err := handler(ctx, s, user, inventory, itemToUse, quantity, args)
    if err != nil {
        log.Error("Handler error", "error", err, "itemName", itemName)
        return "", err
    }
    if err := s.repo.UpdateInventory(ctx, user.ID, *inventory); err != nil {
        log.Error("Failed to update inventory after use", "error", err, "userID", user.ID)
        return "", fmt.Errorf("failed to update inventory: %w", err)
    }
    log.Info("Item used", "username", username, "item", itemName, "quantity", quantity, "message", message)
    return message, nil
}

func (s *service) GetInventory(ctx context.Context, username string) ([]UserInventoryItem, error) {
    log := logger.FromContext(ctx)
    log.Info("GetInventory called", "username", username)
    user, err := s.repo.GetUserByUsername(ctx, username)
    if err != nil {
        log.Error("Failed to get user", "error", err, "username", username)
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    if user == nil {
        log.Warn("User not found", "username", username)
        return nil, fmt.Errorf("user not found: %s", username)
    }
    inventory, err := s.repo.GetInventory(ctx, user.ID)
    if err != nil {
        log.Error("Failed to get inventory", "error", err, "userID", user.ID)
        return nil, fmt.Errorf("failed to get inventory: %w", err)
    }
    var items []UserInventoryItem
    for _, slot := range inventory.Slots {
        item, err := s.repo.GetItemByID(ctx, slot.ItemID)
        if err != nil {
            log.Error("Failed to get item details", "error", err, "itemID", slot.ItemID)
            return nil, fmt.Errorf("failed to get item details for id %d: %w", slot.ItemID, err)
        }
        if item == nil {
            log.Warn("Item missing for slot", "itemID", slot.ItemID)
            continue
        }
        items = append(items, UserInventoryItem{Name: item.Name, Description: item.Description, Quantity: slot.Quantity, Value: item.BaseValue})
    }
    log.Info("Inventory retrieved", "username", username, "itemCount", len(items))
    return items, nil
}

// TimeoutUser times out a user for a specified duration.
// Note: Timeouts are currently in-memory and will be lost on server restart. This is a known design choice.
func (s *service) TimeoutUser(ctx context.Context, username string, duration time.Duration, reason string) error {
    log := logger.FromContext(ctx)
    log.Info("TimeoutUser called", "username", username, "duration", duration, "reason", reason)

    s.timeoutMu.Lock()
    defer s.timeoutMu.Unlock()

    // If user is already timed out, stop the existing timer
    if timer, exists := s.timeouts[username]; exists {
        timer.Stop()
        log.Info("Existing timeout cancelled", "username", username)
    }

    // Create a new timer
    timer := time.AfterFunc(duration, func() {
        s.timeoutMu.Lock()
        delete(s.timeouts, username)
        s.timeoutMu.Unlock()
        // In a real app, we might send a message here
        fmt.Printf("User %s timeout expired\n", username)
    })

    s.timeouts[username] = timer
    log.Info("User timed out", "username", username, "duration", duration)
    return nil
}

// Item effect handlers
func (s *service) handleLootbox1(ctx context.Context, _ *service, _ *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, _ map[string]interface{}) (string, error) {
    log := logger.FromContext(ctx)
    log.Info("handleLootbox1 called", "quantity", quantity)
    lootbox0, err := s.repo.GetItemByName(ctx, domain.ItemLootbox0)
    if err != nil {
        log.Error("Failed to get lootbox0", "error", err)
        return "", fmt.Errorf("failed to get lootbox0: %w", err)
    }
    if lootbox0 == nil {
        log.Warn("lootbox0 not found")
        return "", fmt.Errorf("lootbox0 not found")
    }
    // Find lootbox1 slot
    itemSlotIndex := -1
    for i, slot := range inventory.Slots {
        if slot.ItemID == item.ID {
            itemSlotIndex = i
            break
        }
    }
    if itemSlotIndex == -1 {
        log.Warn("lootbox1 not in inventory")
        return "", fmt.Errorf("item not found in inventory")
    }
    if inventory.Slots[itemSlotIndex].Quantity == quantity {
        inventory.Slots = append(inventory.Slots[:itemSlotIndex], inventory.Slots[itemSlotIndex+1:]...)
    } else {
        inventory.Slots[itemSlotIndex].Quantity -= quantity
    }
    // Grant lootbox0
    found := false
    for i, slot := range inventory.Slots {
        if slot.ItemID == lootbox0.ID {
            inventory.Slots[i].Quantity += quantity
            found = true
            break
        }
    }
    if !found {
        inventory.Slots = append(inventory.Slots, domain.InventorySlot{ItemID: lootbox0.ID, Quantity: quantity})
    }
    log.Info("lootbox1 consumed, lootbox0 granted", "quantity", quantity)
    return fmt.Sprintf("Used %d lootbox1", quantity), nil
}

func (s *service) handleBlaster(ctx context.Context, _ *service, _ *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
    log := logger.FromContext(ctx)
    log.Info("handleBlaster called", "quantity", quantity)
    targetUsername, ok := args["targetUsername"].(string)
    if !ok || targetUsername == "" {
        log.Warn("target username missing for blaster")
        return "", fmt.Errorf("target username is required for blaster")
    }
    username, _ := args["username"].(string)
    // Find blaster slot
    itemSlotIndex := -1
    for i, slot := range inventory.Slots {
        if slot.ItemID == item.ID {
            itemSlotIndex = i
            break
        }
    }
    if itemSlotIndex == -1 {
        log.Warn("blaster not in inventory")
        return "", fmt.Errorf("item not found in inventory")
    }
    if inventory.Slots[itemSlotIndex].Quantity == quantity {
        inventory.Slots = append(inventory.Slots[:itemSlotIndex], inventory.Slots[itemSlotIndex+1:]...)
    } else {
        inventory.Slots[itemSlotIndex].Quantity -= quantity
    }

    // Apply timeout
    timeoutDuration := 60 * time.Second
    if err := s.TimeoutUser(ctx, targetUsername, timeoutDuration, "Blasted by " + username); err != nil {
        log.Error("Failed to timeout user", "error", err, "target", targetUsername)
        // Continue anyway, as the item was used
    }

    log.Info("blaster used", "target", targetUsername, "quantity", quantity)
    return fmt.Sprintf("%s has BLASTED %s %d times! They are timed out for %v.", username, targetUsername, quantity, timeoutDuration), nil
}

func (s *service) handleLootbox0(ctx context.Context, _ *service, _ *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, _ map[string]interface{}) (string, error) {
    log := logger.FromContext(ctx)
    log.Info("handleLootbox0 called", "quantity", quantity)
    // Effect: Consume lootbox0, return empty message
    itemSlotIndex := -1
    for i, slot := range inventory.Slots {
        if slot.ItemID == item.ID {
            itemSlotIndex = i
            break
        }
    }
    if itemSlotIndex == -1 {
        log.Warn("lootbox0 not in inventory")
        return "", fmt.Errorf("item not found in inventory")
    }
    if inventory.Slots[itemSlotIndex].Quantity == quantity {
        inventory.Slots = append(inventory.Slots[:itemSlotIndex], inventory.Slots[itemSlotIndex+1:]...)
    } else {
        inventory.Slots[itemSlotIndex].Quantity -= quantity
    }
    log.Info("lootbox0 consumed", "quantity", quantity)
    return "The lootbox was empty!", nil
}

func (s *service) handleLootbox2(ctx context.Context, _ *service, _ *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, _ map[string]interface{}) (string, error) {
    log := logger.FromContext(ctx)
    log.Info("handleLootbox2 called", "quantity", quantity)
    lootbox1, err := s.repo.GetItemByName(ctx, domain.ItemLootbox1)
    if err != nil {
        log.Error("Failed to get lootbox1", "error", err)
        return "", fmt.Errorf("failed to get lootbox1: %w", err)
    }
    if lootbox1 == nil {
        log.Warn("lootbox1 not found")
        return "", fmt.Errorf("lootbox1 not found")
    }
    itemSlotIndex := -1
    for i, slot := range inventory.Slots {
        if slot.ItemID == item.ID {
            itemSlotIndex = i
            break
        }
    }
    if itemSlotIndex == -1 {
        log.Warn("lootbox2 not in inventory")
        return "", fmt.Errorf("item not found in inventory")
    }
    if inventory.Slots[itemSlotIndex].Quantity == quantity {
        inventory.Slots = append(inventory.Slots[:itemSlotIndex], inventory.Slots[itemSlotIndex+1:]...)
    } else {
        inventory.Slots[itemSlotIndex].Quantity -= quantity
    }
    // Grant lootbox1
    found := false
    for i, slot := range inventory.Slots {
        if slot.ItemID == lootbox1.ID {
            inventory.Slots[i].Quantity += quantity
            found = true
            break
        }
    }
    if !found {
        inventory.Slots = append(inventory.Slots, domain.InventorySlot{ItemID: lootbox1.ID, Quantity: quantity})
    }
    log.Info("lootbox2 consumed, lootbox1 granted", "quantity", quantity)
    return fmt.Sprintf("Used %d lootbox2", quantity), nil
}

// Helper methods

func (s *service) validateUser(ctx context.Context, username string) (*domain.User, error) {
    user, err := s.repo.GetUserByUsername(ctx, username)
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    if user == nil {
        return nil, fmt.Errorf("user not found: %s", username)
    }
    return user, nil
}

func (s *service) validateItem(ctx context.Context, itemName string) (*domain.Item, error) {
    item, err := s.repo.GetItemByName(ctx, itemName)
    if err != nil {
        return nil, fmt.Errorf("failed to get item: %w", err)
    }
    if item == nil {
        return nil, fmt.Errorf("item not found: %s", itemName)
    }
    return item, nil
}
