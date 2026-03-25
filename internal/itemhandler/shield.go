package itemhandler

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

func handleShield(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgHandleShieldCalled, "item", item.InternalName, "quantity", quantity)

	// Find total availability
	totalAvailable := utils.GetTotalQuantity(inventory, item.ID)
	if totalAvailable == 0 {
		log.Warn(LogWarnShieldNotInInventory)
		return "", domain.ErrNotInInventory
	}
	if totalAvailable < quantity {
		log.Warn(LogWarnNotEnoughShields)
		return "", domain.ErrInsufficientQuantity
	}
	if err := utils.ConsumeItems(inventory, item.ID, quantity, ec.RandomFloat); err != nil {
		return "", err
	}

	// Determine if this is a mirror shield
	isMirror := item.InternalName == domain.ItemMirrorShield

	// Apply shield status to user
	if err := ec.ApplyShield(ctx, user, quantity, isMirror); err != nil {
		log.Error(LogWarnFailedToApplyShield, "error", err)
		return "", fmt.Errorf("%w: failed to apply shield", domain.ErrInvalidInput)
	}

	displayName := ec.GetDisplayName(item.InternalName, "")
	log.Info(LogMsgShieldApplied, "item", item.InternalName, "quantity", quantity, "is_mirror", isMirror)

	if isMirror {
		return fmt.Sprintf("Activated %d %s! Next %d attacks will be REFLECTED!", quantity, displayName, quantity), nil
	}
	return fmt.Sprintf("Activated %d %s! Protected from next %d attacks.", quantity, displayName, quantity), nil
}
