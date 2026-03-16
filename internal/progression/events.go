package progression

import (
	"context"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// handleNodeUnlocked invalidates caches when any node is unlocked
func (s *service) handleNodeUnlocked(ctx context.Context, e event.Event) error {
	// Invalidate modifier cache - values may have changed
	s.modifierCache.InvalidateAll()

	// Invalidate unlock cache - new features may be available
	s.unlockCache.InvalidateAll()

	log := logger.FromContext(ctx)
	if payload, ok := e.Payload.(map[string]interface{}); ok {
		log.Info("Invalidated caches due to node unlock",
			"node_key", payload["node_key"],
			"level", payload["level"])
	}
	return nil
}

// handleNodeRelocked invalidates caches when any node is relocked
func (s *service) handleNodeRelocked(ctx context.Context, e event.Event) error {
	// Invalidate modifier cache - values have changed
	s.modifierCache.InvalidateAll()

	// Invalidate unlock cache - features may no longer be available
	s.unlockCache.InvalidateAll()

	log := logger.FromContext(ctx)
	if payload, ok := e.Payload.(map[string]interface{}); ok {
		log.Info("Invalidated caches due to node relock",
			"node_key", payload["node_key"],
			"level", payload["level"])
	}
	return nil
}

// handleEngagement records engagement metrics from events
func (s *service) handleEngagement(ctx context.Context, e event.Event) error {
	// Skip if already recorded by the service to prevent infinite loop
	if recorded, ok := e.GetMetadataValue(domain.MetadataKeyRecorded).(bool); ok && recorded {
		return nil
	}

	metric, err := event.DecodePayload[*domain.EngagementMetric](e.Payload)
	if err != nil {
		// Log error but don't fail event processing
		logger.FromContext(ctx).Error("Failed to decode engagement metric payload", "error", err)
		return nil
	}

	if metric == nil || metric.UserID == "" {
		return nil
	}

	if err := s.RecordEngagement(ctx, metric.UserID, metric.MetricType, metric.MetricValue); err != nil {
		logger.FromContext(ctx).Error("Failed to record engagement from event",
			"error", err,
			"user_id", metric.UserID,
			"metric", metric.MetricType)
		return err
	}

	return nil
}
