package expedition

import (
	"context"
	"fmt"
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
	Shutdown(ctx context.Context) error
}

// ProgressionService defines the interface for progression system
type ProgressionService interface {
	RecordEngagement(ctx context.Context, username string, action string, amount int) error
}

// JobService defines the interface for job operations
type JobService interface {
	AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) error
}

// LootboxService defines the interface for lootbox operations
type LootboxService interface {
	OpenLootbox(ctx context.Context, lootboxKey string, quantity int, boxShine domain.ShineLevel) ([]DroppedItem, error)
}

// DroppedItem represents an item dropped from a lootbox
type DroppedItem struct {
	ItemID     int
	ItemName   string
	Quantity   int
	Value      int
	ShineLevel domain.ShineLevel
}

type service struct {
	repo           repository.Expedition
	eventBus       event.Bus
	progressionSvc ProgressionService
	jobSvc         JobService
	lootboxSvc     LootboxService
	joinDuration   time.Duration
	wg             sync.WaitGroup
}

// NewService creates a new expedition service
func NewService(repo repository.Expedition, eventBus event.Bus, progressionSvc ProgressionService, jobSvc JobService, lootboxSvc LootboxService, joinDuration time.Duration) Service {
	return &service{
		repo:           repo,
		eventBus:       eventBus,
		progressionSvc: progressionSvc,
		jobSvc:         jobSvc,
		lootboxSvc:     lootboxSvc,
		joinDuration:   joinDuration,
	}
}

// StartExpedition creates a new expedition
func (s *service) StartExpedition(ctx context.Context, platform, platformID, username, expeditionType string) (*domain.Expedition, error) {
	// Get initiator
	initiator, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to get initiator: %w", err)
	}

	// Create expedition
	now := time.Now()
	initiatorID, err := uuid.Parse(initiator.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid initiator ID: %w", err)
	}

	expedition := &domain.Expedition{
		ID:                 uuid.New(),
		InitiatorID:        initiatorID,
		ExpeditionType:     expeditionType,
		State:              domain.ExpeditionStateRecruiting,
		CreatedAt:          now,
		JoinDeadline:       now.Add(s.joinDuration),
		CompletionDeadline: now.Add(s.joinDuration + 30*time.Minute), // TODO: Make configurable
	}

	if err := s.repo.CreateExpedition(ctx, expedition); err != nil {
		return nil, fmt.Errorf("failed to create expedition: %w", err)
	}

	// Add initiator as first participant
	participant := &domain.ExpeditionParticipant{
		ExpeditionID: expedition.ID,
		UserID:       initiatorID,
		JoinedAt:     now,
	}

	if err := s.repo.AddParticipant(ctx, participant); err != nil {
		return nil, fmt.Errorf("failed to add initiator as participant: %w", err)
	}

	return expedition, nil
}

// JoinExpedition adds a user to an expedition
func (s *service) JoinExpedition(ctx context.Context, platform, platformID, username string, expeditionID uuid.UUID) error {
	// Get user
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Add participant
	participant := &domain.ExpeditionParticipant{
		ExpeditionID: expeditionID,
		UserID:       userID,
		JoinedAt:     time.Now(),
	}

	if err := s.repo.AddParticipant(ctx, participant); err != nil {
		return fmt.Errorf("failed to add participant: %w", err)
	}

	return nil
}

// GetExpedition retrieves expedition details
func (s *service) GetExpedition(ctx context.Context, expeditionID uuid.UUID) (*domain.ExpeditionDetails, error) {
	return s.repo.GetExpedition(ctx, expeditionID)
}

// GetActiveExpedition retrieves the current active expedition
func (s *service) GetActiveExpedition(ctx context.Context) (*domain.ExpeditionDetails, error) {
	return s.repo.GetActiveExpedition(ctx)
}

// ExecuteExpedition processes an expedition and generates rewards
func (s *service) ExecuteExpedition(ctx context.Context, expeditionID uuid.UUID) error {
	// TODO: Implement expedition execution logic
	// This is a placeholder - actual implementation will be done later
	return fmt.Errorf("not implemented")
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
