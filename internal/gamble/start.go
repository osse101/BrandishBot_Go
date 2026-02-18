package gamble

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Ensure naming.Resolver is used (it's referenced in resolveItemName).
var _ naming.Resolver

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

	if err := s.executeGambleStartTx(ctx, user.ID, username, bets, resolvedItemIDs, gamble); err != nil {
		return nil, err
	}

	s.publishGambleStartedEvent(ctx, gamble)
	s.publishGambleParticipatedEvent(ctx, gamble.ID.String(), user.ID, calculateTotalLootboxes(bets), "start")

	return gamble, nil
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

func (s *service) executeGambleStartTx(ctx context.Context, userID, username string, bets []domain.LootboxBet, resolvedItemIDs []int, gamble *domain.Gamble) error {
	tx, err := s.repo.BeginGambleTx(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToBeginTx, err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Get Inventory with lock
	inventory, err := tx.GetInventory(ctx, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToGetInventory, err)
	}

	// Create a local copy of bets to avoid modifying the caller's slice and race conditions
	gambleBets := make([]domain.LootboxBet, len(bets))
	copy(gambleBets, bets)

	// Consume bet items from inventory using resolved IDs
	for i := range gambleBets {
		itemID := resolvedItemIDs[i]
		qualityLevel, err := consumeItem(inventory, itemID, gambleBets[i].Quantity)
		if err != nil {
			return fmt.Errorf("%s (item %d): %w", ErrContextFailedToConsumeBet, itemID, err)
		}
		gambleBets[i].QualityLevel = qualityLevel
	}

	if err := tx.UpdateInventory(ctx, userID, *inventory); err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToUpdateInventory, err)
	}

	if err := s.repo.CreateGamble(ctx, gamble); err != nil {
		if errors.Is(err, domain.ErrGambleAlreadyActive) {
			return domain.ErrGambleAlreadyActive
		}
		return fmt.Errorf("%s: %w", ErrContextFailedToCreateGamble, err)
	}

	participant := &domain.Participant{
		GambleID:    gamble.ID,
		UserID:      userID,
		LootboxBets: gambleBets,
		Username:    username,
	}

	if err := s.repo.JoinGamble(ctx, participant); err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToAddInitiator, err)
	}

	return tx.Commit(ctx)
}
