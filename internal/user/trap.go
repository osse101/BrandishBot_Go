package user

import (
	"context"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// triggerTrap executes trap trigger logic when a user sends a message
func (s *service) triggerTrap(ctx context.Context, trap *domain.Trap, victim *domain.User) error {
	log := logger.FromContext(ctx)

	// 1. Mark trap as triggered
	if err := s.trapRepo.TriggerTrap(ctx, trap.ID); err != nil {
		return fmt.Errorf("failed to mark trap as triggered: %w", err)
	}

	// 2. Apply timeout
	timeout := time.Duration(trap.CalculateTimeout()) * time.Second
	if err := s.TimeoutUser(ctx, victim.Username, timeout, "BOOM! Stepped on a trap!"); err != nil {
		return fmt.Errorf("failed to timeout user: %w", err)
	}

	// 3. Remove from active chatters (prevent immediate re-targeting by grenades)
	s.activeChatterTracker.Remove(domain.PlatformTwitch, victim.ID)

	// 4. Publish event
	if s.statsService != nil {
		// Fetch setter info for event
		setter, err := s.repo.GetUserByID(ctx, trap.SetterID.String())
		if err != nil {
			log.Warn("Failed to get trap setter for event", "setter_id", trap.SetterID)
		} else {
			eventData := &domain.TrapTriggeredData{
				TrapID:           trap.ID,
				SetterID:         trap.SetterID,
				SetterUsername:   setter.Username,
				TargetID:         trap.TargetID,
				TargetUsername:   victim.Username,
				QualityLevel:     trap.QualityLevel,
				TimeoutSeconds:   trap.CalculateTimeout(),
				WasSelfTriggered: false,
			}
			_ = s.statsService.RecordUserEvent(ctx, victim.ID, domain.EventTrapTriggered, eventData.ToMap())
		}
	}

	log.Info(LogMsgTrapTriggered,
		"victim", victim.Username,
		"timeout", timeout.Seconds(),
		"trap_id", trap.ID)

	return nil
}
