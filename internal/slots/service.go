package slots

import (
	"context"
	"sync"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// CooldownService defines the interface for cooldown operations
type CooldownService interface {
	EnforceCooldown(ctx context.Context, userID, action string, fn func() error) error
}

// Service defines the interface for slots operations
type Service interface {
	SpinSlots(ctx context.Context, platform, platformID, username string, betAmount int) (*domain.SlotsResult, error)
	Shutdown(ctx context.Context) error
}

type service struct {
	userRepo           repository.User
	progressionService progression.Service
	cooldownSvc        CooldownService
	eventBus           event.Bus
	resilientPublisher *event.ResilientPublisher
	namingResolver     naming.Resolver
	rng                func(int) int // Injectable for testing
	wg                 sync.WaitGroup
	shutdown           chan struct{}
}

// NewService creates a new slots service
func NewService(
	userRepo repository.User,
	progressionService progression.Service,
	cooldownSvc CooldownService,
	eventBus event.Bus,
	resilientPublisher *event.ResilientPublisher,
	namingResolver naming.Resolver,
) Service {
	return &service{
		userRepo:           userRepo,
		progressionService: progressionService,
		cooldownSvc:        cooldownSvc,
		eventBus:           eventBus,
		resilientPublisher: resilientPublisher,
		namingResolver:     namingResolver,
		rng:                utils.SecureRandomInt,
		shutdown:           make(chan struct{}),
	}
}

// Shutdown gracefully stops the service
func (s *service) Shutdown(ctx context.Context) error {
	close(s.shutdown)

	// Wait for all async operations to complete
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
