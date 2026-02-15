package gamble

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// Service defines the interface for gamble operations
type Service interface {
	StartGamble(ctx context.Context, platform, platformID, username string, bets []domain.LootboxBet) (*domain.Gamble, error)
	JoinGamble(ctx context.Context, gambleID uuid.UUID, platform, platformID, username string) error
	GetGamble(ctx context.Context, id uuid.UUID) (*domain.Gamble, error)
	ExecuteGamble(ctx context.Context, id uuid.UUID) (*domain.GambleResult, error)
	GetActiveGamble(ctx context.Context) (*domain.Gamble, error)
}

// ProgressionService defines the interface for progression system
type ProgressionService interface {
	GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error)
}

// ResilientPublisher defines the interface for resilient event publishing
type ResilientPublisher interface {
	PublishWithRetry(ctx context.Context, evt event.Event)
}

type service struct {
	repo               repository.Gamble
	eventBus           event.Bus
	resilientPublisher ResilientPublisher
	lootboxSvc         lootbox.Service
	progressionSvc     ProgressionService
	namingResolver     naming.Resolver
	joinDuration       time.Duration
	rng                func(int) int
}

// NewService creates a new gamble service
func NewService(repo repository.Gamble, eventBus event.Bus, resilientPublisher ResilientPublisher, lootboxSvc lootbox.Service, joinDuration time.Duration, progressionSvc ProgressionService, namingResolver naming.Resolver, rng func(int) int) Service {
	if rng == nil {
		rng = utils.SecureRandomInt
	}
	return &service{
		repo:               repo,
		eventBus:           eventBus,
		resilientPublisher: resilientPublisher,
		lootboxSvc:         lootboxSvc,
		progressionSvc:     progressionSvc,
		namingResolver:     namingResolver,
		joinDuration:       joinDuration,
		rng:                rng,
	}
}

// GetGamble retrieves a gamble by ID
func (s *service) GetGamble(ctx context.Context, id uuid.UUID) (*domain.Gamble, error) {
	return s.repo.GetGamble(ctx, id)
}

// GetActiveGamble retrieves the current active gamble
func (s *service) GetActiveGamble(ctx context.Context) (*domain.Gamble, error) {
	return s.repo.GetActiveGamble(ctx)
}
