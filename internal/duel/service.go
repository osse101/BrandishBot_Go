package duel

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// UserService defines the interface for interacting with users and their inventory/timeouts
type UserService interface {
	RemoveItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) (int, error)
	AddItemByUsername(ctx context.Context, platform, username, itemName string, quantity int) error
	AddTimeout(ctx context.Context, platform, username string, duration time.Duration, reason string) error
	GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error)
	FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
}

// UserRepo interface strictly for extracting username from ID easily
type UserRepo interface {
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
}

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
	userRepo       repository.User
	eventBus       event.Bus
	progressionSvc ProgressionService
	userSvc        UserService
	expireDuration time.Duration
}

// NewService creates a new duel service
func NewService(repo repository.Duel, userRepo repository.User, eventBus event.Bus, progressionSvc ProgressionService, userSvc UserService, expireDuration time.Duration) Service {
	return &service{
		repo:           repo,
		userRepo:       userRepo,
		eventBus:       eventBus,
		progressionSvc: progressionSvc,
		userSvc:        userSvc,
		expireDuration: expireDuration,
	}
}

// Challenge creates a new duel challenge
func (s *service) Challenge(ctx context.Context, platform, platformID, opponentUsername string, stakes domain.DuelStakes) (*domain.Duel, error) {
	// Get challenger
	challenger, err := s.userSvc.FindUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenger: %w", err)
	}

	// Get opponent
	opponent, err := s.userSvc.GetUserByPlatformUsername(ctx, platform, opponentUsername)
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

	// Validate challenger has stakes if it's an item wager
	if stakes.WagerItemKey != "" && stakes.WagerAmount > 0 {
		_, err := s.userSvc.RemoveItemByUsername(ctx, platform, challenger.Username, stakes.WagerItemKey, stakes.WagerAmount)
		if err != nil {
			return nil, fmt.Errorf("challenger lacks required wager items: %w", err)
		}
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
		// Restore item wager on failure
		if stakes.WagerItemKey != "" && stakes.WagerAmount > 0 {
			_ = s.userSvc.AddItemByUsername(ctx, platform, challenger.Username, stakes.WagerItemKey, stakes.WagerAmount)
		}
		return nil, fmt.Errorf("failed to create duel: %w", err)
	}

	return duel, nil
}

func (s *service) processDuelStakes(ctx context.Context, platform string, duel *domain.Duel, winnerUsername, loserUsername string) error {
	if duel.Stakes.WagerItemKey != "" && duel.Stakes.WagerAmount > 0 {
		wagerPool := duel.Stakes.WagerAmount * 2
		err := s.userSvc.AddItemByUsername(ctx, platform, winnerUsername, duel.Stakes.WagerItemKey, wagerPool)
		if err != nil {
			return fmt.Errorf("failed to reward winner: %w", err)
		}
	}

	if duel.Stakes.TimeoutDuration > 0 {
		err := s.userSvc.AddTimeout(ctx, platform, loserUsername, time.Duration(duel.Stakes.TimeoutDuration)*time.Second, "Lost duel against "+winnerUsername)
		if err != nil {
			return fmt.Errorf("failed to timeout loser: %w", err)
		}
	}
	return nil
}

func (s *service) validateAndGetDuel(ctx context.Context, tx repository.DuelTx, platform, platformID string, duelID uuid.UUID) (*domain.Duel, *domain.User, error) {
	duel, err := tx.GetDuel(ctx, duelID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get duel: %w", err)
	}

	if duel.State != domain.DuelStatePending {
		return nil, nil, fmt.Errorf("duel is not pending")
	}

	if duel.ExpiresAt.Before(time.Now()) {
		_ = tx.UpdateDuelState(ctx, duelID, domain.DuelStateExpired)
		_ = tx.Commit(ctx)
		return nil, nil, fmt.Errorf("duel has expired")
	}

	// Verify accept caller is the opponent
	opponent, err := s.userSvc.FindUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get opponent: %w", err)
	}

	if duel.OpponentID == nil || opponent.ID != duel.OpponentID.String() {
		return nil, nil, fmt.Errorf("unauthorized to accept this duel")
	}

	return duel, opponent, nil
}

func (s *service) ensureOpponentHasStakes(ctx context.Context, platform, username string, stakes domain.DuelStakes) error {
	if stakes.WagerItemKey != "" && stakes.WagerAmount > 0 {
		_, err := s.userSvc.RemoveItemByUsername(ctx, platform, username, stakes.WagerItemKey, stakes.WagerAmount)
		if err != nil {
			return fmt.Errorf("opponent lacks required wager items: %w", err)
		}
	}
	return nil
}

func (s *service) resolveWinner(duel *domain.Duel, challenger, opponent *domain.User) (uuid.UUID, uuid.UUID, string, string) {
	n, _ := rand.Int(rand.Reader, big.NewInt(2))
	if n.Int64() == 0 {
		return duel.ChallengerID, *duel.OpponentID, challenger.Username, opponent.Username
	}
	return *duel.OpponentID, duel.ChallengerID, opponent.Username, challenger.Username
}

// Accept accepts a duel challenge and executes it
func (s *service) Accept(ctx context.Context, platform, platformID string, duelID uuid.UUID) (*domain.DuelResult, error) {
	tx, err := s.repo.BeginDuelTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	duel, opponent, err := s.validateAndGetDuel(ctx, tx, platform, platformID, duelID)
	if err != nil {
		return nil, err
	}

	challenger, err := s.userRepo.GetUserByID(ctx, duel.ChallengerID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get challenger: %w", err)
	}

	if err := s.ensureOpponentHasStakes(ctx, platform, opponent.Username, duel.Stakes); err != nil {
		return nil, err
	}

	winnerID, loserID, winnerUsername, loserUsername := s.resolveWinner(duel, challenger, opponent)

	result := &domain.DuelResult{
		WinnerID: winnerID,
		LoserID:  loserID,
		Method:   "coin_flip",
		Details:  "50/50 random selection",
	}

	if err := s.processDuelStakes(ctx, platform, duel, winnerUsername, loserUsername); err != nil {
		return nil, err
	}

	if err := tx.AcceptDuel(ctx, duelID, result); err != nil {
		return nil, fmt.Errorf("failed to accept duel: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}

// Decline declines a duel challenge
func (s *service) Decline(ctx context.Context, platform, platformID string, duelID uuid.UUID) error {
	tx, err := s.repo.BeginDuelTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	duel, err := tx.GetDuel(ctx, duelID)
	if err == nil && duel.Stakes.WagerItemKey != "" && duel.Stakes.WagerAmount > 0 {
		challenger, errChallenger := s.userRepo.GetUserByID(ctx, duel.ChallengerID.String())
		if errChallenger == nil {
			_ = s.userSvc.AddItemByUsername(ctx, platform, challenger.Username, duel.Stakes.WagerItemKey, duel.Stakes.WagerAmount)
		}
	}

	err = tx.UpdateDuelState(ctx, duelID, domain.DuelStateDeclined)
	if err == nil {
		_ = tx.Commit(ctx)
	}
	return err
}

// GetPendingDuels retrieves all pending duels for a user
func (s *service) GetPendingDuels(ctx context.Context, platform, platformID string) ([]domain.Duel, error) {
	user, err := s.userSvc.FindUserByPlatformID(ctx, platform, platformID)
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
