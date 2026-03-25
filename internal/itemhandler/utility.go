package itemhandler

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// RareCandyHandler handles rare candy items.
type RareCandyHandler struct{}

// CanHandle returns true for rare candy items.
func (h *RareCandyHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemRareCandy ||
		strings.HasPrefix(itemName, "xp_")
}

// Handle processes rare candy usage.
func (h *RareCandyHandler) Handle(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	return handleRareCandy(ctx, ec, user, inventory, item, quantity, args)
}

// ResourceGeneratorHandler handles items that generate other items.
type ResourceGeneratorHandler struct{}

// CanHandle returns true for resource generator items.
func (h *ResourceGeneratorHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemShovel
}

// Handle processes resource generation.
func (h *ResourceGeneratorHandler) Handle(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	return handleResourceGenerator(ctx, ec, inventory, item, quantity, args)
}

// UtilityHandler handles miscellaneous items with simple effects.
type UtilityHandler struct{}

// CanHandle returns true for utility items.
func (h *UtilityHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemStick
}

// Handle processes utility item usage.
func (h *UtilityHandler) Handle(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	return handleUtility(ctx, ec, inventory, item, quantity, args)
}

// VideoFilterHandler handles video filter items.
type VideoFilterHandler struct{}

// CanHandle returns true for video filter items.
func (h *VideoFilterHandler) CanHandle(itemName string) bool {
	return itemName == domain.ItemVideoFilter
}

// Handle processes video filter application.
func (h *VideoFilterHandler) Handle(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	return handleVideoFilter(ctx, ec, user, inventory, item, quantity, args)
}
