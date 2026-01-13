package cooldown_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/cooldown"
)

// MockBackend is a simple mock for testing the Service interface
type MockBackend struct {
	checkCooldownFunc   func(ctx context.Context, userID, action string) (bool, time.Duration, error)
	enforceCooldownFunc func(ctx context.Context, userID, action string, fn func() error) error
	resetCooldownFunc   func(ctx context.Context, userID, action string) error
	getLastUsedFunc     func(ctx context.Context, userID, action string) (*time.Time, error)
}

func (m *MockBackend) CheckCooldown(ctx context.Context, userID, action string) (bool, time.Duration, error) {
	if m.checkCooldownFunc != nil {
		return m.checkCooldownFunc(ctx, userID, action)
	}
	return false, 0, nil
}

func (m *MockBackend) EnforceCooldown(ctx context.Context, userID, action string, fn func() error) error {
	if m.enforceCooldownFunc != nil {
		return m.enforceCooldownFunc(ctx, userID, action, fn)
	}
	return fn()
}

func (m *MockBackend) ResetCooldown(ctx context.Context, userID, action string) error {
	if m.resetCooldownFunc != nil {
		return m.resetCooldownFunc(ctx, userID, action)
	}
	return nil
}

func (m *MockBackend) GetLastUsed(ctx context.Context, userID, action string) (*time.Time, error) {
	if m.getLastUsedFunc != nil {
		return m.getLastUsedFunc(ctx, userID, action)
	}
	return nil, nil
}

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
