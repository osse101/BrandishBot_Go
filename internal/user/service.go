package user

import (
	"errors"
	"sync"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Service defines the interface for user operations
type Service interface {
	RegisterUser(user domain.User) error
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

// RegisterUser registers a new user
func (s *service) RegisterUser(user domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.ID]; exists {
		return errors.New("user already exists")
	}

	s.users[user.ID] = user
	return nil
}
