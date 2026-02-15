package progression

import (
	"context"
	"fmt"
	"sort"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/utils"
)

const (
	// MaxVotingOptions is the maximum number of options shown in a voting session
	MaxVotingOptions = 4

	// MaxRolloverPoints is the maximum number of contribution points that can rollover
	// to the next node after an unlock. This prevents a single massive contribution
	// from clearing multiple nodes in the tree at once.
	MaxRolloverPoints = 200
)

// StartVotingSession creates a new voting session with 4 random options from available nodes
// unlockedNodeID should be provided if this session is being started after a node unlock
func (s *service) StartVotingSession(ctx context.Context, unlockedNodeID *int) error {
	log := logger.FromContext(ctx)

	// 1. Check for existing active or frozen session
	existingSession, _ := s.repo.GetActiveOrFrozenSession(ctx)
	if existingSession != nil {
		log.Warn("Attempted to start voting while session already active", "sessionID", existingSession.ID, "status", existingSession.Status)
		return domain.ErrSessionAlreadyActive
	}

	// 2. Ensure we have an active unlock progress record
	progress, err := s.ensureActiveUnlockProgress(ctx)
	if err != nil {
		return err
	}

	// 3. Get available nodes and filter current target
	available, err := s.getFilteredAvailableNodes(ctx, progress)
	if err != nil {
		return err
	}

	// 4. Handle node count edge cases
	if len(available) == 0 {
		return fmt.Errorf("no nodes available for voting")
	}
	if len(available) == 1 {
		if progress.NodeID != nil {
			return s.startVotingWithOptions(ctx, available, nil)
		}
		return s.handleSingleOptionAutoSelect(ctx, progress, available[0])
	}

	// 5. Normal path: multiple options available
	return s.startVotingWithMultipleOptions(ctx, available, unlockedNodeID)
}

func (s *service) getFilteredAvailableNodes(ctx context.Context, progress *domain.UnlockProgress) ([]*domain.ProgressionNode, error) {
	available, err := s.GetAvailableUnlocksWithFutureTarget(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get available nodes: %w", err)
	}

	if progress != nil && progress.NodeID != nil {
		filtered := make([]*domain.ProgressionNode, 0, len(available))
		for _, n := range available {
			if n.ID != *progress.NodeID {
				filtered = append(filtered, n)
			}
		}
		return filtered, nil
	}
	return available, nil
}

func (s *service) startVotingWithMultipleOptions(ctx context.Context, available []*domain.ProgressionNode, unlockedNodeID *int) error {
	log := logger.FromContext(ctx)
	selected := selectRandomNodes(available, MaxVotingOptions)

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].NodeKey < selected[j].NodeKey
	})

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

	previousUnlock := ""
	if unlockedNodeID != nil {
		s.publishCycleCompletedEvent(ctx, *unlockedNodeID)
		if node, err := s.repo.GetNodeByID(ctx, *unlockedNodeID); err == nil && node != nil {
			previousUnlock = node.DisplayName
		}
	}

	s.publishVotingStartedEvent(ctx, sessionID, selected, previousUnlock)
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
		if err := s.bus.Publish(ctx, event.NewProgressionTargetEvent(node.NodeKey, targetLevel, true, sessionID)); err != nil {
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

func (s *service) publishCycleCompletedEvent(ctx context.Context, unlockedNodeID int) {
	log := logger.FromContext(ctx)
	unlockedNode, err := s.repo.GetNodeByID(ctx, unlockedNodeID)
	if err != nil {
		log.Warn("Failed to get unlocked node for event", "nodeID", unlockedNodeID, "error", err)
		return
	}

	if s.bus != nil {
		if err := s.bus.Publish(ctx, event.NewProgressionCycleEvent(unlockedNode.NodeKey, unlockedNode.DisplayName, unlockedNode.Description)); err != nil {
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
	optionsList := make([]event.ProgressionVotingOptionV1, 0, len(options))
	for _, node := range options {
		unlockDuration := FormatUnlockDuration(node.Size)

		optionsList = append(optionsList, event.ProgressionVotingOptionV1{
			NodeKey:        node.NodeKey,
			DisplayName:    node.DisplayName,
			Description:    node.Description,
			UnlockDuration: unlockDuration,
		})
	}

	if err := s.bus.Publish(ctx, event.NewProgressionVotingStartedEvent(optionsList, previousUnlock, sessionID)); err != nil {
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

	if session == nil || session.Status != domain.VotingStatusVoting {
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
		if err := s.repo.SetUnlockTarget(ctx, progress.ID, winner.NodeID, winner.TargetLevel, session.ID); err != nil {
			log.Warn("Failed to set unlock target", "error", err)
		} else if winner.NodeDetails != nil {
			s.mu.Lock()
			s.cachedTargetCost = winner.NodeDetails.UnlockCost
			s.cachedProgressID = progress.ID
			s.mu.Unlock()
		}
	}

	log.Info("Voting ended", "sessionID", session.ID, "winningNode", winner.NodeID, "votes", winner.VoteCount, "voterCount", len(voters))

	return winner, nil
}

// AddContribution adds contribution points to current unlock progress
// Uses cache-based estimation when far from threshold, atomic write+check when close (within 3-5 contributions)
func (s *service) AddContribution(ctx context.Context, amount int) error {
	progress, err := s.repo.GetActiveUnlockProgress(ctx)
	if err != nil {
		return fmt.Errorf("failed to get unlock progress: %w", err)
	}

	var progressID int
	var currentTotal int
	if progress == nil {
		progressID, err = s.repo.CreateUnlockProgress(ctx)
		if err != nil {
			return fmt.Errorf("failed to create unlock progress: %w", err)
		}
	} else {
		progressID = progress.ID
		currentTotal = progress.ContributionsAccumulated
	}

	amount = s.applyContributionBoosts(ctx, amount)

	useAtomic, cachedCost := s.determineContributionStrategy(progressID, currentTotal, amount)

	if err = s.repo.AddContribution(ctx, progressID, amount); err != nil {
		return err
	}

	if useAtomic {
		return s.handleAtomicContribution(ctx, amount, cachedCost)
	}

	s.handleCacheContribution(ctx, currentTotal, amount, cachedCost)
	return nil
}

func (s *service) applyContributionBoosts(ctx context.Context, amount int) int {
	isBoosted, _ := s.IsFeatureUnlocked(ctx, "upgrade_contribution_boost")
	if isBoosted {
		return (amount * 3) / 2
	}
	return amount
}

func (s *service) determineContributionStrategy(progressID int, currentTotal, amount int) (bool, int) {
	s.mu.RLock()
	cachedCost := s.cachedTargetCost
	cachedID := s.cachedProgressID
	s.mu.RUnlock()

	if cachedCost <= 0 || cachedID != progressID {
		return false, 0
	}

	estimatedTotal := currentTotal + amount
	remaining := cachedCost - estimatedTotal
	return remaining <= amount*5, cachedCost
}

func (s *service) handleAtomicContribution(ctx context.Context, amount, cachedCost int) error {
	log := logger.FromContext(ctx)
	updated, err := s.repo.GetActiveUnlockProgress(ctx)
	if err != nil {
		log.Warn("Failed to get updated progress", "error", err)
		return nil
	}

	actualTotal := updated.ContributionsAccumulated
	remaining := cachedCost - actualTotal
	if remaining < 0 {
		remaining = 0
	}

	log.Info("Contribution progress updated", "current", actualTotal, "required", cachedCost, "remaining", remaining, "added", amount, "strategy", "atomic")

	if actualTotal >= cachedCost {
		s.triggerUnlock(ctx, actualTotal, cachedCost)
	}
	return nil
}

func (s *service) triggerUnlock(ctx context.Context, actualTotal, cachedCost int) {
	log := logger.FromContext(ctx)
	log.Info("Unlock threshold met, triggering unlock", "accumulated", actualTotal, "required", cachedCost)

	select {
	case s.unlockSem <- struct{}{}:
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer func() { <-s.unlockSem }()
			if _, err := s.CheckAndUnlockNode(s.shutdownCtx); err != nil {
				log.Error("Failed to check and unlock node", "error", err)
			}
		}()
	default:
		log.Debug("Unlock already in progress, skipping duplicate trigger")
	}
}

func (s *service) handleCacheContribution(ctx context.Context, currentTotal, amount, cachedCost int) {
	log := logger.FromContext(ctx)
	estimatedTotal := currentTotal + amount
	if cachedCost > 0 {
		remaining := cachedCost - estimatedTotal
		if remaining < 0 {
			remaining = 0
		}
		log.Info("Contribution progress updated", "current", estimatedTotal, "required", cachedCost, "remaining", remaining, "added", amount, "strategy", "cache")
	} else {
		log.Info("Contribution added", "added", amount, "estimated_total", estimatedTotal, "target", "not_set")
	}
}

// CheckAndUnlockNode checks if unlock target is set and threshold is met
func (s *service) CheckAndUnlockNode(ctx context.Context) (*domain.ProgressionUnlock, error) {
	log := logger.FromContext(ctx)

	progress, err := s.repo.GetActiveUnlockProgress(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get unlock progress: %w", err)
	}

	if progress == nil {
		_, err = s.repo.CreateUnlockProgress(ctx)
		return nil, err
	}

	if progress.NodeID == nil {
		return nil, nil
	}

	node, err := s.repo.GetNodeByID(ctx, *progress.NodeID)
	if err != nil || node == nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	if progress.ContributionsAccumulated < node.UnlockCost {
		log.Debug("Waiting for contribution threshold", "current", progress.ContributionsAccumulated, "required", node.UnlockCost)
		return nil, nil
	}

	return s.performNodeUnlock(ctx, progress, node)
}

func (s *service) performNodeUnlock(ctx context.Context, progress *domain.UnlockProgress, node *domain.ProgressionNode) (*domain.ProgressionUnlock, error) {
	log := logger.FromContext(ctx)
	rollover := progress.ContributionsAccumulated - node.UnlockCost

	if err := s.repo.UnlockNode(ctx, *progress.NodeID, *progress.TargetLevel, "vote", progress.ContributionsAccumulated); err != nil {
		return nil, fmt.Errorf("failed to unlock node: %w", err)
	}

	s.cleanupVotingSessionOnUnlock(ctx, progress)

	// Cap rollover to prevent chain-unlocking the entire tree
	if rollover > MaxRolloverPoints {
		log.Info("Capping contribution rollover", "original", rollover, "capped", MaxRolloverPoints)
		rollover = MaxRolloverPoints
	}

	newProgressID, err := s.repo.CompleteUnlock(ctx, progress.ID, rollover)
	if err != nil {
		log.Warn("Failed to complete unlock progress", "error", err)
	}

	s.clearTargetCache()

	log.Info("Node unlocked", "nodeKey", node.NodeKey, "level", *progress.TargetLevel)

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

func (s *service) cleanupVotingSessionOnUnlock(ctx context.Context, progress *domain.UnlockProgress) {
	if progress.VotingSessionID == nil {
		return
	}

	session, err := s.repo.GetSessionByID(ctx, *progress.VotingSessionID)
	if err != nil || session == nil {
		return
	}

	for _, opt := range session.Options {
		if opt.NodeID == *progress.NodeID {
			optID := opt.ID
			if err := s.repo.EndVotingSession(ctx, *progress.VotingSessionID, &optID); err != nil {
				logger.FromContext(ctx).Warn("Failed to end voting session on unlock", "error", err)
			}
			return
		}
	}
}

func (s *service) clearTargetCache() {
	s.mu.Lock()
	s.cachedTargetCost = 0
	s.cachedProgressID = 0
	s.mu.Unlock()
}

// handlePostUnlockTransition handles the transition after a node unlocks
// It ends any active/frozen parallel voting session, sets winner as new target,
// and starts voting for the next cycle
// Note: rollover is already added by CompleteUnlock, passed here for logging only
func (s *service) handlePostUnlockTransition(ctx context.Context, unlockedNodeID int, newProgressID int, _ int) {
	log := logger.FromContext(ctx)

	// 1. Resolve parallel/frozen session to find next target
	newTargetNode, newTargetLevel := s.resolveNextTarget(ctx)

	// 2. Setup the new target
	if newTargetNode == nil {
		log.Info("All nodes unlocked after this unlock")
		s.publishAllUnlockedEvent(ctx)
		return
	}

	sessionID, err := s.setupNewTarget(ctx, newProgressID, newTargetNode, newTargetLevel)
	if err != nil {
		log.Error("Failed to setup new target", "error", err)
		return
	}

	// 3. Cleanup and notify
	s.publishCycleCompletedEvent(ctx, unlockedNodeID)

	// 4. Initialize voting for the cycle AFTER the next one
	s.initNextVotingCycle(ctx, newTargetNode)

	// End the placeholder session
	if err := s.repo.EndVotingSession(ctx, sessionID, nil); err != nil {
		log.Warn("Failed to end placeholder session", "error", err)
	}
}

func (s *service) resolveNextTarget(ctx context.Context) (*domain.ProgressionNode, int) {
	log := logger.FromContext(ctx)

	// Check parallel/frozen/recent session
	session, _ := s.repo.GetActiveOrFrozenSession(ctx)
	if session == nil {
		recent, err := s.repo.GetMostRecentSession(ctx)
		if err == nil && recent != nil && recent.Status == domain.VotingStatusCompleted {
			session = recent
		}
	}

	if session != nil {
		winner := s.resolveSessionWinner(ctx, session)
		if winner != nil && winner.NodeDetails != nil {
			log.Info("Found target from voting session", "sessionID", session.ID, "status", session.Status, "winnerNodeID", winner.NodeID)
			return winner.NodeDetails, winner.TargetLevel
		}
	}

	// Fallback to random if no winner found
	available, err := s.GetAvailableUnlocks(ctx)
	if err != nil || len(available) == 0 {
		return nil, 0
	}

	newTargetNode := available[utils.SecureRandomInt(len(available))]
	newTargetLevel := s.calculateNextTargetLevel(ctx, newTargetNode)
	log.Info("No active vote, picked random next target", "nodeKey", newTargetNode.NodeKey)
	return newTargetNode, newTargetLevel
}

func (s *service) resolveSessionWinner(ctx context.Context, session *domain.ProgressionVotingSession) *domain.ProgressionVotingOption {
	log := logger.FromContext(ctx)

	if session.Status == domain.VotingStatusCompleted && session.WinningOptionID != nil {
		for _, opt := range session.Options {
			if opt.ID == *session.WinningOptionID {
				return &opt
			}
		}
		return nil
	}

	// Active/Frozen: needs to be ended
	if session.Status == domain.VotingStatusFrozen {
		_ = s.repo.ResumeVotingSession(ctx, session.ID)
	}

	winner := findWinningOption(session.Options)
	if winner != nil {
		winnerID := winner.ID
		if err := s.repo.EndVotingSession(ctx, session.ID, &winnerID); err != nil {
			log.Warn("Failed to end parallel voting session", "sessionID", session.ID, "error", err)
		}
	}
	return winner
}

func (s *service) setupNewTarget(ctx context.Context, progressID int, node *domain.ProgressionNode, level int) (int, error) {
	// Create placeholder for FK
	sessionID, err := s.repo.CreateVotingSession(ctx)
	if err != nil {
		return 0, err
	}

	if err := s.repo.SetUnlockTarget(ctx, progressID, node.ID, level, sessionID); err != nil {
		return sessionID, err
	}

	// Cache new target
	s.mu.Lock()
	s.cachedTargetCost = node.UnlockCost
	s.cachedProgressID = progressID
	s.mu.Unlock()

	return sessionID, nil
}

func (s *service) initNextVotingCycle(ctx context.Context, currentTarget *domain.ProgressionNode) {
	log := logger.FromContext(ctx)
	available, err := s.GetAvailableUnlocks(ctx)
	if err != nil {
		return
	}

	remaining := make([]*domain.ProgressionNode, 0, len(available))
	for _, n := range available {
		if n.ID != currentTarget.ID {
			remaining = append(remaining, n)
		}
	}

	if len(remaining) >= 2 {
		_ = s.startVotingWithOptions(ctx, remaining, &currentTarget.ID)
	} else if len(remaining) == 1 {
		log.Info("Only one option remaining, no voting needed for next cycle")
		s.publishVotingStartedEvent(ctx, 0, remaining, currentTarget.DisplayName)
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

	if session.Status == domain.VotingStatusFrozen {
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

	// 1. Check for frozen/active session
	if session, _ := s.repo.GetActiveOrFrozenSession(ctx); session != nil {
		if session.Status == domain.VotingStatusFrozen {
			return s.resumeFrozenVotingSession(ctx, session.ID)
		}
		log.Info("Voting session already active", "sessionID", session.ID)
		return domain.ErrSessionAlreadyActive
	}

	// 2. No session - check available nodes
	available, err := s.GetAvailableUnlocks(ctx)
	if err != nil {
		return fmt.Errorf("failed to get available nodes: %w", err)
	}
	if len(available) == 0 {
		return domain.ErrNoNodesAvailable
	}

	// 3. Set initial target if none set
	progress, _ := s.repo.GetActiveUnlockProgress(ctx)
	if progress == nil || progress.NodeID == nil {
		if err := s.setInitialTarget(ctx, available); err != nil {
			return err
		}
		// Refresh available nodes after setting target (target is removed)
		available, _ = s.getFilteredAvailableNodes(ctx, progress)
	}

	// 4. Start voting if options remain
	if len(available) >= 1 {
		return s.StartVotingSession(ctx, nil)
	}

	log.Info("Only one node available, no voting needed")
	return nil
}

func (s *service) resumeFrozenVotingSession(ctx context.Context, sessionID int) error {
	if err := s.repo.ResumeVotingSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to resume voting session: %w", err)
	}
	logger.FromContext(ctx).Info("Admin resumed frozen voting session", "sessionID", sessionID)
	return nil
}

func (s *service) setInitialTarget(ctx context.Context, available []*domain.ProgressionNode) error {
	log := logger.FromContext(ctx)
	node := available[utils.SecureRandomInt(len(available))]

	progress, err := s.ensureActiveUnlockProgress(ctx)
	if err != nil {
		return err
	}

	targetLevel := s.calculateNextTargetLevel(ctx, node)
	sessionID, err := s.repo.CreateVotingSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to create voting session: %w", err)
	}

	if err = s.repo.AddVotingOption(ctx, sessionID, node.ID, targetLevel); err != nil {
		log.Warn("Failed to add voting option for initial target", "nodeID", node.ID, "error", err)
	}

	if err := s.repo.SetUnlockTarget(ctx, progress.ID, node.ID, targetLevel, sessionID); err != nil {
		return fmt.Errorf("failed to set unlock target: %w", err)
	}

	s.mu.Lock()
	s.cachedTargetCost = node.UnlockCost
	s.cachedProgressID = progress.ID
	s.mu.Unlock()

	log.Info("Admin set initial target", "nodeKey", node.NodeKey, "targetLevel", targetLevel)
	s.publishVotingStartedEvent(ctx, sessionID, []*domain.ProgressionNode{node}, "")
	s.publishTargetSetEvent(ctx, node, targetLevel, sessionID)

	if err := s.repo.EndVotingSession(ctx, sessionID, nil); err != nil {
		log.Warn("Failed to end placeholder session", "error", err)
	}
	return nil
}

func (s *service) publishTargetSetEvent(ctx context.Context, node *domain.ProgressionNode, level, sessionID int) {
	if s.bus != nil {
		if err := s.bus.Publish(ctx, event.NewProgressionTargetEvent(node.NodeKey, level, true, sessionID)); err != nil {
			logger.FromContext(ctx).Error("Failed to publish progression target set event", "error", err)
		}
	}
}

// startVotingWithOptions starts a voting session with specific options (used when we've already filtered)
func (s *service) startVotingWithOptions(ctx context.Context, options []*domain.ProgressionNode, unlockedNodeID *int) error {
	log := logger.FromContext(ctx)

	if len(options) == 0 {
		return fmt.Errorf("no options provided for voting")
	}

	selected := selectRandomNodes(options, MaxVotingOptions)

	// Enforce consistent ordering of options (sort by NodeKey)
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].NodeKey < selected[j].NodeKey
	})

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

	// 1. Check if target already exists
	if progress != nil && progress.NodeID != nil {
		s.populateTargetCache(ctx, progress)
		return nil
	}

	// 2. No target - check available nodes
	available, err := s.GetAvailableUnlocks(ctx)
	if err != nil {
		return fmt.Errorf("failed to get available nodes: %w", err)
	}

	if len(available) == 0 {
		log.Info("Progression state: all nodes unlocked")
		s.publishAllUnlockedEvent(ctx)
		return nil
	}

	// 3. Set initial target
	if err := s.setInitialTarget(ctx, available); err != nil {
		return err
	}

	// 4. Start voting if more nodes remain
	return s.startNextCycleVoting(ctx, progress)
}

func (s *service) populateTargetCache(ctx context.Context, progress *domain.UnlockProgress) {
	node, err := s.repo.GetNodeByID(ctx, *progress.NodeID)
	if err == nil && node != nil {
		s.mu.Lock()
		s.cachedTargetCost = node.UnlockCost
		s.cachedProgressID = progress.ID
		s.mu.Unlock()
		logger.FromContext(ctx).Info("Progression state: target already set", "nodeKey", node.NodeKey)
	}
}

func (s *service) startNextCycleVoting(ctx context.Context, progress *domain.UnlockProgress) error {
	filtered, _ := s.getFilteredAvailableNodes(ctx, progress)
	if len(filtered) >= 1 {
		if err := s.startVotingWithOptions(ctx, filtered, nil); err != nil {
			logger.FromContext(ctx).Warn("Failed to start parallel voting session", "error", err)
		}
	}
	return nil
}

// publishAllUnlockedEvent publishes an event when all nodes are unlocked
func (s *service) publishAllUnlockedEvent(ctx context.Context) {
	log := logger.FromContext(ctx)

	if s.bus != nil {
		if err := s.bus.Publish(ctx, event.NewProgressionAllUnlockedEvent("All progression nodes have been unlocked! Waiting for new content.")); err != nil {
			log.Error("Failed to publish all unlocked event", "error", err)
		} else {
			log.Info("Published progression all unlocked event")
		}
	}
}
