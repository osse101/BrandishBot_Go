package user

import (
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Service defines the interface for user operations
type Service interface {
	RegisterUser(user domain.User) (domain.User, error)
	FindUserByPlatformID(platform, platformID string) (*domain.User, error)
	HandleIncomingMessage(platform, platformID, username string) (domain.User, error)
}

// service implements the Service interface
type service struct {
	users map[string]domain.User
	mu    sync.RWMutex
}

// NewService creates a new user service
func NewService() Service {
	return &service{
		users: make(map[string]domain.User),
	}
}

// RegisterUser registers or updates a user.
// If the user's InternalID is empty, a new user is created.
// If the user's InternalID is not empty, the user's details are updated.
func (s *service) RegisterUser(user domain.User) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if user.InternalID == "" {
		user.InternalID = uuid.New().String()
	}

	s.users[user.InternalID] = user
	return user, nil
}

// FindUserByPlatformID finds a user by their platform-specific ID
func (s *service) FindUserByPlatformID(platform, platformID string) (*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		switch platform {
		case "twitch":
			if u.TwitchId == platformID {
				return &u, nil
			}
		case "youtube":
			if u.YoutubeId == platformID {
				return &u, nil
			}
		case "discord":
			if u.DiscordId == platformID {
				return &u, nil
			}
		}
	}
	return nil, errors.New("user not found")
}

// HandleIncomingMessage checks if a user exists for an incoming message and creates one if not.
func (s *service) HandleIncomingMessage(platform, platformID, username string) (domain.User, error) {
	user, err := s.FindUserByPlatformID(platform, platformID)
	if err == nil {
		// User found, return it
		return *user, nil
	}

	// User not found, create a new one
	newUser := domain.User{
		Username: username,
	}

	switch platform {
	case "twitch":
		newUser.TwitchId = platformID
	case "youtube":
		newUser.YoutubeId = platformID
	case "discord":
		newUser.DiscordId = platformID
	}

	return s.RegisterUser(newUser)
}
