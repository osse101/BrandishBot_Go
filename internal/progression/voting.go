package progression

import (
	"context"
	"errors"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

func (s *service) VoteForUnlock(ctx context.Context, platform, platformID, username string, optionIndex int) error {
	log := logger.FromContext(ctx)

	// 1. Resolve or auto-register user
	user, err := s.resolveUserByPlatform(ctx, platform, platformID, username)
	if err != nil {
		return err
	}

	// 2. Validate session and option
	session, selectedOption, err := s.validateVotingSession(ctx, optionIndex)
	if err != nil {
		return err
	}

	// 3. Record vote atomically
	if err := s.repo.CheckAndRecordVoteAtomic(ctx, user.ID, session.ID, selectedOption.ID, selectedOption.NodeID); err != nil {
		return err
	}

	// 4. Record engagement
	if err := s.RecordEngagement(ctx, user.ID, "vote_cast", 1); err != nil {
		log.Warn("Failed to record vote engagement", "userID", user.ID, "error", err)
	}

	log.Info("Vote recorded", "userID", user.ID, "platform", platform, "platformID", platformID, "optionIndex", optionIndex, "nodeKey", selectedOption.NodeDetails.NodeKey, "sessionID", session.ID)
	return nil
}

func (s *service) resolveUserByPlatform(ctx context.Context, platform, platformID, username string) (*domain.User, error) {
	log := logger.FromContext(ctx)
	user, err := s.user.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("failed to resolve user: %w", err)
	}

	if user != nil {
		return user, nil
	}

	// Auto-registration
	if username == "" {
		return nil, fmt.Errorf("user not found and no username provided for auto-registration")
	}

	log.Info("Auto-registering new user from vote", "platform", platform, "platformID", platformID, "username", username)
	newUser := domain.User{Username: username}
	switch platform {
	case domain.PlatformTwitch:
		newUser.TwitchID = platformID
	case domain.PlatformYoutube:
		newUser.YoutubeID = platformID
	case domain.PlatformDiscord:
		newUser.DiscordID = platformID
	}

	if err := s.user.UpsertUser(ctx, &newUser); err != nil {
		return nil, fmt.Errorf("failed to auto-register user: %w", err)
	}

	user, err = s.user.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("failed to fetch newly registered user")
	}
	return user, nil
}

func (s *service) validateVotingSession(ctx context.Context, optionIndex int) (*domain.ProgressionVotingSession, *domain.ProgressionVotingOption, error) {
	session, err := s.repo.GetActiveSession(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get active session: %w", err)
	}

	if session == nil || session.Status != domain.VotingStatusVoting {
		return nil, nil, fmt.Errorf("no active voting session")
	}

	if optionIndex < 1 || optionIndex > len(session.Options) {
		return nil, nil, fmt.Errorf("invalid option index: %d (must be between 1 and %d)", optionIndex, len(session.Options))
	}

	return session, &session.Options[optionIndex-1], nil
}

// enrichSessionWithEstimates adds unlock time estimates to session options
func (s *service) enrichSessionWithEstimates(ctx context.Context, session *domain.ProgressionVotingSession) {
	if session == nil {
		return
	}
	for i := range session.Options {
		if session.Options[i].NodeDetails != nil {
			estimate, err := s.EstimateUnlockTime(ctx, session.Options[i].NodeDetails.NodeKey)
			if err == nil && estimate != nil {
				session.Options[i].EstimatedUnlockDate = estimate.EstimatedUnlockDate
			}
		}
	}
}

// GetActiveVotingSession returns the current voting session
func (s *service) GetActiveVotingSession(ctx context.Context) (*domain.ProgressionVotingSession, error) {
	session, err := s.repo.GetActiveSession(ctx)
	if err != nil {
		return nil, err
	}
	s.enrichSessionWithEstimates(ctx, session)
	return session, nil
}

func (s *service) GetMostRecentVotingSession(ctx context.Context) (*domain.ProgressionVotingSession, error) {
	session, err := s.repo.GetMostRecentSession(ctx)
	if err != nil {
		return nil, err
	}
	s.enrichSessionWithEstimates(ctx, session)
	return session, nil
}
