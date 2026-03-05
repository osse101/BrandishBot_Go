package compost

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

type Service interface {
	Deposit(ctx context.Context, platform, platformID string, items []DepositItem) (*domain.CompostBin, error)
	Harvest(ctx context.Context, platform, platformID, username string) (*domain.HarvestResult, error)
	Shutdown(ctx context.Context) error
}

type DepositItem struct {
	ItemName string `json:"item_name"`
	Quantity int    `json:"quantity"`
}

type resolvedDeposit struct {
	item     *domain.Item
	quantity int
}

type service struct {
	repo           repository.CompostRepository
	userRepo       repository.User
	progressionSvc progression.Service
	jobSvc         job.Service
	publisher      *event.ResilientPublisher
	engine         *Engine
	wg             sync.WaitGroup
}

func NewService(
	repo repository.CompostRepository,
	userRepo repository.User,
	progressionSvc progression.Service,
	jobSvc job.Service,
	publisher *event.ResilientPublisher,
) Service {
	return &service{
		repo:           repo,
		userRepo:       userRepo,
		progressionSvc: progressionSvc,
		jobSvc:         jobSvc,
		publisher:      publisher,
		engine:         NewEngine(),
	}
}

func (s *service) Shutdown(ctx context.Context) error {
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

func (s *service) validateFeature(ctx context.Context, userID string) error {
	unlocked, err := s.progressionSvc.IsFeatureUnlocked(ctx, progression.FeatureCompost)
	if err != nil {
		return fmt.Errorf("failed to check compost feature: %w", err)
	}
	if !unlocked {
		return fmt.Errorf("compost requires feature unlock: %w", domain.ErrFeatureLocked)
	}

	if userID != "" {
		jobUnlocked, err := s.jobSvc.IsJobFeatureUnlocked(ctx, userID, progression.FeatureCompost)
		if err == nil && !jobUnlocked {
			return fmt.Errorf("compost requires job progression: %w", domain.ErrFeatureLocked)
		}
	}
	return nil
}

func (s *service) getUserAndBin(ctx context.Context, platform, platformID string, createIfMissing bool) (*domain.User, *domain.CompostBin, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}

	compostBin, err := s.repo.GetBin(ctx, user.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get bin: %w", err)
	}
	if compostBin == nil && createIfMissing {
		compostBin, err = s.repo.CreateBin(ctx, user.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create bin: %w", err)
		}
	}
	return user, compostBin, nil
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return MsgReadyNow
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
