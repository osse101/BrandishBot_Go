package expedition

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
)

// StartExpedition creates a new expedition
func (s *service) StartExpedition(ctx context.Context, platform, platformID, username string, expeditionType domain.ExpeditionType) (*domain.Expedition, error) {
	if err := s.validateType(expeditionType); err != nil {
		return nil, err
	}

	initiator, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to get initiator: %w", err)
	}

	if err := s.checkConstraints(ctx, initiator, platform, username); err != nil {
		return nil, err
	}

	active, err := s.repo.GetActiveExpedition(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check active expedition: %w", err)
	}
	if active != nil {
		return nil, fmt.Errorf("an expedition is already active")
	}

	initiatorID, _ := uuid.Parse(initiator.ID)
	now := time.Now()
	expedition := &domain.Expedition{
		ID:                 uuid.New(),
		InitiatorID:        initiatorID,
		ExpeditionType:     expeditionType,
		State:              domain.ExpeditionStateRecruiting,
		CreatedAt:          now,
		JoinDeadline:       now.Add(s.joinDuration),
		CompletionDeadline: now.Add(s.joinDuration + 30*time.Minute),
	}

	if err := s.repo.CreateExpedition(ctx, expedition); err != nil {
		return nil, fmt.Errorf("failed to create expedition: %w", err)
	}

	// Add initiator as first participant (leader)
	participant := &domain.ExpeditionParticipant{
		ExpeditionID: expedition.ID,
		UserID:       initiatorID,
		Username:     username,
		JoinedAt:     now,
		IsLeader:     true,
	}

	if err := s.repo.AddParticipant(ctx, participant); err != nil {
		return nil, fmt.Errorf("failed to add initiator as participant: %w", err)
	}

	// Publish event for worker to schedule execution
	_ = s.eventBus.Publish(ctx, event.Event{
		Version: "1.0",
		Type:    event.Type(domain.EventExpeditionStarted),
		Payload: expedition,
	})

	return expedition, nil
}

func (s *service) validateType(expeditionType domain.ExpeditionType) error {
	isValid := false
	for _, t := range domain.ValidExpeditionTypes {
		if expeditionType == t {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("%w: %s", domain.ErrInvalidExpeditionType, expeditionType)
	}
	return nil
}

func (s *service) checkConstraints(ctx context.Context, initiator *domain.User, platform, username string) error {
	// Check global cooldown (10 minutes after completion)
	last, err := s.repo.GetLastCompletedExpedition(ctx)
	if err != nil {
		return fmt.Errorf("failed to check global cooldown: %w", err)
	}
	if last != nil && last.CompletedAt != nil {
		cooldownEnd := last.CompletedAt.Add(10 * time.Minute)
		if time.Now().Before(cooldownEnd) {
			remaining := time.Until(cooldownEnd).Truncate(time.Second)
			return fmt.Errorf("%w: expedition on global cooldown for %s", domain.ErrOnCooldown, remaining)
		}
	}

	// Check initiator-specific cooldown
	if s.cooldownSvc != nil {
		onCooldown, remaining, err := s.cooldownSvc.CheckCooldown(ctx, initiator.ID, "expedition")
		if err != nil {
			return fmt.Errorf("failed to check cooldown: %w", err)
		}
		if onCooldown {
			return fmt.Errorf("%w: %s", domain.ErrOnCooldown, remaining.Truncate(time.Second))
		}
	}

	// Check Explorer level 5 requirement
	level, err := s.jobSvc.GetJobLevel(ctx, initiator.ID, domain.JobKeyExplorer)
	if err != nil {
		return fmt.Errorf("failed to check explorer level: %w", err)
	}
	if level < 5 {
		return fmt.Errorf("%w: 5 (currently level %d)", domain.ErrInsufficientLevel, level)
	}

	// Deduct 500 money cost
	removed, err := s.userSvc.RemoveItemByUsername(ctx, platform, username, domain.ItemMoney, 500)
	if err != nil {
		return fmt.Errorf("failed to deduct cost: %w", err)
	}
	if removed < 500 {
		return fmt.Errorf("%w: cost 500", domain.ErrInsufficientFunds)
	}

	return nil
}

// JoinExpedition adds a user to an expedition
func (s *service) JoinExpedition(ctx context.Context, platform, platformID, username string, expeditionID uuid.UUID) error {
	// If ID is nil, try to join the currently active expedition
	if expeditionID == uuid.Nil {
		active, err := s.repo.GetActiveExpedition(ctx)
		if err != nil {
			return fmt.Errorf("failed to check for active expedition: %w", err)
		}
		if active == nil {
			return domain.ErrNoActiveExpedition
		}
		expeditionID = active.Expedition.ID
	}

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
		Username:     username,
		JoinedAt:     time.Now(),
	}

	if err := s.repo.AddParticipant(ctx, participant); err != nil {
		return fmt.Errorf("failed to add participant: %w", err)
	}

	return nil
}
