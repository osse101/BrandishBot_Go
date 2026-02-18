package gamble

import (
	"context"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

func (s *service) publishGambleStartedEvent(ctx context.Context, gamble *domain.Gamble) {
	if s.eventBus == nil {
		logger.FromContext(ctx).Error("Failed to publish "+LogContextGambleStartedEvent, "reason", LogReasonEventBusNil)
		return
	}
	err := s.eventBus.Publish(ctx, event.Event{
		Version: EventSchemaVersion,
		Type:    domain.EventGambleStarted,
		Payload: gamble,
	})
	if err != nil {
		logger.FromContext(ctx).Error("Failed to publish "+LogContextGambleStartedEvent, "error", err)
	}
}

func (s *service) publishGambleParticipatedEvent(ctx context.Context, gambleID, userID string, lootboxCount int, source string) {
	if s.resilientPublisher == nil {
		return
	}
	s.resilientPublisher.PublishWithRetry(ctx, event.Event{
		Version: EventSchemaVersion,
		Type:    event.Type(domain.EventTypeGambleParticipated),
		Payload: domain.GambleParticipatedPayload{
			GambleID:     gambleID,
			UserID:       userID,
			LootboxCount: lootboxCount,
			Source:       source,
			Timestamp:    time.Now().Unix(),
		},
	})
}

func (s *service) publishGambleCompletedEvent(ctx context.Context, result *domain.GambleResult, participantCount int, participants []domain.GambleParticipantOutcome) {
	log := logger.FromContext(ctx)

	if s.resilientPublisher == nil {
		log.Error("Failed to publish GambleCompleted event", "reason", "resilientPublisher is nil")
		return
	}

	evt := event.NewGambleCompletedEvent(result.GambleID.String(), result.WinnerID, result.TotalValue, participantCount, participants)
	s.resilientPublisher.PublishWithRetry(ctx, evt)
}
