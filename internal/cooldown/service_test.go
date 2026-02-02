package cooldown_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/cooldown"
)

// TestErrOnCooldown_Error tests the error message formatting
func TestErrOnCooldown_Error(t *testing.T) {
	tests := []struct {
		name      string
		err       cooldown.ErrOnCooldown
		wantRegex string
	}{
		{
			name:      "minutes and seconds",
			err:       cooldown.ErrOnCooldown{Action: "search", Remaining: 2*time.Minute + 30*time.Second},
			wantRegex: "action 'search' on cooldown: 2m 30s remaining",
		},
		{
			name:      "seconds only",
			err:       cooldown.ErrOnCooldown{Action: "attack", Remaining: 45 * time.Second},
			wantRegex: "action 'attack' on cooldown 45s remaining",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			assert.Contains(t, got, tt.err.Action)
			assert.Contains(t, got, "cooldown")
		})
	}
}

// TestErrOnCooldown_Is tests the errors.Is() compatibility
func TestErrOnCooldown_Is(t *testing.T) {
	err := cooldown.ErrOnCooldown{Action: "test", Remaining: time.Minute}

	// Should match another ErrOnCooldown
	assert.True(t, errors.Is(err, cooldown.ErrOnCooldown{}))

	// Should not match other errors
	assert.False(t, errors.Is(err, errors.New("other error")))
}
