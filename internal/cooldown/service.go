package cooldown

import (
	"context"
	"fmt"
	"time"
)

// ProgressionService defines the interface for progression system
type ProgressionService interface {
	GetModifiedValue(ctx context.Context, featureKey string, baseValue float64) (float64, error)
}

// Service manages action cooldowns for users
type Service interface {
	// CheckCooldown checks if a user's action is on cooldown
	// Returns: (onCooldown bool, remaining time.Duration, error)
	CheckCooldown(ctx context.Context, userID, action string) (bool, time.Duration, error)

	// EnforceCooldown atomically checks cooldown and executes action if allowed
	// This is the primary method - prevents race conditions
	EnforceCooldown(ctx context.Context, userID, action string, fn func() error) error

	// ResetCooldown manually resets a cooldown (admin/testing)
	ResetCooldown(ctx context.Context, userID, action string) error

	// GetLastUsed returns when action was last performed (for UI display)
	GetLastUsed(ctx context.Context, userID, action string) (*time.Time, error)
}

// ErrOnCooldown is returned when action is still on cooldown
type ErrOnCooldown struct {
	Action    string
	Remaining time.Duration
}

func (e ErrOnCooldown) Error() string {
	minutes := int(e.Remaining.Minutes())
	seconds := int(e.Remaining.Seconds()) % SecondsPerMinute

	if minutes > 0 {
		return fmt.Sprintf(ErrFmtCooldownWithMinutes, e.Action, minutes, seconds)
	}
	return fmt.Sprintf(ErrFmtCooldownSecondsOnly, e.Action, seconds)
}

// Is allows errors.Is() to work with ErrOnCooldown
func (e ErrOnCooldown) Is(target error) bool {
	_, ok := target.(ErrOnCooldown)
	return ok
}
