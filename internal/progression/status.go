package progression

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// GetProgressionStatus returns current community progression status
func (s *service) GetProgressionStatus(ctx context.Context) (*domain.ProgressionStatus, error) {
	unlocks, err := s.repo.GetAllUnlocks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get unlocks: %w", err)
	}

	// Get total node count to determine if all are unlocked
	allNodes, err := s.repo.GetAllNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all nodes: %w", err)
	}

	// Check if all nodes are unlocked at their max level
	allUnlocked := s.checkAllNodesUnlocked(allNodes, unlocks)

	contributionScore, err := s.GetEngagementScore(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get contribution score: %w", err)
	}

	activeSession, _ := s.GetActiveVotingSession(ctx) // Use service method to get enriched session
	unlockProgress, _ := s.GetUnlockProgress(ctx)     // Use service method (which currently just calls repo, but consistent)

	// Enrich unlock progress with estimate
	if unlockProgress != nil && unlockProgress.NodeID != nil {
		node, err := s.repo.GetNodeByID(ctx, *unlockProgress.NodeID)
		if err == nil && node != nil {
			estimate, err := s.EstimateUnlockTime(ctx, node.NodeKey)
			if err == nil && estimate != nil {
				unlockProgress.EstimatedUnlockDate = estimate.EstimatedUnlockDate
			}
		}
	}

	isTransitioning := false
	if activeSession == nil && unlockProgress != nil && unlockProgress.UnlockedAt != nil {
		isTransitioning = true
	}

	return &domain.ProgressionStatus{
		TotalUnlocked:        len(unlocks),
		TotalNodes:           len(allNodes),
		AllNodesUnlocked:     allUnlocked,
		ContributionScore:    contributionScore,
		ActiveSession:        activeSession,
		ActiveUnlockProgress: unlockProgress,
		IsTransitioning:      isTransitioning,
	}, nil
}
