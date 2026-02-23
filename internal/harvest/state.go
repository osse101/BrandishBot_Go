package harvest

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

func (s *service) ensureHarvestState(ctx context.Context, userID string) (*domain.HarvestResponse, error) {
	_, err := s.harvestRepo.GetHarvestState(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrHarvestStateNotFound) {
			if _, err = s.harvestRepo.CreateHarvestState(ctx, userID); err != nil {
				return nil, fmt.Errorf("failed to create harvest state: %w", err)
			}
			return &domain.HarvestResponse{
				ItemsGained:       map[string]int{},
				HoursSinceHarvest: 0,
				NextHarvestAt:     time.Now().Add(time.Hour),
				Message:           "Harvest tracking initialized! Come back in 1 hour for your first harvest.",
			}, nil
		}
		return nil, fmt.Errorf("failed to get harvest state: %w", err)
	}
	return nil, nil
}

func (s *service) recordEmptyHarvest(ctx context.Context, tx repository.HarvestTx, userID string, now time.Time, hoursElapsed float64) (*domain.HarvestResponse, error) {
	logger.FromContext(ctx).Warn("No rewards available - all items locked by progression")

	if err := tx.UpdateHarvestState(ctx, userID, now); err != nil {
		return nil, fmt.Errorf("failed to update harvest state: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &domain.HarvestResponse{
		ItemsGained:       map[string]int{},
		HoursSinceHarvest: hoursElapsed,
		NextHarvestAt:     now.Add(time.Hour),
		Message:           "No rewards available - unlock progression nodes to receive harvest items!",
	}, nil
}
