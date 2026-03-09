package activechatter

import (
	"time"
)

const (
	// ExpiryDuration is how long a user remains targetable after their last message
	ExpiryDuration = 30 * time.Minute
	// CleanupInterval is how often we clean up expired chatters
	CleanupInterval = 5 * time.Minute
)

// Chatter holds information about an active chatter
type Chatter struct {
	UserID        string    `json:"user_id"`
	Username      string    `json:"username"`
	Platform      string    `json:"platform"`
	LastMessageAt time.Time `json:"last_message_at"`
}

// TargetInfo holds information about a selected target
type TargetInfo struct {
	Username string
	UserID   string
}
