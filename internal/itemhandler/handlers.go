package itemhandler

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// ============================================================================
// Lootbox Handler
// ============================================================================

// ProcessLootbox handles lootbox opening: validates, consumes, opens, and returns feedback.
func ProcessLootbox(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, lootboxItem *domain.Item, quantity int) (string, error) {
	log := logger.FromContext(ctx)

	// 1. Validate and consume lootboxes
	totalAvailable := utils.GetTotalQuantity(inventory, lootboxItem.ID)
	if totalAvailable == 0 {
		return "", domain.ErrNotInInventory
	}
	if totalAvailable < quantity {
		return "", domain.ErrInsufficientQuantity
	}

	consumedSlots, err := utils.ConsumeItemsWithTracking(inventory, lootboxItem.ID, quantity, ec.RandomFloat)
	if err != nil {
		return "", err
	}

	// 2. Use lootbox service to open lootboxes
	var allDrops []lootbox.DroppedItem
	for _, slot := range consumedSlots {
		drops, err := ec.OpenLootbox(ctx, lootboxItem.InternalName, slot.Quantity, slot.QualityLevel)
		if err != nil {
			log.Error("Failed to open lootbox", "error", err, "lootbox", lootboxItem.InternalName)
			return "", fmt.Errorf("failed to open lootbox: %w", err)
		}
		allDrops = append(allDrops, drops...)
	}

	if len(allDrops) == 0 {
		return MsgLootboxEmpty, nil
	}

	// 3. Process drops and generate feedback
	return ProcessLootboxDrops(ctx, ec, user, inventory, lootboxItem, quantity, allDrops)
}

// ProcessLootboxDrops processes drops from a lootbox opening and generates feedback.
func ProcessLootboxDrops(ctx context.Context, ec EffectContext, user *domain.User, inventory *domain.Inventory, lootboxItem *domain.Item, quantity int, drops []lootbox.DroppedItem) (string, error) {
	var msgBuilder strings.Builder
	// Use alias for the lootbox when opening
	displayName := ec.GetDisplayName(lootboxItem.InternalName, "")

	msgBuilder.WriteString(MsgLootboxOpened)
	msgBuilder.WriteString(" ")
	if quantity > 1 {
		msgBuilder.WriteString(strconv.Itoa(quantity))
		msgBuilder.WriteString(" ")
		msgBuilder.WriteString(ec.Pluralize(displayName, quantity))
	} else {
		msgBuilder.WriteString(getIndefiniteArticle(displayName))
		msgBuilder.WriteString(" ")
		msgBuilder.WriteString(displayName)
	}
	msgBuilder.WriteString(MsgLootboxReceived)

	stats := aggregateDropsAndUpdateInventory(ec, inventory, drops, &msgBuilder)

	if stats.hasLegendary {
		_ = ec.RecordUserEvent(ctx, user.ID, domain.EventTypeLootboxJackpot, &domain.LootboxEventData{
			Item:   lootboxItem.InternalName,
			Drops:  drops,
			Value:  stats.totalValue,
			Source: "lootbox",
		})
		msgBuilder.WriteString(MsgLootboxJackpot)
	} else if stats.hasEpic {
		_ = ec.RecordUserEvent(ctx, user.ID, domain.EventTypeLootboxBigWin, &domain.LootboxEventData{
			Item:   lootboxItem.InternalName,
			Drops:  drops,
			Value:  stats.totalValue,
			Source: "lootbox",
		})
		msgBuilder.WriteString(MsgLootboxBigWin)
	} else if stats.totalValue > 0 && quantity >= domain.BulkFeedbackThreshold {
		msgBuilder.WriteString(MsgLootboxNiceHaul)
	}

	return msgBuilder.String(), nil
}

type dropStats struct {
	totalValue   int
	hasLegendary bool
	hasEpic      bool
}

func aggregateDropsAndUpdateInventory(ec EffectContext, inventory *domain.Inventory, drops []lootbox.DroppedItem, msgBuilder *strings.Builder) dropStats {
	var stats dropStats

	// Convert drops to inventory slots for batch adding
	itemsToAdd := make([]domain.InventorySlot, 0, len(drops))

	// Group items by their resolved display name (which includes quality where applicable)
	type dropGroup struct {
		Quantity int
		Name     string
	}
	displayGroups := make(map[string]*dropGroup)
	var displayOrder []string

	for _, drop := range drops {
		// Track stats for feedback
		stats.totalValue += drop.Value
		if drop.QualityLevel == domain.QualityLegendary {
			stats.hasLegendary = true
		} else if drop.QualityLevel == domain.QualityEpic {
			stats.hasEpic = true
		}

		// Prepare item for batch add - preserve quality level from loot table
		itemsToAdd = append(itemsToAdd, domain.InventorySlot{
			ItemID:       drop.ItemID,
			Quantity:     drop.Quantity,
			QualityLevel: drop.QualityLevel,
		})

		// Get display name
		itemDisplayName := ec.GetDisplayName(drop.ItemName, drop.QualityLevel)

		if group, exists := displayGroups[itemDisplayName]; exists {
			group.Quantity += drop.Quantity
		} else {
			displayOrder = append(displayOrder, itemDisplayName)
			displayGroups[itemDisplayName] = &dropGroup{
				Quantity: drop.Quantity,
				Name:     itemDisplayName,
			}
		}
	}

	// Format output with grouped items
	first := true
	for _, displayName := range displayOrder {
		group := displayGroups[displayName]

		if !first {
			msgBuilder.WriteString(LootboxDropSeparator)
		}

		// Simplify output: "Quantity Name"
		msgBuilder.WriteString(strconv.Itoa(group.Quantity))
		msgBuilder.WriteString(" ")
		msgBuilder.WriteString(ec.Pluralize(group.Name, group.Quantity))

		first = false
	}

	// Add all items to inventory using optimized helper
	utils.AddItemsToInventory(inventory, itemsToAdd, nil)

	return stats
}

// ============================================================================
// Weapon Handler
// ============================================================================

// getWeaponTimeout returns the timeout duration for a weapon, with a default fallback.
func getWeaponTimeout(itemName string) time.Duration {
	if timeout, ok := weaponTimeouts[itemName]; ok {
		return timeout
	}
	return domain.BlasterTimeoutDuration // default fallback
}

func handleWeapon(ctx context.Context, ec EffectContext, _ *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args HandlerArgs) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgHandleWeaponCalled, "item", item.InternalName, "quantity", quantity)

	targetUsername := args.TargetUsername
	username := args.Username
	platform := args.Platform

	// Find total availability first (before target selection)
	totalAvailable := utils.GetTotalQuantity(inventory, item.ID)
	if totalAvailable == 0 {
		log.Warn(LogWarnWeaponNotInInventory, "item", item.InternalName)
		return "", domain.ErrNotInInventory
	}
	if totalAvailable < quantity {
		log.Warn(LogWarnNotEnoughWeapons, "item", item.InternalName)
		return "", domain.ErrInsufficientQuantity
	}

	consumedSlots, err := utils.ConsumeItemsWithTracking(inventory, item.ID, quantity, ec.RandomFloat)
	if err != nil {
		return "", err
	}

	var timeout time.Duration
	var displayName string
	for i, slot := range consumedSlots {
		baseTimeout := getWeaponTimeout(item.InternalName) + slot.QualityLevel.GetTimeoutAdjustment()
		timeout += baseTimeout * time.Duration(slot.Quantity)
		if i == 0 {
			displayName = ec.GetDisplayName(item.InternalName, slot.QualityLevel)
		}
	}

	// Route to special handlers if applicable
	switch item.InternalName {
	case domain.ItemTNT:
		return handleTNT(ctx, ec, username, platform, timeout, displayName)
	case domain.ItemGrenade:
		return handleGrenade(ctx, ec, username, platform, timeout)
	case domain.ItemThis:
		return handleThis(ctx, ec, username, timeout, displayName)
	}

	// Standard weapons require a user-provided target
	if targetUsername == "" {
		log.Warn(LogWarnTargetUsernameMissingWeapon)
		return "", fmt.Errorf("%w: target username is required for weapon", domain.ErrInvalidInput)
	}

	// Apply timeout
	if err := ec.TimeoutUser(ctx, targetUsername, timeout, MsgBlasterReasonBy+username); err != nil {
		log.Error(LogWarnFailedToTimeoutUser, "error", err, "target", targetUsername)
		// Continue anyway, as the item was used
	}

	log.Info(LogMsgWeaponUsed, "target", targetUsername, "item", item.InternalName, "quantity", quantity)
	return fmt.Sprintf("%s used %s on %s! %d %s(s) fired. Timed out for %v.", username, displayName, targetUsername, quantity, displayName, timeout), nil
}

func handleTNT(ctx context.Context, ec EffectContext, username, platform string, timeout time.Duration, displayName string) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("TNT used, selecting 5-9 random targets")

	// Select 5-9 random targets
	numTargets := 5 + rand.Intn(5) //nolint:gosec // weak random is fine for games
	targets, err := ec.GetRandomTargets(platform, numTargets)
	if err != nil {
		log.Warn("No active targets available for TNT", "error", err)
		return "", fmt.Errorf("%w: no active users to target", domain.ErrInvalidInput)
	}

	// Apply timeout to all targets and collect names
	hitUsernames := make([]string, 0, len(targets))
	for _, target := range targets {
		if err := ec.TimeoutUser(ctx, target.Username, timeout, MsgBlasterReasonBy+username); err != nil {
			log.Error(LogWarnFailedToTimeoutUser, "error", err, "target", target.Username)
			// Continue with other targets even if one fails
		}

		// Remove from active chatters
		ec.RemoveActiveChatter(platform, target.UserID)
		hitUsernames = append(hitUsernames, target.Username)
	}

	log.Info("TNT hit multiple targets", "count", len(hitUsernames), "targets", hitUsernames)

	// Format message with all hit users
	targetsStr := FormatTargetList(hitUsernames)
	return fmt.Sprintf("%s used %s! Hit %d targets: %s! Timed out for %v.",
		username, displayName, len(hitUsernames), targetsStr, timeout), nil
}

func handleGrenade(ctx context.Context, ec EffectContext, username, platform string, timeout time.Duration) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("Grenade used, selecting single random target")

	randomUsername, randomUserID, err := ec.GetRandomTarget(platform)
	if err != nil {
		log.Warn("No active targets available for grenade", "error", err)
		return "", fmt.Errorf("%w: no active users to target", domain.ErrInvalidInput)
	}

	// Apply timeout
	if err := ec.TimeoutUser(ctx, randomUsername, timeout, MsgBlasterReasonBy+username); err != nil {
		log.Error(LogWarnFailedToTimeoutUser, "error", err, "target", randomUsername)
		// Continue anyway, as the item was used
	}

	// Remove from active chatters
	ec.RemoveActiveChatter(platform, randomUserID)
	log.Info("Grenade hit target", "target", randomUsername)

	return fmt.Sprintf("%s hit: %s!",
		username, randomUsername), nil
}

func handleThis(ctx context.Context, ec EffectContext, username string, timeout time.Duration, displayName string) (string, error) {
	log := logger.FromContext(ctx)
	log.Info("This used, targeting self")

	// Apply timeout to self
	if err := ec.TimeoutUser(ctx, username, timeout, "Played yourself"); err != nil {
		log.Error(LogWarnFailedToTimeoutUser, "error", err, "target", username)
	}

	return fmt.Sprintf("%s used %s... Congratulations, you played yourself. Timed out for %v.", username, displayName, timeout), nil
}

// FormatTargetList formats a list of usernames for display.
func FormatTargetList(usernames []string) string {
	if len(usernames) == 0 {
		return ""
	}
	if len(usernames) == 1 {
		return usernames[0]
	}
	if len(usernames) == 2 {
		return usernames[0] + " and " + usernames[1]
	}
	// For 3+, use comma-separated with "and" before last
	result := ""
	for i, name := range usernames {
		if i == len(usernames)-1 {
			result += ", and " + name
		} else if i > 0 {
			result += ", " + name
		} else {
			result += name
		}
	}
	return result
}

// ============================================================================
// Revive Handler
// ============================================================================

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

// ============================================================================
// Trap Handler
// ============================================================================

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

// ============================================================================
// Shield Handler
// ============================================================================

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

// ============================================================================
// Rare Candy Handler
// ============================================================================

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

// ============================================================================
// Resource Generator Handler
// ============================================================================

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

// ============================================================================
// Utility Handler
// ============================================================================

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

// ============================================================================
// Video Filter Handler
// ============================================================================

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

// ============================================================================
// Utility Functions
// ============================================================================

// Pluralize handles simple pluralization for game items and phrases.
func Pluralize(name string, quantity int) string {
	if quantity <= 1 || name == "" {
		return name
	}

	// Check for quality emojis at the end (Legendary/Cursed)
	suffix := ""
	baseName := name
	// Emojis are multi-byte
	if strings.HasSuffix(name, "👑") {
		suffix = "👑"
		baseName = strings.TrimSuffix(name, "👑")
	} else if strings.HasSuffix(name, "👻") {
		suffix = "👻"
		baseName = strings.TrimSuffix(name, "👻")
	}

	// Handle "of" phrases: "pouch of coins" -> "pouches of coins"
	if strings.Contains(baseName, " of ") {
		parts := strings.SplitN(baseName, " of ", 2)
		return Pluralize(parts[0], quantity) + " of " + parts[1] + suffix
	}

	// Common uncountable or collective nouns in game context
	lower := strings.ToLower(baseName)
	switch lower {
	case domain.PublicNameMoney, "ghost-gold", "coins", "scrap", "junk", "credits":
		return baseName + suffix
	}
	if strings.HasSuffix(lower, " coins") {
		return baseName + suffix
	}

	// Basic pluralization rules
	if strings.HasSuffix(baseName, "y") && len(baseName) > 1 {
		vowels := "aeiouAEIOU"
		if !strings.ContainsAny(string(baseName[len(baseName)-2]), vowels) {
			return baseName[:len(baseName)-1] + "ies" + suffix
		}
	}

	if strings.HasSuffix(baseName, "s") || strings.HasSuffix(baseName, "x") ||
		strings.HasSuffix(baseName, "ch") || strings.HasSuffix(baseName, "sh") {
		return baseName + "es" + suffix
	}

	return baseName + "s" + suffix
}

// getIndefiniteArticle returns "a" or "an" based on the first letter of the word.
func getIndefiniteArticle(word string) string {
	if len(word) == 0 {
		return "a"
	}
	first := strings.ToLower(string(word[0]))
	if strings.ContainsAny(first, "aeiou") {
		return "an"
	}
	return "a"
}
