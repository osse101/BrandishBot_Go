package prediction

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/stats"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// Service defines the interface for prediction operations
type Service interface {
	ProcessOutcome(ctx context.Context, req *domain.PredictionOutcomeRequest) (*domain.PredictionResult, error)
	Shutdown(ctx context.Context) error
}

type service struct {
	progressionService progression.Service
	jobService         job.Service
	userService        user.Service
	statsService       stats.Service
	eventBus           event.Bus
	resilientPublisher *event.ResilientPublisher
	wg                 sync.WaitGroup
}

// NewService creates a new prediction service
func NewService(
	progressionService progression.Service,
	jobService job.Service,
	userService user.Service,
	statsService stats.Service,
	eventBus event.Bus,
	resilientPublisher *event.ResilientPublisher,
) Service {
	return &service{
		progressionService: progressionService,
		jobService:         jobService,
		userService:        userService,
		statsService:       statsService,
		eventBus:           eventBus,
		resilientPublisher: resilientPublisher,
	}
}

// ProcessOutcome processes a prediction outcome, converting channel points to progression
// contributions and awarding XP to participants
func (s *service) ProcessOutcome(ctx context.Context, req *domain.PredictionOutcomeRequest) (*domain.PredictionResult, error) {
	log := logger.FromContext(ctx)

	log.Info("Processing prediction outcome",
		"platform", req.Platform,
		"total_points", req.TotalPointsSpent,
		"participants", len(req.Participants),
		"winner", req.Winner.Username)

	// 1. Calculate base contribution using logarithmic scaling
	baseContribution := s.calculateContribution(req.TotalPointsSpent)
	log.Debug("Calculated base contribution",
		"total_points", req.TotalPointsSpent,
		"base_contribution", baseContribution)

	// 2. Apply contribution modifier if unlocked (1.5x boost)
	finalContribution, err := s.applyContributionModifier(ctx, baseContribution)
	if err != nil {
		log.Error("Failed to apply contribution modifier", "error", err)
		// Continue with base contribution if modifier check fails
		finalContribution = baseContribution
	}

	// 3. Record engagement and add to progression
	if err := s.recordTotalEngagement(ctx, req.TotalPointsSpent, finalContribution); err != nil {
		log.Error("Failed to record total engagement", "error", err)
		return nil, fmt.Errorf("failed to record engagement: %w", err)
	}

	// 4. Award winner XP and grenade (async, 100 XP to Gambler + 1 grenade if unlocked)
	winnerXP := s.awardWinnerRewards(ctx, req.Platform, req.Winner)

	// 5. Award participants XP (async, 10 each to Gambler)
	s.awardParticipantsXP(ctx, req.Platform, req.Participants)

	// 6. Record stats for all participants
	s.recordParticipantStats(ctx, req.Platform, req.Participants, req.TotalPointsSpent)

	// 7. Publish event
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

// calculateContribution converts channel points to progression contribution using logarithmic scaling
// Formula: 1 + (log10(points/1000) / 3) * 99 + 10
// Examples: 1k points → 11, 10k points → 44, 100k points → 77, 1M points → 110
func (s *service) calculateContribution(points int) int {
	if points <= 0 {
		return 0
	}

	// Scale points to thousands
	scaledPoints := float64(points) / PointsScaleDivisor
	if scaledPoints < 1.0 {
		scaledPoints = 1.0
	}

	// Apply logarithmic formula
	logComponent := math.Log10(scaledPoints) / LogDivisor
	contribution := BaseContribution + (logComponent * ScaleMultiplier) + BonusContribution

	return int(math.Round(contribution))
}

// applyContributionModifier checks if the contribution boost is unlocked and applies it
func (s *service) applyContributionModifier(ctx context.Context, baseContribution int) (int, error) {
	modifiedValue, err := s.progressionService.GetModifiedValue(ctx, "contribution", float64(baseContribution))
	if err != nil {
		return baseContribution, err
	}
	return int(math.Round(modifiedValue)), nil
}

// recordTotalEngagement records the total channel points and progression contribution
func (s *service) recordTotalEngagement(ctx context.Context, totalPoints, contribution int) error {
	// Record the engagement metric using a system identifier for prediction totals
	if err := s.progressionService.RecordEngagement(ctx, "prediction_system", TotalPointsMetricType, totalPoints); err != nil {
		return fmt.Errorf("failed to record engagement: %w", err)
	}

	return nil
}

// awardWinnerRewards awards XP and grenade to the prediction winner (async, graceful degradation)
func (s *service) awardWinnerRewards(_ context.Context, platform string, winner domain.PredictionWinner) int {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		// Use background context to avoid cancellation
		bgCtx := context.Background()
		log := logger.FromContext(bgCtx)

		// Auto-register user if needed
		if err := s.ensureUserRegistered(bgCtx, winner.Username, platform, winner.PlatformID); err != nil {
			log.Error("Failed to register winner", "username", winner.Username, "error", err)
			return
		}

		// Award XP to Gambler job
		metadata := map[string]interface{}{
			"platform": platform,
			"source":   "prediction_winner",
		}
		_, err := s.jobService.AwardXP(bgCtx, winner.Username, GamblerJobKey, WinnerXP, "prediction", metadata)
		if err != nil {
			log.Error("Failed to award winner XP",
				"username", winner.Username,
				"xp", WinnerXP,
				"error", err)
			return
		}

		log.Info("Awarded winner XP",
			"username", winner.Username,
			"xp", WinnerXP,
			"job", GamblerJobKey)

		// Award grenade if unlocked
		grenadeUnlocked, err := s.progressionService.IsItemUnlocked(bgCtx, GrenadeItemName)
		if err != nil {
			log.Warn("Failed to check grenade unlock status", "error", err)
			return
		}

		if grenadeUnlocked {
			if err := s.userService.AddItemByUsername(bgCtx, platform, winner.Username, GrenadeItemName, GrenadeQuantity); err != nil {
				log.Error("Failed to award grenade to winner",
					"username", winner.Username,
					"error", err)
				return
			}

			log.Info("Awarded grenade to prediction winner",
				"username", winner.Username,
				"item", GrenadeItemName,
				"quantity", GrenadeQuantity)
		}
	}()

	return WinnerXP
}

// awardParticipantsXP awards XP to all participants (async, graceful degradation)
func (s *service) awardParticipantsXP(_ context.Context, platform string, participants []domain.PredictionParticipant) {
	for _, participant := range participants {
		s.wg.Add(1)
		go func(p domain.PredictionParticipant) {
			defer s.wg.Done()

			// Use background context to avoid cancellation
			bgCtx := context.Background()
			log := logger.FromContext(bgCtx)

			// Auto-register user if needed
			if err := s.ensureUserRegistered(bgCtx, p.Username, platform, p.PlatformID); err != nil {
				log.Error("Failed to register participant", "username", p.Username, "error", err)
				return
			}

			// Award XP to Gambler job
			metadata := map[string]interface{}{
				"platform":     platform,
				"source":       "prediction_participant",
				"points_spent": p.PointsSpent,
			}
			_, err := s.jobService.AwardXP(bgCtx, p.Username, GamblerJobKey, ParticipantXP, "prediction", metadata)
			if err != nil {
				log.Error("Failed to award participant XP",
					"username", p.Username,
					"xp", ParticipantXP,
					"error", err)
				return
			}

			log.Debug("Awarded participant XP",
				"username", p.Username,
				"xp", ParticipantXP,
				"job", GamblerJobKey)
		}(participant)
	}
}

// recordParticipantStats records prediction participation stats for all participants
func (s *service) recordParticipantStats(_ context.Context, platform string, participants []domain.PredictionParticipant, totalPoints int) {
	for _, participant := range participants {
		s.wg.Add(1)
		go func(p domain.PredictionParticipant) {
			defer s.wg.Done()

			// Use background context to avoid cancellation
			bgCtx := context.Background()
			log := logger.FromContext(bgCtx)

			// Auto-register user if needed
			if err := s.ensureUserRegistered(bgCtx, p.Username, platform, p.PlatformID); err != nil {
				log.Error("Failed to register participant for stats", "username", p.Username, "error", err)
				return
			}

			// Record participation stat
			metadata := map[string]interface{}{
				"platform":     platform,
				"points_spent": p.PointsSpent,
			}
			if err := s.statsService.RecordUserEvent(bgCtx, p.Username, PredictionStatType, metadata); err != nil {
				log.Error("Failed to record participant stat",
					"username", p.Username,
					"error", err)
				return
			}

			log.Debug("Recorded participant stat",
				"username", p.Username,
				"stat_type", PredictionStatType,
				"points", p.PointsSpent)
		}(participant)
	}
}

// ensureUserRegistered checks if a user exists and registers them if not
func (s *service) ensureUserRegistered(ctx context.Context, username, platform, platformID string) error {
	log := logger.FromContext(ctx)

	// Try to get user first
	_, err := s.userService.GetUserByPlatformUsername(ctx, platform, username)
	if err == nil {
		return nil // User exists
	}

	// User doesn't exist, register them
	log.Info("Auto-registering user for prediction",
		"username", username,
		"platform", platform)

	user := domain.User{
		Username: username,
	}

	// Set platform-specific ID
	switch platform {
	case "twitch":
		user.TwitchID = platformID
	case "youtube":
		user.YoutubeID = platformID
	case "discord":
		user.DiscordID = platformID
	}

	_, err = s.userService.RegisterUser(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to auto-register user: %w", err)
	}

	return nil
}

// publishPredictionEvent publishes a prediction processed event to the event bus
func (s *service) publishPredictionEvent(ctx context.Context, req *domain.PredictionOutcomeRequest, contribution int) {
	evt := event.Event{
		Version: "1.0",
		Type:    event.Type(domain.EventTypePredictionProcessed),
		Payload: map[string]interface{}{
			"platform":           req.Platform,
			"winner":             req.Winner.Username,
			"total_points":       req.TotalPointsSpent,
			"contribution":       contribution,
			"participants_count": len(req.Participants),
		},
		Metadata: make(map[string]interface{}),
	}

	s.resilientPublisher.PublishWithRetry(ctx, evt)
}

// Shutdown gracefully shuts down the prediction service
func (s *service) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down prediction service")

	// Wait for all async operations to complete
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
