package cooldown

import (
	"context"
	"fmt"
	"time"
)

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
	seconds := int(e.Remaining.Seconds()) % 60
	
	if minutes > 0 {
		return fmt.Sprintf("action '%s' on cooldown: %dm %ds remaining", e.Action, minutes, seconds)
	}
	return fmt.Sprintf("action '%s' on cooldown: %ds remaining", e.Action, seconds)
}

// Is allows errors.Is() to work with ErrOnCooldown
func (e ErrOnCooldown) Is(target error) bool {
	_, ok := target.(ErrOnCooldown)
	return ok
}
