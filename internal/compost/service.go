package compost

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Service defines the interface for compost operations
type Service interface {
	Deposit(ctx context.Context, platform, platformID, itemKey string, quantity int) (*domain.CompostDeposit, error)
	GetStatus(ctx context.Context, platform, platformID string) (*domain.CompostStatus, error)
	Harvest(ctx context.Context, platform, platformID string) (int, error) // Returns total gems awarded
}

// ProgressionService defines the interface for progression system
type ProgressionService interface {
	RecordEngagement(ctx context.Context, username string, action string, amount int) error
}

type service struct {
	repo           repository.Compost
	eventBus       event.Bus
	progressionSvc ProgressionService
}

// NewService creates a new compost service
func NewService(repo repository.Compost, eventBus event.Bus, progressionSvc ProgressionService) Service {
	return &service{
		repo:           repo,
		eventBus:       eventBus,
		progressionSvc: progressionSvc,
	}
}

// Deposit creates a new compost deposit
func (s *service) Deposit(ctx context.Context, platform, platformID, itemKey string, quantity int) (*domain.CompostDeposit, error) {
	// Get user
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// TODO: Validate user has items in inventory
	// TODO: Remove items from inventory
	// TODO: Calculate ready_at based on item rarity

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Placeholder: 24 hours for now
	readyAt := time.Now().Add(24 * time.Hour)

	deposit := &domain.CompostDeposit{
		ID:          uuid.New(),
		UserID:      userID,
		ItemKey:     itemKey,
		Quantity:    quantity,
		DepositedAt: time.Now(),
		ReadyAt:     readyAt,
	}

	if err := s.repo.CreateDeposit(ctx, deposit); err != nil {
		return nil, fmt.Errorf("failed to create deposit: %w", err)
	}

	return deposit, nil
}

// GetStatus retrieves the compost status for a user
func (s *service) GetStatus(ctx context.Context, platform, platformID string) (*domain.CompostStatus, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	activeDeposits, err := s.repo.GetActiveDepositsForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active deposits: %w", err)
	}

	readyDeposits, err := s.repo.GetReadyDepositsForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ready deposits: %w", err)
	}

	// Calculate total pending gems
	totalGems := 0
	for _, deposit := range readyDeposits {
		// TODO: Calculate gems based on item rarity and quantity
		totalGems += deposit.Quantity * 10 // Placeholder
	}

	return &domain.CompostStatus{
		ActiveDeposits:   activeDeposits,
		ReadyCount:       len(readyDeposits),
		TotalGemsPending: totalGems,
	}, nil
}

// Harvest harvests all ready deposits for a user
func (s *service) Harvest(ctx context.Context, platform, platformID string) (int, error) {
	// TODO: Implement harvest logic
	// This is a placeholder - actual implementation will be done later
	return 0, fmt.Errorf("not implemented")
}
