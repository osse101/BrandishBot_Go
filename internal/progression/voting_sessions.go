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
	existingSession, _ := s.repo.GetActiveSession(ctx)
	if existingSession != nil {
		log.Warn("Attempted to start voting while session already active", "sessionID", existingSession.ID)
		return domain.ErrSessionAlreadyActive
	}

	progress, err := s.ensureActiveUnlockProgress(ctx)
	if err != nil {
		return err
	}

	available, err := s.GetAvailableUnlocks(ctx)
	if err != nil {
		return fmt.Errorf("failed to get available nodes: %w", err)
	}

	if len(available) == 0 {
		return fmt.Errorf("no nodes available for voting")
	}

	if len(available) == 1 {
		return s.handleSingleOptionAutoSelect(ctx, progress, available[0])
	}

	selected := selectRandomNodes(available, MaxVotingOptions)

	sessionID, err := s.repo.CreateVotingSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	for _, node := range selected {
		targetLevel := s.calculateNextTargetLevel(ctx, node)
		if err = s.repo.AddVotingOption(ctx, sessionID, node.ID, targetLevel); err != nil {
			log.Warn("Failed to add voting option", "nodeID", node.ID, "error", err)
		}
	}

	log.Info("Started new voting session", "sessionID", sessionID, "options", len(selected))

	if unlockedNodeID != nil {
		s.publishCycleCompletedEvent(ctx, *unlockedNodeID, sessionID)
	} else {
		// Publish voting started event for fresh voting sessions
		s.publishVotingStartedEvent(ctx, sessionID, selected, "")
	}

	return nil
}

func (s *service) ensureActiveUnlockProgress(ctx context.Context) (*domain.UnlockProgress, error) {
	progress, _ := s.repo.GetActiveUnlockProgress(ctx)
	if progress == nil {
		_, err := s.repo.CreateUnlockProgress(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create initial unlock progress: %w", err)
		}
		progress, err = s.repo.GetActiveUnlockProgress(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get active unlock progress after creation: %w", err)
		}
		logger.FromContext(ctx).Debug("Created initial unlock progress for new voting session")
	}
	return progress, nil
}

func (s *service) calculateNextTargetLevel(ctx context.Context, node *domain.ProgressionNode) int {
	targetLevel := 1
	isUnlocked, _ := s.repo.IsNodeUnlocked(ctx, node.NodeKey, 1)
	if isUnlocked {
		for level := 2; level <= node.MaxLevel; level++ {
			unlocked, _ := s.repo.IsNodeUnlocked(ctx, node.NodeKey, level)
			if !unlocked {
				targetLevel = level
				break
			}
		}
	}
	return targetLevel
}

func (s *service) handleSingleOptionAutoSelect(ctx context.Context, progress *domain.UnlockProgress, node *domain.ProgressionNode) error {
	log := logger.FromContext(ctx)
	log.Info("Only one option available, auto-selecting without voting", "nodeKey", node.NodeKey)

	targetLevel := s.calculateNextTargetLevel(ctx, node)

	// Create a voting session to satisfy FK constraint
	sessionID, err := s.repo.CreateVotingSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to create auto-select session: %w", err)
	}

	// Add the single option to the session
	if err = s.repo.AddVotingOption(ctx, sessionID, node.ID, targetLevel); err != nil {
		log.Warn("Failed to add voting option for auto-select", "nodeID", node.ID, "error", err)
	}

	// Set the unlock target with a valid session ID
	if err := s.repo.SetUnlockTarget(ctx, progress.ID, node.ID, targetLevel, sessionID); err != nil {
		return fmt.Errorf("failed to set unlock target: %w", err)
	}

	s.mu.Lock()
	s.cachedTargetCost = node.UnlockCost
	s.cachedProgressID = progress.ID
	s.mu.Unlock()

	if s.bus != nil {
		if err := s.bus.Publish(ctx, event.Event{
			Version: "1.0",
			Type:    event.ProgressionTargetSet,
			Payload: map[string]interface{}{
				"node_key":      node.NodeKey,
				"target_level":  targetLevel,
				"auto_selected": true,
				"session_id":    sessionID,
			},
		}); err != nil {
			log.Error("Failed to publish progression target set event", "error", err)
		}
	}

	log.Info("Auto-selected target set", "nodeKey", node.NodeKey, "targetLevel", targetLevel, "sessionID", sessionID)

	s.publishVotingStartedEvent(ctx, sessionID, []*domain.ProgressionNode{node}, "")

	if node.UnlockCost == 0 {
		log.Info("Zero-cost node, unlocking immediately", "nodeKey", node.NodeKey)
		// Use semaphore pattern to avoid concurrent unlock attempts
		select {
		case s.unlockSem <- struct{}{}:
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				defer func() { <-s.unlockSem }()
				if _, err := s.CheckAndUnlockNode(s.shutdownCtx); err != nil {
					log.Error("Failed to unlock zero-cost node", "error", err)
				}
				// CheckAndUnlockNode already starts next session via goroutine
			}()
		default:
			log.Debug("Unlock already in progress, skipping zero-cost auto-unlock")
		}
	}

	return nil
}

func (s *service) publishCycleCompletedEvent(ctx context.Context, unlockedNodeID int, sessionID int) {
	log := logger.FromContext(ctx)
	unlockedNode, err := s.repo.GetNodeByID(ctx, unlockedNodeID)
	if err != nil {
		log.Warn("Failed to get unlocked node for event", "nodeID", unlockedNodeID, "error", err)
		return
	}

	session, err := s.repo.GetActiveSession(ctx)
	if err != nil {
		log.Warn("Failed to get session for event", "error", err)
		return
	}

	if s.bus != nil {
		if err := s.bus.Publish(ctx, event.Event{
			Version: "1.0",
			Type:    event.ProgressionCycleCompleted,
			Payload: map[string]interface{}{
				"unlocked_node":  unlockedNode,
				"voting_session": session,
			},
		}); err != nil {
			log.Error("Failed to publish progression cycle completed event", "error", err)
		} else {
			log.Info("Published progression cycle completed event", "unlockedNode", unlockedNode.NodeKey, "sessionID", sessionID)
		}
	}
}

func (s *service) publishVotingStartedEvent(ctx context.Context, sessionID int, options []*domain.ProgressionNode, previousUnlock string) {
	log := logger.FromContext(ctx)

	if s.bus == nil {
		return
	}

	// Build options list for event payload
	optionsList := make([]map[string]interface{}, 0, len(options))
	for _, node := range options {
		optionsList = append(optionsList, map[string]interface{}{
			"node_key":     node.NodeKey,
			"display_name": node.DisplayName,
		})
	}

	if err := s.bus.Publish(ctx, event.Event{
		Version: "1.0",
		Type:    event.ProgressionVotingStarted,
		Payload: map[string]interface{}{
			"session_id":      sessionID,
			"options":         optionsList,
			"previous_unlock": previousUnlock,
		},
	}); err != nil {
		log.Error("Failed to publish voting started event", "error", err)
	} else {
		log.Info("Published voting started event", "sessionID", sessionID, "options", len(options))
	}
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

	winner := findWinningOption(session.Options)
	if winner == nil {
		return nil, fmt.Errorf("no voting options found")
	}

	if err = s.repo.EndVotingSession(ctx, session.ID, winner.ID); err != nil {
		return nil, fmt.Errorf("failed to end voting session: %w", err)
	}

	progress, err := s.ensureProgressForEndVoting(ctx)
	if err != nil {
		log.Warn("Failed to ensure unlock progress", "error", err)
	}

	if progress != nil {
		s.setUnlockTargetInternal(ctx, progress, winner, session.ID)
	}

	voters := s.awardVoterContributions(ctx, session.ID, progress)

	log.Info("Voting ended", "sessionID", session.ID, "winningNode", winner.NodeID, "votes", winner.VoteCount, "voterCount", len(voters))

	return winner, nil
}

func (s *service) ensureProgressForEndVoting(ctx context.Context) (*domain.UnlockProgress, error) {
	progress, err := s.repo.GetActiveUnlockProgress(ctx)
	if err != nil {
		return nil, err
	}
	if progress == nil {
		id, err := s.repo.CreateUnlockProgress(ctx)
		if err != nil {
			return nil, err
		}
		return &domain.UnlockProgress{ID: id}, nil
	}
	return progress, nil
}

func (s *service) setUnlockTargetInternal(ctx context.Context, progress *domain.UnlockProgress, winner *domain.ProgressionVotingOption, sessionID int) {
	if err := s.repo.SetUnlockTarget(ctx, progress.ID, winner.NodeID, winner.TargetLevel, sessionID); err != nil {
		logger.FromContext(ctx).Warn("Failed to set unlock target", "error", err)
	} else if winner.NodeDetails != nil {
		s.mu.Lock()
		s.cachedTargetCost = winner.NodeDetails.UnlockCost
		s.cachedProgressID = progress.ID
		s.mu.Unlock()
	}
}

func (s *service) awardVoterContributions(ctx context.Context, sessionID int, progress *domain.UnlockProgress) []string {
	log := logger.FromContext(ctx)
	voters, err := s.repo.GetSessionVoters(ctx, sessionID)
	if err != nil {
		log.Warn("Failed to get session voters", "error", err)
		return nil
	}

	for _, voterID := range voters {
		metric := &domain.EngagementMetric{
			UserID:      voterID,
			MetricType:  "vote_cast",
			MetricValue: 1,
			Metadata:    map[string]interface{}{"session_id": sessionID},
		}
		if err := s.repo.RecordEngagement(ctx, metric); err != nil {
			log.Warn("Failed to record contribution", "userID", voterID, "error", err)
		}
	}

	if progress != nil && len(voters) > 0 {
		if err := s.repo.AddContribution(ctx, progress.ID, len(voters)); err != nil {
			log.Warn("Failed to add contributions to progress", "error", err)
		}
	}
	return voters
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

		// End the associated voting session if there is one
		if progress.VotingSessionID != nil {
			session, err := s.repo.GetSessionByID(ctx, *progress.VotingSessionID)
			if err == nil && session != nil && len(session.Options) > 0 {
				// Find the option that matches our target node
				var winningOptionID int
				for _, opt := range session.Options {
					if opt.NodeID == *progress.NodeID {
						winningOptionID = opt.ID
						break
					}
				}
				if winningOptionID > 0 {
					if err := s.repo.EndVotingSession(ctx, *progress.VotingSessionID, winningOptionID); err != nil {
						log.Warn("Failed to end voting session on unlock", "sessionID", *progress.VotingSessionID, "error", err)
					}
				}
			}
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
	// Checked for empty slice above
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
