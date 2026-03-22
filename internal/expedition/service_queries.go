package expedition

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// GetExpedition retrieves expedition details
func (s *service) GetExpedition(ctx context.Context, expeditionID uuid.UUID) (*domain.ExpeditionDetails, error) {
	return s.repo.GetExpedition(ctx, expeditionID)
}

// GetActiveExpedition retrieves the current active expedition
func (s *service) GetActiveExpedition(ctx context.Context) (*domain.ExpeditionDetails, error) {
	return s.repo.GetActiveExpedition(ctx)
}

// GetJournal retrieves journal entries for a completed expedition
func (s *service) GetJournal(ctx context.Context, expeditionID uuid.UUID) ([]domain.ExpeditionJournalEntry, error) {
	return s.repo.GetJournalEntries(ctx, expeditionID)
}

// GetStatus returns the current expedition system status including cooldown info
func (s *service) GetStatus(ctx context.Context) (*domain.ExpeditionStatus, error) {
	status := &domain.ExpeditionStatus{}

	// Check for active expedition
	active, err := s.repo.GetActiveExpedition(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active expedition: %w", err)
	}
	if active != nil {
		status.HasActive = true
		status.ActiveDetails = active
	}

	// Check for global cooldown
	last, err := s.repo.GetLastCompletedExpedition(ctx)
	if err == nil && last != nil && last.CompletedAt != nil {
		cooldownEnd := last.CompletedAt.Add(10 * time.Minute)
		if time.Now().Before(cooldownEnd) {
			status.OnCooldown = true
			status.CooldownExpires = &cooldownEnd
		}
	}

	return status, nil
}
