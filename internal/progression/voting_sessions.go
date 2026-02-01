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

	// Check for existing active or frozen session
	existingSession, _ := s.repo.GetActiveOrFrozenSession(ctx)
	if existingSession != nil {
		log.Warn("Attempted to start voting while session already active", "sessionID", existingSession.ID, "status", existingSession.Status)
		return domain.ErrSessionAlreadyActive
	}

	// Check if accumulation is already in progress (target already set)
	progress, err := s.repo.GetActiveUnlockProgress(ctx)
	if err != nil {
		return fmt.Errorf("failed to get unlock progress: %w", err)
	}

	// Note: Parallel voting is allowed. A voting session can be active
	// while a node is currently being unlocked (accumulation in progress).
	// When that node finishes, the winner of this session becomes the next target.
	if progress != nil && progress.NodeID != nil {
		log.Debug("Starting parallel voting session while accumulation in progress", "progressID", progress.ID, "nodeID", *progress.NodeID)
	}

	// Ensure we have an active unlock progress record
	progress, err = s.ensureActiveUnlockProgress(ctx)
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
		// Bug #1 Fix: Publish both cycle completed and voting started events
		// The voting started event needs the previous unlock name for SSE clients
		s.publishCycleCompletedEvent(ctx, *unlockedNodeID, sessionID)

		// Get the unlocked node name for the voting started event
		unlockedNode, err := s.repo.GetNodeByID(ctx, *unlockedNodeID)
		previousUnlock := ""
		if err == nil && unlockedNode != nil {
			previousUnlock = unlockedNode.DisplayName
		}
		s.publishVotingStartedEvent(ctx, sessionID, selected, previousUnlock)
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

	if s.bus != nil {
		if err := s.bus.Publish(ctx, event.Event{
			Version: "1.0",
			Type:    event.ProgressionCycleCompleted,
			Payload: map[string]interface{}{
				"unlocked_node": map[string]interface{}{
					"node_key":     unlockedNode.NodeKey,
					"display_name": unlockedNode.DisplayName,
					"description":  unlockedNode.Description,
				},
			},
		}); err != nil {
			log.Error("Failed to publish progression cycle completed event", "error", err)
		} else {
			log.Info("Published progression cycle completed event", "unlockedNode", unlockedNode.NodeKey)
		}
	}
}

func (s *service) publishVotingStartedEvent(ctx context.Context, sessionID int, options []*domain.ProgressionNode, previousUnlock string) {
	log := logger.FromContext(ctx)

	if s.bus == nil {
		return
	}

	// Build options list for event payload
	optionsList := make([]interface{}, 0, len(options))
	for _, node := range options {
		unlockDuration := FormatUnlockDuration(node.Size)

		optionsList = append(optionsList, map[string]interface{}{
			"node_key":        node.NodeKey,
			"display_name":    node.DisplayName,
			"description":     node.Description,
			"unlock_duration": unlockDuration,
		})
	}

	if err := s.bus.Publish(ctx, event.Event{
		Version: "1.0",
		Type:    event.ProgressionVotingStarted,
		Payload: map[string]interface{}{
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

	winnerID := winner.ID
	if err = s.repo.EndVotingSession(ctx, session.ID, &winnerID); err != nil {
		return nil, fmt.Errorf("failed to end voting session: %w", err)
	}

	progress, err := s.repo.GetActiveUnlockProgress(ctx)
	if err != nil {
		log.Warn("Failed to get active unlock progress", "error", err)
	}

	// Get voter IDs for logging
	voters, _ := s.repo.GetSessionVoters(ctx, session.ID)

	if progress != nil && progress.NodeID == nil {
		s.setUnlockTargetInternal(ctx, progress, winner, session.ID)
	}

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
	voters, err := s.repo.GetSessionVoters(ctx, sessionID)
	if err != nil {
		logger.FromContext(ctx).Warn("Failed to get session voters", "error", err)
		return nil
	}
	return voters
}

// AddContribution adds contribution points to current unlock progress
// Uses cache-based estimation when far from threshold, atomic write+check when close (within 3-5 contributions)
func (s *service) AddContribution(ctx context.Context, amount int) error {
	log := logger.FromContext(ctx)

	progress, err := s.repo.GetActiveUnlockProgress(ctx)
	if err != nil {
		return fmt.Errorf("failed to get unlock progress: %w", err)
	}

	var progressID int
	var currentTotal int
	if progress == nil {
		// Create new progress if none exists
		progressID, err = s.repo.CreateUnlockProgress(ctx)
		if err != nil {
			return fmt.Errorf("failed to create unlock progress: %w", err)
		}
		currentTotal = 0
	} else {
		progressID = progress.ID
		currentTotal = progress.ContributionsAccumulated
	}

	// Check for contribution boost upgrade
	isBoosted, _ := s.IsFeatureUnlocked(ctx, "upgrade_contribution_boost")
	if isBoosted {
		// Apply 1.5x multiplier (integer math: * 3 / 2)
		amount = (amount * 3) / 2
	}

	// Read cache to determine strategy
	s.mu.RLock()
	cachedCost := s.cachedTargetCost
	cachedID := s.cachedProgressID
	s.mu.RUnlock()

	// Calculate estimated total and remaining
	estimatedTotal := currentTotal + amount
	var remaining int
	var useAtomicCheck bool

	if cachedCost > 0 && cachedID == progressID {
		remaining = cachedCost - estimatedTotal
		if remaining < 0 {
			remaining = 0
		}
		// Within 3-5 contributions of threshold? Use atomic write+check
		// This prevents race conditions when multiple contributions could trigger unlock
		if remaining <= amount*5 {
			useAtomicCheck = true
		}
	}

	// Write contribution to DB
	err = s.repo.AddContribution(ctx, progressID, amount)
	if err != nil {
		return err
	}

	if useAtomicCheck {
		// Close to threshold - re-query for exact total to accurately detect unlock
		updatedProgress, err := s.repo.GetActiveUnlockProgress(ctx)
		if err != nil {
			log.Warn("Failed to get updated progress after contribution", "error", err)
			return nil
		}

		actualTotal := updatedProgress.ContributionsAccumulated
		remaining = cachedCost - actualTotal
		if remaining < 0 {
			remaining = 0
		}

		log.Info("Contribution progress updated",
			"current", actualTotal,
			"required", cachedCost,
			"remaining", remaining,
			"added", amount,
			"strategy", "atomic")

		// Threshold met - trigger unlock asynchronously
		if actualTotal >= cachedCost {
			log.Info("Unlock threshold met, triggering unlock",
				"accumulated", actualTotal,
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
	} else {
		// Far from threshold - use cache-based estimation (no extra query)
		if cachedCost > 0 && cachedID == progressID {
			log.Info("Contribution progress updated",
				"current", estimatedTotal,
				"required", cachedCost,
				"remaining", remaining,
				"added", amount,
				"strategy", "cache")
		} else {
			// No target set yet, just log the contribution
			log.Info("Contribution added",
				"added", amount,
				"estimated_total", estimatedTotal,
				"target", "not_set")
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

		// End the associated voting session if there is one (the one that selected THIS node)
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
					optID := winningOptionID
					if err := s.repo.EndVotingSession(ctx, *progress.VotingSessionID, &optID); err != nil {
						log.Warn("Failed to end voting session on unlock", "sessionID", *progress.VotingSessionID, "error", err)
					}
				}
			}
		}

		// Complete unlock and start next with rollover
		newProgressID, err := s.repo.CompleteUnlock(ctx, progress.ID, rollover)
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

		// NEW FLOW: Check for parallel voting session (active or frozen)
		// If found, end it and winner becomes new target
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handlePostUnlockTransition(s.shutdownCtx, node.ID, newProgressID, rollover)
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

// handlePostUnlockTransition handles the transition after a node unlocks
// It ends any active/frozen parallel voting session, sets winner as new target,
// and starts voting for the next cycle
// Note: rollover is already added by CompleteUnlock, passed here for logging only
func (s *service) handlePostUnlockTransition(ctx context.Context, unlockedNodeID int, newProgressID int, rollover int) {
	log := logger.FromContext(ctx)

	// Check for active or frozen parallel voting session
	session, _ := s.repo.GetActiveOrFrozenSession(ctx)

	// If no active session, check if there was a recently ended one that we can pick up
	if session == nil {
		recent, err := s.repo.GetMostRecentSession(ctx)
		if err == nil && recent != nil && recent.Status == SessionStatusCompleted && recent.WinningOptionID != nil {
			session = recent
		}
	}

	var newTargetNode *domain.ProgressionNode
	var newTargetLevel int

	if session != nil {
		var winner *domain.ProgressionVotingOption

		if session.Status == SessionStatusCompleted && session.WinningOptionID != nil {
			// Already completed, just find the winning option in the session data
			for _, opt := range session.Options {
				if opt.ID == *session.WinningOptionID {
					winner = &opt
					break
				}
			}
		} else {
			// Still active/frozen, resume if frozen then end it
			if session.Status == SessionStatusFrozen {
				if err := s.repo.ResumeVotingSession(ctx, session.ID); err != nil {
					log.Warn("Failed to resume frozen session before ending", "sessionID", session.ID, "error", err)
				}
			}

			// Determine winner and end session
			winner = findWinningOption(session.Options)
			if winner != nil {
				winnerID := winner.ID
				if err := s.repo.EndVotingSession(ctx, session.ID, &winnerID); err != nil {
					log.Warn("Failed to end parallel voting session", "sessionID", session.ID, "error", err)
				}
			}
		}

		if winner != nil {
			newTargetNode = winner.NodeDetails
			newTargetLevel = winner.TargetLevel
			log.Info("Found target from voting session", "sessionID", session.ID, "status", session.Status, "winnerNodeID", winner.NodeID)
		}
	}

	// If no winner from voting, pick random from available
	if newTargetNode == nil {
		available, err := s.GetAvailableUnlocks(ctx)
		if err != nil {
			log.Error("Failed to get available nodes for next target", "error", err)
			return
		}

		if len(available) == 0 {
			log.Info("All nodes unlocked after this unlock")
			s.publishAllUnlockedEvent(ctx)
			return
		}

		// Pick random
		newTargetNode = available[utils.SecureRandomInt(len(available))]
		newTargetLevel = s.calculateNextTargetLevel(ctx, newTargetNode)
		log.Info("No active vote, picked random next target", "nodeKey", newTargetNode.NodeKey)
	}

	// Create session placeholder for FK and set target
	sessionID, err := s.repo.CreateVotingSession(ctx)
	if err != nil {
		log.Error("Failed to create session for new target", "error", err)
		return
	}

	if err := s.repo.SetUnlockTarget(ctx, newProgressID, newTargetNode.ID, newTargetLevel, sessionID); err != nil {
		log.Error("Failed to set new unlock target", "error", err)
		return
	}

	// Note: rollover was already added by CompleteUnlock (InsertNextUnlockProgress)
	// Don't add it again here

	// Cache the new target
	s.mu.Lock()
	s.cachedTargetCost = newTargetNode.UnlockCost
	s.cachedProgressID = newProgressID
	s.mu.Unlock()

	// End the placeholder session (no winning option for placeholder)
	if err := s.repo.EndVotingSession(ctx, sessionID, nil); err != nil {
		log.Warn("Failed to end placeholder session", "error", err)
	}

	// Publish cycle completed event
	s.publishCycleCompletedEvent(ctx, unlockedNodeID, sessionID)

	// Get remaining available nodes (excluding new target)
	available, err := s.GetAvailableUnlocks(ctx)
	if err != nil {
		log.Warn("Failed to get available nodes for next voting", "error", err)
		return
	}

	remainingAvailable := make([]*domain.ProgressionNode, 0, len(available))
	for _, n := range available {
		if n.ID != newTargetNode.ID {
			remainingAvailable = append(remainingAvailable, n)
		}
	}

	// Start voting for next cycle if 2+ options remain
	if len(remainingAvailable) >= 2 {
		if err := s.startVotingWithOptions(ctx, remainingAvailable, &unlockedNodeID); err != nil {
			log.Warn("Failed to start next voting session", "error", err)
		}
	} else if len(remainingAvailable) == 1 {
		log.Info("Only one option remaining, no voting needed for next cycle")
		// Publish voting started with auto-selected info
		s.publishVotingStartedEvent(ctx, 0, remainingAvailable, newTargetNode.DisplayName)
	} else {
		log.Info("No more options available after new target")
	}
}

// GetUnlockProgress returns current unlock progress status
func (s *service) GetUnlockProgress(ctx context.Context) (*domain.UnlockProgress, error) {
	return s.repo.GetActiveUnlockProgress(ctx)
}

// AdminForceEndVoting allows admins to force end current voting
// Deprecated: Use AdminFreezeVoting for new flow
func (s *service) AdminForceEndVoting(ctx context.Context) (*domain.ProgressionVotingOption, error) {
	return s.EndVoting(ctx)
}

// AdminFreezeVoting freezes the current voting session (pauses until unlock completes)
func (s *service) AdminFreezeVoting(ctx context.Context) error {
	log := logger.FromContext(ctx)

	session, err := s.repo.GetActiveSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active session: %w", err)
	}

	if session == nil {
		return domain.ErrNoActiveSession
	}

	if session.Status == SessionStatusFrozen {
		log.Info("Voting session already frozen", "sessionID", session.ID)
		return domain.ErrSessionAlreadyFrozen
	}

	if err := s.repo.FreezeVotingSession(ctx, session.ID); err != nil {
		return fmt.Errorf("failed to freeze voting session: %w", err)
	}

	log.Info("Admin froze voting session", "sessionID", session.ID)
	return nil
}

// AdminStartVoting resumes a frozen vote OR starts a new voting session if nodes are available
func (s *service) AdminStartVoting(ctx context.Context) error {
	log := logger.FromContext(ctx)

	// Check for frozen session first
	session, _ := s.repo.GetActiveOrFrozenSession(ctx)
	if session != nil {
		if session.Status == SessionStatusFrozen {
			// Resume frozen vote
			if err := s.repo.ResumeVotingSession(ctx, session.ID); err != nil {
				return fmt.Errorf("failed to resume voting session: %w", err)
			}
			log.Info("Admin resumed frozen voting session", "sessionID", session.ID)
			return nil
		}
		// Already an active voting session
		log.Info("Voting session already active", "sessionID", session.ID)
		return domain.ErrSessionAlreadyActive
	}

	// No active/frozen session - check for available nodes
	available, err := s.GetAvailableUnlocks(ctx)
	if err != nil {
		return fmt.Errorf("failed to get available nodes: %w", err)
	}

	if len(available) == 0 {
		return domain.ErrNoNodesAvailable
	}

	// Check if we need to set an initial target
	progress, _ := s.repo.GetActiveUnlockProgress(ctx)
	if progress == nil || progress.NodeID == nil {
		// Pick random node as target
		node := available[utils.SecureRandomInt(len(available))]

		// Ensure we have an unlock progress record
		if progress == nil {
			progressID, err := s.repo.CreateUnlockProgress(ctx)
			if err != nil {
				return fmt.Errorf("failed to create unlock progress: %w", err)
			}
			progress = &domain.UnlockProgress{ID: progressID}
		}

		targetLevel := s.calculateNextTargetLevel(ctx, node)

		// Create a session to satisfy FK constraint
		sessionID, err := s.repo.CreateVotingSession(ctx)
		if err != nil {
			return fmt.Errorf("failed to create voting session: %w", err)
		}

		// Set the target
		if err := s.repo.SetUnlockTarget(ctx, progress.ID, node.ID, targetLevel, sessionID); err != nil {
			return fmt.Errorf("failed to set unlock target: %w", err)
		}

		// Cache the target cost
		s.mu.Lock()
		s.cachedTargetCost = node.UnlockCost
		s.cachedProgressID = progress.ID
		s.mu.Unlock()

		log.Info("Admin set initial target via AdminStartVoting", "nodeKey", node.NodeKey, "targetLevel", targetLevel)

		// Publish target set event
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

		// Start voting for next node if 2+ remain
		remainingAvailable := make([]*domain.ProgressionNode, 0, len(available)-1)
		for _, n := range available {
			if n.ID != node.ID {
				remainingAvailable = append(remainingAvailable, n)
			}
		}

		if len(remainingAvailable) >= 2 {
			// End the placeholder session and start a real voting session
			if err := s.repo.EndVotingSession(ctx, sessionID, nil); err != nil {
				log.Warn("Failed to end placeholder session", "error", err)
			}
			return s.startVotingWithOptions(ctx, remainingAvailable, nil)
		}

		return nil
	}

	// Target already set, just start voting if 2+ options
	if len(available) >= 2 {
		return s.StartVotingSession(ctx, nil)
	}

	log.Info("Only one node available, no voting needed")
	return nil
}

// startVotingWithOptions starts a voting session with specific options (used when we've already filtered)
func (s *service) startVotingWithOptions(ctx context.Context, options []*domain.ProgressionNode, unlockedNodeID *int) error {
	log := logger.FromContext(ctx)

	if len(options) == 0 {
		return fmt.Errorf("no options provided for voting")
	}

	selected := selectRandomNodes(options, MaxVotingOptions)

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

	log.Info("Started new voting session via startVotingWithOptions", "sessionID", sessionID, "options", len(selected))

	previousUnlock := ""
	if unlockedNodeID != nil {
		unlockedNode, err := s.repo.GetNodeByID(ctx, *unlockedNodeID)
		if err == nil && unlockedNode != nil {
			previousUnlock = unlockedNode.DisplayName
		}
	}
	s.publishVotingStartedEvent(ctx, sessionID, selected, previousUnlock)

	return nil
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

// InitializeProgressionState ensures the progression system is in a valid state on startup
// If no active target exists, it picks a random available node
// If 2+ nodes remain, it starts a voting session for the next target
func (s *service) InitializeProgressionState(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Info("Initializing progression state")

	progress, err := s.repo.GetActiveUnlockProgress(ctx)
	if err != nil {
		return fmt.Errorf("failed to get unlock progress: %w", err)
	}

	// If we have an active target, just ensure cache is populated
	if progress != nil && progress.NodeID != nil {
		node, err := s.repo.GetNodeByID(ctx, *progress.NodeID)
		if err == nil && node != nil {
			s.mu.Lock()
			s.cachedTargetCost = node.UnlockCost
			s.cachedProgressID = progress.ID
			s.mu.Unlock()
			log.Info("Progression state: target already set", "nodeKey", node.NodeKey, "progressID", progress.ID)
		}
		return nil
	}

	// No active target - check for available nodes
	available, err := s.GetAvailableUnlocks(ctx)
	if err != nil {
		return fmt.Errorf("failed to get available nodes: %w", err)
	}

	if len(available) == 0 {
		log.Info("Progression state: all nodes unlocked, waiting for new content")
		s.publishAllUnlockedEvent(ctx)
		return nil
	}

	// Pick random node as initial target
	node := available[utils.SecureRandomInt(len(available))]
	log.Info("Progression state: selecting random initial target", "nodeKey", node.NodeKey)

	// Ensure we have an unlock progress record
	var progressID int
	if progress == nil {
		progressID, err = s.repo.CreateUnlockProgress(ctx)
		if err != nil {
			return fmt.Errorf("failed to create unlock progress: %w", err)
		}
	} else {
		progressID = progress.ID
	}

	targetLevel := s.calculateNextTargetLevel(ctx, node)

	// Create a session to satisfy FK constraint
	sessionID, err := s.repo.CreateVotingSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to create voting session: %w", err)
	}

	// Set the target
	if err := s.repo.SetUnlockTarget(ctx, progressID, node.ID, targetLevel, sessionID); err != nil {
		return fmt.Errorf("failed to set unlock target: %w", err)
	}

	// Cache the target cost
	s.mu.Lock()
	s.cachedTargetCost = node.UnlockCost
	s.cachedProgressID = progressID
	s.mu.Unlock()

	// End the placeholder session (no winning option for placeholder)
	if err := s.repo.EndVotingSession(ctx, sessionID, nil); err != nil {
		log.Warn("Failed to end placeholder session", "error", err)
	}

	// Publish target set event
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

	// Start voting for next node if 2+ remain (excluding the one we just selected)
	remainingAvailable := make([]*domain.ProgressionNode, 0, len(available)-1)
	for _, n := range available {
		if n.ID != node.ID {
			remainingAvailable = append(remainingAvailable, n)
		}
	}

	if len(remainingAvailable) >= 2 {
		log.Info("Starting parallel voting session", "options", len(remainingAvailable))
		if err := s.startVotingWithOptions(ctx, remainingAvailable, nil); err != nil {
			log.Warn("Failed to start parallel voting session", "error", err)
			// Don't fail initialization if voting fails
		}
	} else if len(remainingAvailable) == 1 {
		log.Info("Only one remaining option after target selection, no voting needed")
	}

	log.Info("Progression state initialized", "targetNode", node.NodeKey, "targetLevel", targetLevel)
	return nil
}

// publishAllUnlockedEvent publishes an event when all nodes are unlocked
func (s *service) publishAllUnlockedEvent(ctx context.Context) {
	log := logger.FromContext(ctx)

	if s.bus == nil {
		return
	}

	if err := s.bus.Publish(ctx, event.Event{
		Version: "1.0",
		Type:    event.ProgressionAllUnlocked,
		Payload: map[string]interface{}{
			"message": "All progression nodes have been unlocked! Waiting for new content.",
		},
	}); err != nil {
		log.Error("Failed to publish all unlocked event", "error", err)
	} else {
		log.Info("Published progression all unlocked event")
	}
}
