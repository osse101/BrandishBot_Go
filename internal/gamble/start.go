package gamble

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Ensure naming.Resolver is used (it's referenced in resolveItemName).
var _ naming.Resolver

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

// StartGamble initiates a new gamble
func (s *service) StartGamble(ctx context.Context, platform, platformID, username string, bets []domain.LootboxBet) (*domain.Gamble, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgStartGambleCalled, "platform", platform, "platformID", platformID, "username", username, "bets", bets)

	if err := s.validateGambleStartInput(bets); err != nil {
		return nil, err
	}

	user, err := s.getAndValidateGambleUser(ctx, platform, platformID)
	if err != nil {
		return nil, err
	}

	if err := s.ensureNoActiveGamble(ctx); err != nil {
		return nil, err
	}

	gamble := s.createGambleRecord(user.ID)

	// Validate bets and resolve item names to IDs
	resolvedItemIDs, err := s.validateGambleBets(ctx, bets)
	if err != nil {
		return nil, err
	}

	inventory, err := s.repo.GetInventory(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrContextFailedToGetInventory, err)
	}

	// Create a local copy of bets to avoid modifying the caller's slice and race conditions
	gambleBets := make([]domain.LootboxBet, len(bets))
	copy(gambleBets, bets)

	// Consume bet items from inventory using resolved IDs
	for i := range gambleBets {
		itemID := resolvedItemIDs[i]
		qualityLevel, err := consumeItem(inventory, itemID, gambleBets[i].Quantity)
		if err != nil {
			return nil, fmt.Errorf("%s (item %d): %w", ErrContextFailedToConsumeBet, itemID, err)
		}
		gambleBets[i].QualityLevel = qualityLevel
	}

	participant := &domain.Participant{
		GambleID:    gamble.ID,
		UserID:      user.ID,
		LootboxBets: gambleBets,
		Username:    username,
	}

	if err := s.executeGambleStartTx(ctx, user.ID, inventory, gamble, participant); err != nil {
		return nil, err
	}

	s.publishGambleStartedEvent(ctx, gamble)
	s.publishGambleParticipatedEvent(ctx, gamble.ID.String(), user.ID, calculateTotalLootboxes(gambleBets), "start")

	return gamble, nil
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

func (s *service) createGambleRecord(initiatorID string) *domain.Gamble {
	return &domain.Gamble{
		ID:           uuid.New(),
		InitiatorID:  initiatorID,
		State:        domain.GambleStateJoining,
		CreatedAt:    time.Now(),
		JoinDeadline: time.Now().Add(s.joinDuration),
	}
}

func (s *service) executeGambleStartTx(ctx context.Context, userID string, inventory *domain.Inventory, gamble *domain.Gamble, participant *domain.Participant) error {
	tx, err := s.repo.BeginGambleTx(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToBeginTx, err)
	}
	defer repository.SafeRollback(ctx, tx)

	if err := tx.UpdateInventory(ctx, userID, *inventory); err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToUpdateInventory, err)
	}

	if err := s.repo.CreateGamble(ctx, gamble); err != nil {
		if errors.Is(err, domain.ErrGambleAlreadyActive) {
			return domain.ErrGambleAlreadyActive
		}
		return fmt.Errorf("%s: %w", ErrContextFailedToCreateGamble, err)
	}

	if err := s.repo.JoinGamble(ctx, participant); err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToAddInitiator, err)
	}

	return tx.Commit(ctx)
}

// Helper to consume item from inventory and return its quality level
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

// calculateTotalLootboxes sums up lootbox quantities from bets
func calculateTotalLootboxes(bets []domain.LootboxBet) int {
	total := 0
	for _, bet := range bets {
		total += bet.Quantity
	}
	return total
}

func (s *service) publishGambleStartedEvent(ctx context.Context, gamble *domain.Gamble) {
	if s.eventBus == nil {
		logger.FromContext(ctx).Error("Failed to publish "+LogContextGambleStartedEvent, "reason", LogReasonEventBusNil)
		return
	}
	err := s.eventBus.Publish(ctx, event.Event{
		Version: EventSchemaVersion,
		Type:    domain.EventGambleStarted,
		Payload: gamble,
	})
	if err != nil {
		logger.FromContext(ctx).Error("Failed to publish "+LogContextGambleStartedEvent, "error", err)
	}
}

func (s *service) publishGambleParticipatedEvent(ctx context.Context, gambleID, userID string, lootboxCount int, source string) {
	if s.resilientPublisher == nil {
		return
	}
	s.resilientPublisher.PublishWithRetry(ctx, event.Event{
		Version: EventSchemaVersion,
		Type:    event.Type(domain.EventTypeGambleParticipated),
		Payload: domain.GambleParticipatedPayload{
			GambleID:     gambleID,
			UserID:       userID,
			LootboxCount: lootboxCount,
			Source:       source,
			Timestamp:    time.Now().Unix(),
		},
	})
}
