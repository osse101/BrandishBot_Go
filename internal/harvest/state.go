package harvest

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func (s *service) initializeHarvestStateIfNeeded(ctx context.Context, userID string) (*domain.HarvestResponse, error) {
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
