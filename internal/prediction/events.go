package prediction

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
)

func (s *service) publishPredictionEvent(ctx context.Context, req *domain.PredictionOutcomeRequest, contribution int) {
	evt := event.Event{
		Version: "1.0",
		Type:    event.Type(domain.EventTypePredictionProcessed),
		Payload: map[string]interface{}{
			"platform":           req.Platform,
			"winner":             req.Winner.Username,
			"total_points":       req.TotalPointsSpent,
			"contribution":       contribution,
			"participants_count": len(req.Participants),
		},
		Metadata: make(map[string]interface{}),
	}

	s.resilientPublisher.PublishWithRetry(ctx, evt)
}
