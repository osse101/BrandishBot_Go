package subscription

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// Service defines the interface for subscription operations
type Service interface {
	// Event handling (called by webhook handler)
	HandleSubscriptionEvent(ctx context.Context, evt domain.SubscriptionEvent) error

	// Query operations
	GetUserSubscription(ctx context.Context, userID, platform string) (*domain.SubscriptionWithTier, error)
	GetUserSubscriptions(ctx context.Context, userID string) ([]domain.SubscriptionWithTier, error)

	// Subscription status check (cached)
	IsSubscribed(ctx context.Context, userID, platform string) (bool, error)
	GetSubscriptionTier(ctx context.Context, userID, platform string) (string, int, error)

	// Verification (called by background worker)
	VerifyAndUpdateSubscription(ctx context.Context, userID, platform string) error

	// Lifecycle
	Shutdown(ctx context.Context) error
}

// StreamerbotClient defines the interface for Streamer.bot operations
type StreamerbotClient interface {
	DoAction(actionName string, args map[string]string) error
}

// UserRepository defines user-related operations needed by subscription service
type UserRepository interface {
	GetUserByPlatformID(ctx context.Context, platform, platformID string) (*domain.User, error)
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
}

type service struct {
	repo               repository.Subscription
	userRepo           UserRepository
	sbClient           StreamerbotClient
	resilientPublisher *event.ResilientPublisher
	cache              *StatusCache
	wg                 sync.WaitGroup
}

// NewService creates a new subscription service
func NewService(
	repo repository.Subscription,
	userRepo UserRepository,
	sbClient StreamerbotClient,
	resilientPublisher *event.ResilientPublisher,
) Service {
	return &service{
		repo:               repo,
		userRepo:           userRepo,
		sbClient:           sbClient,
		resilientPublisher: resilientPublisher,
		cache:              NewStatusCache(5 * time.Minute), // 5 minute cache TTL
	}
}

// HandleSubscriptionEvent processes an incoming subscription event from Streamer.bot
func (s *service) HandleSubscriptionEvent(ctx context.Context, evt domain.SubscriptionEvent) error {
	slog.Info("Handling subscription event",
		"platform", evt.Platform,
		"platform_user_id", evt.PlatformUserID,
		"username", evt.Username,
		"tier_name", evt.TierName,
		"event_type", evt.EventType,
	)

	// Get or validate user
	user, err := s.userRepo.GetUserByPlatformID(ctx, evt.Platform, evt.PlatformUserID)
	if err != nil {
		return fmt.Errorf("failed to get user by platform ID: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found for platform %s with ID %s", evt.Platform, evt.PlatformUserID)
	}

	// Lookup tier
	tier, err := s.repo.GetTierByPlatformAndName(ctx, evt.Platform, evt.TierName)
	if err != nil {
		return fmt.Errorf("failed to get tier: %w", err)
	}
	if tier == nil {
		return fmt.Errorf("%w: platform=%s tier=%s", domain.ErrInvalidSubscriptionTier, evt.Platform, evt.TierName)
	}

	// Determine status and expiration based on event type
	status := domain.SubscriptionStatusActive
	now := time.Now()
	expiresAt := now.Add(domain.DefaultSubscriptionDuration)

	if evt.EventType == domain.HistoryEventTypeCancelled {
		status = domain.SubscriptionStatusCancelled
		// Keep existing expiration or set to now
		expiresAt = now
	}

	// Check if subscription exists to determine event type for history/events
	existingSub, err := s.repo.GetUserSubscription(ctx, user.ID, evt.Platform)
	var historyEventType string
	var busEventType event.Type

	if err != nil || existingSub == nil {
		// New subscription
		historyEventType = domain.HistoryEventTypeSubscribed
		busEventType = event.Type(domain.EventTypeSubscriptionActivated)
	} else {
		// Existing subscription - determine type of change
		switch evt.EventType {
		case domain.HistoryEventTypeCancelled:
			historyEventType = domain.HistoryEventTypeCancelled
			busEventType = event.Type(domain.EventTypeSubscriptionCancelled)
		case domain.HistoryEventTypeRenewed:
			historyEventType = domain.HistoryEventTypeRenewed
			busEventType = event.Type(domain.EventTypeSubscriptionRenewed)
		case domain.HistoryEventTypeUpgraded:
			historyEventType = domain.HistoryEventTypeUpgraded
			busEventType = event.Type(domain.EventTypeSubscriptionUpgraded)
		case domain.HistoryEventTypeDowngraded:
			historyEventType = domain.HistoryEventTypeDowngraded
			busEventType = event.Type(domain.EventTypeSubscriptionDowngraded)
		default:
			historyEventType = domain.HistoryEventTypeRenewed
			busEventType = event.Type(domain.EventTypeSubscriptionRenewed)
		}
	}

	// Create/update subscription
	subscription := domain.Subscription{
		UserID:         user.ID,
		Platform:       evt.Platform,
		TierID:         tier.TierID,
		Status:         status,
		SubscribedAt:   now,
		ExpiresAt:      expiresAt,
		LastVerifiedAt: &now,
		UpdatedAt:      now,
	}

	if err := s.repo.UpdateSubscription(ctx, subscription); err != nil {
		return fmt.Errorf("failed to upsert subscription: %w", err)
	}

	slog.Info("Subscription updated",
		"user_id", user.ID,
		"platform", evt.Platform,
		"tier", tier.DisplayName,
		"status", status,
		"expires_at", expiresAt,
	)

	// Record history (non-blocking, log error but don't fail)
	history := domain.SubscriptionHistory{
		UserID:       user.ID,
		Platform:     evt.Platform,
		TierID:       tier.TierID,
		EventType:    historyEventType,
		SubscribedAt: now,
		ExpiresAt:    expiresAt,
		Metadata: map[string]interface{}{
			"source":           "webhook",
			"username":         evt.Username,
			"platform_user_id": evt.PlatformUserID,
		},
		CreatedAt: now,
	}

	if err := s.repo.RecordSubscriptionHistory(ctx, history); err != nil {
		slog.Warn("Failed to record subscription history", "error", err)
	}

	// Invalidate cache for this user's subscription
	s.cache.Invalidate(user.ID, evt.Platform)

	// Publish event to event bus (non-blocking)
	s.publishSubscriptionEvent(ctx, busEventType, user.ID, evt.Platform, tier.TierName, now)

	return nil
}

// GetUserSubscription retrieves a user's subscription for a specific platform
func (s *service) GetUserSubscription(ctx context.Context, userID, platform string) (*domain.SubscriptionWithTier, error) {
	return s.repo.GetUserSubscription(ctx, userID, platform)
}

// GetUserSubscriptions retrieves all subscriptions for a user
func (s *service) GetUserSubscriptions(ctx context.Context, userID string) ([]domain.SubscriptionWithTier, error) {
	return s.repo.GetUserSubscriptions(ctx, userID)
}

// IsSubscribed checks if a user has an active subscription (uses cache)
func (s *service) IsSubscribed(ctx context.Context, userID, platform string) (bool, error) {
	// Check cache first
	if cached, ok := s.cache.Get(userID, platform); ok {
		return cached.IsActive, nil
	}

	// Cache miss - fetch from database
	sub, err := s.repo.GetUserSubscription(ctx, userID, platform)
	if err != nil {
		// Not found is not an error - just means not subscribed
		return false, nil
	}

	// Update cache
	isActive := sub.Status == domain.SubscriptionStatusActive
	s.cache.Set(userID, platform, isActive, sub.TierName, sub.TierLevel)

	return isActive, nil
}

// GetSubscriptionTier returns the tier name and level if subscribed (uses cache)
// Returns empty string and 0 if not subscribed
func (s *service) GetSubscriptionTier(ctx context.Context, userID, platform string) (string, int, error) {
	// Check cache first
	if cached, ok := s.cache.Get(userID, platform); ok {
		if cached.IsActive {
			return cached.TierName, cached.TierLevel, nil
		}
		return "", 0, nil
	}

	// Cache miss - fetch from database
	sub, err := s.repo.GetUserSubscription(ctx, userID, platform)
	if err != nil {
		// Not found is not an error - just means not subscribed
		return "", 0, nil
	}

	// Update cache
	isActive := sub.Status == domain.SubscriptionStatusActive
	s.cache.Set(userID, platform, isActive, sub.TierName, sub.TierLevel)

	if isActive {
		return sub.TierName, sub.TierLevel, nil
	}
	return "", 0, nil
}

// VerifyAndUpdateSubscription requests verification from Streamer.bot
// This is async - Streamer.bot will call the webhook with the result
func (s *service) VerifyAndUpdateSubscription(ctx context.Context, userID, platform string) error {
	// Get user to retrieve platform ID
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Get platform-specific user ID
	var platformUserID string
	switch platform {
	case domain.PlatformTwitch:
		platformUserID = user.TwitchID
	case domain.PlatformYoutube:
		platformUserID = user.YoutubeID
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}

	if platformUserID == "" {
		return fmt.Errorf("user %s has no %s ID", userID, platform)
	}

	slog.Debug("Requesting subscription verification",
		"user_id", userID,
		"platform", platform,
		"platform_user_id", platformUserID,
	)

	// Call Streamer.bot DoAction to verify subscription
	// Streamer.bot will call our webhook with the result
	args := map[string]string{
		"platform":         platform,
		"platform_user_id": platformUserID,
	}

	if err := s.sbClient.DoAction("BrandishBot_VerifySubscription", args); err != nil {
		return fmt.Errorf("failed to request verification from Streamer.bot: %w", err)
	}

	slog.Debug("Subscription verification requested", "user_id", userID, "platform", platform)
	return nil
}

// publishSubscriptionEvent publishes a subscription event to the event bus
func (s *service) publishSubscriptionEvent(ctx context.Context, eventType event.Type, userID, platform, tierName string, timestamp time.Time) {
	payload := event.SubscriptionPayloadV1{
		UserID:    userID,
		Platform:  platform,
		TierName:  tierName,
		Timestamp: timestamp.Unix(),
	}

	evt := event.Event{
		Type:    eventType,
		Payload: payload,
	}

	s.resilientPublisher.PublishWithRetry(ctx, evt)
}

// Shutdown gracefully shuts down the service
func (s *service) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down subscription service")

	// Wait for any in-flight operations
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Subscription service shutdown complete")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("subscription service shutdown timeout: %w", ctx.Err())
	}
}
