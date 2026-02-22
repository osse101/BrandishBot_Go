package harvest

import (
	"context"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Harvest collects accumulated rewards for a user
func (s *service) Harvest(ctx context.Context, platform, platformID, username string) (*domain.HarvestResponse, error) {
	log := logger.FromContext(ctx)
	log.Info("Harvest called", "platform", platform, "platformID", platformID, "username", username)

	user, err := s.ensureUser(ctx, platform, platformID, username)
	if err != nil {
		return nil, err
	}

	if err := s.ensureFarmingUnlocked(ctx); err != nil {
		return nil, err
	}

	if initialResp, err := s.initializeHarvestStateIfNeeded(ctx, user.ID); err != nil || initialResp != nil {
		return initialResp, err
	}

	return s.performHarvestTransaction(ctx, user)
}

func (s *service) performHarvestTransaction(ctx context.Context, user *domain.User) (*domain.HarvestResponse, error) {
	tx, err := s.harvestRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer repository.SafeRollback(ctx, tx)

	harvestState, err := tx.GetHarvestStateWithLock(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get harvest state with lock: %w", err)
	}

	now := time.Now()
	hoursElapsed := now.Sub(harvestState.LastHarvestedAt).Hours()

	if hoursElapsed < minHarvestInterval {
		nextHarvest := harvestState.LastHarvestedAt.Add(time.Hour)
		return nil, fmt.Errorf("%w: next harvest available at %s", domain.ErrHarvestTooSoon, nextHarvest.Format(time.RFC3339))
	}

	yieldMultiplier, growthMultiplier := s.getBonusMultipliers(ctx, user.ID)
	effectiveHours := hoursElapsed * growthMultiplier

	rewards, message := s.calculateHarvestRewards(ctx, effectiveHours, yieldMultiplier)

	s.fireAsyncEvents(ctx, user.ID, hoursElapsed)

	if len(rewards) == 0 {
		return s.handleEmptyHarvest(ctx, tx, user.ID, now, hoursElapsed)
	}

	if err := s.applyHarvestRewards(ctx, tx, user.ID, rewards); err != nil {
		return nil, err
	}

	if err := tx.UpdateHarvestState(ctx, user.ID, now); err != nil {
		return nil, fmt.Errorf("failed to update harvest state: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.FromContext(ctx).Info("Harvest successful", "userID", user.ID, "rewards", rewards, "hours", hoursElapsed)

	return &domain.HarvestResponse{
		ItemsGained:       rewards,
		HoursSinceHarvest: hoursElapsed,
		NextHarvestAt:     now.Add(time.Hour),
		Message:           message,
	}, nil
}
