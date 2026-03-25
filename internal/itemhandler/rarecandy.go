package itemhandler

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

func handleRareCandy(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgHandleRareCandyCalled, "quantity", quantity)

	jobName := args.JobName
	if jobName == "" {
		log.Warn(LogWarnJobNameMissing)
		return "", fmt.Errorf("%w: job name is required for rare candy", domain.ErrInvalidInput)
	}

	// Find total availability
	totalAvailable := utils.GetTotalQuantity(inventory, item.ID)
	if totalAvailable == 0 {
		log.Warn(LogWarnRareCandyNotInInventory)
		return "", domain.ErrNotInInventory
	}
	if totalAvailable < quantity {
		log.Warn(LogWarnNotEnoughRareCandy)
		return "", domain.ErrInsufficientQuantity
	}
	if err := utils.ConsumeItems(inventory, item.ID, quantity, ec.RandomFloat); err != nil {
		return "", err
	}

	// Award XP to the specified job via event
	totalXP := quantity * rarecandyXPAmount
	ec.PublishItemUsedEvent(ctx, user.ID, item.InternalName, quantity, map[string]interface{}{
		"job_name": jobName,
		"xp_total": totalXP,
		"source":   job.SourceRareCandy,
	})

	log.Info(LogMsgRareCandyUsed, "job", jobName, "xp", totalXP, "quantity", quantity)
	return fmt.Sprintf("Used %d rare candy! Granted %d XP to %s.", quantity, totalXP, jobName), nil
}
