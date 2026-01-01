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

	// Optimization: Use linear scan for small number of drops to avoid map allocation overhead
	// Benchmarks show linear scan is faster for small M (drops) even with large N (inventory)
	// Map overhead ~30µs vs Linear ~2µs for M=5, N=1000
	const linearScanThreshold = 10
	useMap := len(drops) >= linearScanThreshold

	var slotMap map[int]int
	if useMap {
		slotMap = make(map[int]int, len(inventory.Slots))
		for i, slot := range inventory.Slots {
			slotMap[slot.ItemID] = i
		}
	}

	first := true
	for _, drop := range drops {
		// Track stats for feedback
		stats.totalValue += drop.Value
		if drop.ShineLevel == lootbox.ShineLegendary {
			stats.hasLegendary = true
		} else if drop.ShineLevel == lootbox.ShineEpic {
			stats.hasEpic = true
		}

		// Add to inventory
		if useMap {
			if idx, exists := slotMap[drop.ItemID]; exists {
				inventory.Slots[idx].Quantity += drop.Quantity
			} else {
				inventory.Slots = append(inventory.Slots, domain.InventorySlot{ItemID: drop.ItemID, Quantity: drop.Quantity})
				slotMap[drop.ItemID] = len(inventory.Slots) - 1
			}
		} else {
			// Linear scan
			found := false
			for i := range inventory.Slots {
				if inventory.Slots[i].ItemID == drop.ItemID {
					inventory.Slots[i].Quantity += drop.Quantity
					found = true
					break
				}
			}
			if !found {
				inventory.Slots = append(inventory.Slots, domain.InventorySlot{ItemID: drop.ItemID, Quantity: drop.Quantity})
			}
		}

		if !first {
			msgBuilder.WriteString(", ")
		}

		// Get display name with shine level
		itemDisplayName := s.namingResolver.GetDisplayName(drop.ItemName, drop.ShineLevel)

		// Add shine annotation for visual impact
		shineAnnotation := ""
		if drop.ShineLevel != "" && drop.ShineLevel != lootbox.ShineCommon {
			shineAnnotation = fmt.Sprintf(" [%s!]", drop.ShineLevel)
		}

		msgBuilder.WriteString(fmt.Sprintf("%dx %s%s", drop.Quantity, itemDisplayName, shineAnnotation))
		first = false
	}

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
