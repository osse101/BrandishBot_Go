package job

import (
	"context"
	"sync"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

// ProgressionService defines the interface for progression system
type ProgressionService interface {
	IsFeatureUnlocked(ctx context.Context, featureKey string) (bool, error)
	GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error)
	IsNodeUnlocked(ctx context.Context, nodeKey string, level int) (bool, error)
}

// Service defines the job system business logic
type Service interface {
	// Core operations
	GetUserJobs(ctx context.Context, userID string) ([]domain.UserJobInfo, error)
	GetUserJobsByPlatform(ctx context.Context, platform, platformID string) ([]domain.UserJobInfo, error)
	GetPrimaryJob(ctx context.Context, platform, platformID string) (*domain.UserJobInfo, error)
	GetJobBonus(ctx context.Context, userID, jobKey string, bonusType string) (float64, error)

	// XP operations
	AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata domain.JobXPMetadata) (*domain.XPAwardResult, error)
	AwardXPByPlatform(ctx context.Context, platform, platformID, jobKey string, baseAmount int, source string, metadata domain.JobXPMetadata) (*domain.XPAwardResult, error)
	GetJobLevel(ctx context.Context, userID, jobKey string) (int, error)

	// Daily reset operations
	ResetDailyJobXP(ctx context.Context) (int64, error)
	GetDailyResetStatus(ctx context.Context) (*domain.DailyResetStatus, error)

	// Utility
	GetAllJobs(ctx context.Context) ([]domain.Job, error)
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	CalculateLevel(totalXP int64) int
	GetXPForLevel(level int) int64
	GetXPProgress(currentXP int64) (currentLevel int, xpToNext int64)
	Shutdown(ctx context.Context) error
}

type service struct {
	repo           repository.Job
	progressionSvc ProgressionService
	eventBus       event.Bus
	publisher      *event.ResilientPublisher
	rnd            func() float64 // For RNG

	// Cache for daily reset status
	resetCache   *domain.DailyResetStatus
	resetCacheMu sync.RWMutex
}

// NewService creates a new job service
func NewService(repo repository.Job, progressionSvc ProgressionService, eventBus event.Bus, publisher *event.ResilientPublisher) Service {
	return &service{
		repo:           repo,
		progressionSvc: progressionSvc,
		eventBus:       eventBus,
		publisher:      publisher,
		rnd:            utils.RandomFloat,
	}
}

// Shutdown gracefully shuts down the job service
func (s *service) Shutdown(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("Job service shutting down...")

	if s.publisher != nil {
		if err := s.publisher.Shutdown(ctx); err != nil {
			log.Error("Failed to shut down job publisher", "error", err)
			return err
		}
	}

	log.Info("Job service shutdown complete")
	return nil
}
