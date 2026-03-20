package user

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/itemhandler"
	"github.com/osse101/BrandishBot_Go/internal/lootbox"
)

// Compile-time check: service implements itemhandler.EffectContext
var _ itemhandler.EffectContext = (*service)(nil)

// GetDisplayName returns a display name for an item with quality prefix.
func (s *service) GetDisplayName(itemName string, quality domain.QualityLevel) string {
	return s.namingResolver.GetDisplayName(itemName, quality)
}

// Pluralize delegates to the itemhandler package's exported Pluralize function.
func (s *service) Pluralize(name string, quantity int) string {
	return itemhandler.Pluralize(name, quantity)
}

// GetRandomTarget returns a single random active chatter.
func (s *service) GetRandomTarget(platform string) (username, userID string, err error) {
	return s.activeChatterTracker.GetRandomTarget(platform)
}

// GetRandomTargets returns multiple random active chatters.
func (s *service) GetRandomTargets(platform string, count int) ([]itemhandler.ActiveTarget, error) {
	targets, err := s.activeChatterTracker.GetRandomTargets(platform, count)
	if err != nil {
		return nil, err
	}
	result := make([]itemhandler.ActiveTarget, len(targets))
	for i, t := range targets {
		result[i] = itemhandler.ActiveTarget{
			UserID:   t.UserID,
			Username: t.Username,
		}
	}
	return result, nil
}

// RemoveActiveChatter removes a user from the active chatter tracker.
func (s *service) RemoveActiveChatter(platform, userID string) {
	s.activeChatterTracker.Remove(platform, userID)
}

// GetItemByName returns an item by its internal name.
func (s *service) GetItemByName(ctx context.Context, name string) (*domain.Item, error) {
	return s.getItemByNameCached(ctx, name)
}

// RecordUserEvent records a stats event for a user.
func (s *service) RecordUserEvent(ctx context.Context, userID string, eventType domain.EventType, data interface{}) error {
	if s.statsService == nil {
		return nil
	}
	return s.statsService.RecordUserEvent(ctx, userID, eventType, data)
}

// PublishItemUsedEvent publishes an item-used event through the resilient publisher.
func (s *service) PublishItemUsedEvent(ctx context.Context, userID, itemName string, quantity int, metadata map[string]interface{}) {
	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.Event{
			Version: "1.1",
			Type:    domain.EventTypeItemUsed,
			Payload: domain.ItemUsedPayload{
				UserID:    userID,
				ItemName:  itemName,
				Quantity:  quantity,
				Metadata:  metadata,
				Timestamp: time.Now().Unix(),
			},
		})
	}
}

// OpenLootbox opens a lootbox via the lootbox service.
func (s *service) OpenLootbox(ctx context.Context, lootboxName string, quantity int, boxQuality domain.QualityLevel) ([]lootbox.DroppedItem, error) {
	return s.lootboxService.OpenLootbox(ctx, lootboxName, quantity, boxQuality)
}

// GetUserByPlatformUsername is already declared in registration.go
// and satisfies the itemhandler.EffectContext interface.

// CreateTrap creates a new trap in the database.
func (s *service) CreateTrap(ctx context.Context, trap *domain.Trap) error {
	return s.trapRepo.CreateTrap(ctx, trap)
}

// GetActiveTrapForUpdate gets an active trap for a target user (with row lock).
func (s *service) GetActiveTrapForUpdate(ctx context.Context, targetID uuid.UUID) (*domain.Trap, error) {
	return s.trapRepo.GetActiveTrapForUpdate(ctx, targetID)
}

// TriggerTrap marks a trap as triggered.
func (s *service) TriggerTrap(ctx context.Context, trapID uuid.UUID) error {
	return s.trapRepo.TriggerTrap(ctx, trapID)
}

// RandomFloat returns a random float [0.0, 1.0).
func (s *service) RandomFloat() float64 {
	return s.rnd()
}

// SetPendingBomb adds a bomb to the queue for a platform.
func (s *service) SetPendingBomb(ctx context.Context, platform, setterUsername string, timeout time.Duration) error {
	s.bombMu.Lock()
	defer s.bombMu.Unlock()

	if s.bombQueues[platform] == nil {
		s.bombQueues[platform] = make([]*pendingBomb, 0)
	}

	s.bombQueues[platform] = append(s.bombQueues[platform], &pendingBomb{
		SetterUsername:   setterUsername,
		Timeout:          timeout,
		AccumulatedUsers: make(map[string]bool),
	})

	return nil
}
