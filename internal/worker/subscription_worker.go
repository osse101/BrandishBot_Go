package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/subscription"
)

// SubscriptionWorker periodically checks for expiring subscriptions and requests verification
type SubscriptionWorker struct {
	subscriptionSvc  subscription.Service
	subscriptionRepo repository.Subscription
	ticker           *time.Ticker
	shutdown         chan struct{}
	wg               sync.WaitGroup
	checkInterval    time.Duration
}

// NewSubscriptionWorker creates a new subscription worker
func NewSubscriptionWorker(
	subscriptionSvc subscription.Service,
	subscriptionRepo repository.Subscription,
	checkInterval time.Duration,
) *SubscriptionWorker {
	if checkInterval <= 0 {
		checkInterval = 6 * time.Hour // Default to 6 hours
	}

	return &SubscriptionWorker{
		subscriptionSvc:  subscriptionSvc,
		subscriptionRepo: subscriptionRepo,
		shutdown:         make(chan struct{}),
		checkInterval:    checkInterval,
	}
}

// Start starts the subscription worker
func (w *SubscriptionWorker) Start() {
	slog.Info("Starting subscription worker", "check_interval", w.checkInterval)

	w.ticker = time.NewTicker(w.checkInterval)

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		// Run check immediately on startup to catch any missed expirations
		w.checkExpiringSubscriptions()

		// Then run on ticker interval
		for {
			select {
			case <-w.ticker.C:
				w.checkExpiringSubscriptions()
			case <-w.shutdown:
				slog.Info("Subscription worker shutdown signal received")
				return
			}
		}
	}()
}

// checkExpiringSubscriptions checks for subscriptions that have ALREADY expired
// and verifies if they should be renewed or remain expired
func (w *SubscriptionWorker) checkExpiringSubscriptions() {
	ctx := context.Background()

	// Check for subscriptions that have already expired (not future expiry)
	now := time.Now()

	slog.Debug("Checking for expired subscriptions", "now", now)

	subscriptions, err := w.subscriptionRepo.GetExpiringSubscriptions(ctx, now)
	if err != nil {
		slog.Error("Failed to get expired subscriptions", "error", err)
		return
	}

	if len(subscriptions) == 0 {
		slog.Debug("No expired subscriptions found")
		return
	}

	slog.Info("Found expired subscriptions", "count", len(subscriptions))

	// Process each expired subscription
	for _, sub := range subscriptions {
		// Check if we're shutting down
		select {
		case <-w.shutdown:
			slog.Info("Subscription worker stopping verification checks")
			return
		default:
		}

		slog.Debug("Processing expired subscription",
			"user_id", sub.UserID,
			"platform", sub.Platform,
			"tier", sub.TierName,
			"expires_at", sub.ExpiresAt)

		// First, mark as expired
		if err := w.subscriptionRepo.MarkSubscriptionExpired(ctx, sub.UserID, sub.Platform); err != nil {
			slog.Error("Failed to mark subscription as expired",
				"user_id", sub.UserID,
				"platform", sub.Platform,
				"error", err)
			continue
		}

		// Then request verification to see if it should be renewed
		if err := w.subscriptionSvc.VerifyAndUpdateSubscription(ctx, sub.UserID, sub.Platform); err != nil {
			slog.Warn("Failed to verify expired subscription",
				"user_id", sub.UserID,
				"platform", sub.Platform,
				"error", err)
			continue
		}

		// Rate limiting - sleep between requests to avoid overwhelming Streamer.bot
		time.Sleep(100 * time.Millisecond)
	}

	slog.Info("Completed expired subscription verification", "count", len(subscriptions))
}

// Shutdown gracefully shuts down the worker
func (w *SubscriptionWorker) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down subscription worker")

	// Stop the ticker if it exists
	if w.ticker != nil {
		w.ticker.Stop()
	}

	// Signal shutdown
	close(w.shutdown)

	// Wait for worker goroutine to finish with timeout
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Subscription worker shutdown complete")
		return nil
	case <-ctx.Done():
		slog.Warn("Subscription worker shutdown timeout")
		return ctx.Err()
	}
}
