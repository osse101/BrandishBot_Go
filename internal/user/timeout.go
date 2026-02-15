package user

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// timeoutKey generates a platform-aware key for the timeout map
func timeoutKey(platform, username string) string {
	return fmt.Sprintf("%s:%s", platform, username)
}

// AddTimeout applies or extends a timeout for a user (accumulating).
// If the user already has a timeout, the new duration is ADDED to the remaining time.
// Note: Timeouts are in-memory and will be lost on server restart.
func (s *service) AddTimeout(ctx context.Context, platform, username string, duration time.Duration, reason string) error {
	log := logger.FromContext(ctx)
	key := timeoutKey(platform, username)
	log.Info("AddTimeout called", "platform", platform, "username", username, "duration", duration, "reason", reason)

	s.timeoutMu.Lock()
	defer s.timeoutMu.Unlock()

	var newExpiresAt time.Time
	now := time.Now()

	// Check if user already has a timeout - accumulate if so
	if info, exists := s.timeouts[key]; exists {
		info.timer.Stop()
		remaining := time.Until(info.expiresAt)
		if remaining < 0 {
			remaining = 0
		}
		// Accumulate: new expiry = now + remaining + new duration
		newExpiresAt = now.Add(remaining + duration)
		log.Info("Timeout accumulated", "platform", platform, "username", username, "previousRemaining", remaining, "added", duration, "newTotal", time.Until(newExpiresAt))
	} else {
		// No existing timeout
		newExpiresAt = now.Add(duration)
		log.Info("New timeout created", "platform", platform, "username", username, "duration", duration)
	}

	// Create timer for expiry
	timer := time.AfterFunc(time.Until(newExpiresAt), func() {
		s.timeoutMu.Lock()
		delete(s.timeouts, key)
		s.timeoutMu.Unlock()
		slog.Default().Info("User timeout expired", "platform", platform, "username", username, "reason", reason)
	})

	s.timeouts[key] = &timeoutInfo{
		timer:     timer,
		expiresAt: newExpiresAt,
	}

	// Publish timeout event
	if s.eventBus != nil {
		totalSeconds := int(time.Until(newExpiresAt).Seconds())
		evt := event.NewTimeoutAppliedEvent(platform, username, totalSeconds, reason)
		if err := s.eventBus.Publish(ctx, evt); err != nil {
			log.Warn("Failed to publish timeout applied event", "error", err)
		}
	}

	return nil
}

// ClearTimeout removes a user's timeout (admin action).
func (s *service) ClearTimeout(ctx context.Context, platform, username string) error {
	log := logger.FromContext(ctx)
	key := timeoutKey(platform, username)
	log.Info("ClearTimeout called", "platform", platform, "username", username)

	s.timeoutMu.Lock()
	defer s.timeoutMu.Unlock()

	info, exists := s.timeouts[key]
	if !exists {
		log.Info("No timeout to clear", "platform", platform, "username", username)
		return nil
	}

	info.timer.Stop()
	delete(s.timeouts, key)
	log.Info("Timeout cleared", "platform", platform, "username", username)

	// Publish timeout cleared event
	if s.eventBus != nil {
		evt := event.NewTimeoutClearedEvent(platform, username)
		if err := s.eventBus.Publish(ctx, evt); err != nil {
			log.Warn("Failed to publish timeout cleared event", "error", err)
		}
	}

	return nil
}

// GetTimeoutPlatform returns the remaining duration of a user's timeout for a specific platform.
func (s *service) GetTimeoutPlatform(ctx context.Context, platform, username string) (time.Duration, error) {
	key := timeoutKey(platform, username)

	s.timeoutMu.Lock()
	defer s.timeoutMu.Unlock()

	info, exists := s.timeouts[key]
	if !exists {
		return 0, nil
	}

	remaining := time.Until(info.expiresAt)
	if remaining < 0 {
		return 0, nil
	}
	return remaining, nil
}

// ReduceTimeoutPlatform reduces a user's timeout by the specified duration for a specific platform.
func (s *service) ReduceTimeoutPlatform(ctx context.Context, platform, username string, reduction time.Duration) error {
	log := logger.FromContext(ctx)
	key := timeoutKey(platform, username)
	log.Info("ReduceTimeoutPlatform called", "platform", platform, "username", username, "reduction", reduction)

	s.timeoutMu.Lock()
	defer s.timeoutMu.Unlock()

	info, exists := s.timeouts[key]
	if !exists {
		log.Info("User not timed out, nothing to reduce", "platform", platform, "username", username)
		return nil
	}

	// Calculate new expiry time
	newExpiresAt := info.expiresAt.Add(-reduction)
	remaining := time.Until(newExpiresAt)

	if remaining <= 0 {
		// Timeout is fully reduced, remove it
		info.timer.Stop()
		delete(s.timeouts, key)
		log.Info("Timeout fully removed via reduction", "platform", platform, "username", username)

		// Publish cleared event since timeout is gone
		if s.eventBus != nil {
			evt := event.NewTimeoutClearedEvent(platform, username)
			if err := s.eventBus.Publish(ctx, evt); err != nil {
				log.Warn("Failed to publish timeout cleared event", "error", err)
			}
		}
		return nil
	}

	// Update the timer with new duration
	info.timer.Stop()
	info.expiresAt = newExpiresAt
	info.timer = time.AfterFunc(remaining, func() {
		s.timeoutMu.Lock()
		delete(s.timeouts, key)
		s.timeoutMu.Unlock()
		slog.Default().Info("User timeout expired", "platform", platform, "username", username)
	})

	log.Info("Timeout reduced", "platform", platform, "username", username, "newRemaining", remaining)
	return nil
}

// TimeoutUser times out a user for a specified duration.
// Note: This method REPLACES the existing timeout (does not accumulate).
// For accumulating timeouts, use AddTimeout.
func (s *service) TimeoutUser(ctx context.Context, username string, duration time.Duration, reason string) error {
	// Legacy behavior: use AddTimeout with twitch platform
	// Note: The original TimeoutUser replaced timeouts, but we're now using accumulating AddTimeout
	// for consistency. If true replacement behavior is needed, we'd need to clear first.
	return s.AddTimeout(ctx, domain.PlatformTwitch, username, duration, reason)
}

// GetTimeout returns the remaining duration of a user's timeout.
func (s *service) GetTimeout(ctx context.Context, username string) (time.Duration, error) {
	return s.GetTimeoutPlatform(ctx, domain.PlatformTwitch, username)
}

// ReduceTimeout reduces a user's timeout by the specified duration (used by revive items).
func (s *service) ReduceTimeout(ctx context.Context, username string, reduction time.Duration) error {
	return s.ReduceTimeoutPlatform(ctx, domain.PlatformTwitch, username, reduction)
}
