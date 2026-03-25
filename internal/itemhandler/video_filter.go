package itemhandler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

func handleVideoFilter(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("handleVideoFilter called", "item", item.InternalName, "quantity", quantity)

	filterKey := strings.ToLower(strings.TrimSpace(args.TargetUsername))
	if filterKey == "" {
		return "", errors.New("must specify a video filter to use! Valid filters: " + validVideoFiltersList)
	}

	if !strings.Contains(validVideoFiltersList, filterKey) {
		return "", fmt.Errorf("invalid video filter '%s'. Valid filters: %s", filterKey, validVideoFiltersList)
	}

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

	displayName := ec.GetDisplayName(item.InternalName, "")
	return fmt.Sprintf("%s applied the %s %s!", user.Username, filterKey, displayName), nil
}
