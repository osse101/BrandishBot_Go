package economy

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/naming"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// Service defines the interface for economy operations
type Service interface {
	GetSellablePrices(ctx context.Context) ([]domain.Item, error)
	GetBuyablePrices(ctx context.Context) ([]domain.Item, error)
	SellItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (int, int, error)
	BuyItem(ctx context.Context, platform, platformID, username, itemName string, quantity int) (int, error)
	Shutdown(ctx context.Context) error
}

// ProgressionService defines the interface for progression operations
type ProgressionService interface {
	IsItemUnlocked(ctx context.Context, itemName string) (bool, error)
	AreItemsUnlocked(ctx context.Context, itemNames []string) (map[string]bool, error)
	IsFeatureUnlocked(ctx context.Context, featureKey string) (bool, error)
	GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error)
}

type service struct {
	repo               repository.Economy
	publisher          *event.ResilientPublisher
	namingResolver     naming.Resolver
	progressionService ProgressionService
	rnd                func() float64 // For RNG - allows deterministic testing
	now                func() time.Time
	weeklySales        []domain.WeeklySale
	weeklySalesMu      sync.RWMutex
}

// NewService creates a new economy service
func NewService(repo repository.Economy, publisher *event.ResilientPublisher, namingResolver naming.Resolver, progressionService ProgressionService) Service {
	s := &service{
		repo:               repo,
		publisher:          publisher,
		namingResolver:     namingResolver,
		progressionService: progressionService,
		rnd:                utils.RandomFloat,
		now:                time.Now,
	}

	// Load weekly sales configuration (log errors but don't fail startup)
	if err := s.loadWeeklySales(); err != nil {
		slog.Warn("Failed to load weekly sales configuration", "error", err)
	}

	return s
}

func (s *service) Shutdown(ctx context.Context) error {
	logger.FromContext(ctx).Info(LogMsgEconomyShuttingDown)
	return nil
}
