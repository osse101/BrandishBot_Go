package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// RegisterUser registers a new user
func (s *service) RegisterUser(ctx context.Context, user domain.User) (domain.User, error) {
	log := logger.FromContext(ctx)
	log.Info(domain.LogMsgRegisterUserCalled, "username", user.Username)
	if err := s.repo.UpsertUser(ctx, &user); err != nil {
		log.Error(domain.LogErrFailedToUpsertUser, "error", err, "username", user.Username)
		return domain.User{}, err
	}

	// Cache the newly registered user for all their platforms
	keys := getPlatformKeysFromUser(user)
	for platform, platformID := range keys {
		s.userCache.Set(platform, platformID, &user)
	}

	log.Info(domain.LogMsgUserRegistered, "user_id", user.ID, "username", user.Username)
	return user, nil
}

// UpdateUser updates an existing user
func (s *service) UpdateUser(ctx context.Context, user domain.User) error {
	log := logger.FromContext(ctx)
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		log.Error("Failed to update user", "error", err, "userID", user.ID)
		return err
	}

	// Invalidate cache for all platforms to force refresh on next lookup
	keys := getPlatformKeysFromUser(user)
	for platform, platformID := range keys {
		s.userCache.Invalidate(platform, platformID)
	}

	return nil
}

// FindUserByPlatformID finds a user by their platform-specific ID
func (s *service) FindUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error) {
	log := logger.FromContext(ctx)
	log.Info("FindUserByPlatformID called", "platform", platform, "platformID", platformID)
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		log.Error("Failed to find user by platform ID", "error", err, "platform", platform, "platformID", platformID)
		return nil, err
	}
	if user != nil {
		log.Info("User found", "userID", user.ID, "username", user.Username)
	}
	return user, nil
}

// GetUserIDByPlatformID resolves a user's internal UUID from their platform identity.
// It checks the in-memory userCache first; only falls back to the DB on a miss.
// Returns ("", nil) if the user is not yet registered (no auto-registration).
func (s *service) GetUserIDByPlatformID(ctx context.Context, platform, platformID string) (string, error) {
	if user, ok := s.userCache.Get(platform, platformID); ok {
		return user.ID, nil
	}

	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", nil
	}

	// Warm the cache for subsequent lookups
	s.userCache.Set(platform, platformID, user)
	return user.ID, nil
}

// HandleIncomingMessage checks if a user exists for an incoming message, creates one if not, and finds string matches.
func (s *service) HandleIncomingMessage(ctx context.Context, platform, platformID, username, message string) (*domain.MessageResult, error) {
	log := logger.FromContext(ctx)
	log.Debug("HandleIncomingMessage called", "platform", platform, "platformID", platformID, "username", username)

	user, err := s.getUserOrRegister(ctx, platform, platformID, username)
	if err != nil {
		log.Error("Failed to get user", "error", err, "platform", platform, "platformID", platformID)
		return nil, domain.ErrFailedToGetUser
	}

	// Update platform username if it has changed
	if user.PlatformUsernames == nil {
		user.PlatformUsernames = make(map[string]string)
	}
	if user.PlatformUsernames[platform] != username {
		log.Debug("Updating platform username", "platform", platform, "old", user.PlatformUsernames[platform], "new", username)
		setPlatformInfo(user, platform, platformID, username)
		if err := s.UpdateUser(ctx, *user); err != nil {
			log.Warn("Failed to update platform username", "error", err)
		}
	}

	// Track this user for bomb peak detection (2s window)
	s.recentChatterMu.Lock()
	if s.recentChatterWindow[platform] == nil {
		s.recentChatterWindow[platform] = make(map[string]bool)
	}
	s.recentChatterWindow[platform][user.ID] = true
	s.recentChatterMu.Unlock()

	// Track this user as an active chatter for random targeting
	s.activeChatterTracker.Track(platform, user.ID, username)

	// Find matches in message
	matches := s.stringFinder.FindMatches(message)

	// Check for active trap on this user and trigger if it exists
	if s.trapRepo != nil {
		userUUID, _ := uuid.Parse(user.ID)
		trap, err := s.trapRepo.GetActiveTrap(ctx, userUUID)
		if err != nil {
			log.Warn("Failed to check for trap", "user_id", user.ID, "error", err)
		} else if trap != nil {
			// Trigger trap asynchronously (don't block message processing)
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				asyncCtx := context.Background() // New context for async operation
				if err := s.triggerTrap(asyncCtx, trap, user); err != nil {
					log.Error(domain.LogMsgTrapTriggered, "trap_id", trap.ID, "error", err)
				}
			}()

			// Add trap trigger to matches so client can display it
			matches = append(matches, domain.FoundString{
				Code:  "trap_triggered",
				Value: fmt.Sprintf("%s stepped on a trap!", username),
			})
		}
	}

	result := &domain.MessageResult{
		User:    *user,
		Matches: matches,
	}

	return result, nil
}

// getUserOrRegister gets a user by platform ID, or auto-registers them if not found
func (s *service) getUserOrRegister(ctx context.Context, platform, platformID, username string) (*domain.User, error) {
	log := logger.FromContext(ctx)
	if username == "" || platform == "" || !validPlatforms[platform] {
		log.Error("Invalid platform or username", "platform", platform, "username", username)
		return nil, domain.ErrInvalidInput
	}

	// Try cache first
	if user, ok := s.userCache.Get(platform, platformID); ok {
		log.Debug("User cache hit", "userID", user.ID, "platform", platform)
		return user, nil
	}

	// Cache miss - fetch from database
	user, err := s.repo.GetUserByPlatformID(ctx, platform, platformID)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		log.Error("Failed to get user by platform ID", "error", err, "platform", platform, "platformID", platformID)
		return nil, domain.ErrFailedToGetUser
	}

	if user != nil {
		log.Debug("Found existing user", "userID", user.ID, "platform", platform)
		// Cache the user for future lookups
		s.userCache.Set(platform, platformID, user)
		return user, nil
	}

	// User not found, auto-register
	log.Info("Auto-registering new user", "platform", platform, "platformID", platformID, "username", username)
	newUser := domain.User{Username: username}
	setPlatformInfo(&newUser, platform, platformID, username)

	registered, err := s.RegisterUser(ctx, newUser)
	if err != nil {
		log.Error("Failed to auto-register user", "error", err)
		return nil, domain.ErrFailedToRegisterUser
	}

	log.Info("User auto-registered", "userID", registered.ID)
	return &registered, nil
}

// GetUserByPlatformUsername retrieves a user by platform and username
func (s *service) GetUserByPlatformUsername(ctx context.Context, platform, username string) (*domain.User, error) {
	return s.repo.GetUserByPlatformUsername(ctx, platform, username)
}
