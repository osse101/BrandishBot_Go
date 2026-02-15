package repository

import (
	"context"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Subscription defines the interface for subscription persistence
type Subscription interface {
	// Core CRUD operations
	GetUserSubscription(ctx context.Context, userID, platform string) (*domain.SubscriptionWithTier, error)
	GetUserSubscriptions(ctx context.Context, userID string) ([]domain.SubscriptionWithTier, error)
	CreateSubscription(ctx context.Context, sub domain.Subscription) error
	UpdateSubscription(ctx context.Context, sub domain.Subscription) error
	DeleteSubscription(ctx context.Context, userID, platform string) error

	// Expiration management
	GetExpiringSubscriptions(ctx context.Context, before time.Time) ([]domain.SubscriptionWithTier, error)
	MarkSubscriptionExpired(ctx context.Context, userID, platform string) error

	// Tier lookups
	GetTierByPlatformAndName(ctx context.Context, platform, tierName string) (*domain.SubscriptionTier, error)
	GetAllTiers(ctx context.Context) ([]domain.SubscriptionTier, error)

	// Audit trail
	RecordSubscriptionHistory(ctx context.Context, history domain.SubscriptionHistory) error
	GetUserSubscriptionHistory(ctx context.Context, userID string, limit int) ([]domain.SubscriptionHistory, error)
}
