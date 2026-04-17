package itemhandler

import (
	"context"
	"fmt"
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

func HandleLootbox(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, lootboxItem *domain.Item, quantity int) (string, error) {
	// 1. Consume lootboxes
	consumedSlots, err := utils.ConsumeItemsWithTracking(inventory, lootboxItem.ID, quantity, ec.RandomFloat)
	if err != nil {
		if total := utils.GetTotalQuantity(inventory, lootboxItem.ID); total == 0 {
			return "", domain.ErrNotInInventory
		}
		return "", domain.ErrInsufficientQuantity
	}

	// 2. Open lootboxes
	allDrops := make([]lootbox.DroppedItem, 0, quantity)
	for _, slot := range consumedSlots {
		drops, err := ec.OpenLootbox(ctx, lootboxItem.InternalName, slot.Quantity, slot.QualityLevel)
		if err != nil {
			logger.FromContext(ctx).Error("Failed to open lootbox", "error", err, "lootbox", lootboxItem.InternalName)
			return "", fmt.Errorf("failed to open lootbox: %w", err)
		}
		allDrops = append(allDrops, drops...)
	}

	if len(allDrops) == 0 {
		return MsgLootboxEmpty, nil
	}

	return HandleLootboxDrops(ctx, ec, user, inventory, lootboxItem, quantity, allDrops)
}

func HandleLootboxDrops(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, lootboxItem *domain.Item, quantity int, drops []lootbox.DroppedItem) (string, error) {
	var stats dropStats
	itemsToAdd := make([]domain.InventorySlot, 0, len(drops))
	displayGroups := make(map[string]int)
	displayOrder := make([]string, 0)

	for _, drop := range drops {
		stats.totalValue += drop.Value
		if drop.QualityLevel == domain.QualityLegendary {
			stats.hasLegendary = true
		} else if drop.QualityLevel == domain.QualityEpic {
			stats.hasEpic = true
		}

		itemsToAdd = append(itemsToAdd, domain.InventorySlot{
			ItemID: drop.ItemID, Quantity: drop.Quantity, QualityLevel: drop.QualityLevel,
		})

		if count := displayGroups[drop.ItemName]; count == 0 {
			displayOrder = append(displayOrder, drop.ItemName)
		}
		displayGroups[drop.ItemName] += drop.Quantity
	}

	utils.AddItemsToInventory(inventory, itemsToAdd, nil)

	// Build message
	displayName := ec.GetDisplayName(lootboxItem.InternalName, "")
	boxPart := fmt.Sprintf("%d %s", quantity, ec.Pluralize(displayName, quantity))
	if quantity == 1 {
		boxPart = fmt.Sprintf("%s %s", getIndefiniteArticle(displayName), displayName)
	}

	dropStrings := make([]string, 0, len(displayOrder))
	for _, name := range displayOrder {
		dropStrings = append(dropStrings, fmt.Sprintf("%d %s", displayGroups[name], ec.Pluralize(name, displayGroups[name])))
	}

	msg := fmt.Sprintf("%s %s%s %s", MsgLootboxOpened, boxPart, MsgLootboxReceived, strings.Join(dropStrings, LootboxDropSeparator))

	// Handle events and special messages
	eventData := &domain.LootboxEventData{Item: lootboxItem.InternalName, Drops: drops, Value: stats.totalValue, Source: "lootbox"}
	if stats.hasLegendary {
		_ = ec.RecordUserEvent(ctx, user.ID, domain.EventTypeLootboxJackpot, eventData)
		msg += MsgLootboxJackpot
	} else if stats.hasEpic {
		_ = ec.RecordUserEvent(ctx, user.ID, domain.EventTypeLootboxBigWin, eventData)
		msg += MsgLootboxBigWin
	} else if stats.totalValue > 0 && quantity >= domain.BulkFeedbackThreshold {
		msg += MsgLootboxNiceHaul
	}

	return msg, nil
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
	return HandleLootbox(ctx, ec, user, inventory, item, quantity)
}
