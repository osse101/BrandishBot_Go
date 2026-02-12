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

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/cooldown"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
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

// timeoutInfo tracks active timeouts
type timeoutInfo struct {
	timer     *time.Timer
	expiresAt time.Time
}

// service implements the Service interface
type service struct {
	repo            repository.User
	trapRepo        repository.TrapRepository
	handlerRegistry *HandlerRegistry
	timeoutMu       sync.Mutex
	timeouts        map[string]*timeoutInfo // Keyed by "platform:username"
	lootboxService  lootbox.Service
	publisher       *event.ResilientPublisher
	statsService    stats.Service
	stringFinder    *StringFinder
	namingResolver  naming.Resolver
	cooldownService cooldown.Service
	eventBus        event.Bus  // Event bus for publishing timeout events
	devMode         bool       // When true, bypasses cooldowns
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

	activeChatterTracker *ActiveChatterTracker // Tracks users eligible for random targeting

	rnd func() float64 // For RNG - allows deterministic testing

	wg sync.WaitGroup // Track background tasks for graceful shutdown
}

// Compile-time interface checks
var _ Service = (*service)(nil)
var _ InventoryService = (*service)(nil)
var _ ManagementService = (*service)(nil)
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
func NewService(repo repository.User, trapRepo repository.TrapRepository, statsService stats.Service, publisher *event.ResilientPublisher, lootboxService lootbox.Service, namingResolver naming.Resolver, cooldownService cooldown.Service, eventBus event.Bus, devMode bool) Service {
	return &service{
		repo:                 repo,
		trapRepo:             trapRepo,
		handlerRegistry:      NewHandlerRegistry(),
		timeouts:             make(map[string]*timeoutInfo),
		lootboxService:       lootboxService,
		publisher:            publisher,
		statsService:         statsService,
		stringFinder:         NewStringFinder(),
		namingResolver:       namingResolver,
		cooldownService:      cooldownService,
		eventBus:             eventBus,
		devMode:              devMode,
		itemCacheByName:      make(map[string]domain.Item),
		itemIDToName:         make(map[int]string),
		userCache:            newUserCache(loadCacheConfig()),
		activeChatterTracker: NewActiveChatterTracker(),
		rnd:                  utils.RandomFloat,
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
	log.Debug("HandleIncomingMessage called", "platform", platform, "platformID", platformID, "username", username)

	user, err := s.getUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		log.Error("Failed to get user", "error", err, "platform", platform, "platformID", platformID)
		return nil, domain.ErrFailedToGetUser
	}

	// Track this user as an active chatter for random targeting
	s.activeChatterTracker.Track(platform, user.ID, username)

	// Check for active trap on this user and trigger if it exists
	if s.trapRepo != nil {
		userUUID, _ := uuid.Parse(user.ID)
		trap, err := s.trapRepo.GetActiveTrap(ctx, userUUID)
		if err != nil {
			log.Warn("Failed to check for trap", "user_id", user.ID, "error", err)
		} else if trap != nil {
			// Trigger trap asynchronously (don't block message processing)
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				asyncCtx := context.Background() // New context for async operation
				if err := s.triggerTrap(asyncCtx, trap, user); err != nil {
					log.Error(LogMsgTrapTriggered, "trap_id", trap.ID, "error", err)
				}
			}()
		}
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

// addItemToUserInternal adds an item to a user's inventory within a transaction.
// Used for admin adds - defaults to COMMON quality level.
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

		// Admin adds default to COMMON quality
		qualityLevel := domain.QualityCommon

		// Find slot with matching ItemID AND QualityLevel
		i, _ := utils.FindSlotWithQuality(inventory, item.ID, qualityLevel)
		if i != -1 {
			inventory.Slots[i].Quantity += quantity
		} else {
			inventory.Slots = append(inventory.Slots, domain.InventorySlot{
				ItemID:       item.ID,
				Quantity:     quantity,
				QualityLevel: qualityLevel,
			})
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

		// Remove item from inventory using random selection (in case multiple slots with different quality levels exist)
		i, slotQty := utils.FindRandomSlot(inventory, item.ID, s.rnd)
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
func (s *service) useItemInternal(ctx context.Context, user *domain.User, platform, itemName string, quantity int, targetName string) (string, error) {
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

		// Find item in inventory using random selection (in case multiple slots exist with different quality levels)
		itemSlotIndex, slotQty := utils.FindRandomSlot(inventory, itemToUse.ID, s.rnd)
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
		args := map[string]interface{}{
			ArgsUsername: user.Username,
			ArgsPlatform: platform,
		}
		if targetName != "" {
			args[ArgsTargetUsername] = targetName
			args[ArgsJobName] = targetName
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
	itemMap, err := s.ensureItemsInCache(ctx, inventory)
	if err != nil {
		return nil, err
	}

	// Group items to merge identical items (same ID and quality)
	type itemKey struct {
		ItemID  int
		Quality domain.QualityLevel
	}
	itemsMap := make(map[itemKey]int)
	itemOrder := make([]itemKey, 0)

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

		key := itemKey{ItemID: slot.ItemID, Quality: slot.QualityLevel}
		if _, exists := itemsMap[key]; !exists {
			itemOrder = append(itemOrder, key)
		}
		itemsMap[key] += slot.Quantity
	}

	// Convert back to array in order of first appearance
	items := make([]InventoryItem, 0, len(itemsMap))
	for _, key := range itemOrder {
		item := itemMap[key.ItemID]
		quality := string(key.Quality)
		if quality == "" {
			quality = string(domain.QualityCommon)
		}

		items = append(items, InventoryItem{
			InternalName: item.InternalName,
			PublicName:   item.PublicName,
			Quantity:     itemsMap[key],
			QualityLevel: quality,
		})
	}

	return items, nil
}

// ensureItemsInCache ensures all items in the inventory are present in the service's item cache
// and returns a map of itemID -> Item.
func (s *service) ensureItemsInCache(ctx context.Context, inventory *domain.Inventory) (map[int]domain.Item, error) {
	log := logger.FromContext(ctx)
	itemMap := make(map[int]domain.Item)
	missingIDs := make([]int, 0, len(inventory.Slots))

	s.itemCacheMu.RLock()
	for _, slot := range inventory.Slots {
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

	return itemMap, nil
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
// Items added via this method default to COMMON quality level.
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
	// Items default to COMMON quality level
	slotsToAdd := make([]domain.InventorySlot, 0, len(items))
	for itemName, quantity := range items {
		itemID, ok := itemIDMap[itemName]
		if !ok {
			log.Warn("Item not found", "itemName", itemName)
			return domain.ErrItemNotFound
		}
		slotsToAdd = append(slotsToAdd, domain.InventorySlot{
			ItemID:       itemID,
			Quantity:     quantity,
			QualityLevel: domain.QualityCommon,
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

	if quantity <= 0 || quantity > domain.MaxTransactionQuantity {
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

		// Find item in owner's inventory using random selection (in case multiple slots with different quality levels exist)
		ownerSlotIndex, ownerSlotQty := utils.FindRandomSlot(ownerInventory, item.ID, s.rnd)
		if ownerSlotIndex == -1 {
			log.Warn("Item not found in owner's inventory", "item", item.InternalName)
			return domain.ErrNotInInventory
		}
		if ownerSlotQty < quantity {
			log.Warn("Insufficient quantity in owner's inventory", "item", item.InternalName, "quantity", quantity)
			return domain.ErrInsufficientQuantity
		}

		// Capture the quality level being transferred
		transferredQuality := ownerInventory.Slots[ownerSlotIndex].QualityLevel

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

		// Add to receiver - must match BOTH ItemID and QualityLevel to preserve exact item quality
		receiverSlotIndex, _ := utils.FindSlotWithQuality(receiverInventory, item.ID, transferredQuality)
		if receiverSlotIndex != -1 {
			receiverInventory.Slots[receiverSlotIndex].Quantity += quantity
		} else {
			receiverInventory.Slots = append(receiverInventory.Slots, domain.InventorySlot{
				ItemID:       item.ID,
				Quantity:     quantity,
				QualityLevel: transferredQuality,
			})
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

func (s *service) UseItem(ctx context.Context, platform, platformID, username, itemName string, quantity int, targetName string) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("UseItem called",
		"platform", platform, "platformID", platformID, "username", username,
		"itemName", itemName, "quantity", quantity, "targetName", targetName)

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

	return s.useItemInternal(ctx, user, platform, resolvedName, quantity, targetName)
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

// timeoutKey generates a platform-aware key for the timeout map
func timeoutKey(platform, username string) string {
	return fmt.Sprintf("%s:%s", platform, username)
}

// AddTimeout applies or extends a timeout for a user (accumulating).
// If the user already has a timeout, the new duration is ADDED to the remaining time.
// Note: Timeouts are in-memory and will be lost on server restart.
func (s *service) AddTimeout(ctx context.Context, platform, username string, duration time.Duration, reason string) error {
	log := logger.FromContext(ctx)
	key := timeoutKey(platform, username)
	log.Info("AddTimeout called", "platform", platform, "username", username, "duration", duration, "reason", reason)

	s.timeoutMu.Lock()
	defer s.timeoutMu.Unlock()

	var newExpiresAt time.Time
	now := time.Now()

	// Check if user already has a timeout - accumulate if so
	if info, exists := s.timeouts[key]; exists {
		info.timer.Stop()
		remaining := time.Until(info.expiresAt)
		if remaining < 0 {
			remaining = 0
		}
		// Accumulate: new expiry = now + remaining + new duration
		newExpiresAt = now.Add(remaining + duration)
		log.Info("Timeout accumulated", "platform", platform, "username", username, "previousRemaining", remaining, "added", duration, "newTotal", time.Until(newExpiresAt))
	} else {
		// No existing timeout
		newExpiresAt = now.Add(duration)
		log.Info("New timeout created", "platform", platform, "username", username, "duration", duration)
	}

	// Create timer for expiry
	timer := time.AfterFunc(time.Until(newExpiresAt), func() {
		s.timeoutMu.Lock()
		delete(s.timeouts, key)
		s.timeoutMu.Unlock()
		slog.Default().Info("User timeout expired", "platform", platform, "username", username, "reason", reason)
	})

	s.timeouts[key] = &timeoutInfo{
		timer:     timer,
		expiresAt: newExpiresAt,
	}

	// Publish timeout event
	if s.eventBus != nil {
		totalSeconds := int(time.Until(newExpiresAt).Seconds())
		evt := event.NewTimeoutAppliedEvent(platform, username, totalSeconds, reason)
		if err := s.eventBus.Publish(ctx, evt); err != nil {
			log.Warn("Failed to publish timeout applied event", "error", err)
		}
	}

	return nil
}

// ClearTimeout removes a user's timeout (admin action).
func (s *service) ClearTimeout(ctx context.Context, platform, username string) error {
	log := logger.FromContext(ctx)
	key := timeoutKey(platform, username)
	log.Info("ClearTimeout called", "platform", platform, "username", username)

	s.timeoutMu.Lock()
	defer s.timeoutMu.Unlock()

	info, exists := s.timeouts[key]
	if !exists {
		log.Info("No timeout to clear", "platform", platform, "username", username)
		return nil
	}

	info.timer.Stop()
	delete(s.timeouts, key)
	log.Info("Timeout cleared", "platform", platform, "username", username)

	// Publish timeout cleared event
	if s.eventBus != nil {
		evt := event.NewTimeoutClearedEvent(platform, username)
		if err := s.eventBus.Publish(ctx, evt); err != nil {
			log.Warn("Failed to publish timeout cleared event", "error", err)
		}
	}

	return nil
}

// GetTimeoutPlatform returns the remaining duration of a user's timeout for a specific platform.
func (s *service) GetTimeoutPlatform(ctx context.Context, platform, username string) (time.Duration, error) {
	key := timeoutKey(platform, username)

	s.timeoutMu.Lock()
	defer s.timeoutMu.Unlock()

	info, exists := s.timeouts[key]
	if !exists {
		return 0, nil
	}

	remaining := time.Until(info.expiresAt)
	if remaining < 0 {
		return 0, nil
	}
	return remaining, nil
}

// ReduceTimeoutPlatform reduces a user's timeout by the specified duration for a specific platform.
func (s *service) ReduceTimeoutPlatform(ctx context.Context, platform, username string, reduction time.Duration) error {
	log := logger.FromContext(ctx)
	key := timeoutKey(platform, username)
	log.Info("ReduceTimeoutPlatform called", "platform", platform, "username", username, "reduction", reduction)

	s.timeoutMu.Lock()
	defer s.timeoutMu.Unlock()

	info, exists := s.timeouts[key]
	if !exists {
		log.Info("User not timed out, nothing to reduce", "platform", platform, "username", username)
		return nil
	}

	// Calculate new expiry time
	newExpiresAt := info.expiresAt.Add(-reduction)
	remaining := time.Until(newExpiresAt)

	if remaining <= 0 {
		// Timeout is fully reduced, remove it
		info.timer.Stop()
		delete(s.timeouts, key)
		log.Info("Timeout fully removed via reduction", "platform", platform, "username", username)

		// Publish cleared event since timeout is gone
		if s.eventBus != nil {
			evt := event.NewTimeoutClearedEvent(platform, username)
			if err := s.eventBus.Publish(ctx, evt); err != nil {
				log.Warn("Failed to publish timeout cleared event", "error", err)
			}
		}
		return nil
	}

	// Update the timer with new duration
	info.timer.Stop()
	info.expiresAt = newExpiresAt
	info.timer = time.AfterFunc(remaining, func() {
		s.timeoutMu.Lock()
		delete(s.timeouts, key)
		s.timeoutMu.Unlock()
		slog.Default().Info("User timeout expired", "platform", platform, "username", username)
	})

	log.Info("Timeout reduced", "platform", platform, "username", username, "newRemaining", remaining)
	return nil
}

// TimeoutUser times out a user for a specified duration.
// Note: This method REPLACES the existing timeout (does not accumulate).
// For accumulating timeouts, use AddTimeout.
func (s *service) TimeoutUser(ctx context.Context, username string, duration time.Duration, reason string) error {
	// Legacy behavior: use AddTimeout with twitch platform
	// Note: The original TimeoutUser replaced timeouts, but we're now using accumulating AddTimeout
	// for consistency. If true replacement behavior is needed, we'd need to clear first.
	return s.AddTimeout(ctx, domain.PlatformTwitch, username, duration, reason)
}

// GetTimeout returns the remaining duration of a user's timeout.
func (s *service) GetTimeout(ctx context.Context, username string) (time.Duration, error) {
	return s.GetTimeoutPlatform(ctx, domain.PlatformTwitch, username)
}

// ReduceTimeout reduces a user's timeout by the specified duration (used by revive items).
func (s *service) ReduceTimeout(ctx context.Context, username string, reduction time.Duration) error {
	return s.ReduceTimeoutPlatform(ctx, domain.PlatformTwitch, username, reduction)
}

// ApplyShield activates shield protection for a user (blocks next weapon attacks)
// Note: Shield count is stored in-memory and will be lost on server restart
func (s *service) ApplyShield(ctx context.Context, user *domain.User, quantity int, isMirror bool) error {
	log := logger.FromContext(ctx)
	log.Info("ApplyShield called", "userID", user.ID, "quantity", quantity, "is_mirror", isMirror)

	// For now, shields are stored in user metadata or a simple map
	// This is a placeholder implementation - full implementation would need persistent storage
	// The shield check would be integrated into the weapon handler

	// TODO: Implement persistent shield storage
	// For now, just log and return success
	shieldType := "standard"
	if isMirror {
		shieldType = "mirror"
	}
	log.Info("Shield applied (placeholder implementation)", "userID", user.ID, "quantity", quantity, "type", shieldType)
	return nil
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
	streak             int
}

// executeSearch performs the actual search logic (called within cooldown enforcement)
func (s *service) executeSearch(ctx context.Context, user *domain.User) (string, error) {
	params := s.calculateSearchParameters(ctx, user)

	// Perform search roll
	roll := s.rnd()

	var resultMessage string
	isSuccess := roll <= params.successThreshold
	var isCritical, isNearMiss, isCritFail bool
	var itemName string
	var quantity int

	if isSuccess {
		var err error
		resultMessage, err = s.processSearchSuccess(ctx, user, roll, params)
		if err != nil {
			return "", err
		}
		isCritical = roll <= SearchCriticalRate
		quantity = 1
		if isCritical {
			quantity = 2
		}
		itemName = domain.ItemLootbox0
	} else {
		failureType := determineSearchFailureType(roll, params.successThreshold)
		isNearMiss = failureType == searchFailureNearMiss
		isCritFail = failureType == searchFailureCritical
		resultMessage = s.processSearchFailure(roll, params.successThreshold, params)
	}

	xpAmount := int(float64(job.ExplorerXPPerItem) * params.xpMultiplier)
	if xpAmount < 1 {
		xpAmount = 1
	}

	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventTypeSearchPerformed),
			Payload: domain.SearchPerformedPayload{
				UserID:         user.ID,
				Success:        isSuccess,
				IsCritical:     isCritical,
				IsNearMiss:     isNearMiss,
				IsCriticalFail: isCritFail,
				XPAmount:       xpAmount,
				ItemName:       itemName,
				Quantity:       quantity,
				Timestamp:      time.Now().Unix(),
			},
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

	if params.isFirstSearchDaily && s.statsService != nil {
		streak, err := s.statsService.GetUserCurrentStreak(ctx, user.ID)
		if err != nil {
			log.Warn("Failed to get user streak", "error", err)
		} else {
			params.streak = streak
		}
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
	qualityLevel := s.calculateSearchQuality(isCritical, params)
	if err := s.grantSearchReward(ctx, user, quantity, qualityLevel); err != nil {
		return "", err
	}

	// Get item for message formatting and event recording
	item, err := s.getItemByNameCached(ctx, domain.ItemLootbox0)
	if err != nil {
		return "", fmt.Errorf("failed to get reward item: %w", err)
	}

	// Format and return result message
	return s.formatSearchSuccessMessage(ctx, item, quantity, isCritical, params), nil
}

func (s *service) addItemToTx(ctx context.Context, tx repository.UserTx, userID string, itemID int, quantity int, qualityLevel domain.QualityLevel) error {
	log := logger.FromContext(ctx)
	inventory, err := tx.GetInventory(ctx, userID)
	if err != nil {
		log.Error("Failed to get inventory", "error", err, "userID", userID)
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	// Find slot with matching ItemID AND QualityLevel to prevent quality corruption
	i, _ := utils.FindSlotWithQuality(inventory, itemID, qualityLevel)
	if i != -1 {
		inventory.Slots[i].Quantity += quantity
	} else {
		inventory.Slots = append(inventory.Slots, domain.InventorySlot{
			ItemID:       itemID,
			Quantity:     quantity,
			QualityLevel: qualityLevel,
		})
	}

	if err := tx.UpdateInventory(ctx, userID, *inventory); err != nil {
		log.Error("Failed to update inventory", "error", err, "userID", userID)
		return fmt.Errorf("failed to update inventory: %w", err)
	}
	return nil
}

func (s *service) processSearchFailure(roll float64, successThreshold float64, params searchParams) string {
	// Determine failure type
	failureType := determineSearchFailureType(roll, successThreshold)

	// Format failure message
	resultMessage := formatSearchFailureMessage(failureType)

	// Append streak and exhausted status if applicable
	return s.formatSearchFailureMessageWithMeta(resultMessage, params)
}

// getUserOrRegister gets a user by platform ID, or auto-registers them if not found
func (s *service) getUserOrRegister(ctx context.Context, platform, platformID, username string) (*domain.User, error) {
	log := logger.FromContext(ctx)
	if username == "" || platform == "" || !validPlatforms[platform] {
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

// triggerTrap executes trap trigger logic when a user sends a message
func (s *service) triggerTrap(ctx context.Context, trap *domain.Trap, victim *domain.User) error {
	log := logger.FromContext(ctx)

	// 1. Mark trap as triggered
	if err := s.trapRepo.TriggerTrap(ctx, trap.ID); err != nil {
		return fmt.Errorf("failed to mark trap as triggered: %w", err)
	}

	// 2. Apply timeout
	timeout := time.Duration(trap.CalculateTimeout()) * time.Second
	if err := s.TimeoutUser(ctx, victim.Username, timeout, "BOOM! Stepped on a trap!"); err != nil {
		return fmt.Errorf("failed to timeout user: %w", err)
	}

	// 3. Remove from active chatters (prevent immediate re-targeting by grenades)
	s.activeChatterTracker.Remove(domain.PlatformTwitch, victim.ID)

	// 4. Publish event
	if s.statsService != nil {
		// Fetch setter info for event
		setter, err := s.repo.GetUserByID(ctx, trap.SetterID.String())
		if err != nil {
			log.Warn("Failed to get trap setter for event", "setter_id", trap.SetterID)
		} else {
			eventData := &domain.TrapTriggeredData{
				TrapID:           trap.ID,
				SetterID:         trap.SetterID,
				SetterUsername:   setter.Username,
				TargetID:         trap.TargetID,
				TargetUsername:   victim.Username,
				QualityLevel:     trap.QualityLevel,
				TimeoutSeconds:   trap.CalculateTimeout(),
				WasSelfTriggered: false,
			}
			_ = s.statsService.RecordUserEvent(ctx, victim.ID, domain.EventTrapTriggered, eventData.ToMap())
		}
	}

	log.Info(LogMsgTrapTriggered,
		"victim", victim.Username,
		"timeout", timeout.Seconds(),
		"trap_id", trap.ID)

	return nil
}

func (s *service) Shutdown(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info(LogMsgUserServiceShuttingDown)

	// 1. Stop the chatter tracker (stops cleanup loop)
	if s.activeChatterTracker != nil {
		s.activeChatterTracker.Stop()
	}

	// 2. Wait for local async tasks (like trap triggers)
	s.wg.Wait()

	// 3. Shut down the publisher (waits for pending events)
	if s.publisher != nil {
		if err := s.publisher.Shutdown(ctx); err != nil {
			log.Error("Failed to shut down publisher", "error", err)
		}
	}

	log.Info("User service shutdown complete")
	return nil
}

func (s *service) GetCacheStats() CacheStats {
	return s.userCache.GetStats()
}
func (s *service) GetActiveChatters() []ActiveChatter {
	chatters := s.activeChatterTracker.GetActiveChatters()
	result := make([]ActiveChatter, len(chatters))
	for i, c := range chatters {
		result[i] = ActiveChatter(c)
	}
	return result
}
