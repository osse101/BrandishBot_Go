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

type CooldownService interface {
	EnforceCooldown(ctx context.Context, userID, action string, fn func() error) error
}

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
	rng                func(int) int
	wg                 sync.WaitGroup
	shutdown           chan struct{}
}

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

func (s *service) Shutdown(ctx context.Context) error {
	close(s.shutdown)

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
