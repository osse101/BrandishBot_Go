package harvest

import (
	"context"
	"errors"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

func (s *service) ensureUser(ctx context.Context, platform, platformID, username string) (*domain.User, error) {
	user, err := s.userRepo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			newUser := &domain.User{Username: username}
			switch platform {
			case "discord":
				newUser.DiscordID = platformID
			case "twitch":
				newUser.TwitchID = platformID
			case "youtube":
				newUser.YoutubeID = platformID
			}
			if err := s.userRepo.UpsertUser(ctx, newUser); err != nil {
				return nil, fmt.Errorf("failed to register user: %w", err)
			}
			return s.userRepo.GetUserByPlatformID(ctx, platform, platformID)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (s *service) checkFarmingUnlocked(ctx context.Context) error {
	unlocked, err := s.progressionSvc.IsFeatureUnlocked(ctx, "feature_farming")
	if err != nil {
		return fmt.Errorf("failed to check farming feature unlock: %w", err)
	}
	if !unlocked {
		return fmt.Errorf("harvest requires farming feature to be unlocked: %w", domain.ErrFeatureLocked)
	}
	return nil
}
