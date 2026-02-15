package harvest

import (
	"context"
	"sync"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Service defines the harvest system business logic
type Service interface {
	// Harvest collects accumulated rewards for a user
	Harvest(ctx context.Context, platform, platformID, username string) (*domain.HarvestResponse, error)
	// Shutdown gracefully shuts down the service
	Shutdown(ctx context.Context) error
}

type service struct {
	harvestRepo    repository.HarvestRepository
	userRepo       repository.User
	progressionSvc progression.Service
	jobSvc         job.Service
	publisher      *event.ResilientPublisher
	wg             sync.WaitGroup
}

// NewService creates a new harvest service
func NewService(
	harvestRepo repository.HarvestRepository,
	userRepo repository.User,
	progressionSvc progression.Service,
	jobSvc job.Service,
	publisher *event.ResilientPublisher,
) Service {
	return &service{
		harvestRepo:    harvestRepo,
		userRepo:       userRepo,
		progressionSvc: progressionSvc,
		jobSvc:         jobSvc,
		publisher:      publisher,
	}
}

// Shutdown gracefully shuts down the service
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
