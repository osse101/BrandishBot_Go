package user

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

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

// addItemToTx adds an item to an inventory within a transaction
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
