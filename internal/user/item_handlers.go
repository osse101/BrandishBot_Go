package user

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
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

const (
	// Error messages
	ErrMsgInvalidQuantity = "invalid quantity"
)

// Item effect handlers

func (s *service) processLootbox(ctx context.Context, user *domain.User, inventory *domain.Inventory, lootboxItem *domain.Item, quantity int) (string, error) {
	log := logger.FromContext(ctx)

	// 1. Validate and consume lootboxes
	qualityLevel, err := s.consumeLootboxFromInventory(inventory, lootboxItem, quantity)
	if err != nil {
		return "", err
	}

	// 2. Use lootbox service to open lootboxes
	drops, err := s.lootboxService.OpenLootbox(ctx, lootboxItem.InternalName, quantity, qualityLevel)
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

func (s *service) consumeLootboxFromInventory(inventory *domain.Inventory, item *domain.Item, quantity int) (domain.QualityLevel, error) {
	itemSlotIndex, slotQuantity := utils.FindRandomSlot(inventory, item.ID, s.rnd)
	if itemSlotIndex == -1 {
		return "", errors.New(ErrMsgItemNotFoundInInventory)
	}

	if slotQuantity < quantity {
		return "", errors.New(ErrMsgNotEnoughItemsInInventory)
	}

	qualityLevel := inventory.Slots[itemSlotIndex].QualityLevel
	utils.RemoveFromSlot(inventory, itemSlotIndex, quantity)
	return qualityLevel, nil
}

func (s *service) processLootboxDrops(ctx context.Context, user *domain.User, inventory *domain.Inventory, lootboxItem *domain.Item, quantity int, drops []lootbox.DroppedItem) (string, error) {
	var msgBuilder strings.Builder
	// User Request: Use alias for the lootbox when opening
	displayName := s.namingResolver.GetDisplayName(lootboxItem.InternalName, "")

	msgBuilder.WriteString(MsgLootboxOpened)
	msgBuilder.WriteString(" ")
	if quantity > 1 {
		msgBuilder.WriteString(strconv.Itoa(quantity))
		msgBuilder.WriteString(" ")
		msgBuilder.WriteString(s.pluralize(displayName, quantity))
	} else {
		msgBuilder.WriteString(getIndefiniteArticle(displayName))
		msgBuilder.WriteString(" ")
		msgBuilder.WriteString(displayName)
	}
	msgBuilder.WriteString(MsgLootboxReceived)

	stats := s.aggregateDropsAndUpdateInventory(inventory, drops, &msgBuilder)

	// User Request: "All lootbox open messages were too verbose and should be at the level I gave as example"
	// Example: "Opened Junkbox and received: 1 Shiny credit" or " ... 5 Shiny credits"
	// LevelUp Philosophy: "If a number goes up, the player should feel it."
	// Removing explicit Value output as per user request to reduce verbosity

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

		if !first {
			msgBuilder.WriteString(LootboxDropSeparator)
		}

		// Get display name (which might be "Shiny credit" for money or "Ray Gun" for blaster)
		// We trust the resolver to give the base name, and we handle pluralization
		itemDisplayName := s.namingResolver.GetDisplayName(drop.ItemName, drop.QualityLevel)

		// Simplify output: "Quantity Name"
		msgBuilder.WriteString(strconv.Itoa(drop.Quantity))
		msgBuilder.WriteString(" ")
		msgBuilder.WriteString(s.pluralize(itemDisplayName, drop.Quantity))

		first = false
	}

	// Add all items to inventory using optimized helper
	utils.AddItemsToInventory(inventory, itemsToAdd, nil)

	return stats
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

	targetUsername, targetProvided := args[ArgsTargetUsername].(string)
	username, _ := args[ArgsUsername].(string)
	platform, _ := args[ArgsPlatform].(string)

	// Find item slot first (before target selection), randomly if multiple exist with different qualities
	itemSlotIndex, slotQuantity := utils.FindRandomSlot(inventory, item.ID, s.rnd)
	if itemSlotIndex == -1 {
		log.Warn(LogWarnWeaponNotInInventory, "item", item.InternalName)
		return "", errors.New(ErrMsgItemNotFoundInInventory)
	}
	if slotQuantity < quantity {
		log.Warn(LogWarnNotEnoughWeapons, "item", item.InternalName)
		return "", errors.New(ErrMsgNotEnoughItemsInInventory)
	}

	qualityLevel := inventory.Slots[itemSlotIndex].QualityLevel
	timeout := getWeaponTimeout(item.InternalName) + qualityLevel.GetTimeoutAdjustment()
	displayName := s.namingResolver.GetDisplayName(item.InternalName, qualityLevel)

	// Special handling for TNT - multi-target (5-9 targets)
	if item.InternalName == domain.ItemTNT {
		log.Info("TNT used, selecting 5-9 random targets")

		// Select 5-9 random targets
		numTargets := 5 + rand.Intn(5) //nolint:gosec // weak random is fine for games
		targets, err := s.activeChatterTracker.GetRandomTargets(platform, numTargets)
		if err != nil {
			log.Warn("No active targets available for TNT", "error", err)
			return "", errors.New(ErrMsgNoActiveTargets)
		}

		// Remove item from inventory
		utils.RemoveFromSlot(inventory, itemSlotIndex, quantity)

		// Apply timeout to all targets and collect names
		hitUsernames := make([]string, 0, len(targets))
		for _, target := range targets {
			if err := s.TimeoutUser(ctx, target.Username, timeout, MsgBlasterReasonBy+username); err != nil {
				log.Error(LogWarnFailedToTimeoutUser, "error", err, "target", target.Username)
				// Continue with other targets even if one fails
			}

			// Remove from active chatters
			s.activeChatterTracker.Remove(platform, target.UserID)
			hitUsernames = append(hitUsernames, target.Username)
		}

		log.Info("TNT hit multiple targets", "count", len(hitUsernames), "targets", hitUsernames)

		// Format message with all hit users
		targetsStr := formatTargetList(hitUsernames)
		return fmt.Sprintf("%s used %s! Hit %d targets: %s! Timed out for %v.",
			username, displayName, len(hitUsernames), targetsStr, timeout), nil
	}

	// Special handling for grenade - single random target
	if item.InternalName == domain.ItemGrenade {
		log.Info("Grenade used, selecting single random target")

		randomUsername, randomUserID, err := s.activeChatterTracker.GetRandomTarget(platform)
		if err != nil {
			log.Warn("No active targets available for grenade", "error", err)
			return "", errors.New(ErrMsgNoActiveTargets)
		}

		// Remove item from inventory
		utils.RemoveFromSlot(inventory, itemSlotIndex, quantity)

		// Apply timeout
		if err := s.TimeoutUser(ctx, randomUsername, timeout, MsgBlasterReasonBy+username); err != nil {
			log.Error(LogWarnFailedToTimeoutUser, "error", err, "target", randomUsername)
			// Continue anyway, as the item was used
		}

		// Remove from active chatters
		s.activeChatterTracker.Remove(platform, randomUserID)
		log.Info("Grenade hit target", "target", randomUsername)

		return fmt.Sprintf("%s used %s! Hit random target: %s! Timed out for %v.",
			username, displayName, randomUsername, timeout), nil
	}

	// Standard weapons require a user-provided target
	if !targetProvided || targetUsername == "" {
		log.Warn(LogWarnTargetUsernameMissingWeapon)
		return "", errors.New(ErrMsgTargetUsernameRequired)
	}

	// Remove item from inventory
	utils.RemoveFromSlot(inventory, itemSlotIndex, quantity)

	// Apply timeout
	if err := s.TimeoutUser(ctx, targetUsername, timeout, MsgBlasterReasonBy+username); err != nil {
		log.Error(LogWarnFailedToTimeoutUser, "error", err, "target", targetUsername)
		// Continue anyway, as the item was used
	}

	log.Info(LogMsgWeaponUsed, "target", targetUsername, "item", item.InternalName, "quantity", quantity)
	return fmt.Sprintf("%s used %s on %s! %d %s(s) fired. Timed out for %v.", username, displayName, targetUsername, quantity, displayName, timeout), nil
}

// formatTargetList formats a list of usernames for display
func formatTargetList(usernames []string) string {
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
		return "", errors.New(ErrMsgTargetUsernameRequiredRevive)
	}
	username, _ := args[ArgsUsername].(string)

	// Find item slot (randomly if multiple exist with different qualities)
	itemSlotIndex, slotQuantity := utils.FindRandomSlot(inventory, item.ID, s.rnd)
	if itemSlotIndex == -1 {
		log.Warn(LogWarnReviveNotInInventory, "item", item.InternalName)
		return "", errors.New(ErrMsgItemNotFoundInInventory)
	}
	if slotQuantity < quantity {
		log.Warn(LogWarnNotEnoughRevives, "item", item.InternalName)
		return "", errors.New(ErrMsgNotEnoughItemsInInventory)
	}
	utils.RemoveFromSlot(inventory, itemSlotIndex, quantity)

	// Get recovery time for this revive type
	qualityLevel := inventory.Slots[itemSlotIndex].QualityLevel
	recovery := getReviveRecovery(item.InternalName) + qualityLevel.GetTimeoutAdjustment()
	totalRecovery := time.Duration(quantity) * recovery

	// Reduce timeout for target user
	if err := s.ReduceTimeout(ctx, targetUsername, totalRecovery); err != nil {
		log.Error(LogWarnFailedToReduceTimeout, "error", err, "target", targetUsername)
		// Continue anyway, as the item was used
	}

	displayName := s.namingResolver.GetDisplayName(item.InternalName, qualityLevel)
	log.Info(LogMsgReviveUsed, "target", targetUsername, "item", item.InternalName, "quantity", quantity)
	return fmt.Sprintf("%s used %d %s on %s! Reduced timeout by %v.", username, quantity, displayName, targetUsername, totalRecovery), nil
}

func (s *service) handleTrap(ctx context.Context, _ *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgHandleTrapCalled, "item", item.InternalName, "quantity", quantity)

	if quantity < 1 {
		return "", errors.New(ErrMsgInvalidQuantity)
	}

	platform, _ := args[ArgsPlatform].(string)
	if platform == "" {
		platform = domain.PlatformTwitch
	}

	potentialTargets, isMine, err := s.getTrapTargets(ctx, item, quantity, user, platform, args)
	if err != nil {
		return "", err
	}

	if err := s.validateTrapInventory(inventory, item, quantity, isMine); err != nil {
		log.Warn("Trap inventory validation failed", "item", item.InternalName, "error", err)
		return "", err
	}

	itemsConsumed, trapsPlaced, selfTriggered, badLuckSelf, err := s.executeTrapTransaction(ctx, user, item, quantity, platform, potentialTargets, isMine)
	if err != nil {
		return "", err
	}

	return s.formatTrapResponse(user, item, potentialTargets, itemsConsumed, trapsPlaced, selfTriggered, badLuckSelf), nil
}

func (s *service) getTrapTargets(ctx context.Context, item *domain.Item, quantity int, user *domain.User, platform string, args map[string]interface{}) ([]string, bool, error) {
	log := logger.FromContext(ctx)
	if item.InternalName == domain.ItemMine {
		log.Info("Mine used, selecting random targets", "count", quantity)
		targets, err := s.activeChatterTracker.GetRandomTargets(platform, quantity)
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

	targetUsername, ok := args[ArgsTargetUsername].(string)
	if !ok || targetUsername == "" {
		return nil, false, errors.New(ErrMsgTargetUsernameRequired)
	}
	return []string{targetUsername}, false, nil
}

func (s *service) validateTrapInventory(inventory *domain.Inventory, item *domain.Item, quantity int, isMine bool) error {
	itemSlotIndex, slotQuantity := utils.FindRandomSlot(inventory, item.ID, s.rnd)
	if itemSlotIndex == -1 {
		return errors.New(ErrMsgItemNotFoundInInventory)
	}
	if slotQuantity < 1 {
		return errors.New(ErrMsgNotEnoughItemsInInventory)
	}
	if !isMine && slotQuantity < quantity {
		return errors.New(ErrMsgNotEnoughItemsInInventory)
	}
	return nil
}

func (s *service) executeTrapTransaction(ctx context.Context, user *domain.User, item *domain.Item, quantity int, platform string, potentialTargets []string, isMine bool) (int, int, bool, bool, error) {
	itemsConsumed := 0
	trapsPlaced := 0
	selfTriggered := false
	badLuckSelf := false

	err := s.withTx(ctx, func(tx repository.UserTx) error {
		txInventory, err := tx.GetInventory(ctx, user.ID)
		if err != nil {
			return fmt.Errorf("failed to get inventory: %w", err)
		}

		txSlotIndex, txSlotQty := utils.FindRandomSlot(txInventory, item.ID, s.rnd)
		if txSlotIndex == -1 || txSlotQty < 1 {
			return fmt.Errorf("item no longer available")
		}

		maxPossible := quantity
		if txSlotQty < maxPossible {
			maxPossible = txSlotQty
		}

		if isMine && len(potentialTargets) == 1 && strings.EqualFold(potentialTargets[0], user.Username) {
			// Special case: if we ONLY targeted ourselves (e.g. no other active chatters)
			badLuckSelf = true
			itemsConsumed = 1
			trapsPlaced = 1
			s.handleSelfTargetMine(ctx, user)
		} else {
			consumed, placed, triggered, badLuck, err := s.processTrapTargets(ctx, user, potentialTargets, platform, maxPossible, txInventory.Slots[txSlotIndex].QualityLevel, isMine)
			if err != nil {
				return err
			}
			itemsConsumed = consumed
			trapsPlaced = placed
			selfTriggered = triggered
			badLuckSelf = badLuck
		}

		if itemsConsumed > 0 {
			utils.RemoveFromSlot(txInventory, txSlotIndex, itemsConsumed)
			if err := tx.UpdateInventory(ctx, user.ID, *txInventory); err != nil {
				return fmt.Errorf("failed to update inventory: %w", err)
			}
		}
		return nil
	})

	return itemsConsumed, trapsPlaced, selfTriggered, badLuckSelf, err
}

func (s *service) processTrapTargets(ctx context.Context, user *domain.User, potentialTargets []string, platform string, maxPossible int, qualityLevel domain.QualityLevel, isMine bool) (int, int, bool, bool, error) {
	log := logger.FromContext(ctx)
	itemsConsumed := 0
	trapsPlaced := 0
	selfTriggered := false
	badLuckSelf := false

	if qualityLevel == "" {
		qualityLevel = domain.QualityCommon
	}

	for _, targetName := range potentialTargets {
		if itemsConsumed >= maxPossible {
			break
		}

		if isMine && strings.EqualFold(targetName, user.Username) {
			badLuckSelf = true
			itemsConsumed++
			trapsPlaced++
			s.handleSelfTargetMine(ctx, user)
			break
		}

		targetUser, err := s.repo.GetUserByPlatformUsername(ctx, platform, targetName)
		if err != nil {
			log.Warn("Target user not found, skipping", "username", targetName)
			continue
		}

		targetUserID, _ := uuid.Parse(targetUser.ID)
		existingTrap, err := s.trapRepo.GetActiveTrapForUpdate(ctx, targetUserID)
		if err != nil {
			return itemsConsumed, trapsPlaced, selfTriggered, false, fmt.Errorf("failed to check existing trap: %w", err)
		}

		if existingTrap != nil {
			itemsConsumed++
			selfTriggered = true
			if err := s.handleExistingTrapTrigger(ctx, user, targetName, existingTrap); err != nil {
				return itemsConsumed, trapsPlaced, true, false, err
			}
			break
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
		if err := s.trapRepo.CreateTrap(ctx, newTrap); err != nil {
			return itemsConsumed, trapsPlaced, false, false, fmt.Errorf("failed to create trap: %w", err)
		}
	}
	return itemsConsumed, trapsPlaced, selfTriggered, badLuckSelf, nil
}

func (s *service) formatTrapResponse(user *domain.User, item *domain.Item, potentialTargets []string, itemsConsumed, trapsPlaced int, selfTriggered, badLuckSelf bool) string {
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
	if len(potentialTargets) == 1 {
		targetMsg = potentialTargets[0]
	} else if trapsPlaced > 0 {
		targetMsg = fmt.Sprintf("%d people", trapsPlaced)
	}
	return fmt.Sprintf("%s set %d %s for %s!", user.Username, trapsPlaced, item.PublicName, targetMsg)
}

func (s *service) handleSelfTargetMine(ctx context.Context, user *domain.User) {
	log := logger.FromContext(ctx)
	if err := s.TimeoutUser(ctx, user.Username, 60*time.Second, "tripped on your own mine immediately!"); err != nil {
		log.Error("Failed to timeout user for bad luck", "error", err)
	}
}

func (s *service) handleExistingTrapTrigger(ctx context.Context, user *domain.User, targetName string, existingTrap *domain.Trap) error {
	timeout := time.Duration(existingTrap.CalculateTimeout()) * time.Second
	if err := s.TimeoutUser(ctx, user.Username, timeout, fmt.Sprintf("stepped on %s's trap entirely by accident!", targetName)); err != nil {
		return fmt.Errorf("failed to apply self-trap timeout: %w", err)
	}
	if err := s.trapRepo.TriggerTrap(ctx, existingTrap.ID); err != nil {
		return fmt.Errorf("failed to trigger existing trap: %w", err)
	}

	s.recordTrapSelfTriggerStats(ctx, user, targetName, existingTrap)
	return nil
}

func (s *service) recordTrapSelfTriggerStats(ctx context.Context, user *domain.User, targetName string, trap *domain.Trap) {
	if s.statsService == nil {
		return
	}
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
	_ = s.statsService.RecordUserEvent(ctx, user.ID, domain.EventTrapSelfTriggered, eventData.ToMap())
}

func (s *service) handleShield(ctx context.Context, _ *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, _ map[string]interface{}) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgHandleShieldCalled, "item", item.InternalName, "quantity", quantity)

	// Find item slot (randomly if multiple exist with different qualities)
	itemSlotIndex, slotQuantity := utils.FindRandomSlot(inventory, item.ID, s.rnd)
	if itemSlotIndex == -1 {
		log.Warn(LogWarnShieldNotInInventory)
		return "", errors.New(ErrMsgItemNotFoundInInventory)
	}
	if slotQuantity < quantity {
		log.Warn(LogWarnNotEnoughShields)
		return "", errors.New(ErrMsgNotEnoughItemsInInventory)
	}
	utils.RemoveFromSlot(inventory, itemSlotIndex, quantity)

	// Determine if this is a mirror shield
	isMirror := item.InternalName == domain.ItemMirrorShield

	// Apply shield status to user
	if err := s.ApplyShield(ctx, user, quantity, isMirror); err != nil {
		log.Error(LogWarnFailedToApplyShield, "error", err)
		return "", errors.New(ErrMsgFailedToApplyShield)
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
		return "", errors.New(ErrMsgJobNameRequired)
	}

	// Find item slot (randomly if multiple exist with different qualities)
	itemSlotIndex, slotQuantity := utils.FindRandomSlot(inventory, item.ID, s.rnd)
	if itemSlotIndex == -1 {
		log.Warn(LogWarnRareCandyNotInInventory)
		return "", errors.New(ErrMsgItemNotFoundInInventory)
	}
	if slotQuantity < quantity {
		log.Warn(LogWarnNotEnoughRareCandy)
		return "", errors.New(ErrMsgNotEnoughItemsInInventory)
	}
	utils.RemoveFromSlot(inventory, itemSlotIndex, quantity)

	// Award XP to the specified job via event
	totalXP := quantity * rarecandyXPAmount
	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.Event{
			Version: "1.1",
			Type:    domain.EventTypeItemUsed,
			Payload: domain.ItemUsedPayload{
				UserID:   user.ID,
				ItemName: item.InternalName,
				Quantity: quantity,
				Metadata: map[string]interface{}{
					"job_name": jobName,
					"xp_total": totalXP,
					"source":   job.SourceRareCandy,
				},
				Timestamp: time.Now().Unix(),
			},
		})
	}

	log.Info(LogMsgRareCandyUsed, "job", jobName, "xp", totalXP, "quantity", quantity)
	return fmt.Sprintf("Used %d rare candy! Granted %d XP to %s.", quantity, totalXP, jobName), nil
}

func (s *service) handleResourceGenerator(ctx context.Context, _ *service, _ *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgResourceGeneratorCalled, "item", item.InternalName, "quantity", quantity)

	username, _ := args[ArgsUsername].(string)

	// Find item slot (randomly if multiple exist with different qualities)
	itemSlotIndex, slotQuantity := utils.FindRandomSlot(inventory, item.ID, s.rnd)
	if itemSlotIndex == -1 {
		return "", errors.New(ErrMsgItemNotFoundInInventory)
	}
	if slotQuantity < quantity {
		return "", errors.New(ErrMsgNotEnoughItemsInInventory)
	}
	utils.RemoveFromSlot(inventory, itemSlotIndex, quantity)

	// Generate sticks (shovel generates 2 sticks per use)
	stickItem, err := s.getItemByNameCached(ctx, domain.ItemStick)
	if err != nil {
		return "", fmt.Errorf("failed to get stick item: %w", err)
	}

	sticksGenerated := quantity * ShovelSticksPerUse
	utils.AddItemsToInventory(inventory, []domain.InventorySlot{
		{ItemID: stickItem.ID, Quantity: sticksGenerated, QualityLevel: domain.QualityCommon},
	}, nil)

	displayName := s.namingResolver.GetDisplayName(domain.ItemStick, "")
	return fmt.Sprintf("%s%d %s!", username+MsgShovelUsed, sticksGenerated, displayName), nil
}

func (s *service) handleUtility(ctx context.Context, _ *service, _ *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgUtilityCalled, "item", item.InternalName, "quantity", quantity)

	username, _ := args[ArgsUsername].(string)

	// Find item slot (randomly if multiple exist with different qualities)
	itemSlotIndex, slotQuantity := utils.FindRandomSlot(inventory, item.ID, s.rnd)
	if itemSlotIndex == -1 {
		return "", errors.New(ErrMsgItemNotFoundInInventory)
	}
	if slotQuantity < quantity {
		return "", errors.New(ErrMsgNotEnoughItemsInInventory)
	}
	utils.RemoveFromSlot(inventory, itemSlotIndex, quantity)

	return username + MsgStickUsed, nil
}

// pluralize handles simple pluralization for game items and phrases
func (s *service) pluralize(name string, quantity int) string {
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
		return s.pluralize(parts[0], quantity) + " of " + parts[1] + suffix
	}

	// Common uncountable or collective nouns in game context
	lower := strings.ToLower(baseName)
	switch lower {
	case "money", "ghost-gold", "coins", "scrap", "junk", "credits":
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

// getIndefiniteArticle returns "a" or "an" based on the first letter of the word
func getIndefiniteArticle(word string) string {
	if len(word) == 0 {
		return "a"
	}
	// Simplified a/an logic logic
	first := strings.ToLower(string(word[0]))
	if strings.ContainsAny(first, "aeiou") {
		return "an"
	}
	return "a"
}
