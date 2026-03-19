package prediction

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

func (s *service) ensureUserRegistered(ctx context.Context, username, platform, platformID string) (*domain.User, error) {
	log := logger.FromContext(ctx)

	user, err := s.userService.GetUserByPlatformUsername(ctx, platform, username)
	if err == nil {
		return user, nil
	}

	user, err = s.userService.FindUserByPlatformID(ctx, platform, platformID)
	if err == nil && user != nil {
		return user, nil
	}

	log.Info("Auto-registering user for prediction",
		"username", username,
		"platform", platform)

	newUser := domain.User{
		Username: username,
	}

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
