package user

import (
	"time"
)

const (
	// ChatterExpiryDuration is how long a user remains targetable after their last message
	ChatterExpiryDuration = 30 * time.Minute
	// CleanupInterval is how often we clean up expired chatters
	CleanupInterval = 5 * time.Minute
)

// ChatterInfo holds information about an active chatter
type ChatterInfo struct {
	UserID        string
	Username      string
	Platform      string
	LastMessageAt time.Time
}

// TargetInfo holds information about a selected target
type TargetInfo struct {
	Username string
	UserID   string
}
