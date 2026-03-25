package itemhandler

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

func handleResourceGenerator(ctx context.Context, ec EffectContext, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgResourceGeneratorCalled, "item", item.InternalName, "quantity", quantity)

	username := args.Username

	// Find total availability
	totalAvailable := utils.GetTotalQuantity(inventory, item.ID)
	if totalAvailable == 0 {
		return "", domain.ErrNotInInventory
	}
	if totalAvailable < quantity {
		return "", domain.ErrInsufficientQuantity
	}
	if err := utils.ConsumeItems(inventory, item.ID, quantity, ec.RandomFloat); err != nil {
		return "", err
	}

	// Generate sticks (shovel generates 2 sticks per use)
	stickItem, err := ec.GetItemByName(ctx, domain.ItemStick)
	if err != nil {
		return "", fmt.Errorf("failed to get stick item: %w", err)
	}

	sticksGenerated := quantity * domain.ShovelSticksPerUse
	utils.AddItemsToInventory(inventory, []domain.InventorySlot{
		{ItemID: stickItem.ID, Quantity: sticksGenerated, QualityLevel: domain.QualityCommon},
	}, nil)

	displayName := ec.GetDisplayName(domain.ItemStick, "")
	return fmt.Sprintf("%s%d %s!", username+MsgShovelUsed, sticksGenerated, displayName), nil
}
