package expedition

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Service defines the interface for expedition operations
type Service interface {
	StartExpedition(ctx context.Context, platform, platformID, username, expeditionType string) (*domain.Expedition, error)
	JoinExpedition(ctx context.Context, platform, platformID, username string, expeditionID uuid.UUID) error
	GetExpedition(ctx context.Context, expeditionID uuid.UUID) (*domain.ExpeditionDetails, error)
	GetActiveExpedition(ctx context.Context) (*domain.ExpeditionDetails, error)
	ExecuteExpedition(ctx context.Context, expeditionID uuid.UUID) error
	GetJournal(ctx context.Context, expeditionID uuid.UUID) ([]domain.ExpeditionJournalEntry, error)
	GetStatus(ctx context.Context) (*domain.ExpeditionStatus, error)
	Shutdown(ctx context.Context) error
}

// ProgressionService defines the interface for progression system
type ProgressionService interface {
	RecordEngagement(ctx context.Context, username string, action string, amount int) error
}

// JobService defines the interface for job operations
// Kept for GetUserJobs (read-only) used during expedition skill checks
type JobService interface {
	GetUserJobs(ctx context.Context, userID string) ([]domain.UserJobInfo, error)
}

// EventPublisher defines the interface for publishing events with retry
type EventPublisher interface {
	PublishWithRetry(ctx context.Context, evt event.Event)
}

// UserService defines the interface for user operations needed by expedition
type UserService interface {
	AddItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) error
}

// CooldownService defines the interface for cooldown operations
type CooldownService interface {
	CheckCooldown(ctx context.Context, userID, action string) (bool, time.Duration, error)
	EnforceCooldown(ctx context.Context, userID, action string, fn func() error) error
}

type service struct {
	repo           repository.Expedition
	eventBus       event.Bus
	progressionSvc ProgressionService
	jobSvc         JobService
	publisher      EventPublisher
	userSvc        UserService
	cooldownSvc    CooldownService
	config         *EncounterConfig
	joinDuration   time.Duration
	cooldownDur    time.Duration
	wg             sync.WaitGroup
}

// NewService creates a new expedition service
func NewService(
	repo repository.Expedition,
	eventBus event.Bus,
	progressionSvc ProgressionService,
	jobSvc JobService,
	publisher EventPublisher,
	userSvc UserService,
	cooldownSvc CooldownService,
	config *EncounterConfig,
	joinDuration time.Duration,
	cooldownDur time.Duration,
) Service {
	return &service{
		repo:           repo,
		eventBus:       eventBus,
		progressionSvc: progressionSvc,
		jobSvc:         jobSvc,
		publisher:      publisher,
		userSvc:        userSvc,
		cooldownSvc:    cooldownSvc,
		config:         config,
		joinDuration:   joinDuration,
		cooldownDur:    cooldownDur,
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
