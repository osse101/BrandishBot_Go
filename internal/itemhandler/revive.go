package itemhandler

import (
	"context"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// getReviveRecovery returns the recovery duration for a revive item.
func getReviveRecovery(itemName string) time.Duration {
	if recovery, ok := reviveRecoveryTimes[itemName]; ok {
		return recovery
	}
	return 60 * time.Second // default fallback
}

func handleRevive(ctx context.Context, ec EffectContext, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgHandleReviveCalled, "item", item.InternalName, "quantity", quantity)
	targetUsername := args.TargetUsername
	if targetUsername == "" {
		log.Warn(LogWarnTargetUsernameMissingRevive)
		return "", fmt.Errorf("%w: target username is required for revive", domain.ErrInvalidInput)
	}
	username := args.Username

	// Find total availability
	totalAvailable := utils.GetTotalQuantity(inventory, item.ID)
	if totalAvailable == 0 {
		log.Warn(LogWarnReviveNotInInventory, "item", item.InternalName)
		return "", domain.ErrNotInInventory
	}
	if totalAvailable < quantity {
		log.Warn(LogWarnNotEnoughRevives, "item", item.InternalName)
		return "", domain.ErrInsufficientQuantity
	}

	consumedSlots, err := utils.ConsumeItemsWithTracking(inventory, item.ID, quantity, ec.RandomFloat)
	if err != nil {
		return "", err
	}

	// Get recovery time for this revive type
	var totalRecovery time.Duration
	var displayName string
	for i, slot := range consumedSlots {
		recovery := getReviveRecovery(item.InternalName) + slot.QualityLevel.GetTimeoutAdjustment()
		totalRecovery += time.Duration(slot.Quantity) * recovery
		if i == 0 {
			displayName = ec.GetDisplayName(item.InternalName, slot.QualityLevel)
		}
	}

	// Reduce timeout for target user
	if err := ec.ReduceTimeout(ctx, targetUsername, totalRecovery); err != nil {
		log.Error(LogWarnFailedToReduceTimeout, "error", err, "target", targetUsername)
		// Continue anyway, as the item was used
	}

	log.Info(LogMsgReviveUsed, "target", targetUsername, "item", item.InternalName, "quantity", quantity)
	return fmt.Sprintf("%s used %d %s on %s! Reduced timeout by %v.", username, quantity, displayName, targetUsername, totalRecovery), nil
}
