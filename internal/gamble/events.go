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

	winnerUsername := ""
	if result.WinnerID != "" {
		for _, p := range participants {
			if p.UserID == result.WinnerID {
				winnerUsername = p.Username
				break
			}
		}
	}

	evt := event.NewGambleCompletedEvent(result.GambleID.String(), result.WinnerID, winnerUsername, result.TotalValue, participantCount, participants, result.Items)
	s.resilientPublisher.PublishWithRetry(ctx, evt)
}

func (s *service) publishGambleRefundedEvent(ctx context.Context, gamble *domain.Gamble) {
	if s.resilientPublisher == nil {
		return
	}

	// We reuse GambleCompletedEvent but with no winner and 0 value to signify cancellation/refund
	// In the future, we could add a specific GambleRefunded event, but this is sufficient for now

	evt := event.NewGambleCompletedEvent(
		gamble.ID.String(),
		"", // No winner
		"",
		0, // 0 value
		len(gamble.Participants),
		s.buildParticipantOutcomes(gamble, make(map[string]int64), "", nil, nil, nil),
		nil,
	)
	s.resilientPublisher.PublishWithRetry(ctx, evt)
}
