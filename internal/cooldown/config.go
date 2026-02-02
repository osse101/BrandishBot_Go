package cooldown

import (
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Config holds cooldown service configuration
type Config struct {
	// DevMode bypasses all cooldowns when true
	DevMode bool

	// Cooldowns maps action names to their durations
	// If not specified, defaults from domain package are used
	Cooldowns map[string]time.Duration
}

// GetCooldownDuration returns the cooldown duration for an action
func (c *Config) GetCooldownDuration(action string) time.Duration {
	// Check custom overrides first
	if c.Cooldowns != nil {
		if duration, ok := c.Cooldowns[action]; ok {
			return duration
		}
	}

	// Fall back to defaults
	switch action {
	case domain.ActionSearch:
		return domain.SearchCooldownDuration
	default:
		// Unknown action - use default
		return DefaultCooldownDuration
	}
}
