package itemhandler

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

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
