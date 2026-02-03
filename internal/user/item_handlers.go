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
	shineLevel, err := s.consumeLootboxFromInventory(inventory, lootboxItem, quantity)
	if err != nil {
		return "", err
	}

	// 2. Use lootbox service to open lootboxes
	drops, err := s.lootboxService.OpenLootbox(ctx, lootboxItem.InternalName, quantity, shineLevel)
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

func (s *service) consumeLootboxFromInventory(inventory *domain.Inventory, item *domain.Item, quantity int) (domain.ShineLevel, error) {
	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
	if itemSlotIndex == -1 {
		return "", errors.New(ErrMsgItemNotFoundInInventory)
	}

	if slotQuantity < quantity {
		return "", errors.New(ErrMsgNotEnoughItemsInInventory)
	}

	shineLevel := inventory.Slots[itemSlotIndex].ShineLevel
	utils.RemoveFromSlot(inventory, itemSlotIndex, quantity)
	return shineLevel, nil
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
		if drop.ShineLevel == domain.ShineLegendary {
			stats.hasLegendary = true
		} else if drop.ShineLevel == domain.ShineEpic {
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

		// Get display name (which might be "Shiny credit" for money or "Ray Gun" for blaster)
		// We trust the resolver to give the base name, and we handle pluralization
		itemDisplayName := s.namingResolver.GetDisplayName(drop.ItemName, drop.ShineLevel)

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

	// Find item slot first (before target selection)
	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
	if itemSlotIndex == -1 {
		log.Warn(LogWarnWeaponNotInInventory, "item", item.InternalName)
		return "", errors.New(ErrMsgItemNotFoundInInventory)
	}
	if slotQuantity < quantity {
		log.Warn(LogWarnNotEnoughWeapons, "item", item.InternalName)
		return "", errors.New(ErrMsgNotEnoughItemsInInventory)
	}

	shineLevel := inventory.Slots[itemSlotIndex].ShineLevel
	timeout := getWeaponTimeout(item.InternalName) + shineLevel.GetTimeoutAdjustment()
	displayName := s.namingResolver.GetDisplayName(item.InternalName, shineLevel)

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

	// Find item slot
	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
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
	shineLevel := inventory.Slots[itemSlotIndex].ShineLevel
	recovery := getReviveRecovery(item.InternalName) + shineLevel.GetTimeoutAdjustment()
	totalRecovery := time.Duration(quantity) * recovery

	// Reduce timeout for target user
	if err := s.ReduceTimeout(ctx, targetUsername, totalRecovery); err != nil {
		log.Error(LogWarnFailedToReduceTimeout, "error", err, "target", targetUsername)
		// Continue anyway, as the item was used
	}

	displayName := s.namingResolver.GetDisplayName(item.InternalName, shineLevel)
	log.Info(LogMsgReviveUsed, "target", targetUsername, "item", item.InternalName, "quantity", quantity)
	return fmt.Sprintf("%s used %d %s on %s! Reduced timeout by %v.", username, quantity, displayName, targetUsername, totalRecovery), nil
}

func (s *service) handleTrap(ctx context.Context, _ *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, args map[string]interface{}) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgHandleTrapCalled, "item", item.InternalName, "quantity", quantity)

	// 1. Validate quantity
	// 1. Validate input
	if quantity < 1 {
		return "", errors.New(ErrMsgInvalidQuantity)
	}

	// 2. Extract common args
	platform, _ := args[ArgsPlatform].(string)
	if platform == "" {
		platform = domain.PlatformTwitch // default
	}

	// 3. Determine targets based on item type
	var potentialTargets []string
	var isMine bool

	if item.InternalName == domain.ItemMine {
		isMine = true
		log.Info("Mine used, selecting random targets", "count", quantity)

		// Get random targets
		// We request 'quantity' targets to try and use all mines
		targets, err := s.activeChatterTracker.GetRandomTargets(platform, quantity)
		if err != nil {
			// No targets found, add self as single target (fallback)
			potentialTargets = []string{user.Username}
		} else {
			for _, t := range targets {
				potentialTargets = append(potentialTargets, t.Username)
			}
			// If we got fewer targets than quantity, we just use what we found
			if len(potentialTargets) == 0 {
				potentialTargets = []string{user.Username}
			}
		}
	} else {
		// Standard trap requires explicit target and single quantity
		if quantity != 1 {
			return "", errors.New("can only use 1 trap at a time")
		}

		targetUsername, ok := args[ArgsTargetUsername].(string)
		if !ok || targetUsername == "" {
			log.Warn(LogWarnTargetUsernameMissingTrap)
			return "", errors.New(ErrMsgTargetUsernameRequired)
		}
		potentialTargets = []string{targetUsername}
	}

	// 4. Find item in inventory (initial check)
	// We need at least 1 item to start
	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
	if itemSlotIndex == -1 {
		log.Warn(LogWarnTrapNotInInventory, "item", item.InternalName)
		return "", errors.New(ErrMsgItemNotFoundInInventory)
	}
	if slotQuantity < 1 { // Changed from < quantity because we might use fewer
		log.Warn(LogWarnNotEnoughTraps, "item", item.InternalName)
		return "", errors.New(ErrMsgNotEnoughItemsInInventory)
	}
	// Also ensure we have enough for the requested quantity if it's a Trap (mines adjust)
	if !isMine && slotQuantity < quantity {
		return "", errors.New(ErrMsgNotEnoughItemsInInventory)
	}

	// 5. Execute Loop
	itemsConsumed := 0
	trapsPlaced := 0
	selfTriggered := false
	badLuckSelf := false // Self selected by random targeting

	err := s.withTx(ctx, func(tx repository.UserTx) error {
		// Re-fetch inventory inside transaction
		txInventory, err := tx.GetInventory(ctx, user.ID)
		if err != nil {
			return fmt.Errorf("failed to get inventory: %w", err)
		}

		// Re-validate inventory
		txSlotIndex, txSlotQty := utils.FindSlot(txInventory, item.ID)
		if txSlotIndex == -1 || txSlotQty < 1 {
			return fmt.Errorf("item no longer available")
		}

		// Cap targets by available inventory
		maxPossible := txSlotQty
		if quantity < maxPossible {
			maxPossible = quantity
		}

		// Loop through targets
		for _, targetName := range potentialTargets {
			if itemsConsumed >= maxPossible {
				break
			}

			// Check for "bad luck" self-selection (only relevant for mines)
			if isMine && strings.EqualFold(targetName, user.Username) {
				// We targeted ourselves randomly!
				// Consume the item, apply self-trap (if we want that logic), or just break
				// Requirement: "break if self is selected randomly"
				badLuckSelf = true
				itemsConsumed++ // We use the mine that blew us up
				trapsPlaced++   // Technically we placed it on ourselves

				// Apply timeout for bad luck
				if err := s.TimeoutUser(ctx, user.Username, 60*time.Second, "tripped on your own mine immediately!"); err != nil {
					log.Error("Failed to timeout user for bad luck", "error", err)
				}

				break // Stop processing
			}

			targetUser, err := s.repo.GetUserByPlatformUsername(ctx, platform, targetName)
			if err != nil {
				log.Warn("Target user not found, skipping", "username", targetName)
				continue
			}

			targetUserID, _ := uuid.Parse(targetUser.ID)
			existingTrap, err := s.trapRepo.GetActiveTrapForUpdate(ctx, targetUserID)
			if err != nil {
				return fmt.Errorf("failed to check existing trap: %w", err)
			}

			// If trap exists, trigger it on setter (Self-Trigger Logic)
			if existingTrap != nil {
				itemsConsumed++ // Triggering a trap consumes the item used to trigger it
				selfTriggered = true

				timeout := time.Duration(existingTrap.CalculateTimeout()) * time.Second
				if err := s.TimeoutUser(ctx, user.Username, timeout,
					fmt.Sprintf("stepped on %s's trap entirely by accident!", targetName)); err != nil {
					return fmt.Errorf("failed to apply self-trap timeout: %w", err)
				}

				if err := s.trapRepo.TriggerTrap(ctx, existingTrap.ID); err != nil {
					return fmt.Errorf("failed to trigger existing trap: %w", err)
				}

				// Record event
				if s.statsService != nil {
					eventData := &domain.TrapTriggeredData{
						TrapID:           existingTrap.ID,
						SetterID:         existingTrap.SetterID,
						SetterUsername:   targetName,
						TargetID:         existingTrap.TargetID,
						TargetUsername:   user.Username,
						ShineLevel:       existingTrap.ShineLevel,
						TimeoutSeconds:   existingTrap.CalculateTimeout(),
						WasSelfTriggered: true,
					}
					_ = s.statsService.RecordUserEvent(ctx, user.ID, domain.EventTrapSelfTriggered, eventData.ToMap())
				}

				break // Stop processing after self-trigger
			}

			// Place new trap
			itemsConsumed++
			trapsPlaced++

			// Get shine level (safe read)
			shineLevel := txInventory.Slots[txSlotIndex].ShineLevel
			if shineLevel == "" {
				shineLevel = domain.ShineCommon
			}

			userID, _ := uuid.Parse(user.ID)
			newTrap := &domain.Trap{
				ID:             uuid.New(),
				SetterID:       userID,
				TargetID:       targetUserID,
				ShineLevel:     shineLevel,
				TimeoutSeconds: 60,
				PlacedAt:       time.Now(),
			}

			if err := s.trapRepo.CreateTrap(ctx, newTrap); err != nil {
				return fmt.Errorf("failed to create trap: %w", err)
			}
		}

		// Dedudct consumed items
		if itemsConsumed > 0 {
			utils.RemoveFromSlot(txInventory, txSlotIndex, itemsConsumed)
			if err := tx.UpdateInventory(ctx, user.ID, *txInventory); err != nil {
				return fmt.Errorf("failed to update inventory: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	// Construct message
	if selfTriggered {
		return fmt.Sprintf("%s tried to use %s but stepped on a trap!", user.Username, item.PublicName), nil
	}
	if badLuckSelf {
		return fmt.Sprintf("%s dropped a mine straight on their own foot!", user.Username), nil
	}

	if trapsPlaced == 0 && itemsConsumed == 0 {
		return "No targets found.", nil
	}

	targetMsg := "someone"
	if len(potentialTargets) == 1 {
		targetMsg = potentialTargets[0]
	} else if trapsPlaced > 0 {
		targetMsg = fmt.Sprintf("%d people", trapsPlaced)
	}

	return fmt.Sprintf("%s set %d %s for %s!", user.Username, trapsPlaced, item.PublicName, targetMsg), nil

}

func (s *service) handleShield(ctx context.Context, _ *service, user *domain.User, inventory *domain.Inventory, item *domain.Item, quantity int, _ map[string]interface{}) (string, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgHandleShieldCalled, "item", item.InternalName, "quantity", quantity)

	// Find item slot
	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
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

	// Find item slot
	itemSlotIndex, slotQuantity := utils.FindSlot(inventory, item.ID)
	if itemSlotIndex == -1 {
		log.Warn(LogWarnRareCandyNotInInventory)
		return "", errors.New(ErrMsgItemNotFoundInInventory)
	}
	if slotQuantity < quantity {
		log.Warn(LogWarnNotEnoughRareCandy)
		return "", errors.New(ErrMsgNotEnoughItemsInInventory)
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
			return "", errors.New(ErrMsgFailedToAwardXP)
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
		{ItemID: stickItem.ID, Quantity: sticksGenerated, ShineLevel: domain.ShineCommon},
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

	// Check for shine emojis at the end (Legendary/Cursed)
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
