package prediction

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// Service defines the interface for prediction operations
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

// NewService creates a new prediction service
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

	// 3. Record engagement and add to progression (using the final scaled contribution)
	if err := s.recordTotalEngagement(ctx, finalContribution); err != nil {
		log.Error("Failed to record total engagement", "error", err)
		return nil, fmt.Errorf("failed to record engagement: %w", err)
	}

	// 4. Award winner XP and grenade (publishes event for XP; async goroutine for grenade item)
	winnerXP := s.awardWinnerRewards(ctx, req.Platform, req.Winner)

	// 5. Award participants XP via events (10 each to Gambler)
	s.awardParticipantsXP(ctx, req.Platform, req.Participants)

	// 6. Publish event
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

	// Scale points relative to divisor (e.g. 10k)
	scaledPoints := float64(points) / PointsScaleDivisor

	// Apply logarithmic formula
	// Goal: 10,000 points = 1 contribution, 1,000,000 points = 50 contribution
	logComponent := math.Log10(scaledPoints) / LogDivisor
	contribution := BaseContribution + (logComponent * ScaleMultiplier) + BonusContribution

	// Clamp to 0 minimum
	if contribution < 0 {
		contribution = 0
	}

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
func (s *service) recordTotalEngagement(ctx context.Context, contribution int) error {
	// Record the engagement metric using a system identifier for prediction totals
	// We use PredictionContributionMetricType to indicate this value is already scaled
	if err := s.progressionService.RecordEngagement(ctx, "prediction_system", PredictionContributionMetricType, contribution); err != nil {
		return fmt.Errorf("failed to record engagement: %w", err)
	}

	return nil
}

// awardWinnerRewards publishes XP event for the prediction winner and awards grenade async
func (s *service) awardWinnerRewards(ctx context.Context, platform string, winner domain.PredictionWinner) int {
	// Resolve UUID for the winner
	var userID string
	user, err := s.ensureUserRegistered(ctx, winner.Username, platform, winner.PlatformID)
	if err == nil && user != nil {
		userID = user.ID
	}

	// Publish XP + stats event (handled by job and stats event handlers)
	if s.resilientPublisher != nil {
		s.resilientPublisher.PublishWithRetry(context.Background(), event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventTypePredictionParticipated),
			Payload: domain.PredictionParticipantPayload{
				UserID:     userID,
				Username:   winner.Username,
				Platform:   platform,
				PlatformID: winner.PlatformID,
				XP:         WinnerXP,
				IsWinner:   true,
				Timestamp:  time.Now().Unix(),
			},
		})
	}

	// Award grenade if unlocked (async, needs direct service calls)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		bgCtx := context.Background()
		log := logger.FromContext(bgCtx)

		// Auto-register user if needed before item award
		_, err = s.ensureUserRegistered(bgCtx, winner.Username, platform, winner.PlatformID)
		if err != nil {
			log.Error("Failed to register winner", "username", winner.Username, "error", err)
			return
		}

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

// awardParticipantsXP publishes XP + stats events for all participants
func (s *service) awardParticipantsXP(ctx context.Context, platform string, participants []domain.PredictionParticipant) {
	if s.resilientPublisher == nil {
		return
	}
	for _, p := range participants {
		// Resolve UUID for each participant
		var userID string
		user, err := s.ensureUserRegistered(ctx, p.Username, platform, p.PlatformID)
		if err == nil && user != nil {
			userID = user.ID
		}

		s.resilientPublisher.PublishWithRetry(context.Background(), event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventTypePredictionParticipated),
			Payload: domain.PredictionParticipantPayload{
				UserID:     userID,
				Username:   p.Username,
				Platform:   platform,
				PlatformID: p.PlatformID,
				XP:         ParticipantXP,
				IsWinner:   false,
				Timestamp:  time.Now().Unix(),
			},
		})
	}
}

// ensureUserRegistered checks if a user exists and registers them if not
func (s *service) ensureUserRegistered(ctx context.Context, username, platform, platformID string) (*domain.User, error) {
	log := logger.FromContext(ctx)

	// Try to get user first
	user, err := s.userService.GetUserByPlatformUsername(ctx, platform, username)
	if err == nil {
		return user, nil // User exists
	}

	// Try searching by platform ID as well to be safe
	user, err = s.userService.FindUserByPlatformID(ctx, platform, platformID)
	if err == nil && user != nil {
		return user, nil
	}

	// User doesn't exist, register them
	log.Info("Auto-registering user for prediction",
		"username", username,
		"platform", platform)

	newUser := domain.User{
		Username: username,
	}

	// Set platform-specific ID
	switch platform {
	case "twitch":
		newUser.TwitchID = platformID
	case "youtube":
		newUser.YoutubeID = platformID
	case "discord":
		newUser.DiscordID = platformID
	}

	registeredUser, err := s.userService.RegisterUser(ctx, newUser)
	if err != nil {
		return nil, fmt.Errorf("failed to auto-register user: %w", err)
	}

	return &registeredUser, nil
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
