package itemhandler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

func handleTrap(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgHandleTrapCalled, "item", item.InternalName, "quantity", quantity)

	if quantity < 1 {
		return "", domain.ErrInvalidQuantity
	}

	platform := args.Platform
	if platform == "" {
		platform = domain.PlatformTwitch
	}

	potentialTargets, isMine, err := getTrapTargets(ctx, ec, item, quantity, user, platform, args)
	if err != nil {
		return "", err
	}

	if err := validateTrapInventory(ec, inventory, item, quantity, isMine); err != nil {
		log.Warn("Trap inventory validation failed", "item", item.InternalName, "error", err)
		return "", err
	}

	itemsConsumed, trapsPlaced, selfTriggered, badLuckSelf, err := executeTrapTransaction(ctx, ec, user, item, quantity, platform, potentialTargets, isMine, inventory)
	if err != nil {
		return "", err
	}

	return formatTrapResponse(user, item, potentialTargets, itemsConsumed, trapsPlaced, selfTriggered, badLuckSelf), nil
}

func getTrapTargets(ctx context.Context, ec EffectContext, item *domain.Item, quantity int, user *domain.User, platform string, args HandlerArgs) ([]string, bool, error) {
	log := logger.FromContext(ctx)
	if item.InternalName == domain.ItemMine {
		log.Info("Mine used, selecting random targets", "count", quantity)
		targets, err := ec.GetRandomTargets(platform, quantity)
		if err != nil {
			return []string{user.Username}, true, nil
		}
		var potentialTargets []string
		for _, t := range targets {
			potentialTargets = append(potentialTargets, t.Username)
		}
		if len(potentialTargets) == 0 {
			potentialTargets = []string{user.Username}
		}
		return potentialTargets, true, nil
	}

	if quantity != 1 {
		return nil, false, errors.New("can only use 1 trap at a time")
	}

	targetUsername := args.TargetUsername
	if targetUsername == "" {
		return nil, false, fmt.Errorf("%w: target username is required for weapon", domain.ErrInvalidInput)
	}
	return []string{targetUsername}, false, nil
}

func validateTrapInventory(ec EffectContext, inventory *domain.Inventory, item *domain.Item, quantity int, isMine bool) error {
	itemSlotIndex, slotQuantity := utils.FindRandomSlot(inventory, item.ID, ec.RandomFloat)
	if itemSlotIndex == -1 {
		return domain.ErrNotInInventory
	}
	if slotQuantity < 1 {
		return domain.ErrInsufficientQuantity
	}
	if !isMine && slotQuantity < quantity {
		return domain.ErrInsufficientQuantity
	}
	return nil
}

func executeTrapTransaction(ctx context.Context, ec EffectContext, user *domain.User, item *domain.Item, quantity int, platform string, potentialTargets []string, isMine bool, inventory *domain.Inventory) (int, int, bool, bool, error) {
	itemsConsumed := 0
	trapsPlaced := 0
	selfTriggered := false
	badLuckSelf := false

	totalQty := utils.GetTotalQuantity(inventory, item.ID)
	if totalQty < 1 {
		return 0, 0, false, false, fmt.Errorf("item no longer available")
	}

	maxPossible := quantity
	if totalQty < maxPossible {
		maxPossible = totalQty
	}

	if isMine && len(potentialTargets) == 1 && strings.EqualFold(potentialTargets[0], user.Username) {
		// Special case: if we ONLY targeted ourselves (e.g. no other active chatters)
		badLuckSelf = true
		itemsConsumed = 1
		trapsPlaced = 1
		err := utils.ConsumeItems(inventory, item.ID, 1, ec.RandomFloat)
		if err != nil {
			return 0, 0, false, false, err
		}
		handleSelfTargetMine(ctx, ec, user)
	} else {
		var err error
		itemsConsumed, trapsPlaced, selfTriggered, badLuckSelf, err = processTrapTargets(ctx, ec, user, potentialTargets, platform, maxPossible, inventory, item, isMine)
		if err != nil {
			return 0, 0, false, false, err
		}
	}

	return itemsConsumed, trapsPlaced, selfTriggered, badLuckSelf, nil
}

func processTrapTargets(ctx context.Context, ec EffectContext, user *domain.User, potentialTargets []string, platform string, maxPossible int, inventory *domain.Inventory, item *domain.Item, isMine bool) (int, int, bool, bool, error) {
	log := logger.FromContext(ctx)
	itemsConsumed := 0
	trapsPlaced := 0
	selfTriggered := false
	badLuckSelf := false

	for _, targetName := range potentialTargets {
		if itemsConsumed >= maxPossible {
			break
		}

		if isMine && strings.EqualFold(targetName, user.Username) {
			badLuckSelf = true
			itemsConsumed++
			trapsPlaced++
			err := utils.ConsumeItems(inventory, item.ID, 1, ec.RandomFloat)
			if err != nil {
				return itemsConsumed, trapsPlaced, false, false, fmt.Errorf("failed to create trap: %w", err)
			}
			handleSelfTargetMine(ctx, ec, user)
			break
		}

		targetUser, err := ec.GetUserByPlatformUsername(ctx, platform, targetName)
		if err != nil {
			log.Warn("Target user not found, skipping", "username", targetName)
			continue
		}

		targetUserID, _ := uuid.Parse(targetUser.ID)
		existingTrap, err := ec.GetActiveTrapForUpdate(ctx, targetUserID)
		if err != nil {
			return itemsConsumed, trapsPlaced, selfTriggered, false, fmt.Errorf("failed to check existing trap: %w", err)
		}

		if existingTrap != nil {
			itemsConsumed++
			err := utils.ConsumeItems(inventory, item.ID, 1, ec.RandomFloat)
			if err != nil {
				return itemsConsumed, trapsPlaced, false, false, fmt.Errorf("failed to create trap: %w", err)
			}
			selfTriggered = true
			if err := handleExistingTrapTrigger(ctx, ec, user, targetName, existingTrap); err != nil {
				return itemsConsumed, trapsPlaced, true, false, err
			}
			break
		}

		consumedSlots, err := utils.ConsumeItemsWithTracking(inventory, item.ID, 1, ec.RandomFloat)
		if err != nil || len(consumedSlots) == 0 {
			break // Should not happen since we checked maxPossible
		}
		qualityLevel := consumedSlots[0].QualityLevel
		if qualityLevel == "" {
			qualityLevel = domain.QualityCommon
		}

		itemsConsumed++
		trapsPlaced++
		userID, _ := uuid.Parse(user.ID)
		newTrap := &domain.Trap{
			ID:             uuid.New(),
			SetterID:       userID,
			TargetID:       targetUserID,
			QualityLevel:   qualityLevel,
			TimeoutSeconds: 60,
			PlacedAt:       time.Now(),
		}
		if err := ec.CreateTrap(ctx, newTrap); err != nil {
			return itemsConsumed, trapsPlaced, false, false, fmt.Errorf("failed to create trap: %w", err)
		}
	}
	return itemsConsumed, trapsPlaced, selfTriggered, badLuckSelf, nil
}

func formatTrapResponse(user *domain.User, item *domain.Item, potentialTargets []string, itemsConsumed, trapsPlaced int, selfTriggered, badLuckSelf bool) string {
	if selfTriggered {
		return fmt.Sprintf("%s tried to use %s but stepped on a trap!", user.Username, item.PublicName)
	}
	if badLuckSelf {
		return fmt.Sprintf("%s dropped a mine straight on their own foot!", user.Username)
	}
	if trapsPlaced == 0 && itemsConsumed == 0 {
		return "No targets found."
	}

	targetMsg := "someone"
	// Only show the specific target name if it's NOT a mine
	if item.InternalName != domain.ItemMine {
		if len(potentialTargets) == 1 {
			targetMsg = potentialTargets[0]
		} else if trapsPlaced > 0 {
			targetMsg = fmt.Sprintf("%d people", trapsPlaced)
		}
	} else if trapsPlaced > 1 {
		// For multiple mines, use a generic plural
		targetMsg = "multiple people"
	}

	return fmt.Sprintf("%s set %d %s for %s!", user.Username, trapsPlaced, item.PublicName, targetMsg)
}

func handleSelfTargetMine(ctx context.Context, ec EffectContext, user *domain.User) {
	log := logger.FromContext(ctx)
	if err := ec.TimeoutUser(ctx, user.Username, 60*time.Second, "tripped on your own mine immediately!"); err != nil {
		log.Error("Failed to timeout user for bad luck", "error", err)
	}
}

func handleExistingTrapTrigger(ctx context.Context, ec EffectContext, user *domain.User, targetName string, existingTrap *domain.Trap) error {
	timeout := time.Duration(existingTrap.CalculateTimeout()) * time.Second
	if err := ec.TimeoutUser(ctx, user.Username, timeout, fmt.Sprintf("stepped on %s's trap entirely by accident!", targetName)); err != nil {
		return fmt.Errorf("failed to apply self-trap timeout: %w", err)
	}
	if err := ec.TriggerTrap(ctx, existingTrap.ID); err != nil {
		return fmt.Errorf("failed to trigger existing trap: %w", err)
	}

	recordTrapSelfTriggerStats(ctx, ec, user, targetName, existingTrap)
	return nil
}

func recordTrapSelfTriggerStats(ctx context.Context, ec EffectContext, user *domain.User, targetName string, trap *domain.Trap) {
	eventData := &domain.TrapTriggeredData{
		TrapID:           trap.ID,
		SetterID:         trap.SetterID,
		SetterUsername:   targetName,
		TargetID:         trap.TargetID,
		TargetUsername:   user.Username,
		QualityLevel:     trap.QualityLevel,
		TimeoutSeconds:   trap.CalculateTimeout(),
		WasSelfTriggered: true,
	}
	_ = ec.RecordUserEvent(ctx, user.ID, domain.EventTrapSelfTriggered, eventData)
}
