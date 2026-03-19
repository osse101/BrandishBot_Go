package prediction

import (
	"context"
	"fmt"
	"math"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func (s *service) calculateContribution(points int) int {
	if points <= 0 {
		return 0
	}

	scaledPoints := float64(points) / PointsScaleDivisor
	logComponent := math.Log10(scaledPoints) / LogDivisor
	contribution := BaseContribution + (logComponent * ScaleMultiplier) + BonusContribution

	if contribution < 0 {
		contribution = 0
	}

	return int(math.Round(contribution))
}

func (s *service) applyContributionModifier(ctx context.Context, baseContribution int) (int, error) {
	modifiedValue, err := s.progressionService.GetModifiedValue(ctx, "", "contribution", float64(baseContribution))
	if err != nil {
		return baseContribution, err
	}
	return int(math.Round(modifiedValue)), nil
}

func (s *service) recordTotalEngagement(ctx context.Context, contribution int) error {
	if err := s.progressionService.RecordEngagement(ctx, "prediction_system", domain.MetricTypePredictionContribution, contribution); err != nil {
		return fmt.Errorf("failed to record engagement: %w", err)
	}
	return nil
}
