package slots

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

func (s *service) SpinSlots(ctx context.Context, platform, platformID, username string, betAmount int) (*domain.SlotsResult, error) {
	log := logger.FromContext(ctx)

	if betAmount < MinBetAmount {
		return nil, fmt.Errorf("minimum bet is %d money", MinBetAmount)
	}
	if betAmount > MaxBetAmount {
		return nil, fmt.Errorf("maximum bet is %d money", MaxBetAmount)
	}

	user, err := s.userRepo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	isUnlocked, err := s.progressionService.IsFeatureUnlocked(ctx, progression.FeatureSlots)
	if err != nil {
		log.Warn("Failed to check feature lock", "error", err)
	}
	if !isUnlocked {
		return nil, fmt.Errorf("slots feature is not yet unlocked")
	}

	var result *domain.SlotsResult

	err = s.cooldownSvc.EnforceCooldown(ctx, user.ID, domain.ActionSlots, func() error {
		var spinErr error
		result, spinErr = s.executeSpin(ctx, user, username, betAmount)
		return spinErr
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *service) executeSpin(ctx context.Context, user *domain.User, username string, betAmount int) (*domain.SlotsResult, error) {
	tx, err := s.userRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	moneyItem, err := s.userRepo.GetItemByName(ctx, domain.ItemMoney)
	if err != nil {
		return nil, fmt.Errorf("failed to get money item: %w", err)
	}
	if moneyItem == nil {
		return nil, fmt.Errorf("money item not found")
	}

	inventory, err := tx.GetInventory(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}

	moneySlotIndex, currentMoney := utils.FindRandomSlot(inventory, moneyItem.ID, func() float64 {
		return float64(s.rng(1000)) / 1000.0
	})

	if currentMoney < betAmount {
		return nil, fmt.Errorf("insufficient funds. You have %d money", currentMoney)
	}

	reel1, reel2, reel3 := s.spinReels()

	amount, mult, trigger := s.calculatePayout(reel1, reel2, reel3, betAmount)

	netChange := amount - betAmount
	newBalance := currentMoney + netChange

	if newBalance < 0 {
		return nil, fmt.Errorf("transaction would result in negative balance")
	}

	if moneySlotIndex != -1 {
		if newBalance == 0 {
			inventory.Slots = append(inventory.Slots[:moneySlotIndex], inventory.Slots[moneySlotIndex+1:]...)
		} else {
			inventory.Slots[moneySlotIndex].Quantity = newBalance
		}
	}

	if err := tx.UpdateInventory(ctx, user.ID, *inventory); err != nil {
		return nil, fmt.Errorf("failed to update inventory: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	isWin := (reel1 == reel2 && reel2 == reel3)
	isNearMiss := false

	result := &domain.SlotsResult{
		UserID:           user.ID,
		Username:         username,
		Reel1:            reel1,
		Reel2:            reel2,
		Reel3:            reel3,
		BetAmount:        betAmount,
		PayoutAmount:     amount,
		PayoutMultiplier: mult,
		IsWin:            isWin,
		IsNearMiss:       isNearMiss,
		TriggerType:      trigger,
		Message:          s.formatMessage(reel1, reel2, reel3, betAmount, amount, trigger),
	}

	s.wg.Add(1)
	go s.recordAllEngagement(ctx, user.ID, result)

	payload := domain.SlotsCompletedPayload{
		UserID:           user.ID,
		Username:         username,
		BetAmount:        betAmount,
		Reel1:            reel1,
		Reel2:            reel2,
		Reel3:            reel3,
		PayoutAmount:     amount,
		PayoutMultiplier: mult,
		TriggerType:      trigger,
		IsWin:            isWin,
		IsNearMiss:       isNearMiss,
	}

	evt := event.Event{
		Type:    event.Type(domain.EventSlotsCompleted),
		Payload: payload,
	}
	s.resilientPublisher.PublishWithRetry(ctx, evt)

	return result, nil
}
