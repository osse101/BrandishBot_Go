package user

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/stats"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// validPlatforms defines the supported platform values
var validPlatforms = map[string]bool{
	domain.PlatformTwitch:  true,
	domain.PlatformYoutube: true,
	domain.PlatformDiscord: true,
}

// Repository defines the interface for user persistence
type Repository interface {
	UpsertUser(ctx context.Context, user *domain.User) error
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
	UpdateUser(ctx context.Context, user domain.User) error
	DeleteUser(ctx context.Context, userID string) error
	GetInventory(ctx context.Context, userID string) (*domain.Inventory, error)
	UpdateInventory(ctx context.Context, userID string, inventory domain.Inventory) error
	DeleteInventory(ctx context.Context, userID string) error
	GetItemByName(ctx context.Context, itemName string) (*domain.Item, error)
	GetItemByID(ctx context.Context, id int) (*domain.Item, error)
	GetItemsByIDs(ctx context.Context, itemIDs []int) ([]domain.Item, error)

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
	HandleIncomingMessage(ctx context.Context, platform, platformID, username, message string) (*domain.MessageResult, error)
	AddItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) error
	RemoveItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (int, error)
	GiveItem(ctx context.Context, ownerPlatform, ownerPlatformID, ownerUsername, receiverPlatform, receiverPlatformID, receiverUsername, itemName string, quantity int) error
	UseItem(ctx context.Context, platform, platformID, username, itemName string, quantity int, targetUsername string) (string, error)
	GetInventory(ctx context.Context, platform, platformID, username string) ([]UserInventoryItem, error)
	TimeoutUser(ctx context.Context, username string, duration time.Duration, reason string) error
	LoadLootTables(path string) error
	HandleSearch(ctx context.Context, platform, platformID, username string) (string, error)
	// Account linking methods
	MergeUsers(ctx context.Context, primaryUserID, secondaryUserID string) error
	UnlinkPlatform(ctx context.Context, userID, platform string) error
	GetLinkedPlatforms(ctx context.Context, platform, platformID string) ([]string, error)
	Shutdown(ctx context.Context) error
}

type UserInventoryItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	Value       int    `json:"value"`
}

// ItemEffectHandler defines the function signature for item effects
type ItemEffectHandler func(ctx context.Context, s *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error)

// JobService defines the interface for job operations
type JobService interface {
	AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error)
}

// service implements the Service interface
type service struct {
	repo           Repository
	itemHandlers   map[string]ItemEffectHandler
	timeoutMu      sync.Mutex
	timeouts       map[string]*time.Timer
	lootTables     map[string][]LootItem
	jobService     JobService
	statsService   stats.Service
	stringFinder   *StringFinder
	namingResolver naming.Resolver
	devMode        bool // When true, bypasses cooldowns
	wg             sync.WaitGroup
}

// setPlatformID sets the appropriate platform-specific ID field on a user
func setPlatformID(user *domain.User, platform, platformID string) {
	switch platform {
	case domain.PlatformTwitch:
		user.TwitchID = platformID
	case domain.PlatformYoutube:
		user.YoutubeID = platformID
	case domain.PlatformDiscord:
		user.DiscordID = platformID
	}
}

// NewService creates a new user service
func NewService(repo Repository, statsService stats.Service, jobService JobService, namingResolver naming.Resolver, devMode bool) Service {
	s := &service{
		repo:           repo,
		itemHandlers:   make(map[string]ItemEffectHandler),
		timeouts:       make(map[string]*time.Timer),
		lootTables:     make(map[string][]LootItem),
		jobService:     jobService,
		statsService:   statsService,
		stringFinder:   NewStringFinder(),
		namingResolver: namingResolver,
		devMode:        devMode,
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

// HandleIncomingMessage checks if a user exists for an incoming message, creates one if not, and finds string matches.
func (s *service) HandleIncomingMessage(ctx context.Context, platform, platformID, username, message string) (*domain.MessageResult, error) {
	log := logger.FromContext(ctx)
	log.Info("HandleIncomingMessage called", "platform", platform, "platformID", platformID, "username", username)
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	// If user not found, create new user
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		log.Error("Failed to get user", "error", err, "platform", platform, "platformID", platformID)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	var targetUser *domain.User
	if user != nil {
		log.Info("Existing user found", "userID", user.ID)
		targetUser = user
	} else {
		// User not found, register new user
		newUser := domain.User{Username: username}
		switch platform {
		case domain.PlatformTwitch:
			newUser.TwitchID = platformID
		case domain.PlatformYoutube:
			newUser.YoutubeID = platformID
		case domain.PlatformDiscord:
			newUser.DiscordID = platformID
		default:
			log.Error("Unsupported platform", "platform", platform)
			return nil, fmt.Errorf("unsupported platform: %s", platform)
		}
		registered, err := s.RegisterUser(ctx, newUser)
		if err != nil {
			log.Error("Failed to register new user", "error", err, "username", username)
			return nil, err
		}
		log.Info("New user registered", "username", username)
		targetUser = &registered
	}

	// Find matches in message
	matches := s.stringFinder.FindMatches(message)

	result := &domain.MessageResult{
		User:    *targetUser,
		Matches: matches,
	}

	return result, nil
}

func (s *service) AddItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) error {
	log := logger.FromContext(ctx)
	log.Info("AddItem called", "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", quantity)

	user, err := s.getUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		return err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		log.Error("Failed to get item", "error", err, "itemName", itemName)
		return fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		log.Warn("Item not found", "itemName", itemName)
		return fmt.Errorf("item not found: %s", itemName)
	}
	inventory, err := tx.GetInventory(ctx, user.ID)
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
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		log.Error("Failed to update inventory", "error", err, "userID", user.ID)
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("Failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info("Item added successfully", "username", username, "item", itemName, "quantity", quantity)
	return nil
}

func (s *service) RemoveItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (int, error) {
	log := logger.FromContext(ctx)
	log.Info("RemoveItem called", "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", quantity)

	user, err := s.getUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		return 0, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "error", err)
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		log.Error("Failed to get item", "error", err, "itemName", itemName)
		return 0, fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		return 0, fmt.Errorf("%w: %s", domain.ErrItemNotFound, itemName)
	}
	inventory, err := tx.GetInventory(ctx, user.ID)
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
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		log.Error("Failed to update inventory", "error", err, "userID", user.ID)
		return 0, fmt.Errorf("failed to update inventory: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("Failed to commit transaction", "error", err)
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info("Item removed", "username", username, "item", itemName, "removed", removed)
	return removed, nil
}

func (s *service) GiveItem(ctx context.Context, ownerPlatform, ownerPlatformID, ownerUsername, receiverPlatform, receiverPlatformID, receiverUsername, itemName string, quantity int) error {
	log := logger.FromContext(ctx)
	log.Info("GiveItem called",
		"ownerPlatform", ownerPlatform, "ownerPlatformID", ownerPlatformID, "ownerUsername", ownerUsername,
		"receiverPlatform", receiverPlatform, "receiverPlatformID", receiverPlatformID, "receiverUsername", receiverUsername,
		"item", itemName, "quantity", quantity)

	owner, err := s.getUserOrRegister(ctx, ownerPlatform, ownerPlatformID, ownerUsername)
	if err != nil {
		return err
	}

	receiver, err := s.getUserOrRegister(ctx, receiverPlatform, receiverPlatformID, receiverUsername)
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
	defer repository.SafeRollback(ctx, tx)

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
		return fmt.Errorf("owner does not have item %s in inventory", item.InternalName)
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

	log.Info("Item transferred", "owner", owner.Username, "receiver", receiver.Username, "item", item.InternalName, "quantity", quantity)
	return nil
}

func (s *service) UseItem(ctx context.Context, platform, platformID, username, itemName string, quantity int, targetUsername string) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("UseItem called", "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", quantity, "target", targetUsername)

	user, err := s.getUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		return "", err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "error", err)
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	inventory, err := tx.GetInventory(ctx, user.ID)
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
	args := map[string]interface{}{"targetUsername": targetUsername, "username": username}
	message, err := handler(ctx, s, user, inventory, itemToUse, quantity, args)
	if err != nil {
		log.Error("Handler error", "error", err, "itemName", itemName)
		return "", err
	}
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		log.Error("Failed to update inventory after use", "error", err, "userID", user.ID)
		return "", fmt.Errorf("failed to update inventory: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("Failed to commit transaction", "error", err)
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info("Item used", "username", username, "item", itemName, "quantity", quantity, "message", message)
	return message, nil
}

func (s *service) GetInventory(ctx context.Context, platform, platformID, username string) ([]UserInventoryItem, error) {
	log := logger.FromContext(ctx)
	log.Info("GetInventory called", "platform", platform, "platformID", platformID, "username", username)

	user, err := s.getUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		return nil, err
	}
	inventory, err := s.repo.GetInventory(ctx, user.ID)
	if err != nil {
		log.Error("Failed to get inventory", "error", err, "userID", user.ID)
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}
	// Optimization: Batch fetch all item details
	itemIDs := make([]int, 0, len(inventory.Slots))
	for _, slot := range inventory.Slots {
		itemIDs = append(itemIDs, slot.ItemID)
	}

	itemList, err := s.repo.GetItemsByIDs(ctx, itemIDs)
	if err != nil {
		log.Error("Failed to get item details", "error", err)
		return nil, fmt.Errorf("failed to get item details: %w", err)
	}

	itemMap := make(map[int]domain.Item)
	for _, item := range itemList {
		itemMap[item.ID] = item
	}

	var items []UserInventoryItem
	for _, slot := range inventory.Slots {
		item, ok := itemMap[slot.ItemID]
		if !ok {
			log.Warn("Item missing for slot", "itemID", slot.ItemID)
			continue
		}
		items = append(items, UserInventoryItem{Name: item.InternalName, Description: item.Description, Quantity: slot.Quantity, Value: item.BaseValue})
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

// Constants for search mechanic
const (
	SearchSuccessRate  = 0.8
	SearchCriticalRate = 0.05
	SearchNearMissRate = 0.05
)

// HandleSearch performs a search action for a user with cooldown tracking
func (s *service) HandleSearch(ctx context.Context, platform, platformID, username string) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("HandleSearch called", "platform", platform, "platformID", platformID, "username", username)

	// Validate platform
	if username == "" {
		return "", fmt.Errorf("username cannot be empty")
	}

	if platform == "" {
		// Default to twitch for backwards compatibility
		platform = domain.PlatformTwitch
		log.Info("Platform not specified, defaulting to twitch", "username", username)
	} else if !validPlatforms[platform] {
		return "", fmt.Errorf("invalid platform '%s': must be one of: %s, %s, %s", platform, domain.PlatformTwitch, domain.PlatformYoutube, domain.PlatformDiscord)
	}

	// Get or create user
	user, err := s.getUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		return "", err
	}

	// Check cooldown
	lastUsed, err := s.repo.GetLastCooldown(ctx, user.ID, domain.ActionSearch)
	if err != nil {
		log.Error("Failed to get cooldown", "error", err, "userID", user.ID)
		return "", fmt.Errorf("failed to check cooldown: %w", err)
	}

	now := time.Now()
	if lastUsed != nil {
		// Check if dev mode bypasses cooldowns
		if !s.devMode {
			elapsed := now.Sub(*lastUsed)
			if elapsed < domain.SearchCooldownDuration {
				remaining := domain.SearchCooldownDuration - elapsed
				minutes := int(remaining.Minutes())
				seconds := int(remaining.Seconds()) % 60

				log.Info("Search on cooldown", "username", username, "remaining", remaining)
				return fmt.Sprintf("You can search again in %dm %ds", minutes, seconds), nil
			}
		} else {
			log.Info("DEV_MODE: Bypassing cooldown check", "username", username)
		}
	}

	// Perform search
	var resultMessage string
	roll := utils.RandomFloat()

	// Check for First Search of the Day
	isFirstSearchDaily := false
	if lastUsed == nil {
		isFirstSearchDaily = true
	} else {
		y1, m1, d1 := lastUsed.Date()
		y2, m2, d2 := now.Date()
		if y1 != y2 || m1 != m2 || d1 != d2 {
			isFirstSearchDaily = true
		}
	}

	// Apply First Search Bonus
	if isFirstSearchDaily {
		roll = 0.0 // Guaranteed Success (and Critical Success since 0.0 <= 0.05)
		log.Info("First search of the day - applying bonus", "username", username)
	}

	if roll <= SearchSuccessRate {
		// Success case
		isCritical := roll <= SearchCriticalRate
		quantity := 1
		if isCritical {
			quantity = 2
		}

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

		// Begin transaction for inventory update
		tx, err := s.repo.BeginTx(ctx)
		if err != nil {
			log.Error("Failed to begin transaction", "error", err)
			return "", fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer repository.SafeRollback(ctx, tx)

		// Add to inventory
		inventory, err := tx.GetInventory(ctx, user.ID)
		if err != nil {
			log.Error("Failed to get inventory", "error", err, "userID", user.ID)
			return "", fmt.Errorf("failed to get inventory: %w", err)
		}

		i, _ := utils.FindSlot(inventory, item.ID)
		if i != -1 {
			inventory.Slots[i].Quantity += quantity
		} else {
			inventory.Slots = append(inventory.Slots, domain.InventorySlot{ItemID: item.ID, Quantity: quantity})
		}

		if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
			log.Error("Failed to update inventory", "error", err, "userID", user.ID)
			return "", fmt.Errorf("failed to update inventory: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			log.Error("Failed to commit transaction", "error", err)
			return "", fmt.Errorf("failed to commit transaction: %w", err)
		}

		// Award Explorer XP for finding item (async, don't block)
		s.wg.Add(1)
		go s.awardExplorerXP(context.Background(), user.ID, item.InternalName)

		// Get display name with shine (empty shine for search results)
		displayName := s.namingResolver.GetDisplayName(item.InternalName, "")

		if isCritical {
			resultMessage = fmt.Sprintf("%s You found %dx %s", domain.MsgSearchCriticalSuccess, quantity, displayName)
			log.Info("Search CRITICAL success", "username", username, "item", item.InternalName, "quantity", quantity)
		} else {
			resultMessage = fmt.Sprintf("You have found %dx %s", quantity, displayName)
			log.Info("Search successful - lootbox found", "username", username, "item", item.InternalName)
		}

		if isFirstSearchDaily {
			resultMessage += domain.MsgFirstSearchBonus
		}
	} else if roll <= SearchSuccessRate+SearchNearMissRate {
		// Near Miss case
		if s.statsService != nil {
			_ = s.statsService.RecordUserEvent(ctx, user.ID, domain.EventSearchNearMiss, map[string]interface{}{
				"roll":      roll,
				"threshold": SearchSuccessRate,
			})
		}
		resultMessage = domain.MsgSearchNearMiss
		log.Info("Search NEAR MISS", "username", username, "roll", roll)
	} else {
		// Failure case - Pick a random funny message
		resultMessage = domain.MsgSearchNothingFound
		if len(domain.SearchFailureMessages) > 0 {
			idx := utils.RandomInt(0, len(domain.SearchFailureMessages)-1)
			resultMessage = domain.SearchFailureMessages[idx]
		}
		log.Info("Search successful - nothing found", "username", username, "message", resultMessage)
	}

	// Update cooldown
	if err := s.repo.UpdateCooldown(ctx, user.ID, domain.ActionSearch, now); err != nil {
		log.Error("Failed to update cooldown", "error", err, "userID", user.ID)
		// Don't fail the search, just log the error
	}

	log.Info("Search completed", "username", username, "result", resultMessage)
	return resultMessage, nil
}

// getUserOrRegister gets a user by platform ID, or auto-registers them if not found
func (s *service) getUserOrRegister(ctx context.Context, platform, platformID, username string) (*domain.User, error) {
	log := logger.FromContext(ctx)

	// Try to find existing user by platform ID
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		log.Error("Failed to get user by platform ID", "error", err, "platform", platform, "platformID", platformID)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user != nil {
		log.Debug("Found existing user", "userID", user.ID, "platform", platform)
		return user, nil
	}

	// User not found, auto-register
	log.Info("Auto-registering new user", "platform", platform, "platformID", platformID, "username", username)
	newUser := domain.User{Username: username}
	setPlatformID(&newUser, platform, platformID)

	registered, err := s.RegisterUser(ctx, newUser)
	if err != nil {
		log.Error("Failed to auto-register user", "error", err)
		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	log.Info("User auto-registered", "userID", registered.ID)
	return &registered, nil
}

// awardExplorerXP awards Explorer job XP for finding items during search
func (s *service) awardExplorerXP(ctx context.Context, userID, itemName string) {
	defer s.wg.Done()

	if s.jobService == nil {
		return // Job system not enabled
	}

	xp := job.ExplorerXPPerItem

	metadata := map[string]interface{}{
		"item_name": itemName,
	}

	result, err := s.jobService.AwardXP(ctx, userID, job.JobKeyExplorer, xp, "search", metadata)
	if err != nil {
		logger.FromContext(ctx).Warn("Failed to award Explorer XP", "error", err, "user_id", userID)
	} else if result != nil && result.LeveledUp {
		logger.FromContext(ctx).Info("Explorer leveled up!", "user_id", userID, "new_level", result.NewLevel)
	}
}

func (s *service) Shutdown(ctx context.Context) error {
	logger.FromContext(ctx).Info("User service shutting down, waiting for background tasks...")
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
