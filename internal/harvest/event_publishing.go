package harvest

import (
	"context"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
)

func (s *service) fireAsyncEvents(ctx context.Context, userID string, hoursElapsed float64) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		asyncCtx := context.WithoutCancel(ctx)
		s.awardFarmerXP(asyncCtx, userID, hoursElapsed)
	}()
}

func (s *service) awardFarmerXP(ctx context.Context, userID string, hoursElapsed float64) {
	if hoursElapsed < farmerXPThreshold {
		return
	}

	// Cap XP at 120 hours (5 days)
	xpWaitHours := hoursElapsed
	if xpWaitHours > 120.0 {
		xpWaitHours = 120.0
	}

	xpAmount := int(xpWaitHours * farmerXPPerHour)
	spoiled := hoursElapsed > spoiledThreshold

	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.Event{
			Version: "1.0",
			Type:    event.Type(domain.EventTypeHarvestCompleted),
			Payload: domain.HarvestCompletedPayload{
				UserID:       userID,
				HoursElapsed: hoursElapsed,
				XPAmount:     xpAmount,
				Spoiled:      spoiled,
				Timestamp:    time.Now().Unix(),
			},
		})
	}
}
