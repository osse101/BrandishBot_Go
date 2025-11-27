package user

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/concurrency"
	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
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
	GetRecipeByTargetItemID(ctx context.Context, itemID int) (*domain.Recipe, error)
	IsRecipeUnlocked(ctx context.Context, userID string, recipeID int) (bool, error)
	UnlockRecipe(ctx context.Context, userID string, recipeID int) error
	GetUnlockedRecipesForUser(ctx context.Context, userID string) ([]crafting.UnlockedRecipeInfo, error)
	GetLastCooldown(ctx context.Context, userID, action string) (*time.Time, error)
	UpdateCooldown(ctx context.Context, userID, action string, timestamp time.Time) error
}

// Service defines the interface for user operations
type Service interface {
	RegisterUser(ctx context.Context, user domain.User) (domain.User, error)
	FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	HandleIncomingMessage(ctx context.Context, platform, platformID, username string) (domain.User, error)
	AddItem(ctx context.Context, username, platform, itemName string, quantity int) error
	RemoveItem(ctx context.Context, username, platform, itemName string, quantity int) (int, error)
	GiveItem(ctx context.Context, ownerUsername, receiverUsername, platform, itemName string, quantity int) error
	UseItem(ctx context.Context, username, platform, itemName string, quantity int, targetUsername string) (string, error)
	GetInventory(ctx context.Context, username string) ([]UserInventoryItem, error)
	TimeoutUser(ctx context.Context, username string, duration time.Duration, reason string) error
	LoadLootTables(path string) error
	HandleSearch(ctx context.Context, username, platform string) (string, error)
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
	lockManager  *concurrency.LockManager
	lootTables   map[string][]LootItem
}

// NewService creates a new user service
func NewService(repo Repository, lockManager *concurrency.LockManager) Service {
	s := &service{
		repo:         repo,
		itemHandlers: make(map[string]ItemEffectHandler),
		timeouts:     make(map[string]*time.Timer),
		lockManager:  lockManager,
		lootTables:   make(map[string][]LootItem),
	}
	s.registerHandlers()
	// Attempt to load default loot tables, ignore error if file doesn't exist (will be empty)
	// In a real app we might want to pass config path in NewService
	_ = s.LoadLootTables("configs/loot_tables.json")
	return s
}

func (s *service) LoadLootTables(path string) error {
	var tables map[string][]LootItem
	if err := utils.LoadJSON(path, &tables); err != nil {
		return err
	}
	s.lootTables = tables
	return nil
}

func (s *service) registerHandlers() {
    s.itemHandlers[domain.ItemLootbox1] = s.handleLootbox1
    s.itemHandlers[domain.ItemBlaster] = s.handleBlaster
    s.itemHandlers[domain.ItemLootbox0] = s.handleLootbox0
    s.itemHandlers[domain.ItemLootbox2] = s.handleLootbox2
}

func (s *service) getUserLock(userID string) *sync.Mutex {
    return s.lockManager.GetLock(userID)
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
    // If user not found, create new user
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		log.Error("Failed to get user", "error", err, "platform", platform, "platformID", platformID)
		return domain.User{}, fmt.Errorf("failed to get user: %w", err)
	}
    if user != nil {
        log.Info("Existing user found", "userID", user.ID)
        return *user, nil
    }
    // User not found, register new user
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

    lock := s.getUserLock(user.ID)
    lock.Lock()
    defer lock.Unlock()

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
        return 0, fmt.Errorf("%w: %s", domain.ErrUserNotFound, username)
    }

    lock := s.getUserLock(user.ID)
    lock.Lock()
    defer lock.Unlock()

    item, err := s.repo.GetItemByName(ctx, itemName)
    if err != nil {
        log.Error("Failed to get item", "error", err, "itemName", itemName)
        return 0, fmt.Errorf("failed to get item: %w", err)
    }
    if item == nil {
		return 0, fmt.Errorf("%w: %s", domain.ErrItemNotFound, itemName)
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

    // Acquire locks in consistent order to prevent deadlocks
    firstLock := s.getUserLock(owner.ID)
    secondLock := s.getUserLock(receiver.ID)
    
    if owner.ID > receiver.ID {
        firstLock, secondLock = secondLock, firstLock
    }
    
    firstLock.Lock()
    defer firstLock.Unlock()
    
    // If IDs are same (giving to self), we already have the lock
    if owner.ID != receiver.ID {
        secondLock.Lock()
        defer secondLock.Unlock()
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
		return fmt.Errorf("%w: has %d, needs %d", domain.ErrInsufficientQuantity, ownerInventory.Slots[ownerSlotIndex].Quantity, quantity)
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
        return "", fmt.Errorf("%w: %s", domain.ErrUserNotFound, username)
    }

    lock := s.getUserLock(user.ID)
    lock.Lock()
    defer lock.Unlock()

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
        return "", fmt.Errorf("%w: %s", domain.ErrItemNotFound, itemName)
    }
    itemSlotIndex := -1
    for i, slot := range inventory.Slots {
        if slot.ItemID == itemToUse.ID {
            itemSlotIndex = i
            break
        }
    }
    if itemSlotIndex == -1 || inventory.Slots[itemSlotIndex].Quantity < quantity {
		return "", fmt.Errorf("%w: %s", domain.ErrInsufficientQuantity, itemName)
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
		return nil, fmt.Errorf("%w: %s", domain.ErrUserNotFound, username)
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

// Helper methods

func (s *service) validateUser(ctx context.Context, username string) (*domain.User, error) {
    user, err := s.repo.GetUserByUsername(ctx, username)
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    if user == nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrUserNotFound, username)
	}
    return user, nil
}

func (s *service) validateItem(ctx context.Context, itemName string) (*domain.Item, error) {
    item, err := s.repo.GetItemByName(ctx, itemName)
    if err != nil {
        return nil, fmt.Errorf("failed to get item: %w", err)
    }
    if item == nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrItemNotFound, itemName)
	}
    return item, nil
}

// HandleSearch performs a search action for a user with cooldown tracking
func (s *service) HandleSearch(ctx context.Context, username, platform string) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("HandleSearch called", "username", username, "platform", platform)

	// Get or create user
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		log.Error("Failed to get user", "error", err, "username", username)
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	
	// If user doesn't exist, register them first
	if user == nil {
		log.Info("User not found, registering new user", "username", username)
		newUser := domain.User{Username: username}
		
		// Set platform ID based on platform parameter
		// Note: We don't have platformID here, so we'll just set username as ID
		// In production, this should come from the actual platform integration
		switch platform {
		case "twitch":
			newUser.TwitchID = username
		case "youtube":
			newUser.YoutubeID = username
		case "discord":
			newUser.DiscordID = username
		default:
			newUser.TwitchID = username // Default to twitch
		}
		
		registeredUser, err := s.RegisterUser(ctx, newUser)
		if err != nil {
			log.Error("Failed to register new user", "error", err, "username", username)
			return "", fmt.Errorf("failed to register user: %w", err)
		}
		user = &registeredUser
		log.Info("New user registered for search", "userID", user.ID, "username", username)
	}

	// Lock the user for the duration of this operation
	lock := s.getUserLock(user.ID)
	lock.Lock()
	defer lock.Unlock()

	// Check cooldown
	lastUsed, err := s.repo.GetLastCooldown(ctx, user.ID, domain.ActionSearch)
	if err != nil {
		log.Error("Failed to get cooldown", "error", err, "userID", user.ID)
		return "", fmt.Errorf("failed to check cooldown: %w", err)
	}

	now := time.Now()
	if lastUsed != nil {
		elapsed := now.Sub(*lastUsed)
		if elapsed < domain.SearchCooldownDuration {
			remaining := domain.SearchCooldownDuration - elapsed
			minutes := int(remaining.Minutes())
			seconds := int(remaining.Seconds()) % 60
			
			log.Info("Search on cooldown", "username", username, "remaining", remaining)
			return fmt.Sprintf("You can search again in %dm %ds", minutes, seconds), nil
		}
	}

	// Perform search - 80% chance of lootbox0
	var resultMessage string
	if utils.RandomFloat() <= 0.8 {
		// Give lootbox0
		item, err := s.repo.GetItemByName(ctx, domain.ItemLootbox0)
		if err != nil {
			log.Error("Failed to get lootbox0 item", "error", err)
			return "", fmt.Errorf("failed to get reward item: %w", err)
		}
		if item == nil {
			log.Error("Lootbox0 item not found in database")
			return "", fmt.Errorf("reward item not configured")
		}

		// Add to inventory
		inventory, err := s.repo.GetInventory(ctx, user.ID)
		if err != nil {
			log.Error("Failed to get inventory", "error", err, "userID", user.ID)
			return "", fmt.Errorf("failed to get inventory: %w", err)
		}

		found := false
		for i, slot := range inventory.Slots {
			if slot.ItemID == item.ID {
				inventory.Slots[i].Quantity++
				found = true
				break
			}
		}
		if !found {
			inventory.Slots = append(inventory.Slots, domain.InventorySlot{ItemID: item.ID, Quantity: 1})
		}

		if err := s.repo.UpdateInventory(ctx, user.ID, *inventory); err != nil {
			log.Error("Failed to update inventory", "error", err, "userID", user.ID)
			return "", fmt.Errorf("failed to update inventory: %w", err)
		}

		resultMessage = fmt.Sprintf("You have found 1x %s", item.Name)
		log.Info("Search successful - lootbox found", "username", username, "item", item.Name)
	} else {
		resultMessage = "You have found nothing"
		log.Info("Search successful - nothing found", "username", username)
	}

	// Update cooldown
	if err := s.repo.UpdateCooldown(ctx, user.ID, domain.ActionSearch, now); err != nil {
		log.Error("Failed to update cooldown", "error", err, "userID", user.ID)
		// Don't fail the search, just log the error
	}

	log.Info("Search completed", "username", username, "result", resultMessage)
	return resultMessage, nil
}
