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

// ExecuteGamble runs the gamble logic
func (s *service) ExecuteGamble(ctx context.Context, id uuid.UUID) (*domain.GambleResult, error) {
	log := logger.FromContext(ctx)
	log.Info(LogMsgExecuteGambleCalled, "gambleID", id)

	gamble, err := s.repo.GetGamble(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrContextFailedToGetGamble, err)
	}
	if gamble == nil {
		return nil, domain.ErrGambleNotFound
	}

	if gamble.State == domain.GambleStateCompleted {
		log.Info(LogMsgGambleAlreadyCompleted, "gambleID", id)
		return nil, nil
	}

	if err := s.validateGambleExecution(gamble); err != nil {
		return nil, err
	}

	tx, err := s.repo.BeginGambleTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrContextFailedToBeginTx, err)
	}
	defer repository.SafeRollback(ctx, tx)

	if err := s.transitionToOpeningState(ctx, tx, id, gamble.State); err != nil {
		return nil, err
	}

	// Minimum participant check (2+ required to gamble)
	if len(gamble.Participants) < 2 {
		log.Info("Gamble cancelled: not enough participants", "gambleID", id, "count", len(gamble.Participants))
		if err := s.refundGamble(ctx, tx, gamble); err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("%s: %w", ErrContextFailedToCommitTx, err)
		}
		return &domain.GambleResult{GambleID: id}, nil
	}

	userValues, allOpenedItems, totalGambleValue := s.openParticipantsLootboxes(ctx, gamble)

	// Determine critical failures (before determining winner)
	critFailUsers := s.determineCriticalFailures(userValues, totalGambleValue)

	if err := tx.SaveOpenedItems(ctx, allOpenedItems); err != nil {
		return nil, fmt.Errorf("failed to save opened items: %w", err)
	}

	winnerID, highestValue, tieBreakLostUsers := s.determineGambleWinners(userValues)
	nearMissUsers := s.determineNearMisses(winnerID, highestValue, userValues)

	if winnerID != "" {
		if err := s.awardItemsToWinner(ctx, tx, winnerID, allOpenedItems); err != nil {
			return nil, err
		}
	}

	result := &domain.GambleResult{
		GambleID:   id,
		WinnerID:   winnerID,
		TotalValue: totalGambleValue,
		Items:      allOpenedItems,
	}

	if err := tx.CompleteGamble(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to complete gamble: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%s: %w", ErrContextFailedToCommitTx, err)
	}

	// Publish gamble completion event with per-participant outcomes
	participants := s.buildParticipantOutcomes(gamble, userValues, winnerID, critFailUsers, tieBreakLostUsers, nearMissUsers)
	s.publishGambleCompletedEvent(ctx, result, len(gamble.Participants), participants)

	return result, nil
}

func (s *service) transitionToOpeningState(ctx context.Context, tx repository.GambleTx, id uuid.UUID, currentState domain.GambleState) error {
	if currentState == domain.GambleStateOpening {
		// Already in opening state, likely resuming after interruption
		return nil
	}
	rowsAffected, err := tx.UpdateGambleStateIfMatches(ctx, id, domain.GambleStateJoining, domain.GambleStateOpening)
	if err != nil {
		return fmt.Errorf("failed to transition gamble state: %w", err)
	}
	if rowsAffected == 0 {
		return errors.New(ErrMsgGambleAlreadyExecuted)
	}
	return nil
}

func (s *service) refundGamble(ctx context.Context, tx repository.GambleTx, gamble *domain.Gamble) error {
	for _, p := range gamble.Participants {
		inv, err := tx.GetInventory(ctx, p.UserID)
		if err != nil {
			return fmt.Errorf("failed to get inventory for refund (user:%s): %w", p.UserID, err)
		}

		for _, bet := range p.LootboxBets {
			// Resolve bet item name to ID
			itemID, err := s.resolveLootboxBet(ctx, bet)
			if err != nil {
				continue
			}

			// Add items back to inventory
			found := false
			for i, slot := range inv.Slots {
				if slot.ItemID == itemID {
					inv.Slots[i].Quantity += bet.Quantity
					found = true
					break
				}
			}
			if !found {
				inv.Slots = append(inv.Slots, domain.InventorySlot{
					ItemID:   itemID,
					Quantity: bet.Quantity,
				})
			}
		}

		if err := tx.UpdateInventory(ctx, p.UserID, *inv); err != nil {
			return fmt.Errorf("failed to update inventory for refund (user:%s): %w", p.UserID, err)
		}
	}

	if err := tx.RefundGamble(ctx, gamble.ID); err != nil {
		return fmt.Errorf("failed to mark gamble as refunded: %w", err)
	}

	s.publishGambleRefundedEvent(ctx, gamble)
	return nil
}

func (s *service) openParticipantsLootboxes(ctx context.Context, gamble *domain.Gamble) (map[string]int64, []domain.GambleOpenedItem, int64) {
	userValues := make(map[string]int64)
	var allOpenedItems []domain.GambleOpenedItem
	var totalGambleValue int64

	for _, p := range gamble.Participants {
		for _, bet := range p.LootboxBets {
			// Resolve bet item name to ID to get lootbox item
			itemID, err := s.resolveLootboxBet(ctx, bet)
			if err != nil {
				continue
			}

			lootboxItem, err := s.repo.GetItemByID(ctx, itemID)
			if err != nil || lootboxItem == nil {
				continue
			}

			drops, err := s.lootboxSvc.OpenLootbox(ctx, lootboxItem.InternalName, bet.Quantity, bet.QualityLevel)
			if err != nil {
				continue
			}

			for _, drop := range drops {
				totalValue := int64(drop.Value * drop.Quantity)
				if s.progressionSvc != nil {
					modifiedValue, err := s.progressionSvc.GetModifiedValue(ctx, "", ProgressionFeatureGambleWinBonus, float64(totalValue))
					if err == nil {
						totalValue = int64(modifiedValue)
					}
				}

				allOpenedItems = append(allOpenedItems, domain.GambleOpenedItem{
					GambleID:     gamble.ID,
					UserID:       p.UserID,
					ItemID:       drop.ItemID,
					Quantity:     drop.Quantity,
					Value:        totalValue,
					QualityLevel: drop.QualityLevel,
				})

				userValues[p.UserID] += totalValue
				totalGambleValue += totalValue
			}
		}
	}
	return userValues, allOpenedItems, totalGambleValue
}
