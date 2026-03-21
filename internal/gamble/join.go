package gamble

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// JoinGamble adds a user to an existing gamble
func (s *service) JoinGamble(ctx context.Context, gambleID uuid.UUID, platform, platformID, username string) error {
	log := logger.FromContext(ctx)
	log.Info(LogMsgJoinGambleCalled, "gambleID", gambleID, "username", username)

	// Get User
	user, err := s.getAndValidateGambleUser(ctx, platform, platformID)
	if err != nil {
		return err
	}

	// Get Gamble
	gamble, err := s.getAndValidateActiveGamble(ctx, gambleID)
	if err != nil {
		return err
	}

	// Get initiator's bets to use for this joiner
	var initialBets []domain.LootboxBet
	for _, p := range gamble.Participants {
		if p.UserID == gamble.InitiatorID {
			initialBets = p.LootboxBets
			break
		}
	}

	if len(initialBets) == 0 {
		return fmt.Errorf("failed to find initiator bets for gamble %s: %w", gambleID, domain.ErrGambleNotFound)
	}

	// Create a deep copy of bets to use for this joiner to avoid side effects
	joinerBets := make([]domain.LootboxBet, len(initialBets))
	copy(joinerBets, initialBets)

	// Note: Duplicate join prevention is enforced by database constraint
	// (idx_gamble_participants_unique_user on gamble_participants table)

	// Validate bets and resolve item names to IDs
	resolvedItemIDs, err := s.validateGambleBets(ctx, joinerBets)
	if err != nil {
		return err
	}

	// Execute transaction
	if err := s.executeGambleJoinTx(ctx, user.ID, gamble.ID, username, joinerBets, resolvedItemIDs); err != nil {
		return err
	}

	// Publish gamble participated event (job handler awards XP)
	s.publishGambleParticipatedEvent(ctx, gambleID.String(), user.ID, calculateTotalLootboxes(joinerBets), "join")

	return nil
}

// executeGambleJoinTx encapsulates the transactional logic for joining a gamble
func (s *service) executeGambleJoinTx(ctx context.Context, userID string, gambleID uuid.UUID, username string, joinerBets []domain.LootboxBet, resolvedItemIDs []int) error {
	tx, err := s.repo.BeginGambleTx(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToBeginTx, err)
	}
	defer repository.SafeRollback(ctx, tx)

	// Get Inventory
	inventory, err := tx.GetInventory(ctx, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToGetInventory, err)
	}

	// Consume Bets using resolved item IDs
	for i := range joinerBets {
		itemID := resolvedItemIDs[i]
		qualityLevel, err := consumeItem(inventory, itemID, joinerBets[i].Quantity)
		if err != nil {
			return fmt.Errorf("%s (item %d): %w", ErrContextFailedToConsumeBet, itemID, err)
		}
		joinerBets[i].QualityLevel = qualityLevel
	}

	// Update Inventory
	if err := tx.UpdateInventory(ctx, userID, *inventory); err != nil {
		return fmt.Errorf("%s: %w", ErrContextFailedToUpdateInventory, err)
	}

	// Add Participant
	participant := &domain.Participant{
		GambleID:    gambleID,
		UserID:      userID,
		LootboxBets: joinerBets,
		Username:    username,
	}
	if err := s.repo.JoinGamble(ctx, participant); err != nil {
		if errors.Is(err, domain.ErrUserAlreadyJoined) {
			return domain.ErrUserAlreadyJoined
		}
		return fmt.Errorf("%s: %w", ErrContextFailedToJoinGamble, err)
	}

	return tx.Commit(ctx)
}
