package discord

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatFriendlyError(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Insufficient Funds",
			input:    "api error: insufficient funds",
			expected: MsgInsufficientFunds,
		},
		{
			name:     "Cooldown Simple",
			input:    "api error: action on cooldown",
			expected: MsgCooldownActive,
		},
		{
			name:     "Cooldown With Time",
			input:    "api error: action 'beg' on cooldown: 4m 3s remaining",
			expected: "Wait for: **4m 3s**",
		},
		{
			name:     "Generic Error",
			input:    "some random error",
			expected: "❌ some random error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatFriendlyError(tt.input)
			if tt.name == "Cooldown With Time" {
				assert.Contains(t, result, tt.expected)
				assert.Contains(t, result, MsgCooldownActive)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
