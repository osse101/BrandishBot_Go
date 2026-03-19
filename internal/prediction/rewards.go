package prediction

import (
	"context"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

func (s *service) awardWinnerRewards(ctx context.Context, platform string, winner domain.PredictionWinner) int {
	var userID string
	user, err := s.ensureUserRegistered(ctx, winner.Username, platform, winner.PlatformID)
	if err == nil && user != nil {
		userID = user.ID
	}

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

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		bgCtx := context.Background()
		log := logger.FromContext(bgCtx)

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

func (s *service) awardParticipantsXP(ctx context.Context, platform string, participants []domain.PredictionParticipant) {
	if s.resilientPublisher == nil {
		return
	}
	for _, p := range participants {
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
