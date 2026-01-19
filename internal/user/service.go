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
	handlerRegistry *HandlerRegistry
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
	itemCacheByName map[string]domain.Item // Primary cache by internal name
	itemIDToName    map[int]string         // Index for ID -> name lookups
	itemCacheMu     sync.RWMutex           // Protects both maps

	rnd func() float64 // For RNG - allows deterministic testing
}

// Compile-time interface checks
var _ Service = (*service)(nil)
var _ InventoryService = (*service)(nil)
var _ UserManagementService = (*service)(nil)
var _ AccountLinkingService = (*service)(nil)
var _ GameplayService = (*service)(nil)

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

	if val := os.Getenv(EnvUserCacheSize); val != "" {
		if size, err := strconv.Atoi(val); err == nil && size > 0 {
			config.Size = size
		}
	}

	if val := os.Getenv(EnvUserCacheTTL); val != "" {
		if ttl, err := time.ParseDuration(val); err == nil && ttl > 0 {
			config.TTL = ttl
		}
	}

	return config
}

// NewService creates a new user service
func NewService(repo repository.User, statsService stats.Service, jobService JobService, lootboxService lootbox.Service, namingResolver naming.Resolver, cooldownService cooldown.Service, devMode bool) Service {
	return &service{
		repo:            repo,
		handlerRegistry: NewHandlerRegistry(),
		timeouts:        make(map[string]*timeoutInfo),
		lootboxService:  lootboxService,
		jobService:      jobService,
		statsService:    statsService,
		stringFinder:    NewStringFinder(),
		namingResolver:  namingResolver,
		cooldownService: cooldownService,
		devMode:         devMode,
		itemCacheByName: make(map[string]domain.Item),
		itemIDToName:    make(map[int]string),
		userCache:       newUserCache(loadCacheConfig()),
		rnd:             utils.RandomFloat,
	}
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
	log.Info(LogMsgRegisterUserCalled, "username", user.Username)
	if err := s.repo.UpsertUser(ctx, &user); err != nil {
		log.Error(LogErrFailedToUpsertUser, "error", err, "username", user.Username)
		return domain.User{}, err
	}

	// Cache the newly registered user for all their platforms
	keys := getPlatformKeysFromUser(user)
	for platform, platformID := range keys {
		s.userCache.Set(platform, platformID, &user)
	}

	log.Info(LogMsgUserRegistered, "user_id", user.ID, "username", user.Username)
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
		return nil, domain.ErrFailedToGetUser
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

	item, err := s.getItemByNameCached(ctx, itemName)
	if err != nil {
		log.Error("Failed to get item", "error", err, "itemName", itemName)
		return domain.ErrFailedToGetItem
	}
	if item == nil {
		log.Warn("Item not found", "itemName", itemName)
		return domain.ErrItemNotFound
	}

	return s.withTx(ctx, func(tx repository.UserTx) error {
		inventory, err := tx.GetInventory(ctx, user.ID)
		if err != nil {
			log.Error("Failed to get inventory", "error", err, "userID", user.ID)
			return domain.ErrFailedToGetInventory
		}

		// Add item to inventory using utility function
		i, _ := utils.FindSlot(inventory, item.ID)
		if i != -1 {
			inventory.Slots[i].Quantity += quantity
		} else {
			inventory.Slots = append(inventory.Slots, domain.InventorySlot{ItemID: item.ID, Quantity: quantity})
		}

		if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
			log.Error("Failed to update inventory", "error", err, "userID", user.ID)
			return domain.ErrFailedToUpdateInventory
		}

		return nil
	})
}

// removeItemFromUserInternal removes an item from a user's inventory within a transaction
func (s *service) removeItemFromUserInternal(ctx context.Context, user *domain.User, itemName string, quantity int) (int, error) {
	log := logger.FromContext(ctx)

	item, err := s.getItemByNameCached(ctx, itemName)
	if err != nil {
		log.Error("Failed to get item", "error", err, "itemName", itemName)
		return 0, domain.ErrFailedToGetItem
	}
	if item == nil {
		return 0, domain.ErrItemNotFound
	}

	var removed int
	err = s.withTx(ctx, func(tx repository.UserTx) error {
		inventory, err := tx.GetInventory(ctx, user.ID)
		if err != nil {
			log.Error("Failed to get inventory", "error", err, "userID", user.ID)
			return domain.ErrFailedToGetInventory
		}

		// Remove item from inventory using utility function
		i, slotQty := utils.FindSlot(inventory, item.ID)
		if i == -1 {
			log.Warn("Item not in inventory", "itemName", itemName)
			return domain.ErrNotInInventory
		}

		if slotQty >= quantity {
			inventory.Slots[i].Quantity -= quantity
			removed = quantity
		} else {
			removed = slotQty
			inventory.Slots[i].Quantity = 0
		}
		if inventory.Slots[i].Quantity == 0 {
			inventory.Slots = append(inventory.Slots[:i], inventory.Slots[i+1:]...)
		}

		if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
			log.Error("Failed to update inventory", "error", err, "userID", user.ID)
			return domain.ErrFailedToUpdateInventory
		}

		return nil
	})

	return removed, err
}

// useItemInternal handles item usage logic within a transaction
func (s *service) useItemInternal(ctx context.Context, user *domain.User, itemName string, quantity int, targetUser *domain.User) (string, error) {
	log := logger.FromContext(ctx)

	itemToUse, err := s.getItemByNameCached(ctx, itemName)
	if err != nil {
		log.Error("Failed to get item", "error", err, "itemName", itemName)
		return "", domain.ErrFailedToGetItem
	}
	if itemToUse == nil {
		log.Warn("Item not found", "itemName", itemName)
		return "", domain.ErrItemNotFound
	}

	var message string
	err = s.withTx(ctx, func(tx repository.UserTx) error {
		inventory, err := tx.GetInventory(ctx, user.ID)
		if err != nil {
			log.Error("Failed to get inventory", "error", err, "userID", user.ID)
			return domain.ErrFailedToGetInventory
		}

		// Find item in inventory using utility function
		itemSlotIndex, slotQty := utils.FindSlot(inventory, itemToUse.ID)
		if itemSlotIndex == -1 {
			return domain.ErrNotInInventory
		}
		if slotQty < quantity {
			return domain.ErrInsufficientQuantity
		}

		// Execute item handler
		handler := s.handlerRegistry.GetHandler(itemName)
		if handler == nil {
			log.Warn("No handler for item", "itemName", itemName)
			return domain.ErrItemNotHandled
		}
		var args map[string]interface{}
		if targetUser == nil {
			args = map[string]interface{}{"username": user.Username}
		} else {
			args = map[string]interface{}{"targetUsername": targetUser.Username, "username": user.Username}
		}
		message, err = handler.Handle(ctx, s, user, inventory, itemToUse, quantity, args)
		if err != nil {
			log.Error("Handler error", "error", err, "itemName", itemName)
			return err
		}

		if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
			log.Error("Failed to update inventory after use", "error", err, "userID", user.ID)
			return domain.ErrFailedToUpdateInventory
		}

		return nil
	})

	return message, err
}

// getInventoryInternal retrieves a user's inventory with optional filtering
func (s *service) getInventoryInternal(ctx context.Context, user *domain.User, filter string) ([]InventoryItem, error) {
	log := logger.FromContext(ctx)

	inventory, err := s.repo.GetInventory(ctx, user.ID)
	if err != nil {
		log.Error("Failed to get inventory", "error", err, "userID", user.ID)
		return nil, domain.ErrFailedToGetInventory
	}

	// Optimization: Batch fetch all item details using cache
	itemMap := make(map[int]domain.Item)
	var missingIDs []int

	s.itemCacheMu.RLock()
	for _, slot := range inventory.Slots {
		// Use index to find item name, then look up in primary cache
		if itemName, ok := s.itemIDToName[slot.ItemID]; ok {
			if item, ok := s.itemCacheByName[itemName]; ok {
				itemMap[slot.ItemID] = item
				continue
			}
		}
		missingIDs = append(missingIDs, slot.ItemID)
	}
	s.itemCacheMu.RUnlock()

	if len(missingIDs) > 0 {
		itemList, err := s.repo.GetItemsByIDs(ctx, missingIDs)
		if err != nil {
			log.Error("Failed to get item details", "error", err)
			return nil, domain.ErrFailedToGetItemDetails
		}

		s.itemCacheMu.Lock()
		for _, item := range itemList {
			s.itemCacheByName[item.InternalName] = item
			s.itemIDToName[item.ID] = item.InternalName
			itemMap[item.ID] = item
		}
		s.itemCacheMu.Unlock()
	}

	items := make([]InventoryItem, 0, len(inventory.Slots))
	for _, slot := range inventory.Slots {
		item, ok := itemMap[slot.ItemID]
		if !ok {
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

		items = append(items, InventoryItem{
			Name:     item.PublicName,
			Quantity: slot.Quantity,
		})
	}

	return items, nil
}

// AddItemByUsername adds an item by platform username
func (s *service) AddItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) error {
	user, err := s.GetUserByPlatformUsername(ctx, platform, username)
	if err != nil {
		return err
	}
	
	return s.addItemToUserInternal(ctx, user, itemName, quantity)
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
			return domain.ErrItemNotFound
		}

		// Update cache and map
		s.itemCacheMu.Lock()
		for _, item := range missingItems {
			s.itemCacheByName[item.InternalName] = item
			s.itemIDToName[item.ID] = item.InternalName
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
			return domain.ErrItemNotFound
		}
		slotsToAdd = append(slotsToAdd, domain.InventorySlot{
			ItemID:   itemID,
			Quantity: quantity,
		})
	}

	// Start single transaction for all items
	err = s.withTx(ctx, func(tx repository.UserTx) error {
		// Get inventory once
		inventory, err := tx.GetInventory(ctx, user.ID)
		if err != nil {
			log.Error("Failed to get inventory", "error", err, "userID", user.ID)
			return domain.ErrFailedToGetInventory
		}

		// Add all items to inventory using optimized helper
		utils.AddItemsToInventory(inventory, slotsToAdd, nil)

		// Single inventory update
		if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
			log.Error("Failed to update inventory", "error", err, "userID", user.ID)
			return domain.ErrFailedToUpdateInventory
		}

		return nil
	})
	if err != nil {
		return err
	}

	log.Info("Items added successfully", "username", username, "itemCount", len(items))
	return nil
}

// RemoveItemByUsername removes an item by platform username
func (s *service) RemoveItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) (int, error) {
	log := logger.FromContext(ctx)
	log.Info("RemoveItemByUsername called",
		"platform", platform, "username", username,
		"itemName", itemName, "quantity", quantity)
	user, err := s.GetUserByPlatformUsername(ctx, platform, username)
	if err != nil {
		log.Error("Failed to get user", "error", err)
		return 0, domain.ErrFailedToGetUser
	}
	return s.removeItemFromUserInternal(ctx, user, itemName, quantity)
}

func (s *service) GiveItem(ctx context.Context, ownerPlatform, ownerPlatformID, ownerUsername, receiverPlatform, receiverUsername, itemName string, quantity int) error {
	log := logger.FromContext(ctx)
	log.Info("GiveItem called",
		"ownerPlatform", ownerPlatform, "ownerPlatformID", ownerPlatformID, "ownerUsername", ownerUsername,
		"receiverPlatform", receiverPlatform, "receiverUsername", receiverUsername,
		"item", itemName, "quantity", quantity)

	owner, err := s.getUserOrRegister(ctx, ownerPlatform, ownerPlatformID, ownerUsername)
	if err != nil {
		log.Error("Failed to get owner", "error", err)
		return domain.ErrUserNotFound
	}

	receiver, err := s.GetUserByPlatformUsername(ctx, receiverPlatform, receiverUsername)
	if err != nil {
		log.Error("Failed to get receiver", "error", err)
		return domain.ErrUserNotFound
	}

	if( quantity <= 0 || quantity > domain.MaxTransactionQuantity ) {
		log.Error("Quantity validation failed", "error", domain.ErrInvalidInput)
		return domain.ErrInvalidInput
	}

	item, err := s.validateItem(ctx, itemName)
	if err != nil {
		log.Error("Failed to get item", "error", err)
		return domain.ErrItemNotFound
	}

	return s.executeGiveItemTx(ctx, owner, receiver, item, quantity)
}

func (s *service) executeGiveItemTx(ctx context.Context, owner, receiver *domain.User, item *domain.Item, quantity int) error {
	log := logger.FromContext(ctx)

	return s.withTx(ctx, func(tx repository.UserTx) error {
		ownerInventory, err := tx.GetInventory(ctx, owner.ID)
		if err != nil {
			log.Error("Failed to get owner inventory", "error", err)
			return domain.ErrFailedToGetInventory
		}

		// Find item in owner's inventory using utility function
		ownerSlotIndex, ownerSlotQty := utils.FindSlot(ownerInventory, item.ID)
		if ownerSlotIndex == -1 {
			log.Warn("Item not found in owner's inventory", "item", item.InternalName)
			return domain.ErrNotInInventory
		}
		if ownerSlotQty < quantity {
			log.Warn("Insufficient quantity in owner's inventory", "item", item.InternalName, "quantity", quantity)
			return domain.ErrInsufficientQuantity
		}

		receiverInventory, err := tx.GetInventory(ctx, receiver.ID)
		if err != nil {
			log.Error("Failed to get receiver inventory", "error", err)
			return domain.ErrFailedToGetInventory
		}

		// Remove from owner
		if ownerSlotQty == quantity {
			ownerInventory.Slots = append(ownerInventory.Slots[:ownerSlotIndex], ownerInventory.Slots[ownerSlotIndex+1:]...)
		} else {
			ownerInventory.Slots[ownerSlotIndex].Quantity -= quantity
		}

		// Add to receiver using utility function
		receiverSlotIndex, _ := utils.FindSlot(receiverInventory, item.ID)
		if receiverSlotIndex != -1 {
			receiverInventory.Slots[receiverSlotIndex].Quantity += quantity
		} else {
			receiverInventory.Slots = append(receiverInventory.Slots, domain.InventorySlot{ItemID: item.ID, Quantity: quantity})
		}

		if err := tx.UpdateInventory(ctx, owner.ID, *ownerInventory); err != nil {
			log.Error("Failed to update owner inventory", "error", err)
			return domain.ErrFailedToUpdateInventory
		}
		if err := tx.UpdateInventory(ctx, receiver.ID, *receiverInventory); err != nil {
			log.Error("Failed to update receiver inventory", "error", err)
			return domain.ErrFailedToUpdateInventory
		}

		log.Info("Item transferred", "owner", owner.Username, "receiver", receiver.Username, "item", item.InternalName, "quantity", quantity)
		return nil
	})
}

func (s *service) UseItem(ctx context.Context, platform, platformID, username, itemName string, quantity int, targetUsername string) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("UseItem called",
		"platform", platform, "platformID", platformID, "username", username,
		"itemName", itemName, "quantity", quantity, "targetUsername", targetUsername)

	user, err := s.getUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		log.Error("Failed to get user or register", "error", err)
		return "", domain.ErrFailedToGetUser
	}
	
	// Resolve public name to internal name
	resolvedName, err := s.resolveItemName(ctx, itemName)
	if err != nil {
		log.Error("Failed to resolve item name", "error", err)
		return "", domain.ErrInvalidInput
	}

	// targetUser is optional depending on the item
	var targetUser *domain.User = nil
	if targetUsername != "" {
		targetUser, err = s.GetUserByPlatformUsername(ctx, platform, targetUsername)
		if err != nil {
			log.Error("Failed to resolve target username", "error", err)
			return "", domain.ErrFailedToGetUser
		}
	}

	return s.useItemInternal(ctx, user, resolvedName, quantity, targetUser)
}

// resolveItemName attempts to resolve a user-provided item name to its internal name.
// It first tries the naming resolver, then falls back to using the input as-is.
// This allows users to use either public names ("junkbox") or internal names ("lootbox_tier0").
func (s *service) resolveItemName(ctx context.Context, itemName string) (string, error) {
	log := logger.FromContext(ctx)
	// Try naming resolver first (handles public names)
	if s.namingResolver != nil {
		if internalName, ok := s.namingResolver.ResolvePublicName(itemName); ok {
			return internalName, nil
		}
	}

	// Fall back - assume it's already an internal name
	item, err := s.getItemByNameCached(ctx, itemName)
	if err != nil {
		log.Warn("Failed to resolve item name", "error", err)
		return "", domain.ErrItemNotFound
	}
	if item == nil {
		log.Warn("Item not found", "itemName", itemName)
		return "", domain.ErrItemNotFound
	}

	return itemName, nil
}

func (s *service) GetInventory(ctx context.Context, platform, platformID, username, filter string) ([]InventoryItem, error) {
	log := logger.FromContext(ctx)
	log.Info("GetInventory called", "platform", platform, "platformID", platformID, "username", username)

	user, err := s.getUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		log.Error("Failed to get user or register", "error", err)
		return nil, domain.ErrFailedToGetUser
	}

	return s.getInventoryInternal(ctx, user, filter)
}

// GetInventoryByUsername gets inventory by platform username
func (s *service) GetInventoryByUsername(ctx context.Context, platform, username, filter string) ([]InventoryItem, error) {
	log := logger.FromContext(ctx)
	log.Info("GetInventoryByUsername called", "platform", platform, "username", username)

	// Look up user by username
	user, err := s.GetUserByPlatformUsername(ctx, platform, username)
	if err != nil {
		log.Error("Failed to get user by username", "error", err)
		return nil, domain.ErrFailedToGetUser
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
	// Note: Using slog.Default() here since the timer callback runs asynchronously
	// and the original context may no longer be valid
	timer := time.AfterFunc(duration, func() {
		s.timeoutMu.Lock()
		delete(s.timeouts, username)
		s.timeoutMu.Unlock()
		slog.Default().Info("User timeout expired", "username", username, "reason", reason)
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
	log := logger.FromContext(ctx)
	item, err := s.getItemByNameCached(ctx, itemName)
	if err != nil {
		log.Error("Failed to get item by name", "error", err)
		return nil, domain.ErrItemNotFound
	}
	if item == nil {
		log.Error("Item not found", "itemName", itemName)
		return nil, domain.ErrItemNotFound
	}
	return item, nil
}


// HandleSearch performs a search action for a user with cooldown tracking
func (s *service) HandleSearch(ctx context.Context, platform, platformID, username string) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("HandleSearch called", "platform", platform, "platformID", platformID, "username", username)

	// Get or create user
	user, err := s.getUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		log.Error("Failed to get user or register", "error", err)
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
	params := s.calculateSearchParameters(ctx, user)

	// Perform search roll
	roll := s.rnd()

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
		isDiminished:       (dailyCount >= SearchDailyDiminishmentThreshold),
		xpMultiplier:       1.0,
		successThreshold:   SearchSuccessRate,
		dailyCount:         dailyCount,
	}

	if params.isDiminished {
		params.successThreshold = SearchDiminishedSuccessRate
		params.xpMultiplier = SearchDiminishedXPMultiplier
		log.Info(LogMsgDiminishedReturnsApplied, "username", user.Username, "dailyCount", dailyCount)
	}

	return params
}

func (s *service) processSearchSuccess(ctx context.Context, user *domain.User, roll float64, params searchParams) (string, error) {
	isCritical := roll <= SearchCriticalRate
	quantity := 1
	if isCritical {
		quantity = 2
	}

	// Grant reward
	if err := s.grantSearchReward(ctx, user, quantity); err != nil {
		return "", err
	}

	// Get item for message formatting and event recording
	item, err := s.getItemByNameCached(ctx, domain.ItemLootbox0)
	if err != nil {
		return "", fmt.Errorf("failed to get reward item: %w", err)
	}

	// Award Explorer XP for finding item (async, don't block)
	s.wg.Add(1)
	go s.awardExplorerXP(context.Background(), user.ID, item.InternalName, params.xpMultiplier)

	// Record success events
	s.recordSearchSuccessEvents(ctx, user, item, quantity, roll, isCritical)

	// Format and return result message
	return s.formatSearchSuccessMessage(ctx, item, quantity, isCritical, params), nil
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
	// Determine failure type
	failureType := determineSearchFailureType(roll, successThreshold)

	// Record failure events
	s.recordSearchFailureEvents(ctx, user, roll, successThreshold, failureType)

	// Format failure message
	resultMessage := formatSearchFailureMessage(failureType)

	// Record search attempt (to track daily count)
	if s.statsService != nil {
		_ = s.statsService.RecordUserEvent(ctx, user.ID, domain.EventSearch, map[string]interface{}{
			"success":     roll <= successThreshold,
			"daily_count": params.dailyCount + 1, // +1 because we just did one
		})
	}

	// Append streak bonus if applicable
	return s.appendStreakBonus(ctx, user, resultMessage, params.isFirstSearchDaily)
}

// getUserOrRegister gets a user by platform ID, or auto-registers them if not found
func (s *service) getUserOrRegister(ctx context.Context, platform, platformID, username string) (*domain.User, error) {
	log := logger.FromContext(ctx)
	if username == "" || platform == "" || !validPlatforms[platform]{
		log.Error("Invalid platform or username", "platform", platform, "username", username)
		return nil, domain.ErrInvalidInput
	}

	// Try cache first
	if user, ok := s.userCache.Get(platform, platformID); ok {
		log.Debug("User cache hit", "userID", user.ID, "platform", platform)
		return user, nil
	}

	// Cache miss - fetch from database
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		log.Error("Failed to get user by platform ID", "error", err, "platform", platform, "platformID", platformID)
		return nil, domain.ErrFailedToGetUser
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
		return nil, domain.ErrFailedToRegisterUser
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
