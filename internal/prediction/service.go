package prediction

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

type Service interface {
	ProcessOutcome(ctx context.Context, req *domain.PredictionOutcomeRequest) (*domain.PredictionResult, error)
	Shutdown(ctx context.Context) error
}

type service struct {
	progressionService progression.Service
	userService        user.Service
	eventBus           event.Bus
	resilientPublisher *event.ResilientPublisher
	wg                 sync.WaitGroup
}

func NewService(
	progressionService progression.Service,
	userService user.Service,
	eventBus event.Bus,
	resilientPublisher *event.ResilientPublisher,
) Service {
	return &service{
		progressionService: progressionService,
		userService:        userService,
		eventBus:           eventBus,
		resilientPublisher: resilientPublisher,
	}
}

func (s *service) ProcessOutcome(ctx context.Context, req *domain.PredictionOutcomeRequest) (*domain.PredictionResult, error) {
	log := logger.FromContext(ctx)

	log.Info("Processing prediction outcome",
		"platform", req.Platform,
		"total_points", req.TotalPointsSpent,
		"participants", len(req.Participants),
		"winner", req.Winner.Username)

	baseContribution := s.calculateContribution(req.TotalPointsSpent)
	log.Debug("Calculated base contribution",
		"total_points", req.TotalPointsSpent,
		"base_contribution", baseContribution)

	finalContribution, err := s.applyContributionModifier(ctx, baseContribution)
	if err != nil {
		log.Error("Failed to apply contribution modifier", "error", err)
		finalContribution = baseContribution
	}

	if err := s.recordTotalEngagement(ctx, finalContribution); err != nil {
		log.Error("Failed to record total engagement", "error", err)
		return nil, fmt.Errorf("failed to record engagement: %w", err)
	}

	winnerXP := s.awardWinnerRewards(ctx, req.Platform, req.Winner)

	s.awardParticipantsXP(ctx, req.Platform, req.Participants)

	s.publishPredictionEvent(ctx, req, finalContribution)

	result := &domain.PredictionResult{
		TotalPoints:           req.TotalPointsSpent,
		ContributionAwarded:   finalContribution,
		ParticipantsProcessed: len(req.Participants),
		WinnerXPAwarded:       winnerXP,
		Message:               fmt.Sprintf("Processed prediction: %d points → %d contribution", req.TotalPointsSpent, finalContribution),
	}

	log.Info("Prediction outcome processed successfully",
		"contribution", finalContribution,
		"participants", len(req.Participants),
		"winner_xp", winnerXP)

	return result, nil
}

func (s *service) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down prediction service")

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Prediction service shutdown complete")
		return nil
	case <-ctx.Done():
		slog.Warn("Prediction service shutdown timed out")
		return ctx.Err()
	}
}
