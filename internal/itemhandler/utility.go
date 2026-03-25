package itemhandler

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

func handleUtility(ctx context.Context, ec EffectContext, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgUtilityCalled, "item", item.InternalName, "quantity", quantity)

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

	return username + MsgStickUsed, nil
}
