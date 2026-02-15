package gamble

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
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

	if err := s.transitionToOpeningState(ctx, tx, id); err != nil {
		return nil, err
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

func (s *service) transitionToOpeningState(ctx context.Context, tx repository.GambleTx, id uuid.UUID) error {
	rowsAffected, err := tx.UpdateGambleStateIfMatches(ctx, id, domain.GambleStateJoining, domain.GambleStateOpening)
	if err != nil {
		return fmt.Errorf("failed to transition gamble state: %w", err)
	}
	if rowsAffected == 0 {
		return errors.New(ErrMsgGambleAlreadyExecuted)
	}
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
					modifiedValue, err := s.progressionSvc.GetModifiedValue(ctx, ProgressionFeatureGambleWinBonus, float64(totalValue))
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

// determineCriticalFailures returns the set of user IDs who had critical fail scores
func (s *service) determineCriticalFailures(userValues map[string]int64, totalGambleValue int64) map[string]bool {
	critFails := make(map[string]bool)
	if len(userValues) <= 1 || totalGambleValue <= 0 {
		return critFails
	}
	averageScore := float64(totalGambleValue) / float64(len(userValues))
	threshold := int64(averageScore * CriticalFailThreshold)
	for userID, val := range userValues {
		if val <= threshold {
			critFails[userID] = true
		}
	}
	return critFails
}

// determineGambleWinners returns the winner ID, highest score, and set of users who lost a tie-break
func (s *service) determineGambleWinners(userValues map[string]int64) (string, int64, map[string]bool) {
	var highestValue int64 = InitialHighestValue
	var winners []string

	for userID, val := range userValues {
		if val > highestValue {
			highestValue = val
			winners = []string{userID}
		} else if val == highestValue {
			winners = append(winners, userID)
		}
	}

	tieBreakLost := make(map[string]bool)

	if len(winners) == 0 {
		return "", 0, tieBreakLost
	}

	if len(winners) > 1 {
		sort.Strings(winners)
		idx := s.rng(len(winners))
		winnerID := winners[idx]
		for _, uid := range winners {
			if uid != winnerID {
				tieBreakLost[uid] = true
			}
		}
		return winnerID, highestValue, tieBreakLost
	}
	return winners[0], highestValue, tieBreakLost
}

// determineNearMisses returns the set of user IDs who had near-miss scores (not the winner)
func (s *service) determineNearMisses(winnerID string, highestValue int64, userValues map[string]int64) map[string]bool {
	nearMiss := make(map[string]bool)
	if winnerID == "" || highestValue <= 0 {
		return nearMiss
	}
	threshold := int64(float64(highestValue) * NearMissThreshold)
	for userID, val := range userValues {
		if userID == winnerID || val == highestValue {
			continue
		}
		if val >= threshold {
			nearMiss[userID] = true
		}
	}
	return nearMiss
}

// buildParticipantOutcomes constructs per-participant outcome data for the GambleCompletedPayloadV2
func (s *service) buildParticipantOutcomes(gamble *domain.Gamble, userValues map[string]int64, winnerID string, critFailUsers, tieBreakLostUsers, nearMissUsers map[string]bool) []domain.GambleParticipantOutcome {
	outcomes := make([]domain.GambleParticipantOutcome, 0, len(gamble.Participants))
	for _, p := range gamble.Participants {
		outcomes = append(outcomes, domain.GambleParticipantOutcome{
			UserID:         p.UserID,
			Score:          userValues[p.UserID],
			LootboxCount:   calculateTotalLootboxes(p.LootboxBets),
			IsWinner:       p.UserID == winnerID,
			IsNearMiss:     nearMissUsers[p.UserID],
			IsCritFail:     critFailUsers[p.UserID],
			IsTieBreakLost: tieBreakLostUsers[p.UserID],
		})
	}
	return outcomes
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

func (s *service) publishGambleCompletedEvent(ctx context.Context, result *domain.GambleResult, participantCount int, participants []domain.GambleParticipantOutcome) {
	log := logger.FromContext(ctx)

	if s.resilientPublisher == nil {
		log.Error("Failed to publish GambleCompleted event", "reason", "resilientPublisher is nil")
		return
	}

	evt := event.NewGambleCompletedEvent(result.GambleID.String(), result.WinnerID, result.TotalValue, participantCount, participants)
	s.resilientPublisher.PublishWithRetry(ctx, evt)
}
