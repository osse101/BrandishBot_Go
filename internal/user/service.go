package user

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Repository defines the interface for user persistence
type Repository interface {
	UpsertUser(ctx context.Context, user *domain.User) error
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
}

// Service defines the interface for user operations
type Service interface {
	RegisterUser(ctx context.Context, user domain.User) (domain.User, error)
	FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	HandleIncomingMessage(ctx context.Context, platform, platformID, username string) (domain.User, error)
}

// service implements the Service interface
type service struct {
	repo Repository
}

// NewService creates a new user service
func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

// RegisterUser registers a new user
func (s *service) RegisterUser(ctx context.Context, user domain.User) (domain.User, error) {
	if err := s.repo.UpsertUser(ctx, &user); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

// FindUserByPlatformID finds a user by their platform-specific ID
func (s *service) FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	return s.repo.GetUserByPlatformID(ctx, platform, platformID)
}

// HandleIncomingMessage checks if a user exists for an incoming message and creates one if not.
func (s *service) HandleIncomingMessage(ctx context.Context, platform, platformID, username string) (domain.User, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err == nil {
		return *user, nil
	}

	// TODO: Check if error is actually "not found"

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
	default:
		return domain.User{}, fmt.Errorf("unsupported platform: %s", platform)
	}

	if _, err := s.RegisterUser(ctx, newUser); err != nil {
		return domain.User{}, err
	}

	return newUser, nil
}
