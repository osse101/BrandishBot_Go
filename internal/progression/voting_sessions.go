package progression

import (
	"context"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

const (
	// MaxVotingOptions is the maximum number of options shown in a voting session
	MaxVotingOptions = 4
)

// StartVotingSession creates a new voting session with 4 random options from available nodes
// unlockedNodeID should be provided if this session is being started after a node unlock
func (s *service) StartVotingSession(ctx context.Context, unlockedNodeID *int) error {
	log := logger.FromContext(ctx)

	// Ensure unlock progress exists (bootstrap case or after manual reset)
	progress, _ := s.repo.GetActiveUnlockProgress(ctx)
	if progress == nil {
		_, err := s.repo.CreateUnlockProgress(ctx)
		if err != nil {
			return fmt.Errorf("failed to create initial unlock progress: %w", err)
		}

		// Fetch it now so we have the ID for later
		progress, err = s.repo.GetActiveUnlockProgress(ctx)
		if err != nil {
			return fmt.Errorf("failed to get active unlock progress after creation: %w", err)
		}

		log.Debug("Created initial unlock progress for new voting session")
	}

	// Get available nodes with met prerequisites
	available, err := s.GetAvailableUnlocks(ctx)
	if err != nil {
		return fmt.Errorf("failed to get available nodes: %w", err)
	}

	if len(available) == 0 {
		return fmt.Errorf("no nodes available for voting")
	}

	// SPECIAL CASE: Auto-select if only one option available
	if len(available) == 1 {
		node := available[0]
		log.Info("Only one option available, auto-selecting without voting", "nodeKey", node.NodeKey)

		// Determine target level
		targetLevel := 1
		isUnlocked, _ := s.repo.IsNodeUnlocked(ctx, node.NodeKey, 1)
		if isUnlocked {
			// Find next level to unlock
			for level := 2; level <= node.MaxLevel; level++ {
				unlocked, _ := s.repo.IsNodeUnlocked(ctx, node.NodeKey, level)
				if !unlocked {
					targetLevel = level
					break
				}
			}
		}

		// Set as unlock target immediately
		err = s.repo.SetUnlockTarget(ctx, progress.ID, node.ID, targetLevel, 0)
		if err != nil {
			return fmt.Errorf("failed to set unlock target: %w", err)
		}

		// Cache the unlock cost
		s.mu.Lock()
		s.cachedTargetCost = node.UnlockCost
		s.cachedProgressID = progress.ID
		s.mu.Unlock()

		// Publish event if needed (target set)
		if s.bus != nil {
			if err := s.bus.Publish(ctx, event.Event{
				Type: event.ProgressionTargetSet,
				Payload: map[string]interface{}{
					"node_key":      node.NodeKey,
					"target_level":  targetLevel,
					"auto_selected": true,
				},
			}); err != nil {
				log.Error("Failed to publish progression target set event", "error", err)
			}
		}

		log.Info("Auto-selected target set", "nodeKey", node.NodeKey, "targetLevel", targetLevel)

		return nil
	}

	// Select up to MaxVotingOptions random options
	selected := selectRandomNodes(available, MaxVotingOptions)

	// Create voting session
	sessionID, err := s.repo.CreateVotingSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Add options to session
	for _, node := range selected {
		// Determine target level (next unlockable level)
		targetLevel := 1
		isUnlocked, _ := s.repo.IsNodeUnlocked(ctx, node.NodeKey, 1)
		if isUnlocked {
			// Find next level to unlock
			for level := 2; level <= node.MaxLevel; level++ {
				unlocked, _ := s.repo.IsNodeUnlocked(ctx, node.NodeKey, level)
				if !unlocked {
					targetLevel = level
					break
				}
			}
		}

		err = s.repo.AddVotingOption(ctx, sessionID, node.ID, targetLevel)
		if err != nil {
			log.Warn("Failed to add voting option", "nodeID", node.ID, "error", err)
		}
	}

	log.Info("Started new voting session", "sessionID", sessionID, "options", len(selected))

	// Publish progression cycle event if triggered by an unlock
	if unlockedNodeID != nil && s.bus != nil {
		// Fetch the unlocked node details
		unlockedNode, err := s.repo.GetNodeByID(ctx, *unlockedNodeID)
		if err != nil {
			log.Warn("Failed to get unlocked node for event", "nodeID", *unlockedNodeID, "error", err)
		} else {
			// Fetch the session with options for the event
			session, err := s.repo.GetActiveSession(ctx)
			if err != nil {
				log.Warn("Failed to get session for event", "error", err)
			} else {
				// Publish the combined event
				if err := s.bus.Publish(ctx, event.Event{
					Type: event.ProgressionCycleCompleted,
					Payload: map[string]interface{}{
						"unlocked_node":  unlockedNode,
						"voting_session": session,
					},
				}); err != nil {
					log.Error("Failed to publish progression cycle completed event", "error", err)
				} else {
					log.Info("Published progression cycle completed event",
						"unlockedNode", unlockedNode.NodeKey,
						"sessionID", sessionID)
				}
			}
		}
	}

	return nil
}

// EndVoting closes voting and determines winner
func (s *service) EndVoting(ctx context.Context) (*domain.ProgressionVotingOption, error) {
	log := logger.FromContext(ctx)

	session, err := s.repo.GetActiveSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active session: %w", err)
	}

	if session == nil || session.Status != SessionStatusVoting {
		return nil, fmt.Errorf("no active voting session")
	}

	// Find winning option (highest votes, earliest LastHighestVoteAt for ties)
	winner := findWinningOption(session.Options)
	if winner == nil {
		return nil, fmt.Errorf("no voting options found")
	}

	// End voting phase
	err = s.repo.EndVotingSession(ctx, session.ID, winner.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to end voting session: %w", err)
	}

	// Set unlock target on current progress
	progress, err := s.repo.GetActiveUnlockProgress(ctx)
	if err != nil {
		log.Warn("Failed to get active unlock progress", "error", err)
	}

	if progress == nil {
		// Defensive: Create progress if missing (shouldn't happen - StartVotingSession should create it)
		log.Warn("No active unlock progress found during EndVoting - creating as fallback (investigate if this occurs frequently)")
		var id int
		id, err = s.repo.CreateUnlockProgress(ctx)
		if err != nil {
			log.Warn("Failed to create unlock progress", "error", err)
		} else {
			// Get the newly created progress
			progress = &domain.UnlockProgress{ID: id} // Minimal object for ID
		}
	}

	if progress != nil {
		err = s.repo.SetUnlockTarget(ctx, progress.ID, winner.NodeID, winner.TargetLevel, session.ID)
		if err != nil {
			log.Warn("Failed to set unlock target", "error", err)
		} else {
			// Cache unlock cost for instant threshold checking
			if winner.NodeDetails != nil {
				s.mu.Lock()
				s.cachedTargetCost = winner.NodeDetails.UnlockCost
				s.cachedProgressID = progress.ID
				s.mu.Unlock()

				log.Debug("Cached unlock threshold",
					"progressID", progress.ID,
					"unlockCost", winner.NodeDetails.UnlockCost)
			}
		}
	}

	// Award contribution points to all voters
	voters, err := s.repo.GetSessionVoters(ctx, session.ID)
	if err != nil {
		log.Warn("Failed to get session voters", "error", err)
	} else {
		for _, voterID := range voters {
			metric := &domain.EngagementMetric{
				UserID:      voterID,
				MetricType:  "vote_cast",
				MetricValue: 1,
				Metadata: map[string]interface{}{
					"session_id": session.ID,
				},
			}
			if err := s.repo.RecordEngagement(ctx, metric); err != nil {
				log.Warn("Failed to record contribution", "userID", voterID, "error", err)
			}
		}

		// Add contribution points to unlock progress
		if progress != nil && len(voters) > 0 {
			err = s.repo.AddContribution(ctx, progress.ID, len(voters))
			if err != nil {
				log.Warn("Failed to add contributions to progress", "error", err)
			}
		}
	}

	log.Info("Voting ended",
		"sessionID", session.ID,
		"winningNode", winner.NodeID,
		"votes", winner.VoteCount,
		"voterCount", len(voters))

	return winner, nil
}

// AddContribution adds contribution points to current unlock progress
func (s *service) AddContribution(ctx context.Context, amount int) error {
	log := logger.FromContext(ctx)

	progress, err := s.repo.GetActiveUnlockProgress(ctx)
	if err != nil {
		return fmt.Errorf("failed to get unlock progress: %w", err)
	}

	var progressID int
	if progress == nil {
		// Create new progress if none exists
		progressID, err = s.repo.CreateUnlockProgress(ctx)
		if err != nil {
			return fmt.Errorf("failed to create unlock progress: %w", err)
		}
	} else {
		progressID = progress.ID
	}

	// Check for contribution boost upgrade
	isBoosted, _ := s.IsFeatureUnlocked(ctx, "upgrade_contribution_boost")
	if isBoosted {
		// Apply 1.5x multiplier (integer math: * 3 / 2)
		amount = (amount * 3) / 2
	}

	// Write contribution to DB
	err = s.repo.AddContribution(ctx, progressID, amount)
	if err != nil {
		return err
	}

	// Check cache for instant unlock detection (zero extra queries!)
	s.mu.RLock()
	cachedCost := s.cachedTargetCost
	cachedID := s.cachedProgressID
	s.mu.RUnlock()

	// If cache is populated and matches current progress
	if cachedCost > 0 && cachedID == progressID {
		// Get updated progress to check new total
		updatedProgress, err := s.repo.GetActiveUnlockProgress(ctx)
		if err == nil && updatedProgress != nil {
			// Threshold met - trigger unlock asynchronously
			if updatedProgress.ContributionsAccumulated >= cachedCost {
				log.Info("Unlock threshold met, triggering unlock",
					"accumulated", updatedProgress.ContributionsAccumulated,
					"required", cachedCost)

				// Non-blocking send to semaphore - if unlock already in progress, skip
				select {
				case s.unlockSem <- struct{}{}:
					// Got the semaphore, proceed with unlock
					s.wg.Add(1)
					go func() {
						defer s.wg.Done()
						defer func() { <-s.unlockSem }() // Release semaphore when done
						s.CheckAndUnlockNode(s.shutdownCtx)
						if _, err := s.CheckAndUnlockNode(s.shutdownCtx); err != nil {
							log.Error("Failed to check and unlock node in background", "error", err)
						}
					}()
				default:
					// Unlock already in progress, skip this trigger
					log.Debug("Unlock already in progress, skipping duplicate trigger")
				}
			}
		}
	}

	return nil
}

// CheckAndUnlockNode checks if unlock target is set and threshold is met
func (s *service) CheckAndUnlockNode(ctx context.Context) (*domain.ProgressionUnlock, error) {
	log := logger.FromContext(ctx)

	// Get current unlock progress
	progress, err := s.repo.GetActiveUnlockProgress(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get unlock progress: %w", err)
	}

	if progress == nil {
		// Create new progress entry
		_, err = s.repo.CreateUnlockProgress(ctx)
		return nil, err
	}

	// If node not set yet (voting in progress), nothing to do
	if progress.NodeID == nil {
		return nil, nil
	}

	// Get node details to check unlock cost
	node, err := s.repo.GetNodeByID(ctx, *progress.NodeID)
	if err != nil || node == nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Check if threshold met
	if progress.ContributionsAccumulated >= node.UnlockCost {
		// Calculate rollover
		rollover := progress.ContributionsAccumulated - node.UnlockCost

		// Unlock the node
		err = s.repo.UnlockNode(ctx, *progress.NodeID, *progress.TargetLevel, "vote", progress.ContributionsAccumulated)
		if err != nil {
			return nil, fmt.Errorf("failed to unlock node: %w", err)
		}

		// Complete unlock and start next with rollover
		_, err = s.repo.CompleteUnlock(ctx, progress.ID, rollover)
		if err != nil {
			log.Warn("Failed to complete unlock progress", "error", err)
		}

		// Clear cache since we're starting a new cycle
		s.mu.Lock()
		s.cachedTargetCost = 0
		s.cachedProgressID = 0
		s.mu.Unlock()

		log.Info("Node unlocked",
			"nodeKey", node.NodeKey,
			"level", *progress.TargetLevel,
			"contributions", progress.ContributionsAccumulated,
			"rollover", rollover)

		// Start next voting session with context about the unlocked node
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			if err := s.StartVotingSession(s.shutdownCtx, &node.ID); err != nil {
				log.Error("Failed to start voting session in background", "error", err)
			}
		}()

		return &domain.ProgressionUnlock{
			NodeID:          *progress.NodeID,
			CurrentLevel:    *progress.TargetLevel,
			UnlockedBy:      "vote",
			EngagementScore: progress.ContributionsAccumulated,
		}, nil
	}

	log.Debug("Waiting for contribution threshold",
		"current", progress.ContributionsAccumulated,
		"required", node.UnlockCost,
		"remaining", node.UnlockCost-progress.ContributionsAccumulated)

	return nil, nil
}

// GetUnlockProgress returns current unlock progress status
func (s *service) GetUnlockProgress(ctx context.Context) (*domain.UnlockProgress, error) {
	return s.repo.GetActiveUnlockProgress(ctx)
}

// AdminForceEndVoting allows admins to force end current voting
func (s *service) AdminForceEndVoting(ctx context.Context) (*domain.ProgressionVotingOption, error) {
	return s.EndVoting(ctx)
}

// Helper functions

func selectRandomNodes(nodes []*domain.ProgressionNode, count int) []*domain.ProgressionNode {
	if len(nodes) <= count {
		return nodes
	}

	// Fisher-Yates shuffle
	shuffled := make([]*domain.ProgressionNode, len(nodes))
	copy(shuffled, nodes)

	for i := len(shuffled) - 1; i > 0; i-- {
		j := utils.SecureRandomInt(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled[:count]
}

func findWinningOption(options []domain.ProgressionVotingOption) *domain.ProgressionVotingOption {
	if len(options) == 0 {
		return nil
	}

	// Check if all options have 0 votes
	allZeroVotes := true
	for _, opt := range options {
		if opt.VoteCount > 0 {
			allZeroVotes = false
			break
		}
	}

	// If 0 votes total, pick random option
	if allZeroVotes {
		randomIndex := utils.SecureRandomInt(len(options))
		return &options[randomIndex]
	}

	// Normal tie-breaking with votes
	//nolint:gosec // Checked for empty slice above
	winner := &options[0]
	for i := 1; i < len(options); i++ {
		opt := &options[i]

		// Higher vote count wins
		if opt.VoteCount > winner.VoteCount {
			winner = opt
			continue
		}

		// Tie-breaker: first to reach highest vote (LastHighestVoteAt)
		if opt.VoteCount == winner.VoteCount {
			if opt.LastHighestVoteAt != nil && winner.LastHighestVoteAt != nil {
				if opt.LastHighestVoteAt.Before(*winner.LastHighestVoteAt) {
					winner = opt
				}
			} else if opt.LastHighestVoteAt != nil {
				winner = opt
			}
		}
	}

	return winner
}
