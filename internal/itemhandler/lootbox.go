package itemhandler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

type dropStats struct {
	totalValue   int
	hasLegendary bool
	hasEpic      bool
}

func ProcessLootbox(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, lootboxItem *domain.Item, quantity int) (string, error) {
	log := logger.FromContext(ctx)

	// 1. Validate and consume lootboxes
	totalAvailable := utils.GetTotalQuantity(inventory, lootboxItem.ID)
	if totalAvailable == 0 {
		return "", domain.ErrNotInInventory
	}
	if totalAvailable < quantity {
		return "", domain.ErrInsufficientQuantity
	}

	consumedSlots, err := utils.ConsumeItemsWithTracking(inventory, lootboxItem.ID, quantity, ec.RandomFloat)
	if err != nil {
		return "", err
	}

	// 2. Use lootbox service to open lootboxes
	var allDrops []lootbox.DroppedItem
	for _, slot := range consumedSlots {
		drops, err := ec.OpenLootbox(ctx, lootboxItem.InternalName, slot.Quantity, slot.QualityLevel)
		if err != nil {
			log.Error("Failed to open lootbox", "error", err, "lootbox", lootboxItem.InternalName)
			return "", fmt.Errorf("failed to open lootbox: %w", err)
		}
		allDrops = append(allDrops, drops...)
	}

	if len(allDrops) == 0 {
		return MsgLootboxEmpty, nil
	}

	// 3. Process drops and generate feedback
	return ProcessLootboxDrops(ctx, ec, user, inventory, lootboxItem, quantity, allDrops)
}

func ProcessLootboxDrops(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, lootboxItem *domain.Item, quantity int, drops []lootbox.DroppedItem) (string, error) {
	var msgBuilder strings.Builder
	// Use alias for the lootbox when opening
	displayName := ec.GetDisplayName(lootboxItem.InternalName, "")

	msgBuilder.WriteString(MsgLootboxOpened)
	msgBuilder.WriteString(" ")
	if quantity > 1 {
		msgBuilder.WriteString(strconv.Itoa(quantity))
		msgBuilder.WriteString(" ")
		msgBuilder.WriteString(ec.Pluralize(displayName, quantity))
	} else {
		msgBuilder.WriteString(getIndefiniteArticle(displayName))
		msgBuilder.WriteString(" ")
		msgBuilder.WriteString(displayName)
	}
	msgBuilder.WriteString(MsgLootboxReceived)

	stats := aggregateDropsAndUpdateInventory(ec, inventory, drops, &msgBuilder)

	if stats.hasLegendary {
		_ = ec.RecordUserEvent(ctx, user.ID, domain.EventTypeLootboxJackpot, &domain.LootboxEventData{
			Item:   lootboxItem.InternalName,
			Drops:  drops,
			Value:  stats.totalValue,
			Source: "lootbox",
		})
		msgBuilder.WriteString(MsgLootboxJackpot)
	} else if stats.hasEpic {
		_ = ec.RecordUserEvent(ctx, user.ID, domain.EventTypeLootboxBigWin, &domain.LootboxEventData{
			Item:   lootboxItem.InternalName,
			Drops:  drops,
			Value:  stats.totalValue,
			Source: "lootbox",
		})
		msgBuilder.WriteString(MsgLootboxBigWin)
	} else if stats.totalValue > 0 && quantity >= domain.BulkFeedbackThreshold {
		msgBuilder.WriteString(MsgLootboxNiceHaul)
	}

	return msgBuilder.String(), nil
}

func aggregateDropsAndUpdateInventory(ec EffectContext, inventory *domain.Inventory, drops []lootbox.DroppedItem, msgBuilder *strings.Builder) dropStats {
	var stats dropStats

	// Convert drops to inventory slots for batch adding
	itemsToAdd := make([]domain.InventorySlot, 0, len(drops))

	// Group items by their resolved display name (which includes quality where applicable)
	type dropGroup struct {
		Quantity int
		Name     string
	}
	displayGroups := make(map[string]*dropGroup)
	var displayOrder []string

	for _, drop := range drops {
		// Track stats for feedback
		stats.totalValue += drop.Value
		if drop.QualityLevel == domain.QualityLegendary {
			stats.hasLegendary = true
		} else if drop.QualityLevel == domain.QualityEpic {
			stats.hasEpic = true
		}

		// Prepare item for batch add - preserve quality level from loot table
		itemsToAdd = append(itemsToAdd, domain.InventorySlot{
			ItemID:       drop.ItemID,
			Quantity:     drop.Quantity,
			QualityLevel: drop.QualityLevel,
		})

		// Get display name
		itemDisplayName := ec.GetDisplayName(drop.ItemName, drop.QualityLevel)

		if group, exists := displayGroups[itemDisplayName]; exists {
			group.Quantity += drop.Quantity
		} else {
			displayOrder = append(displayOrder, itemDisplayName)
			displayGroups[itemDisplayName] = &dropGroup{
				Quantity: drop.Quantity,
				Name:     itemDisplayName,
			}
		}
	}

	// Format output with grouped items
	first := true
	for _, displayName := range displayOrder {
		group := displayGroups[displayName]

		if !first {
			msgBuilder.WriteString(LootboxDropSeparator)
		}

		// Simplify output: "Quantity Name"
		msgBuilder.WriteString(strconv.Itoa(group.Quantity))
		msgBuilder.WriteString(" ")
		msgBuilder.WriteString(ec.Pluralize(group.Name, group.Quantity))

		first = false
	}

	// Add all items to inventory using optimized helper
	utils.AddItemsToInventory(inventory, itemsToAdd, nil)

	return stats
}

// LootboxHandler handles all lootbox tiers.
type LootboxHandler struct{}

// CanHandle returns true for any lootbox item.
func (h *LootboxHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemLootbox0 ||
		itemName == domain.ItemLootbox1 ||
		itemName == domain.ItemLootbox2 ||
		itemName == domain.ItemLootbox3 ||
		strings.HasPrefix(itemName, "lootbox_tier")
}

// Handle processes lootbox opening.
func (h *LootboxHandler) Handle(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	return ProcessLootbox(ctx, ec, user, inventory, item, quantity)
}
