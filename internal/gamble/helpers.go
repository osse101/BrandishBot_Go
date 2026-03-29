package gamble

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// consumeItem consumes an item from the inventory and returns its quality level
func consumeItem(inventory *domain.Inventory, itemID, quantity int) (domain.QualityLevel, error) {
	for i := range inventory.Slots {
		if inventory.Slots[i].ItemID == itemID {
			if inventory.Slots[i].Quantity < quantity {
				return "", domain.ErrInsufficientQuantity
			}
			qualityLevel := inventory.Slots[i].QualityLevel
			if inventory.Slots[i].Quantity == quantity {
				// Remove slot
				inventory.Slots = append(inventory.Slots[:i], inventory.Slots[i+1:]...)
			} else {
				// Reduce quantity
				inventory.Slots[i].Quantity -= quantity
			}
			return qualityLevel, nil
		}
	}
	return domain.QualityLevel(""), domain.ErrItemNotFound
}

func (s *service) awardItemsToWinner(ctx context.Context, tx repository.GambleTx, winnerID string, allOpenedItems []domain.GambleOpenedItem) error {
	inv, err := tx.GetInventory(ctx, winnerID)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToGetWinnerInv, err)
	}

	itemsToAdd := make(map[int]int)
	for _, item := range allOpenedItems {
		itemsToAdd[item.ItemID] += item.Quantity
	}

	for i, slot := range inv.Slots {
		if qty, ok := itemsToAdd[slot.ItemID]; ok {
			inv.Slots[i].Quantity += qty
			delete(itemsToAdd, slot.ItemID)
		}
	}

	var newItemIDs []int
	if len(itemsToAdd) > 0 {
		newItemIDs = make([]int, 0, len(itemsToAdd))
		for itemID := range itemsToAdd {
			newItemIDs = append(newItemIDs, itemID)
		}
		sort.Ints(newItemIDs)
	}

	for _, itemID := range newItemIDs {
		inv.Slots = append(inv.Slots, domain.InventorySlot{ItemID: itemID, Quantity: itemsToAdd[itemID]})
	}

	if err := tx.UpdateInventory(ctx, winnerID, *inv); err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToUpdateWinnerInv, err)
	}
	return nil
}

// resolveItemName attempts to resolve a user-provided item name to its internal name.
// It first tries the naming resolver, then falls back to using the input as-is.
// This allows users to use either public names ("junkbox") or internal names ("lootbox_tier0").
func (s *service) resolveItemName(ctx context.Context, itemName string) (string, error) {
	// Try naming resolver first (handles public names)
	if s.namingResolver != nil {
		if internalName, ok := s.namingResolver.ResolvePublicName(itemName); ok {
			return internalName, nil
		}
	}

	// Fall back - assume it's already an internal name
	// Validate by checking if item exists
	item, err := s.repo.GetItemByName(ctx, itemName)
	if err != nil {
		return "", fmt.Errorf("%s '%s': %w", ErrContextFailedToResolveItemName, itemName, err)
	}
	if item == nil {
		return "", fmt.Errorf("%w: %s (%s)", domain.ErrItemNotFound, itemName, ErrMsgItemNotFoundAsPublicOrInternalName)
	}

	return itemName, nil
}

// resolveLootboxBet resolves a bet's item name to its item ID
// Returns the resolved item ID or an error
func (s *service) resolveLootboxBet(ctx context.Context, bet domain.LootboxBet) (int, error) {
	// Resolve name to internal name
	internalName, err := s.resolveItemName(ctx, bet.ItemName)
	if err != nil {
		return 0, fmt.Errorf("%s '%s': %w", ErrContextFailedToResolveItemName, bet.ItemName, err)
	}

	// Get item by internal name to get ID
	item, err := s.repo.GetItemByName(ctx, internalName)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", ErrContextFailedToGetItem, err)
	}
	if item == nil {
		return 0, fmt.Errorf("%w: %s", domain.ErrItemNotFound, internalName)
	}

	// Validate it's a lootbox
	if len(item.InternalName) < LootboxPrefixLength || item.InternalName[:LootboxPrefixLength] != LootboxPrefix {
		return 0, fmt.Errorf("%w: %s (id:%d)", domain.ErrNotALootbox, item.InternalName, item.ID)
	}

	return item.ID, nil
}

// validateGambleBets validates bets and resolves item names to IDs
// Returns a slice of resolved item IDs corresponding to each bet
func (s *service) validateGambleBets(ctx context.Context, bets []domain.LootboxBet) ([]int, error) {
	resolvedItemIDs := make([]int, len(bets))
	for i, bet := range bets {
		if bet.Quantity > domain.MaxTransactionQuantity {
			return nil, fmt.Errorf("%w: max is %d", domain.ErrQuantityTooHigh, domain.MaxTransactionQuantity)
		}
		itemID, err := s.resolveLootboxBet(ctx, bet)
		if err != nil {
			return nil, err
		}
		resolvedItemIDs[i] = itemID
	}
	return resolvedItemIDs, nil
}

func (s *service) validateGambleStartInput(bets []domain.LootboxBet) error {
	if len(bets) == 0 {
		return domain.ErrAtLeastOneLootboxRequired
	}
	for _, bet := range bets {
		if bet.Quantity <= 0 {
			return domain.ErrBetQuantityMustBePositive
		}
		if bet.Quantity > domain.MaxTransactionQuantity {
			return fmt.Errorf("%w: max is %d", domain.ErrQuantityTooHigh, domain.MaxTransactionQuantity)
		}
	}
	return nil
}

func (s *service) getAndValidateGambleUser(ctx context.Context, platform, platformID string) (*domain.User, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrContextFailedToGetUser, err)
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

func (s *service) ensureNoActiveGamble(ctx context.Context) error {
	active, err := s.repo.GetActiveGamble(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToCheckActive, err)
	}
	if active != nil {
		return domain.ErrGambleAlreadyActive
	}
	return nil
}

func (s *service) validateGambleExecution(gamble *domain.Gamble) error {
	if gamble.State != domain.GambleStateJoining {
		return fmt.Errorf("%w (current: %s)", domain.ErrNotInJoiningState, gamble.State)
	}
	// Allow execution within grace period of deadline to handle clock skew/network delays
	deadlineWithGrace := gamble.JoinDeadline.Add(-ExecutionGracePeriod)
	if time.Now().Before(deadlineWithGrace) {
		return fmt.Errorf("%s (deadline: %v, grace_period: %v)", ErrMsgCannotExecuteBeforeDeadline, gamble.JoinDeadline, ExecutionGracePeriod)
	}
	return nil
}

func (s *service) getAndValidateActiveGamble(ctx context.Context, gambleID uuid.UUID) (*domain.Gamble, error) {
	gamble, err := s.repo.GetGamble(ctx, gambleID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrContextFailedToGetGamble, err)
	}
	if gamble == nil {
		return nil, domain.ErrGambleNotFound
	}
	if gamble.State != domain.GambleStateJoining {
		return nil, domain.ErrNotInJoiningState
	}
	if time.Now().After(gamble.JoinDeadline) {
		return nil, domain.ErrJoinDeadlinePassed
	}
	return gamble, nil
}
