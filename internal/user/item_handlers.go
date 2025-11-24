package user

import (
	"context"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

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
	if err := s.TimeoutUser(ctx, targetUsername, timeoutDuration, "Blasted by "+username); err != nil {
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
