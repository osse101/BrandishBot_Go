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

// LootItem defines an item that can be dropped from a lootbox
// LootItem defines an item that can be dropped from a lootbox
type LootItem struct {
	ItemName string  `json:"item_name"`
	Min      int     `json:"min"`
	Max      int     `json:"max"`
	Chance   float64 `json:"chance"`
}

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

	// 2. Process drops
	table, ok := s.lootTables[lootboxItem.InternalName]
	if !ok {
		log.Warn("No loot table found for item", "item", lootboxItem.InternalName)
		return "Lootbox opened but it was empty (no loot table)!", nil
	}

	drops := make(map[string]int)

	// Seed random source if not already done globally (Go 1.20+ seeds automatically, but good to be safe if older)
	// assuming global rand is seeded or we use a local source.
	// For simplicity using global math/rand here, assuming it's seeded in main.

	for i := 0; i < quantity; i++ {
		for _, loot := range table {
			if utils.SecureRandomFloat() <= loot.Chance {
				qty := loot.Min
				if loot.Max > loot.Min {
					qty = utils.SecureRandomIntRange(loot.Min, loot.Max)
				}
				drops[loot.ItemName] += qty
			}
		}
	}

	if len(drops) == 0 {
		return "The lootbox was empty!", nil
	}

	// 3. Add drops to inventory and build message
	var msgBuilder strings.Builder
	msgBuilder.WriteString(fmt.Sprintf("Opened %d %s and received: ", quantity, lootboxItem.InternalName))

	first := true
	for itemName, qty := range drops {
		item, err := s.repo.GetItemByName(ctx, itemName)
		if err != nil {
			log.Error("Failed to get dropped item", "item", itemName, "error", err)
			continue
		}
		if item == nil {
			log.Warn("Dropped item not found in DB", "item", itemName)
			continue
		}

		// Add to inventory
		i, _ := utils.FindSlot(inventory, item.ID)
		if i != -1 {
			inventory.Slots[i].Quantity += qty
		} else {
			inventory.Slots = append(inventory.Slots, domain.InventorySlot{ItemID: item.ID, Quantity: qty})
		}

		if !first {
			msgBuilder.WriteString(", ")
		}
		msgBuilder.WriteString(fmt.Sprintf("%dx %s", qty, item.InternalName))
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
