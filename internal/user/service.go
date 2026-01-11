package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/cooldown"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
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


// Service defines the interface for user operations
type Service interface {
	RegisterUser(ctx context.Context, user domain.User) (domain.User, error)
	FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	HandleIncomingMessage(ctx context.Context, platform, platformID, username, message string) (*domain.MessageResult, error)

	// Inventory management - by platform ID
	AddItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) error
	AddItems(ctx context.Context, platform, platformID, username string, items map[string]int) error
	RemoveItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (int, error)
	GiveItem(ctx context.Context, ownerPlatform, ownerPlatformID, ownerUsername, receiverPlatform, receiverPlatformID, receiverUsername, itemName string, quantity int) error
	UseItem(ctx context.Context, platform, platformID, username, itemName string, quantity int, targetUsername string) (string, error)
	GetInventory(ctx context.Context, platform, platformID, username, filter string) ([]UserInventoryItem, error)

	// Inventory management - by username
	AddItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) error
	RemoveItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) (int, error)
	UseItemByUsername(ctx context.Context, platform, username, itemName string, quantity int, targetUsername string) (string, error)
	GetInventoryByUsername(ctx context.Context, platform, username, filter string) ([]UserInventoryItem, error)
	GiveItemByUsername(ctx context.Context, fromPlatform, fromUsername, toPlatform, toUsername, itemName string, quantity int) (string, error)

	// User lookup by platform and username
	GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error)
	UpdateUser(ctx context.Context, user domain.User) error

	// Other methods
	TimeoutUser(ctx context.Context, username string, duration time.Duration, reason string) error
	HandleSearch(ctx context.Context, platform, platformID, username string) (string, error)
	// Account linking methods
	MergeUsers(ctx context.Context, primaryUserID, secondaryUserID string) error
	UnlinkPlatform(ctx context.Context, userID, platform string) error
	GetLinkedPlatforms(ctx context.Context, platform, platformID string) ([]string, error)
	GetTimeout(ctx context.Context, username string) (time.Duration, error)
	GetCacheStats() CacheStats
	Shutdown(ctx context.Context) error
}

type UserInventoryItem struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

// ItemEffectHandler defines the function signature for item effects
type ItemEffectHandler func(ctx context.Context, s *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error)

// JobService defines the interface for job operations
type JobService interface {
	AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error)
}

// timeoutInfo tracks active timeouts
type timeoutInfo struct {
	timer     *time.Timer
	expiresAt time.Time
}

// service implements the Service interface
type service struct {
	repo            repository.User
	itemHandlers    map[string]ItemEffectHandler
	timeoutMu       sync.Mutex
	timeouts        map[string]*timeoutInfo
	lootboxService  lootbox.Service
	jobService      JobService
	statsService    stats.Service
	stringFinder    *StringFinder
	namingResolver  naming.Resolver
	cooldownService cooldown.Service
	devMode         bool // When true, bypasses cooldowns
	wg              sync.WaitGroup
	userCache       *userCache // In-memory cache for user lookups

	// Item cache: in-memory cache for item metadata (name, description, value, etc.)
	// Purpose: Reduce database queries for frequently accessed item data
	// Thread-safety: Protected by itemCacheMu (RWMutex)
	// Invalidation: Cache is populated on-demand and persists for server lifetime
	//               Item metadata is assumed immutable - if items are modified in DB,
	//               server restart is required to refresh cache
	itemCache       map[int]domain.Item    // Cache by item ID
	itemCacheByName map[string]domain.Item // Cache by internal name
	itemCacheMu     sync.RWMutex           // Protects both cache maps
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

func loadCacheConfig() CacheConfig {
	config := DefaultCacheConfig()

	if val := os.Getenv("USER_CACHE_SIZE"); val != "" {
		if size, err := strconv.Atoi(val); err == nil && size > 0 {
			config.Size = size
		}
	}

	if val := os.Getenv("USER_CACHE_TTL"); val != "" {
		if ttl, err := time.ParseDuration(val); err == nil && ttl > 0 {
			config.TTL = ttl
		}
	}

	return config
}

// NewService creates a new user service
func NewService(repo repository.User, statsService stats.Service, jobService JobService, lootboxService lootbox.Service, namingResolver naming.Resolver, cooldownService cooldown.Service, devMode bool) Service {
	s := &service{
		repo:            repo,
		itemHandlers:    make(map[string]ItemEffectHandler),
		timeouts:        make(map[string]*timeoutInfo),
		lootboxService:  lootboxService,
		jobService:      jobService,
		statsService:    statsService,
		stringFinder:    NewStringFinder(),
		namingResolver:  namingResolver,
		cooldownService: cooldownService,
		devMode:         devMode,
		itemCache:       make(map[int]domain.Item),
		itemCacheByName: make(map[string]domain.Item),
		userCache:       newUserCache(loadCacheConfig()),
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

func getPlatformKeysFromUser(user domain.User) map[string]string {
	keys := make(map[string]string)
	if user.TwitchID != "" {
		keys[domain.PlatformTwitch] = user.TwitchID
	}
	if user.YoutubeID != "" {
		keys[domain.PlatformYoutube] = user.YoutubeID
	}
	if user.DiscordID != "" {
		keys[domain.PlatformDiscord] = user.DiscordID
	}
	return keys
}

// RegisterUser registers a new user
func (s *service) RegisterUser(ctx context.Context, user domain.User) (domain.User, error) {
	log := logger.FromContext(ctx)
	log.Info("RegisterUser called", "username", user.Username)
	if err := s.repo.UpsertUser(ctx, &user); err != nil {
		log.Error("Failed to upsert user", "error", err, "username", user.Username)
		return domain.User{}, err
	}

	// Cache the newly registered user for all their platforms
	keys := getPlatformKeysFromUser(user)
	for platform, platformID := range keys {
		s.userCache.Set(platform, platformID, &user)
	}

	log.Info("User registered", "user_id", user.ID, "username", user.Username)
	return user, nil
}

// UpdateUser updates an existing user
func (s *service) UpdateUser(ctx context.Context, user domain.User) error {
	log := logger.FromContext(ctx)
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		log.Error("Failed to update user", "error", err, "userID", user.ID)
		return err
	}

	// Invalidate cache for all platforms to force refresh on next lookup
	keys := getPlatformKeysFromUser(user)
	for platform, platformID := range keys {
		s.userCache.Invalidate(platform, platformID)
	}

	return nil
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

	user, err := s.getUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		log.Error("Failed to get user", "error", err, "platform", platform, "platformID", platformID)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Find matches in message
	matches := s.stringFinder.FindMatches(message)

	result := &domain.MessageResult{
		User:    *user,
		Matches: matches,
	}

	return result, nil
}

// ============================================================================
// Internal Helper Methods - Inventory Transaction Logic
// ============================================================================
//
// These helpers centralize the common transaction logic for inventory operations.
// Public methods handle user lookup (auto-register vs username-only), then delegate
// to these helpers for the actual inventory modification.

// addItemToUserInternal adds an item to a user's inventory within a transaction
func (s *service) addItemToUserInternal(ctx context.Context, user *domain.User, itemName string, quantity int) error {
	log := logger.FromContext(ctx)

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	item, err := s.getItemByNameCached(ctx, itemName)
	if err != nil {
		log.Error("Failed to get item", "error", err, "itemName", itemName)
		return fmt.Errorf("failed to get item: %w", err)
	}
	if item == nil {
		log.Warn("Item not found", "itemName", itemName)
		return fmt.Errorf("%w: %s", domain.ErrItemNotFound, itemName)
	}

	inventory, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		log.Error("Failed to get inventory", "error", err, "userID", user.ID)
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	// Add item to inventory
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

	return nil
}

// removeItemFromUserInternal removes an item from a user's inventory within a transaction
func (s *service) removeItemFromUserInternal(ctx context.Context, user *domain.User, itemName string, quantity int) (int, error) {
	log := logger.FromContext(ctx)

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "error", err)
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	item, err := s.getItemByNameCached(ctx, itemName)
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

	// Remove item from inventory
	found := false
	var removed int
	for i, slot := range inventory.Slots {
		if slot.ItemID == item.ID {
			if slot.Quantity >= quantity {
				inventory.Slots[i].Quantity -= quantity
				removed = quantity
			} else {
				removed = slot.Quantity
				inventory.Slots[i].Quantity = 0
			}
			if inventory.Slots[i].Quantity == 0 {
				inventory.Slots = append(inventory.Slots[:i], inventory.Slots[i+1:]...)
			}
			found = true
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

	return removed, nil
}

// useItemInternal handles item usage logic within a transaction
func (s *service) useItemInternal(ctx context.Context, user *domain.User, itemName string, quantity int, targetUsername string, username string) (string, error) {
	log := logger.FromContext(ctx)

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

	itemToUse, err := s.getItemByNameCached(ctx, itemName)
	if err != nil {
		log.Error("Failed to get item", "error", err, "itemName", itemName)
		return "", fmt.Errorf("failed to get item: %w", err)
	}
	if itemToUse == nil {
		log.Warn("Item not found", "itemName", itemName)
		return "", fmt.Errorf("%w: %s", domain.ErrItemNotFound, itemName)
	}

	// Find item in inventory
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

	// Execute item handler
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

	return message, nil
}

// getInventoryInternal retrieves a user's inventory with optional filtering
func (s *service) getInventoryInternal(ctx context.Context, user *domain.User, filter string) ([]UserInventoryItem, error) {
	log := logger.FromContext(ctx)

	inventory, err := s.repo.GetInventory(ctx, user.ID)
	if err != nil {
		log.Error("Failed to get inventory", "error", err, "userID", user.ID)
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	// Optimization: Identify missing items in cache first
	var missingIDs []int

	s.itemCacheMu.RLock()
	for _, slot := range inventory.Slots {
		if _, ok := s.itemCache[slot.ItemID]; !ok {
			missingIDs = append(missingIDs, slot.ItemID)
		}
	}
	s.itemCacheMu.RUnlock()

	// Batch fetch missing items if any
	if len(missingIDs) > 0 {
		itemList, err := s.repo.GetItemsByIDs(ctx, missingIDs)
		if err != nil {
			log.Error("Failed to get item details", "error", err)
			return nil, fmt.Errorf("failed to get item details: %w", err)
		}

		s.itemCacheMu.Lock()
		for _, item := range itemList {
			s.itemCache[item.ID] = item
			s.itemCacheByName[item.InternalName] = item
		}
		s.itemCacheMu.Unlock()
	}

	var items []UserInventoryItem
	// Hold read lock while building result to read directly from cache
	// This avoids allocating and populating a temporary map
	s.itemCacheMu.RLock()
	defer s.itemCacheMu.RUnlock()

	for _, slot := range inventory.Slots {
		item, ok := s.itemCache[slot.ItemID]
		if !ok {
			// This should rarely happen as we just populated it
			log.Warn("Item missing for slot", "itemID", slot.ItemID)
			continue
		}

		// Filter logic - check if item has the specified type
		if filter != "" {
			hasType := false
			for _, t := range item.Types {
				if t == filter {
					hasType = true
					break
				}
			}
			if !hasType {
				continue
			}
		}

		items = append(items, UserInventoryItem{
			Name:     item.PublicName,
			Quantity: slot.Quantity,
		})
	}

	return items, nil
}

func (s *service) AddItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) error {
	log := logger.FromContext(ctx)
	log.Info("AddItem called", "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", quantity)

	user, err := s.getUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		return err
	}

	if err := s.addItemToUserInternal(ctx, user, itemName, quantity); err != nil {
		return err
	}

	log.Info("Item added successfully", "username", username, "item", itemName, "quantity", quantity)
	return nil
}

// AddItemByUsername adds an item by platform username
func (s *service) AddItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) error {
	log := logger.FromContext(ctx)
	log.Info("AddItemByUsername called", "platform", platform, "username", username, "item", itemName, "quantity", quantity)

	user, err := s.repo.GetUserByPlatformUsername(ctx, platform, username)
	if err != nil {
		return err
	}

	if err := s.addItemToUserInternal(ctx, user, itemName, quantity); err != nil {
		return err
	}

	log.Info("Item added successfully by username", "username", username, "item", itemName, "quantity", quantity)
	return nil
}

// AddItems adds multiple items to a user's inventory in a single transaction.
// This is more efficient than calling AddItem multiple times as it reduces transaction overhead.
// Useful for bulk operations like lootbox opening.
func (s *service) AddItems(ctx context.Context, platform, platformID, username string, items map[string]int) error {
	log := logger.FromContext(ctx)
	log.Info("AddItems called", "platform", platform, "platformID", platformID, "username", username, "itemCount", len(items))

	if len(items) == 0 {
		return nil // Nothing to do
	}

	user, err := s.getUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		return err
	}

	// Start single transaction for all items
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Get inventory once
	inventory, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		log.Error("Failed to get inventory", "error", err, "userID", user.ID)
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	// Build map of item names -> item IDs
	itemIDMap := make(map[string]int)

	// Optimization: Identify missing items in cache first
	var missingNames []string

	s.itemCacheMu.RLock()
	for itemName := range items {
		if item, ok := s.itemCacheByName[itemName]; ok {
			itemIDMap[itemName] = item.ID
		} else {
			missingNames = append(missingNames, itemName)
		}
	}
	s.itemCacheMu.RUnlock()

	// Batch fetch missing items from DB
	if len(missingNames) > 0 {
		missingItems, err := s.repo.GetItemsByNames(ctx, missingNames)
		if err != nil {
			log.Error("Failed to get missing items", "error", err)
			return fmt.Errorf("failed to get missing items: %w", err)
		}

		// Update cache and map
		s.itemCacheMu.Lock()
		for _, item := range missingItems {
			s.itemCache[item.ID] = item
			s.itemCacheByName[item.InternalName] = item
			itemIDMap[item.InternalName] = item.ID
		}
		s.itemCacheMu.Unlock()
	}

	// Convert items to InventorySlots for the helper
	// Also verifies that all items were found
	slotsToAdd := make([]domain.InventorySlot, 0, len(items))
	for itemName, quantity := range items {
		itemID, ok := itemIDMap[itemName]
		if !ok {
			log.Warn("Item not found", "itemName", itemName)
			return fmt.Errorf("%w: %s", domain.ErrItemNotFound, itemName)
		}
		slotsToAdd = append(slotsToAdd, domain.InventorySlot{
			ItemID:   itemID,
			Quantity: quantity,
		})
	}

	// Add all items to inventory using optimized helper
	utils.AddItemsToInventory(inventory, slotsToAdd, nil)

	// Single inventory update
	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		log.Error("Failed to update inventory", "error", err, "userID", user.ID)
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	// Single commit
	if err := tx.Commit(ctx); err != nil {
		log.Error("Failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info("Items added successfully", "username", username, "itemCount", len(items))
	return nil
}

func (s *service) RemoveItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (int, error) {
	log := logger.FromContext(ctx)
	log.Info("RemoveItem called", "platform", platform, "platformID", platformID, "username", username, "item", itemName, "quantity", quantity)

	user, err := s.getUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		return 0, err
	}

	removed, err := s.removeItemFromUserInternal(ctx, user, itemName, quantity)
	if err != nil {
		return 0, err
	}

	log.Info("Item removed", "username", username, "item", itemName, "removed", removed)
	return removed, nil
}

// RemoveItemByUsername removes an item by platform username
func (s *service) RemoveItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) (int, error) {
	log := logger.FromContext(ctx)
	log.Info("RemoveItemByUsername called", "platform", platform, "username", username, "item", itemName, "quantity", quantity)

	user, err := s.repo.GetUserByPlatformUsername(ctx, platform, username)
	if err != nil {
		return 0, err
	}

	removed, err := s.removeItemFromUserInternal(ctx, user, itemName, quantity)
	if err != nil {
		return 0, err
	}

	log.Info("Item removed by username", "username", username, "item", itemName, "removed", removed)
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

// GiveItemByUsername transfers an item between users using usernames
func (s *service) GiveItemByUsername(ctx context.Context, fromPlatform, fromUsername, toPlatform, toUsername, itemName string, quantity int) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("GiveItemByUsername called",
		"fromPlatform", fromPlatform, "fromUsername", fromUsername,
		"toPlatform", toPlatform, "toUsername", toUsername,
		"item", itemName, "quantity", quantity)

	owner, err := s.repo.GetUserByPlatformUsername(ctx, fromPlatform, fromUsername)
	if err != nil {
		return "", err
	}

	receiver, err := s.repo.GetUserByPlatformUsername(ctx, toPlatform, toUsername)
	if err != nil {
		return "", err
	}

	item, err := s.validateItem(ctx, itemName)
	if err != nil {
		return "", err
	}

	if err := s.executeGiveItemTx(ctx, owner, receiver, item, quantity); err != nil {
		return "", err
	}

	log.Info("Item given by username", "from", fromUsername, "to", toUsername, "item", itemName, "quantity", quantity)
	return fmt.Sprintf("Successfully gave %d %s to %s", quantity, itemName, toUsername), nil
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
		return fmt.Errorf("%w: %s", domain.ErrNotInInventory, item.InternalName)
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

	// Resolve public name to internal name
	resolvedName, err := s.resolveItemName(ctx, itemName)
	if err != nil {
		return "", err
	}

	message, err := s.useItemInternal(ctx, user, resolvedName, quantity, targetUsername, username)
	if err != nil {
		return "", err
	}

	log.Info("Item used", "username", username, "item", itemName, "resolved", resolvedName, "quantity", quantity, "message", message)
	return message, nil
}

// UseItemByUsername uses an item by platform username
func (s *service) UseItemByUsername(ctx context.Context, platform, username, itemName string, quantity int, targetUsername string) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("UseItemByUsername called", "platform", platform, "username", username, "item", itemName, "quantity", quantity, "target", targetUsername)

	user, err := s.repo.GetUserByPlatformUsername(ctx, platform, username)
	if err != nil {
		return "", err
	}

	// Resolve public name to internal name
	resolvedName, err := s.resolveItemName(ctx, itemName)
	if err != nil {
		return "", err
	}

	message, err := s.useItemInternal(ctx, user, resolvedName, quantity, targetUsername, username)
	if err != nil {
		return "", err
	}

	log.Info("Item used by username", "username", username, "item", itemName, "resolved", resolvedName, "quantity", quantity, "message", message)
	return message, nil
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
	item, err := s.getItemByNameCached(ctx, itemName)
	if err != nil {
		return "", fmt.Errorf("failed to resolve item name '%s': %w", itemName, err)
	}
	if item == nil {
		return "", fmt.Errorf("%w: %s (not found as public or internal name)", domain.ErrItemNotFound, itemName)
	}

	return itemName, nil
}

func (s *service) GetInventory(ctx context.Context, platform, platformID, username, filter string) ([]UserInventoryItem, error) {
	log := logger.FromContext(ctx)
	log.Info("GetInventory called", "platform", platform, "platformID", platformID, "username", username)

	user, err := s.getUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		return nil, err
	}

	return s.getInventoryInternal(ctx, user, filter)
}

// GetInventoryByUsername gets inventory by platform username
func (s *service) GetInventoryByUsername(ctx context.Context, platform, username, filter string) ([]UserInventoryItem, error) {
	log := logger.FromContext(ctx)
	log.Info("GetInventoryByUsername called", "platform", platform, "username", username)

	// Look up user by username
	user, err := s.repo.GetUserByPlatformUsername(ctx, platform, username)
	if err != nil {
		return nil, err
	}

	return s.getInventoryInternal(ctx, user, filter)
}

// GetUserByPlatformUsername retrieves a user by platform and username
func (s *service) GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error) {
	return s.repo.GetUserByPlatformUsername(ctx, platform, username)
}

// TimeoutUser times out a user for a specified duration.
// Note: Timeouts are currently in-memory and will be lost on server restart. This is a known design choice.
func (s *service) TimeoutUser(ctx context.Context, username string, duration time.Duration, reason string) error {
	log := logger.FromContext(ctx)
	log.Info("TimeoutUser called", "username", username, "duration", duration, "reason", reason)

	s.timeoutMu.Lock()
	defer s.timeoutMu.Unlock()

	// If user is already timed out, stop the existing timer
	if info, exists := s.timeouts[username]; exists {
		info.timer.Stop()
		log.Info("Existing timeout cancelled", "username", username)
	}

	// Create a new timer
	timer := time.AfterFunc(duration, func() {
		s.timeoutMu.Lock()
		delete(s.timeouts, username)
		s.timeoutMu.Unlock()
		slog.Default().Info("User timeout expired", "username", username)
	})

	s.timeouts[username] = &timeoutInfo{
		timer:     timer,
		expiresAt: time.Now().Add(duration),
	}
	log.Info("User timed out", "username", username, "duration", duration)
	return nil
}

// GetTimeout returns the remaining duration of a user's timeout
func (s *service) GetTimeout(ctx context.Context, username string) (time.Duration, error) {
	s.timeoutMu.Lock()
	defer s.timeoutMu.Unlock()

	info, exists := s.timeouts[username]
	if !exists {
		return 0, nil
	}

	remaining := time.Until(info.expiresAt)
	if remaining < 0 {
		return 0, nil
	}
	return remaining, nil
}

// Helper methods

func (s *service) validateItem(ctx context.Context, itemName string) (*domain.Item, error) {
	item, err := s.getItemByNameCached(ctx, itemName)
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
	SearchSuccessRate      = 0.8
	SearchCriticalRate     = 0.05
	SearchNearMissRate     = 0.05
	SearchCriticalFailRate = 0.05
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

	// Execute search with atomic cooldown enforcement
	var resultMessage string
	err = s.cooldownService.EnforceCooldown(ctx, user.ID, domain.ActionSearch, func() error {
		var err error
		resultMessage, err = s.executeSearch(ctx, user)
		return err
	})

	if err != nil {
		// Check if it's a cooldown error
		var cooldownErr cooldown.ErrOnCooldown
		if errors.As(err, &cooldownErr) {
			// Return user-friendly cooldown message
			minutes := int(cooldownErr.Remaining.Minutes())
			seconds := int(cooldownErr.Remaining.Seconds()) % 60
			if minutes > 0 {
				return fmt.Sprintf("You can search again in %dm %ds.", minutes, seconds), nil
			}
			return fmt.Sprintf("You can search again in %ds.", seconds), nil
		}
		// Other error
		return "", err
	}

	log.Info("Search completed", "username", username, "result", resultMessage)
	return resultMessage, nil
}

type searchParams struct {
	isFirstSearchDaily bool
	isDiminished       bool
	xpMultiplier       float64
	successThreshold   float64
	dailyCount         int
}

// executeSearch performs the actual search logic (called within cooldown enforcement)
func (s *service) executeSearch(ctx context.Context, user *domain.User) (string, error) {
	log := logger.FromContext(ctx)

	params := s.calculateSearchParameters(ctx, user)

	// Perform search roll
	roll := utils.SecureRandomFloat()
	if params.isFirstSearchDaily {
		roll = 0.0 // Guaranteed Success
		log.Info("First search of the day - applying bonus", "username", user.Username)
	}

	var resultMessage string

	if roll <= params.successThreshold {
		var err error
		resultMessage, err = s.processSearchSuccess(ctx, user, roll, params)
		if err != nil {
			return "", err
		}
	} else {
		resultMessage = s.processSearchFailure(ctx, user, roll, params.successThreshold, params)
	}

	// Record search attempt (to track daily count)
	if s.statsService != nil {
		_ = s.statsService.RecordUserEvent(ctx, user.ID, domain.EventSearch, map[string]interface{}{
			"success":     roll <= params.successThreshold,
			"daily_count": params.dailyCount + 1,
		})
	}

	return resultMessage, nil
}

func (s *service) calculateSearchParameters(ctx context.Context, user *domain.User) searchParams {
	log := logger.FromContext(ctx)
	dailyCount := 0
	if s.statsService != nil {
		stats, err := s.statsService.GetUserStats(ctx, user.ID, domain.PeriodDaily)
		if err != nil {
			log.Warn("Failed to get search counts", "error", err)
		} else if stats != nil && stats.EventCounts != nil {
			dailyCount = stats.EventCounts[domain.EventSearch]
		}
	}

	params := searchParams{
		isFirstSearchDaily: (dailyCount == 0),
		isDiminished:       (dailyCount >= 6),
		xpMultiplier:       1.0,
		successThreshold:   SearchSuccessRate,
		dailyCount:         dailyCount,
	}

	if params.isDiminished {
		params.successThreshold = 0.1 // Reduced success rate
		params.xpMultiplier = 0.1     // Reduced XP
		log.Info("Diminished search returns applied", "username", user.Username, "dailyCount", dailyCount)
	}

	return params
}

func (s *service) processSearchSuccess(ctx context.Context, user *domain.User, roll float64, params searchParams) (string, error) {
	log := logger.FromContext(ctx)
	isCritical := roll <= SearchCriticalRate
	quantity := 1
	if isCritical {
		quantity = 2
	}

	// Give lootbox0
	item, err := s.getItemByNameCached(ctx, domain.ItemLootbox0)
	if err != nil {
		log.Error("Failed to get lootbox0 item", "error", err)
		return "", fmt.Errorf("failed to get reward item: %w", err)
	}
	if item == nil {
		log.Error("Lootbox0 item not found in database")
		return "", fmt.Errorf("%w: %s", domain.ErrItemNotFound, domain.ItemLootbox0)
	}

	// Begin transaction for inventory update
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		log.Error("Failed to begin transaction", "error", err)
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Add to inventory
	if err := s.addItemToTx(ctx, tx, user.ID, item.ID, quantity); err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("Failed to commit transaction", "error", err)
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Award Explorer XP for finding item (async, don't block)
	s.wg.Add(1)
	go s.awardExplorerXP(context.Background(), user.ID, item.InternalName, params.xpMultiplier)

	// Get display name with shine (empty shine for search results)
	displayName := s.namingResolver.GetDisplayName(item.InternalName, "")
	var resultMessage string

	if isCritical {
		// Record critical success event
		if s.statsService != nil {
			_ = s.statsService.RecordUserEvent(ctx, user.ID, domain.EventSearchCriticalSuccess, map[string]interface{}{
				"item":     item.InternalName,
				"quantity": quantity,
				"roll":     roll,
			})
		}
		resultMessage = fmt.Sprintf("%s You found %dx %s", domain.MsgSearchCriticalSuccess, quantity, displayName)
		log.Info("Search CRITICAL success", "username", user.Username, "item", item.InternalName, "quantity", quantity)
	} else {
		resultMessage = fmt.Sprintf("You have found %dx %s", quantity, displayName)
		log.Info("Search successful - lootbox found", "username", user.Username, "item", item.InternalName)
	}

	if params.isFirstSearchDaily {
		resultMessage += domain.MsgFirstSearchBonus
	} else if params.isDiminished {
		resultMessage += " (Exhausted)"
	}

	return resultMessage, nil
}

func (s *service) addItemToTx(ctx context.Context, tx repository.Tx, userID string, itemID int, quantity int) error {
	log := logger.FromContext(ctx)
	inventory, err := tx.GetInventory(ctx, userID)
	if err != nil {
		log.Error("Failed to get inventory", "error", err, "userID", userID)
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	i, _ := utils.FindSlot(inventory, itemID)
	if i != -1 {
		inventory.Slots[i].Quantity += quantity
	} else {
		inventory.Slots = append(inventory.Slots, domain.InventorySlot{ItemID: itemID, Quantity: quantity})
	}

	if err := tx.UpdateInventory(ctx, userID, *inventory); err != nil {
		log.Error("Failed to update inventory", "error", err, "userID", userID)
		return fmt.Errorf("failed to update inventory: %w", err)
	}
	return nil
}

func (s *service) processSearchFailure(ctx context.Context, user *domain.User, roll float64, successThreshold float64, params searchParams) string {
	log := logger.FromContext(ctx)

	var resultMessage string

	if roll <= successThreshold+SearchNearMissRate {
		// Near Miss case
		if s.statsService != nil {
			_ = s.statsService.RecordUserEvent(ctx, user.ID, domain.EventSearchNearMiss, map[string]interface{}{
				"roll":      roll,
				"threshold": successThreshold,
			})
		}
		resultMessage = domain.MsgSearchNearMiss
		log.Info("Search NEAR MISS", "username", user.Username, "roll", roll)
	} else {
		// Check for Critical Failure (roll > 0.95)
		if roll > 1.0-SearchCriticalFailRate {
			// Critical Fail case
			if s.statsService != nil {
				_ = s.statsService.RecordUserEvent(ctx, user.ID, domain.EventSearchCriticalFail, map[string]interface{}{
					"roll": roll,
				})
			}
			resultMessage = domain.MsgSearchCriticalFail
			if len(domain.SearchCriticalFailMessages) > 0 {
				idx := utils.SecureRandomIntRange(0, len(domain.SearchCriticalFailMessages)-1)
				resultMessage = fmt.Sprintf("%s %s", domain.MsgSearchCriticalFail, domain.SearchCriticalFailMessages[idx])
			}
			log.Info("Search CRITICAL FAIL", "username", user.Username, "roll", roll)
		} else {
			// Failure case - Pick a random funny message
			resultMessage = domain.MsgSearchNothingFound
			if len(domain.SearchFailureMessages) > 0 {
				idx := utils.SecureRandomIntRange(0, len(domain.SearchFailureMessages)-1)
				resultMessage = domain.SearchFailureMessages[idx]
			}
			log.Info("Search successful - nothing found", "username", user.Username, "message", resultMessage)
		}
	}

	// Record search attempt (to track daily count)
	if s.statsService != nil {
		_ = s.statsService.RecordUserEvent(ctx, user.ID, domain.EventSearch, map[string]interface{}{
			"success":     roll <= successThreshold,
			"daily_count": params.dailyCount + 1, // +1 because we just did one
		})

		// If this was the first search of the day, show the current streak
		if params.isFirstSearchDaily {
			streak, err := s.statsService.GetUserCurrentStreak(ctx, user.ID)
			if err != nil {
				log.Warn("Failed to get user streak", "error", err)
			} else if streak > 1 {
				resultMessage += fmt.Sprintf(domain.MsgStreakBonus, streak)
			}
		}
	}

	return resultMessage
}

// getUserOrRegister gets a user by platform ID, or auto-registers them if not found
func (s *service) getUserOrRegister(ctx context.Context, platform, platformID, username string) (*domain.User, error) {
	log := logger.FromContext(ctx)

	// Try cache first
	if user, ok := s.userCache.Get(platform, platformID); ok {
		log.Debug("User cache hit", "userID", user.ID, "platform", platform)
		return user, nil
	}

	// Cache miss - fetch from database
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		log.Error("Failed to get user by platform ID", "error", err, "platform", platform, "platformID", platformID)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user != nil {
		log.Debug("Found existing user", "userID", user.ID, "platform", platform)
		// Cache the user for future lookups
		s.userCache.Set(platform, platformID, user)
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
func (s *service) awardExplorerXP(ctx context.Context, userID, itemName string, xpMultiplier float64) {
	defer s.wg.Done()

	if s.jobService == nil {
		return // Job system not enabled
	}

	xp := int(float64(job.ExplorerXPPerItem) * xpMultiplier)
	if xp < 1 {
		xp = 1
	}

	metadata := map[string]interface{}{
		"item_name":  itemName,
		"multiplier": xpMultiplier,
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

func (s *service) GetCacheStats() CacheStats {
	return s.userCache.GetStats()
}
