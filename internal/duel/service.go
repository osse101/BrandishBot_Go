package duel

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Service defines the interface for duel operations
type Service interface {
	Challenge(ctx context.Context, platform, platformID, opponentUsername string, stakes domain.DuelStakes) (*domain.Duel, error)
	Accept(ctx context.Context, platform, platformID string, duelID uuid.UUID) (*domain.DuelResult, error)
	Decline(ctx context.Context, platform, platformID string, duelID uuid.UUID) error
	GetPendingDuels(ctx context.Context, platform, platformID string) ([]domain.Duel, error)
	GetDuel(ctx context.Context, duelID uuid.UUID) (*domain.Duel, error)
}

// ProgressionService defines the interface for progression system
type ProgressionService interface {
	RecordEngagement(ctx context.Context, username string, action string, amount int) error
}

type service struct {
	repo           repository.Duel
	eventBus       event.Bus
	progressionSvc ProgressionService
	expireDuration time.Duration
}

// NewService creates a new duel service
func NewService(repo repository.Duel, eventBus event.Bus, progressionSvc ProgressionService, expireDuration time.Duration) Service {
	return &service{
		repo:           repo,
		eventBus:       eventBus,
		progressionSvc: progressionSvc,
		expireDuration: expireDuration,
	}
}

// Challenge creates a new duel challenge
func (s *service) Challenge(ctx context.Context, platform, platformID, opponentUsername string, stakes domain.DuelStakes) (*domain.Duel, error) {
	// Get challenger
	challenger, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenger: %w", err)
	}

	// Get opponent
	opponent, err := s.repo.GetUserByPlatformID(ctx, "twitch", opponentUsername) // TODO: Make platform configurable
	if err != nil {
		return nil, fmt.Errorf("failed to get opponent: %w", err)
	}

	challengerID, err := uuid.Parse(challenger.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid challenger ID: %w", err)
	}

	opponentID, err := uuid.Parse(opponent.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid opponent ID: %w", err)
	}

	// Create duel
	duel := &domain.Duel{
		ID:           uuid.New(),
		ChallengerID: challengerID,
		OpponentID:   &opponentID,
		State:        domain.DuelStatePending,
		Stakes:       stakes,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(s.expireDuration),
	}

	if err := s.repo.CreateDuel(ctx, duel); err != nil {
		return nil, fmt.Errorf("failed to create duel: %w", err)
	}

	return duel, nil
}

// Accept accepts a duel challenge and executes it
func (s *service) Accept(ctx context.Context, platform, platformID string, duelID uuid.UUID) (*domain.DuelResult, error) {
	// TODO: Implement duel execution logic (coin flip, dice roll, etc.)
	// This is a placeholder - actual implementation will be done later
	return nil, fmt.Errorf("not implemented")
}

// Decline declines a duel challenge
func (s *service) Decline(ctx context.Context, platform, platformID string, duelID uuid.UUID) error {
	return s.repo.DeclineDuel(ctx, duelID)
}

// GetPendingDuels retrieves all pending duels for a user
func (s *service) GetPendingDuels(ctx context.Context, platform, platformID string) ([]domain.Duel, error) {
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	return s.repo.GetPendingDuelsForUser(ctx, userID)
}

// GetDuel retrieves a duel by ID
func (s *service) GetDuel(ctx context.Context, duelID uuid.UUID) (*domain.Duel, error) {
	return s.repo.GetDuel(ctx, duelID)
}
