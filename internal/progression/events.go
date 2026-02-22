package progression

import (
	"context"

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
