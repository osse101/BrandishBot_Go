package progression

import (
	"context"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// updateLiveNodeCostAndCache recalculates the node cost, updates the database if changed, and updates the service's progress cache.
func (s *service) updateLiveNodeCostAndCache(ctx context.Context, node *domain.ProgressionNode, progressID int) {
	log := logger.FromContext(ctx)
	if node == nil {
		return
	}

	// Recalculate cost live to ensure latest tiers/sizes are used
	newCost, err := CalculateUnlockCost(node.Tier, NodeSize(node.Size))
	if err != nil {
		log.Warn("Failed to recalculate node cost, using existing", "error", err, "nodeKey", node.NodeKey)
		newCost = node.UnlockCost
	}

	// Update DB if cost differs from persisted value
	if newCost != node.UnlockCost {
		log.Info("Updating node cost live", "nodeKey", node.NodeKey, "oldCost", node.UnlockCost, "newCost", newCost)
		if err := s.repo.UpdateNodeCost(ctx, node.ID, newCost); err != nil {
			log.Warn("Failed to update node cost in repository", "error", err, "nodeKey", node.NodeKey)
		}
		node.UnlockCost = newCost
	}

	// Update service-level progress cache
	s.mu.Lock()
	s.cachedTargetCost = node.UnlockCost
	s.cachedProgressID = progressID
	s.mu.Unlock()
}

// autoStartNextVotingSession handles the asynchronous startup of a new voting session after an unlock.
func (s *service) autoStartNextVotingSession(ctx context.Context, lastUnlockedNodeID int) {
	log := logger.FromContext(ctx)
	reqID := logger.GetRequestID(ctx)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		// Use a fresh context with timeout for the background task
		bgCtx, cancel := context.WithTimeout(s.shutdownCtx, 1*time.Minute)
		defer cancel()

		// Inject request ID for tracing
		if reqID != "" {
			bgCtx = logger.WithRequestID(bgCtx, reqID)
		}

		if err := s.StartVotingSession(bgCtx, &lastUnlockedNodeID); err != nil {
			log.Error("Failed to auto-start voting session after unlock", "error", err, "lastUnlockedNodeID", lastUnlockedNodeID)
		}
	}()
}
