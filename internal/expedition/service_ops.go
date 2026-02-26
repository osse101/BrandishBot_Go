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
func (s *service) StartExpedition(ctx context.Context, platform, platformID, username, expeditionType string) (*domain.Expedition, error) {
	// Get initiator
	initiator, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to get initiator: %w", err)
	}

	initiatorID, err := uuid.Parse(initiator.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid initiator ID: %w", err)
	}

	// Check cooldown
	if s.cooldownSvc != nil {
		onCooldown, remaining, err := s.cooldownSvc.CheckCooldown(ctx, initiator.ID, "expedition")
		if err != nil {
			return nil, fmt.Errorf("failed to check cooldown: %w", err)
		}
		if onCooldown {
			return nil, fmt.Errorf("expedition on cooldown for %s", remaining.Truncate(time.Second))
		}
	}

	// Check no active expedition
	active, err := s.repo.GetActiveExpedition(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check active expedition: %w", err)
	}
	if active != nil {
		return nil, fmt.Errorf("an expedition is already active")
	}

	// Create expedition
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
		Username:     username,
		JoinedAt:     time.Now(),
	}

	if err := s.repo.AddParticipant(ctx, participant); err != nil {
		return fmt.Errorf("failed to add participant: %w", err)
	}

	return nil
}
