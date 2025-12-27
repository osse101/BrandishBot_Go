package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// Item effect handlers

func (s *service) processLootbox(ctx context.Context, inventory *domain.Inventory, lootboxItem *domain.Item, quantity int) (string, error) {
	log := logger.FromContext(ctx)

	// 1. Validate and consume lootboxes
	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, lootboxItem.ID)
	if itemSlotIndex == -1 {
		return "", fmt.Errorf("item not found in inventory")
	}

	if slotQuantity < quantity {
		return "", fmt.Errorf("not enough items in inventory")
	}

	if inventory.Slots[itemSlotIndex].Quantity == quantity {
		inventory.Slots = append(inventory.Slots[:itemSlotIndex], inventory.Slots[itemSlotIndex+1:]...)
	} else {
		inventory.Slots[itemSlotIndex].Quantity -= quantity
	}

	// 2. Use lootbox service to open lootboxes
	drops, err := s.lootboxService.OpenLootbox(ctx, lootboxItem.InternalName, quantity)
	if err != nil {
		log.Error("Failed to open lootbox", "error", err, "lootbox", lootboxItem.InternalName)
		return "", fmt.Errorf("failed to open lootbox: %w", err)
	}

	if len(drops) == 0 {
		return "The lootbox was empty!", nil
	}

	// 3. Add drops to inventory and build message with shine feedback
	var msgBuilder strings.Builder
	displayName := s.namingResolver.GetDisplayName(lootboxItem.InternalName, "")
	msgBuilder.WriteString(fmt.Sprintf("Opened %d %s and received: ", quantity, displayName))

	// Optimization: Build a map for O(1) inventory lookups instead of O(N) scan per drop
	// This reduces complexity from O(N*M) to O(N+M) where N=inventory size, M=drops
	slotMap := make(map[int]int, len(inventory.Slots))
	for i, slot := range inventory.Slots {
		slotMap[slot.ItemID] = i
	}

	first := true
	for _, drop := range drops {
		// Add to inventory
		if idx, exists := slotMap[drop.ItemID]; exists {
			inventory.Slots[idx].Quantity += drop.Quantity
		} else {
			inventory.Slots = append(inventory.Slots, domain.InventorySlot{ItemID: drop.ItemID, Quantity: drop.Quantity})
			slotMap[drop.ItemID] = len(inventory.Slots) - 1
		}

		if !first {
			msgBuilder.WriteString(", ")
		}

		// Get display name with shine level
		itemDisplayName := s.namingResolver.GetDisplayName(drop.ItemName, drop.ShineLevel)

		// Add shine annotation for visual impact
		shineAnnotation := ""
		if drop.ShineLevel != "" && drop.ShineLevel != "COMMON" {
			shineAnnotation = fmt.Sprintf(" [%s!]", drop.ShineLevel)
		}

		msgBuilder.WriteString(fmt.Sprintf("%dx %s%s", drop.Quantity, itemDisplayName, shineAnnotation))
		first = false
	}

	return msgBuilder.String(), nil
}


func (s *service) handleLootbox1(ctx context.Context, _ *service, _ *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, _ map[string]interface{}) (string, error) {
	return s.processLootbox(ctx, inventory, item, quantity)
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
	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
	if itemSlotIndex == -1 {
		log.Warn("blaster not in inventory")
		return "", fmt.Errorf("item not found in inventory")
	}
	if slotQuantity < quantity {
		log.Warn("not enough blasters in inventory")
		return "", fmt.Errorf("not enough items in inventory")
	}
	if inventory.Slots[itemSlotIndex].Quantity == quantity {
		inventory.Slots = append(inventory.Slots[:itemSlotIndex], inventory.Slots[itemSlotIndex+1:]...)
	} else {
		inventory.Slots[itemSlotIndex].Quantity -= quantity
	}

	// Apply timeout
	timeoutDuration := 60 * time.Second
	if err := s.TimeoutUser(ctx, targetUsername, timeoutDuration, "Blasted by "+username); err != nil {
		log.Error("Failed to timeout user", "error", err, "target", targetUsername)
		// Continue anyway, as the item was used
	}

	log.Info("blaster used", "target", targetUsername, "quantity", quantity)
	return fmt.Sprintf("%s has BLASTED %s %d times! They are timed out for %v.", username, targetUsername, quantity, timeoutDuration), nil
}

func (s *service) handleLootbox0(ctx context.Context, _ *service, _ *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, _ map[string]interface{}) (string, error) {
	return s.processLootbox(ctx, inventory, item, quantity)
}

func (s *service) handleLootbox2(ctx context.Context, _ *service, _ *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, _ map[string]interface{}) (string, error) {
	return s.processLootbox(ctx, inventory, item, quantity)
}
