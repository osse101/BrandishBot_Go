package user

import (
	"context"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// pendingBomb tracks a bomb waiting for a peak and slowdown
type pendingBomb struct {
	SetterUsername   string
	Timeout          time.Duration
	AccumulatedUsers map[string]bool // userID -> true
}

// pulseRecentChatters runs every 2 seconds to manage bombs and recent chatters
func (s *service) pulseRecentChatters() {
	for range s.recentChatterTicker.C {
		s.processRecentChatters()
	}
}

// processRecentChatters handles the 2-second pulse logic
func (s *service) processRecentChatters() {
	s.recentChatterMu.Lock()
	defer s.recentChatterMu.Unlock()

	for platform, window := range s.recentChatterWindow {
		// 1. Process Bombs
		s.handleBombPulse(platform, window)

		// 2. Clear window for next 2s
		s.recentChatterWindow[platform] = make(map[string]bool)
	}
}

// handleBombPulse handles bomb-specific pulse logic
func (s *service) handleBombPulse(platform string, window map[string]bool) {
	queue := s.bombQueues[platform]
	if len(queue) == 0 {
		return
	}

	activeBomb := queue[0]

	if len(window) > 0 {
		// Chat is active, accumulate users
		for userID := range window {
			activeBomb.AccumulatedUsers[userID] = true
		}
	} else {
		// Chat slowed down (window is empty)
		if len(activeBomb.AccumulatedUsers) >= 5 {
			s.detonateBomb(platform)
		}
	}
}

// detonateBomb triggers a bomb explosion for a platform
func (s *service) detonateBomb(platform string) {
	// Assumes bombMu is already locked by processRecentChatters
	queue := s.bombQueues[platform]
	if len(queue) == 0 {
		return
	}

	activeBomb := queue[0]
	ctx := context.Background()
	log := logger.FromContext(ctx)

	log.Info("DETONATING BOMB", "platform", platform, "targets", len(activeBomb.AccumulatedUsers))

	hitUsernames := make([]string, 0, len(activeBomb.AccumulatedUsers))
	for userID := range activeBomb.AccumulatedUsers {
		// Resolve username (ideally we'd have it cached or stored)
		// For now, we'll try to find the user in our cache or DB
		user, err := s.repo.GetUserByID(ctx, userID)
		if err != nil {
			log.Warn("Failed to resolve user for bomb timeout", "userID", userID, "error", err)
			continue
		}

		// Apply timeout
		if err := s.AddTimeout(ctx, platform, user.Username, activeBomb.Timeout, "Caught in a Bomb burst!"); err != nil {
			log.Error("Failed to apply bomb timeout", "username", user.Username, "error", err)
		} else {
			hitUsernames = append(hitUsernames, user.Username)
		}

		// Remove from active chatter tracker
		s.activeChatterTracker.Remove(platform, userID)
	}

	// Publish detonation event
	if s.publisher != nil {
		s.publisher.PublishWithRetry(ctx, event.Event{
			Version: "1.1",
			Type:    domain.EventTypeBombDetonated,
			Payload: domain.BombDetonatedPayload{
				SetterUsername: activeBomb.SetterUsername,
				Platform:       platform,
				HitCount:       len(hitUsernames),
				Targets:        hitUsernames,
				TimeoutSeconds: int(activeBomb.Timeout.Seconds()),
				Timestamp:      time.Now().Unix(),
			},
		})
	}

	// Shift the queue
	s.bombQueues[platform] = s.bombQueues[platform][1:]

	log.Info("Bomb detonation complete", "platform", platform, "hit_count", len(hitUsernames))
}
