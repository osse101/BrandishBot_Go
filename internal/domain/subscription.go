package domain

import (
	"errors"
	"time"
)

// SubscriptionTier represents a subscription tier configuration
type SubscriptionTier struct {
	TierID      int       `json:"tier_id" db:"tier_id"`
	Platform    string    `json:"platform" db:"platform"`
	TierName    string    `json:"tier_name" db:"tier_name"`
	DisplayName string    `json:"display_name" db:"display_name"`
	TierLevel   int       `json:"tier_level" db:"tier_level"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Subscription represents a user's subscription status
type Subscription struct {
	UserID         string     `json:"user_id" db:"user_id"`
	Platform       string     `json:"platform" db:"platform"`
	TierID         int        `json:"tier_id" db:"tier_id"`
	Status         string     `json:"status" db:"status"`
	SubscribedAt   time.Time  `json:"subscribed_at" db:"subscribed_at"`
	ExpiresAt      time.Time  `json:"expires_at" db:"expires_at"`
	LastVerifiedAt *time.Time `json:"last_verified_at,omitempty" db:"last_verified_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// SubscriptionWithTier represents a subscription joined with tier details
type SubscriptionWithTier struct {
	Subscription
	TierName    string `json:"tier_name" db:"tier_name"`
	DisplayName string `json:"display_name" db:"display_name"`
	TierLevel   int    `json:"tier_level" db:"tier_level"`
}

// SubscriptionHistory represents an audit record of subscription lifecycle events
type SubscriptionHistory struct {
	HistoryID    int64                  `json:"history_id" db:"history_id"`
	UserID       string                 `json:"user_id" db:"user_id"`
	Platform     string                 `json:"platform" db:"platform"`
	TierID       int                    `json:"tier_id" db:"tier_id"`
	EventType    string                 `json:"event_type" db:"event_type"`
	SubscribedAt time.Time              `json:"subscribed_at" db:"subscribed_at"`
	ExpiresAt    time.Time              `json:"expires_at" db:"expires_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
}

// SubscriptionEvent represents an incoming webhook payload from Streamer.bot
type SubscriptionEvent struct {
	Platform       string `json:"platform" validate:"required,oneof=twitch youtube"`
	PlatformUserID string `json:"platform_user_id" validate:"required"`
	Username       string `json:"username" validate:"required"`
	TierName       string `json:"tier_name" validate:"required"`
	EventType      string `json:"event_type" validate:"required"` // Use HistoryEventType constants
	Timestamp      int64  `json:"timestamp" validate:"required"`
}

// SubscriptionVerificationRequest represents a request to verify subscription status
type SubscriptionVerificationRequest struct {
	Platform       string `json:"platform"`
	PlatformUserID string `json:"platform_user_id"`
}

// Errors
var (
	ErrSubscriptionNotFound      = errors.New("subscription not found")
	ErrInvalidSubscriptionTier   = errors.New("invalid subscription tier")
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists")
)
