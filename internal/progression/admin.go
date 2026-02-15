package progression

import (
	"context"
	"fmt"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// AdminUnlock forces a node to unlock (for testing)
func (s *service) AdminUnlock(ctx context.Context, nodeKey string, level int) error {
	log := logger.FromContext(ctx)

	node, err := s.repo.GetNodeByKey(ctx, nodeKey)
	if err != nil || node == nil {
		return fmt.Errorf("%w: %s", domain.ErrNodeNotFound, nodeKey)
	}

	if level > node.MaxLevel {
		return fmt.Errorf("%w: level %d exceeds max level %d for node %s", domain.ErrMaxLevelExceeded, level, node.MaxLevel, nodeKey)
	}

	engagementScore, err := s.GetEngagementScore(ctx)
	if err != nil {
		log.Warn("Failed to get engagement score for unlock", "error", err)
		engagementScore = 0
	}

	if err := s.repo.UnlockNode(ctx, node.ID, level, "admin", engagementScore); err != nil {
		return fmt.Errorf("failed to unlock node: %w", err)
	}

	log.Info("Admin unlocked node", "nodeKey", nodeKey, "level", level)
	return nil
}

// AdminUnlockAll unlocks all progression nodes at their max level (for debugging)
func (s *service) AdminUnlockAll(ctx context.Context) error {
	log := logger.FromContext(ctx)

	// Get all nodes
	nodes, err := s.repo.GetAllNodes(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all nodes: %w", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("%w: no nodes found", domain.ErrNodeNotFound)
	}

	// Unlock each node at its max level
	unlockedCount := 0
	for _, node := range nodes {
		if err := s.AdminUnlock(ctx, node.NodeKey, node.MaxLevel); err != nil {
			log.Warn("Failed to unlock node", "nodeKey", node.NodeKey, "error", err)
			continue
		}
		unlockedCount++
	}

	log.Info("Admin unlocked all nodes", "total", len(nodes), "unlocked", unlockedCount)
	return nil
}

// AdminRelock locks a node again (for testing)
func (s *service) AdminRelock(ctx context.Context, nodeKey string, level int) error {
	log := logger.FromContext(ctx)

	node, err := s.repo.GetNodeByKey(ctx, nodeKey)
	if err != nil || node == nil {
		return fmt.Errorf("%w: %s", domain.ErrNodeNotFound, nodeKey)
	}

	if err := s.repo.RelockNode(ctx, node.ID, level); err != nil {
		return fmt.Errorf("failed to relock node: %w", err)
	}

	log.Info("Admin relocked node", "nodeKey", nodeKey, "level", level)
	return nil
}

// ResetProgressionTree performs annual reset
func (s *service) ResetProgressionTree(ctx context.Context, resetBy string, reason string, preserveUserData bool) error {
	log := logger.FromContext(ctx)
	log.Info("Resetting progression tree", "resetBy", resetBy, "reason", reason)

	return s.repo.ResetTree(ctx, resetBy, reason, preserveUserData)
}

// CheckAndUnlockCriteria checks if unlock criteria met
func (s *service) CheckAndUnlockCriteria(ctx context.Context) (*domain.ProgressionUnlock, error) {
	log := logger.FromContext(ctx)
	reqID := logger.GetRequestID(ctx)

	// Check if there's a node waiting to unlock
	unlock, err := s.CheckAndUnlockNode(ctx)
	if err != nil || unlock != nil {
		return unlock, err
	}

	// If no session exists, start one
	session, _ := s.repo.GetActiveSession(ctx)
	if session == nil {
		// Use shutdown context for async operation with timeout
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()

			ctx, cancel := context.WithTimeout(s.shutdownCtx, 1*time.Minute)
			defer cancel()

			// Inject request ID into context for tracing
			if reqID != "" {
				ctx = logger.WithRequestID(ctx, reqID)
			}

			if err := s.StartVotingSession(ctx, nil); err != nil {
				log.Error("Failed to auto-start voting session", "error", err)
			}
		}()
	}

	return nil, nil
}

// ForceInstantUnlock selects highest voted option and unlocks immediately
func (s *service) ForceInstantUnlock(ctx context.Context) (*domain.ProgressionUnlock, error) {
	log := logger.FromContext(ctx)

	// Get active session
	session, err := s.repo.GetActiveSession(ctx)
	if err != nil || session == nil {
		return nil, domain.ErrNoActiveSession
	}

	if session.Status != domain.VotingStatusVoting {
		return nil, domain.ErrNoActiveSession
	}

	// Find winning option
	winner := findWinningOption(session.Options)
	if winner == nil {
		return nil, domain.ErrNoActiveSession
	}

	// End voting session
	winnerID := winner.ID
	if err := s.repo.EndVotingSession(ctx, session.ID, &winnerID); err != nil {
		return nil, fmt.Errorf("failed to end voting: %w", err)
	}

	// Set unlock target
	progress, _ := s.repo.GetActiveUnlockProgress(ctx)
	if progress != nil {
		if err := s.repo.SetUnlockTarget(ctx, progress.ID, winner.NodeID, winner.TargetLevel, session.ID); err != nil {
			log.Warn("Failed to set unlock target during instant unlock", "error", err)
		}
	}

	// Unlock the node immediately
	engagementScore, err := s.GetEngagementScore(ctx)
	if err != nil {
		log.Warn("Failed to get engagement score for instant unlock", "error", err)
		engagementScore = 0
	}

	if err := s.repo.UnlockNode(ctx, winner.NodeID, winner.TargetLevel, "instant_override", engagementScore); err != nil {
		return nil, fmt.Errorf("failed to unlock node: %w", err)
	}

	// Mark progress complete and start new
	if progress != nil {
		if _, err := s.repo.CompleteUnlock(ctx, progress.ID, 0); err != nil {
			log.Error("Failed to complete unlock progress", "error", err)
			// We don't return error here because the node IS unlocked, but we log the inconsistency
		}
	}

	// Start new voting session with the unlocked node context
	reqID := logger.GetRequestID(ctx)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		ctx, cancel := context.WithTimeout(s.shutdownCtx, 1*time.Minute)
		defer cancel()

		// Inject request ID into context for tracing
		if reqID != "" {
			ctx = logger.WithRequestID(ctx, reqID)
		}

		if err := s.StartVotingSession(ctx, &winner.NodeID); err != nil {
			log.Error("Failed to auto-start voting session after instant unlock", "error", err)
		}
	}()

	// Return the unlock
	return s.repo.GetUnlock(ctx, winner.NodeID, winner.TargetLevel)
}
