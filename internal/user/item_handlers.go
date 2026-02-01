package user

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// Item effect handlers

func (s *service) processLootbox(ctx context.Context, user *domain.User, inventory *domain.Inventory, lootboxItem *domain.Item, quantity int) (string, error) {
	log := logger.FromContext(ctx)

	// 1. Validate and consume lootboxes
	if err := s.consumeLootboxFromInventory(inventory, lootboxItem, quantity); err != nil {
		return "", err
	}

	// 2. Use lootbox service to open lootboxes
	drops, err := s.lootboxService.OpenLootbox(ctx, lootboxItem.InternalName, quantity)
	if err != nil {
		log.Error("Failed to open lootbox", "error", err, "lootbox", lootboxItem.InternalName)
		return "", fmt.Errorf("failed to open lootbox: %w", err)
	}

	if len(drops) == 0 {
		return MsgLootboxEmpty, nil
	}

	// 3. Process drops and generate feedback
	return s.processLootboxDrops(ctx, user, inventory, lootboxItem, quantity, drops)
}

func (s *service) consumeLootboxFromInventory(inventory *domain.Inventory, item *domain.Item, quantity int) error {
	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
	if itemSlotIndex == -1 {
		return fmt.Errorf(ErrMsgItemNotFoundInInventory)
	}

	if slotQuantity < quantity {
		return fmt.Errorf(ErrMsgNotEnoughItemsInInventory)
	}

	utils.RemoveFromSlot(inventory, itemSlotIndex, quantity)
	return nil
}

func (s *service) processLootboxDrops(ctx context.Context, user *domain.User, inventory *domain.Inventory, lootboxItem *domain.Item, quantity int, drops []lootbox.DroppedItem) (string, error) {
	var msgBuilder strings.Builder
	displayName := s.namingResolver.GetDisplayName(lootboxItem.InternalName, "")

	msgBuilder.WriteString(MsgLootboxOpened)
	msgBuilder.WriteString(" ")
	msgBuilder.WriteString(strconv.Itoa(quantity))
	msgBuilder.WriteString(" ")
	msgBuilder.WriteString(displayName)
	msgBuilder.WriteString(MsgLootboxReceived)

	stats := s.aggregateDropsAndUpdateInventory(inventory, drops, &msgBuilder)

	// 4. Append "Juice" - Feedback based on results
	// LevelUp Philosophy: "If a number goes up, the player should feel it."
	msgBuilder.WriteString(MsgLootboxValue)
	msgBuilder.WriteString(strconv.Itoa(stats.totalValue))
	msgBuilder.WriteString(MsgLootboxValueEnd)

	if stats.hasLegendary {
		if s.statsService != nil && user != nil {
			eventData := &domain.LootboxEventData{
				Item:   lootboxItem.InternalName,
				Drops:  drops,
				Value:  stats.totalValue,
				Source: "lootbox",
			}
			if err := s.statsService.RecordUserEvent(ctx, user.ID, domain.EventLootboxJackpot, eventData.ToMap()); err != nil {
				log := logger.FromContext(ctx)
				log.Warn(LogWarnFailedToRecordLootboxJackpot, "error", err, "user_id", user.ID)
			}
		}
		msgBuilder.WriteString(MsgLootboxJackpot)
	} else if stats.hasEpic {
		if s.statsService != nil && user != nil {
			eventData := &domain.LootboxEventData{
				Item:   lootboxItem.InternalName,
				Drops:  drops,
				Value:  stats.totalValue,
				Source: "lootbox",
			}
			if err := s.statsService.RecordUserEvent(ctx, user.ID, domain.EventLootboxBigWin, eventData.ToMap()); err != nil {
				log := logger.FromContext(ctx)
				log.Warn(LogWarnFailedToRecordLootboxBigWin, "error", err, "user_id", user.ID)
			}
		}
		msgBuilder.WriteString(MsgLootboxBigWin)
	} else if stats.totalValue > 0 && quantity >= BulkFeedbackThreshold {
		// If opening many boxes and getting nothing special, at least acknowledge the haul
		msgBuilder.WriteString(MsgLootboxNiceHaul)
	}

	return msgBuilder.String(), nil
}

type dropStats struct {
	totalValue   int
	hasLegendary bool
	hasEpic      bool
}

func (s *service) aggregateDropsAndUpdateInventory(inventory *domain.Inventory, drops []lootbox.DroppedItem, msgBuilder *strings.Builder) dropStats {
	var stats dropStats

	// Convert drops to inventory slots for batch adding
	itemsToAdd := make([]domain.InventorySlot, 0, len(drops))

	first := true
	for _, drop := range drops {
		// Track stats for feedback
		stats.totalValue += drop.Value
		if drop.ShineLevel == lootbox.ShineLegendary {
			stats.hasLegendary = true
		} else if drop.ShineLevel == lootbox.ShineEpic {
			stats.hasEpic = true
		}

		// Prepare item for batch add
		itemsToAdd = append(itemsToAdd, domain.InventorySlot{
			ItemID:   drop.ItemID,
			Quantity: drop.Quantity,
		})

		if !first {
			msgBuilder.WriteString(LootboxDropSeparator)
		}

		// Get display name with shine level
		itemDisplayName := s.namingResolver.GetDisplayName(drop.ItemName, drop.ShineLevel)

		// Write drop info directly to builder to minimize allocations
		msgBuilder.WriteString(strconv.Itoa(drop.Quantity))
		msgBuilder.WriteString(LootboxDisplayQuantityPrefix)
		msgBuilder.WriteString(itemDisplayName)

		// Add shine annotation for visual impact
		if drop.ShineLevel != "" && drop.ShineLevel != lootbox.ShineCommon {
			msgBuilder.WriteString(LootboxShineAnnotationOpen)
			msgBuilder.WriteString(drop.ShineLevel)
			msgBuilder.WriteString(LootboxShineAnnotationClose)
		}

		first = false
	}

	// Add all items to inventory using optimized helper
	utils.AddItemsToInventory(inventory, itemsToAdd, nil)

	return stats
}

// handleLootboxGeneric is a unified handler for all lootbox tiers.
// The lootbox type is determined by the item parameter.
func (s *service) handleLootboxGeneric(ctx context.Context, _ *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, _ map[string]interface{}) (string, error) {
	return s.processLootbox(ctx, user, inventory, item, quantity)
}

// weaponTimeouts maps weapon internal names to their timeout durations
var weaponTimeouts = map[string]time.Duration{
	domain.ItemBlaster:     60 * time.Second,
	domain.ItemBigBlaster:  600 * time.Second,
	domain.ItemHugeBlaster: 6000 * time.Second,
	domain.ItemThis:        101 * time.Second,
	domain.ItemDeez:        202 * time.Second,
	domain.ItemMissile:     60 * time.Second,
	domain.ItemGrenade:     60 * time.Second,
	domain.ItemTNT:         60 * time.Second,
}

// getWeaponTimeout returns the timeout duration for a weapon, with a default fallback
func getWeaponTimeout(itemName string) time.Duration {
	if timeout, ok := weaponTimeouts[itemName]; ok {
		return timeout
	}
	return BlasterTimeoutDuration // default fallback
}

func (s *service) handleWeapon(ctx context.Context, _ *service, _ *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgHandleWeaponCalled, "item", item.InternalName, "quantity", quantity)

	targetUsername, ok := args[ArgsTargetUsername].(string)
	username, _ := args[ArgsUsername].(string)

	// Special handling for grenades - no target required (random targeting)
	if item.InternalName == domain.ItemGrenade {
		// Grenade doesn't require a target, it selects randomly
		if !ok || targetUsername == "" {
			log.Info("Grenade used without target - random targeting will be implemented by caller")
			// For now, we require a target until we implement random user selection
			// This will be handled at a higher level (Discord bot/API) which knows active users
			return "", fmt.Errorf(ErrMsgTargetUsernameRequired)
		}
	} else if item.InternalName == domain.ItemTNT {
		// TNT is AoE - doesn't require a specific target
		// The target can be empty for TNT, and it will affect multiple users
		log.Info("TNT used - AoE targeting will be implemented by caller")
		// For now, require target until AoE logic is implemented at caller level
		if !ok || targetUsername == "" {
			return "", fmt.Errorf(ErrMsgTargetUsernameRequired)
		}
	} else {
		// Standard weapons require a target
		if !ok || targetUsername == "" {
			log.Warn(LogWarnTargetUsernameMissingWeapon)
			return "", fmt.Errorf(ErrMsgTargetUsernameRequired)
		}
	}

	// Find item slot
	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
	if itemSlotIndex == -1 {
		log.Warn(LogWarnWeaponNotInInventory, "item", item.InternalName)
		return "", fmt.Errorf(ErrMsgItemNotFoundInInventory)
	}
	if slotQuantity < quantity {
		log.Warn(LogWarnNotEnoughWeapons, "item", item.InternalName)
		return "", fmt.Errorf(ErrMsgNotEnoughItemsInInventory)
	}
	utils.RemoveFromSlot(inventory, itemSlotIndex, quantity)

	// Get timeout for this weapon type
	timeout := getWeaponTimeout(item.InternalName)

	// Apply timeout
	if err := s.TimeoutUser(ctx, targetUsername, timeout, MsgBlasterReasonBy+username); err != nil {
		log.Error(LogWarnFailedToTimeoutUser, "error", err, "target", targetUsername)
		// Continue anyway, as the item was used
	}

	displayName := s.namingResolver.GetDisplayName(item.InternalName, "")
	log.Info(LogMsgWeaponUsed, "target", targetUsername, "item", item.InternalName, "quantity", quantity)
	return fmt.Sprintf("%s used %s on %s! %d %s(s) fired. Timed out for %v.", username, displayName, targetUsername, quantity, displayName, timeout), nil
}

// handleBlaster is a legacy wrapper for backward compatibility
func (s *service) handleBlaster(ctx context.Context, svc *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	return s.handleWeapon(ctx, svc, user, inventory, item, quantity, args)
}

// reviveRecoveryTimes maps revive internal names to their recovery durations
var reviveRecoveryTimes = map[string]time.Duration{
	domain.ItemReviveSmall:  60 * time.Second,
	domain.ItemReviveMedium: 600 * time.Second,
	domain.ItemReviveLarge:  6000 * time.Second,
}

// getReviveRecovery returns the recovery duration for a revive item
func getReviveRecovery(itemName string) time.Duration {
	if recovery, ok := reviveRecoveryTimes[itemName]; ok {
		return recovery
	}
	return 60 * time.Second // default fallback
}

func (s *service) handleRevive(ctx context.Context, _ *service, _ *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgHandleReviveCalled, "item", item.InternalName, "quantity", quantity)
	targetUsername, ok := args[ArgsTargetUsername].(string)
	if !ok || targetUsername == "" {
		log.Warn(LogWarnTargetUsernameMissingRevive)
		return "", fmt.Errorf(ErrMsgTargetUsernameRequiredRevive)
	}
	username, _ := args[ArgsUsername].(string)

	// Find item slot
	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
	if itemSlotIndex == -1 {
		log.Warn(LogWarnReviveNotInInventory, "item", item.InternalName)
		return "", fmt.Errorf(ErrMsgItemNotFoundInInventory)
	}
	if slotQuantity < quantity {
		log.Warn(LogWarnNotEnoughRevives, "item", item.InternalName)
		return "", fmt.Errorf(ErrMsgNotEnoughItemsInInventory)
	}
	utils.RemoveFromSlot(inventory, itemSlotIndex, quantity)

	// Get recovery time for this revive type
	recovery := getReviveRecovery(item.InternalName)
	totalRecovery := time.Duration(quantity) * recovery

	// Reduce timeout for target user
	if err := s.ReduceTimeout(ctx, targetUsername, totalRecovery); err != nil {
		log.Error(LogWarnFailedToReduceTimeout, "error", err, "target", targetUsername)
		// Continue anyway, as the item was used
	}

	displayName := s.namingResolver.GetDisplayName(item.InternalName, "")
	log.Info(LogMsgReviveUsed, "target", targetUsername, "item", item.InternalName, "quantity", quantity)
	return fmt.Sprintf("%s used %d %s on %s! Reduced timeout by %v.", username, quantity, displayName, targetUsername, totalRecovery), nil
}

func (s *service) handleShield(ctx context.Context, _ *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, _ map[string]interface{}) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgHandleShieldCalled, "item", item.InternalName, "quantity", quantity)

	// Find item slot
	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
	if itemSlotIndex == -1 {
		log.Warn(LogWarnShieldNotInInventory)
		return "", fmt.Errorf(ErrMsgItemNotFoundInInventory)
	}
	if slotQuantity < quantity {
		log.Warn(LogWarnNotEnoughShields)
		return "", fmt.Errorf(ErrMsgNotEnoughItemsInInventory)
	}
	utils.RemoveFromSlot(inventory, itemSlotIndex, quantity)

	// Determine if this is a mirror shield
	isMirror := item.InternalName == domain.ItemMirrorShield

	// Apply shield status to user
	if err := s.ApplyShield(ctx, user, quantity, isMirror); err != nil {
		log.Error(LogWarnFailedToApplyShield, "error", err)
		return "", fmt.Errorf(ErrMsgFailedToApplyShield)
	}

	displayName := s.namingResolver.GetDisplayName(item.InternalName, "")
	log.Info(LogMsgShieldApplied, "item", item.InternalName, "quantity", quantity, "is_mirror", isMirror)

	if isMirror {
		return fmt.Sprintf("Activated %d %s! Next %d attacks will be REFLECTED!", quantity, displayName, quantity), nil
	}
	return fmt.Sprintf("Activated %d %s! Protected from next %d attacks.", quantity, displayName, quantity), nil
}

// rarecandyXPAmount defines the XP granted per rare candy
const rarecandyXPAmount = 500

func (s *service) handleRareCandy(ctx context.Context, _ *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgHandleRareCandyCalled, "quantity", quantity)

	jobName, ok := args[ArgsJobName].(string)
	if !ok || jobName == "" {
		log.Warn(LogWarnJobNameMissing)
		return "", fmt.Errorf(ErrMsgJobNameRequired)
	}

	// Find item slot
	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
	if itemSlotIndex == -1 {
		log.Warn(LogWarnRareCandyNotInInventory)
		return "", fmt.Errorf(ErrMsgItemNotFoundInInventory)
	}
	if slotQuantity < quantity {
		log.Warn(LogWarnNotEnoughRareCandy)
		return "", fmt.Errorf(ErrMsgNotEnoughItemsInInventory)
	}
	utils.RemoveFromSlot(inventory, itemSlotIndex, quantity)

	// Award XP to the specified job
	totalXP := quantity * rarecandyXPAmount
	if s.jobService != nil {
		metadata := map[string]interface{}{
			"source":   job.SourceRareCandy,
			"quantity": quantity,
		}
		if _, err := s.jobService.AwardXP(ctx, user.ID, jobName, totalXP, job.SourceRareCandy, metadata); err != nil {
			log.Error(LogWarnFailedToAwardJobXP, "error", err, "job", jobName)
			return "", fmt.Errorf(ErrMsgFailedToAwardXP)
		}
	}

	log.Info(LogMsgRareCandyUsed, "job", jobName, "xp", totalXP, "quantity", quantity)
	return fmt.Sprintf("Used %d rare candy! Granted %d XP to %s.", quantity, totalXP, jobName), nil
}

func (s *service) handleResourceGenerator(ctx context.Context, _ *service, _ *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgResourceGeneratorCalled, "item", item.InternalName, "quantity", quantity)

	username, _ := args[ArgsUsername].(string)

	// Find item slot
	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
	if itemSlotIndex == -1 {
		return "", fmt.Errorf(ErrMsgItemNotFoundInInventory)
	}
	if slotQuantity < quantity {
		return "", fmt.Errorf(ErrMsgNotEnoughItemsInInventory)
	}
	utils.RemoveFromSlot(inventory, itemSlotIndex, quantity)

	// Generate sticks (shovel generates 2 sticks per use)
	stickItem, err := s.getItemByNameCached(ctx, domain.ItemStick)
	if err != nil {
		return "", fmt.Errorf("failed to get stick item: %w", err)
	}

	sticksGenerated := quantity * ShovelSticksPerUse
	utils.AddItemsToInventory(inventory, []domain.InventorySlot{
		{ItemID: stickItem.ID, Quantity: sticksGenerated, ShineLevel: "COMMON"},
	}, nil)

	displayName := s.namingResolver.GetDisplayName(domain.ItemStick, "")
	return fmt.Sprintf("%s%d %s!", username+MsgShovelUsed, sticksGenerated, displayName), nil
}

func (s *service) handleUtility(ctx context.Context, _ *service, _ *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgUtilityCalled, "item", item.InternalName, "quantity", quantity)

	username, _ := args[ArgsUsername].(string)

	// Find item slot
	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
	if itemSlotIndex == -1 {
		return "", fmt.Errorf(ErrMsgItemNotFoundInInventory)
	}
	if slotQuantity < quantity {
		return "", fmt.Errorf(ErrMsgNotEnoughItemsInInventory)
	}
	utils.RemoveFromSlot(inventory, itemSlotIndex, quantity)

	return username + MsgStickUsed, nil
}
