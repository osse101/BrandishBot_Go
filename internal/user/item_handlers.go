package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// BulkFeedbackThreshold defines the number of lootboxes required to trigger "Nice haul" message
const BulkFeedbackThreshold = 5

// Item effect handlers

func (s *service) processLootbox(ctx context.Context, user *domain.User, inventory *domain.Inventory, lootboxItem *domain.Item, quantity int) (string, error) {
	log := logger.FromContext(ctx)

	// 1. Validate and consume lootboxes
	if err := s.consumeLootboxFromInventory(inventory, lootboxItem, quantity); err != nil {
		return "", err
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

	// 3. Process drops and generate feedback
	return s.processLootboxDrops(ctx, user, inventory, lootboxItem, quantity, drops)
}

func (s *service) consumeLootboxFromInventory(inventory *domain.Inventory, item *domain.Item, quantity int) error {
	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
	if itemSlotIndex == -1 {
		return fmt.Errorf("item not found in inventory")
	}

	if slotQuantity < quantity {
		return fmt.Errorf("not enough items in inventory")
	}

	if inventory.Slots[itemSlotIndex].Quantity == quantity {
		inventory.Slots = append(inventory.Slots[:itemSlotIndex], inventory.Slots[itemSlotIndex+1:]...)
	} else {
		inventory.Slots[itemSlotIndex].Quantity -= quantity
	}
	return nil
}

func (s *service) processLootboxDrops(ctx context.Context, user *domain.User, inventory *domain.Inventory, lootboxItem *domain.Item, quantity int, drops []lootbox.DroppedItem) (string, error) {
	var msgBuilder strings.Builder
	displayName := s.namingResolver.GetDisplayName(lootboxItem.InternalName, "")
	msgBuilder.WriteString(fmt.Sprintf("Opened %d %s and received: ", quantity, displayName))

	stats := s.aggregateDropsAndUpdateInventory(inventory, drops, &msgBuilder)

	// 4. Append "Juice" - Feedback based on results
	// LevelUp Philosophy: "If a number goes up, the player should feel it."
	msgBuilder.WriteString(fmt.Sprintf(" (Value: %d)", stats.totalValue))

	if stats.hasLegendary {
		if s.statsService != nil && user != nil {
			eventData := &domain.LootboxEventData{
				Item:   lootboxItem.InternalName,
				Drops:  drops,
				Value:  stats.totalValue,
				Source: "lootbox",
			}
			if err := s.statsService.RecordUserEvent(ctx, user.ID, domain.EventLootboxJackpot, eventData.ToMap()); err != nil {
				log := logger.FromContext(ctx)
				log.Warn("Failed to record lootbox jackpot event", "error", err, "user_id", user.ID)
			}
		}
		msgBuilder.WriteString(" JACKPOT! 🎰✨")
	} else if stats.hasEpic {
		if s.statsService != nil && user != nil {
			eventData := &domain.LootboxEventData{
				Item:   lootboxItem.InternalName,
				Drops:  drops,
				Value:  stats.totalValue,
				Source: "lootbox",
			}
			if err := s.statsService.RecordUserEvent(ctx, user.ID, domain.EventLootboxBigWin, eventData.ToMap()); err != nil {
				log := logger.FromContext(ctx)
				log.Warn("Failed to record lootbox big-win event", "error", err, "user_id", user.ID)
			}
		}
		msgBuilder.WriteString(" BIG WIN! 💰")
	} else if stats.totalValue > 0 && quantity >= BulkFeedbackThreshold {
		// If opening many boxes and getting nothing special, at least acknowledge the haul
		msgBuilder.WriteString(" Nice haul! 📦")
	}

	return msgBuilder.String(), nil
}

type dropStats struct {
	totalValue   int
	hasLegendary bool
	hasEpic      bool
}

func (s *service) aggregateDropsAndUpdateInventory(inventory *domain.Inventory, drops []lootbox.DroppedItem, msgBuilder *strings.Builder) dropStats {
	var stats dropStats

	// Convert drops to inventory slots for batch adding
	itemsToAdd := make([]domain.InventorySlot, 0, len(drops))

	first := true
	for _, drop := range drops {
		// Track stats for feedback
		stats.totalValue += drop.Value
		if drop.ShineLevel == lootbox.ShineLegendary {
			stats.hasLegendary = true
		} else if drop.ShineLevel == lootbox.ShineEpic {
			stats.hasEpic = true
		}

		// Prepare item for batch add
		itemsToAdd = append(itemsToAdd, domain.InventorySlot{
			ItemID:   drop.ItemID,
			Quantity: drop.Quantity,
		})

		if !first {
			msgBuilder.WriteString(", ")
		}

		// Get display name with shine level
		itemDisplayName := s.namingResolver.GetDisplayName(drop.ItemName, drop.ShineLevel)

		// Write drop info directly to builder to minimize allocations
		fmt.Fprintf(msgBuilder, "%dx %s", drop.Quantity, itemDisplayName)

		// Add shine annotation for visual impact
		if drop.ShineLevel != "" && drop.ShineLevel != lootbox.ShineCommon {
			fmt.Fprintf(msgBuilder, " [%s!]", drop.ShineLevel)
		}

		first = false
	}

	// Add all items to inventory using optimized helper
	utils.AddItemsToInventory(inventory, itemsToAdd, nil)

	return stats
}

func (s *service) handleLootbox1(ctx context.Context, _ *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, _ map[string]interface{}) (string, error) {
	return s.processLootbox(ctx, user, inventory, item, quantity)
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

func (s *service) handleLootbox0(ctx context.Context, _ *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, _ map[string]interface{}) (string, error) {
	return s.processLootbox(ctx, user, inventory, item, quantity)
}

func (s *service) handleLootbox2(ctx context.Context, _ *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, _ map[string]interface{}) (string, error) {
	return s.processLootbox(ctx, user, inventory, item, quantity)
}
