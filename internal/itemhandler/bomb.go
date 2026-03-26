package itemhandler

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

func handleBomb(ctx context.Context, ec EffectContext, _ *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("handleBomb called", "username", args.Username)

	platform := args.Platform
	username := args.Username

	totalAvailable := utils.GetTotalQuantity(inventory, item.ID)
	if totalAvailable == 0 {
		return "", domain.ErrNotInInventory
	}
	consumedSlots, err := utils.ConsumeItemsWithTracking(inventory, item.ID, quantity, ec.RandomFloat)
	if err != nil {
		return "", err
	}

	// 2. Queue each bomb (handling multiple quantities)
	var lastDisplayName string
	totalQueued := 0
	for _, slot := range consumedSlots {
		baseTimeout := getWeaponTimeout(item.InternalName) + slot.QualityLevel.GetTimeoutAdjustment()
		displayName := ec.GetDisplayName(item.InternalName, slot.QualityLevel)
		lastDisplayName = displayName

		// A single slot might contain multiple items of the same quality
		for i := 0; i < slot.Quantity; i++ {
			if err := ec.SetPendingBomb(ctx, platform, username, baseTimeout); err != nil {
				log.Error("Failed to set pending bomb", "error", err, "index", i)
			} else {
				totalQueued++
			}
		}
	}

	if totalQueued == 0 {
		return "", fmt.Errorf("failed to queue any bombs: %w", domain.ErrInternalError)
	}

	log.Info("Bombs set successfully", "setter", username, "platform", platform, "count", totalQueued)
	return fmt.Sprintf("%s set %s! Waiting for a crowd...", username, lastDisplayName), nil
}

// BombHandler handles bomb items.
type BombHandler struct{}

// CanHandle returns true for bomb items.
func (h *BombHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemBomb
}

// Handle processes bomb usage.
func (h *BombHandler) Handle(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	return handleBomb(ctx, ec, user, inventory, item, quantity, args)
}
