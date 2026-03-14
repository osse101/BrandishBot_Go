package slots

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

func (s *service) recordAllEngagement(ctx context.Context, userID string, result *domain.SlotsResult) {
	defer s.wg.Done()

	log := logger.FromContext(ctx)

	if err := s.progressionService.RecordEngagement(ctx, userID, domain.MetricTypeSlotsSpin, 1); err != nil {
		log.Warn("Failed to record slots spin engagement", "error", err)
	}

	if result.IsWin {
		if err := s.progressionService.RecordEngagement(ctx, userID, domain.MetricTypeSlotsWin, 1); err != nil {
			log.Warn("Failed to record slots win engagement", "error", err)
		}
	}

	if result.PayoutMultiplier >= BigWinThreshold {
		if err := s.progressionService.RecordEngagement(ctx, userID, domain.MetricTypeSlotsBigWin, 1); err != nil {
			log.Warn("Failed to record slots big win engagement", "error", err)
		}
	}

	if result.PayoutMultiplier >= JackpotThreshold {
		if err := s.progressionService.RecordEngagement(ctx, userID, domain.MetricTypeSlotsJackpot, 1); err != nil {
			log.Warn("Failed to record slots jackpot engagement", "error", err)
		}
	}
}
