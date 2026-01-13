package handler

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
)

// Test boundaries
const (
	MaxUsernameLength = 100
	MinQuantity       = 1
	MaxQuantity       = 10000
)

type TestStruct struct {
	Platform string `validate:"platform"`
	Username string `validate:"required,max=100,excludesall=\x00\n\r\t"`
	Quantity int    `validate:"min=1,max=10000"`
}

// =============================================================================
// Validator Tests - Demonstrating 5-Case Testing Model
// =============================================================================

func TestValidator_PlatformValidation(t *testing.T) {
	InitValidator()
	v := GetValidator()

	tests := []struct {
		name     string
		platform string
		wantErr  bool
	}{
		// CASE 1: Best Case
		{"valid twitch", domain.PlatformTwitch, false},
		{"valid youtube", "youtube", false},
		{"valid discord", domain.PlatformDiscord, false},

		// CASE 2: Boundary - empty allowed (not required)
		{"empty platform allowed", "", false},

		// CASE 3: Edge - case insensitive
		{"uppercase platform", "TWITCH", false},

		// CASE 4: Invalid Case
		{"invalid platform", "invalidplatform", true},
		{"typo", "twich", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestStruct{
				Platform: tt.platform,
				Username: "validuser",
				Quantity: 10,
			}

			err := v.ValidateStruct(input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_UsernameValidation(t *testing.T) {
	InitValidator()
	v := GetValidator()

	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		// CASE 1: Best Case
		{"valid username", "validuser", false},
		{"alphanumeric", "user123", false},
		{"with underscore", "user_name", false},

		// CASE 2: Boundary Case
		{"one char (just inside)", "a", false},
		{"exactly max length", strings.Repeat("a", MaxUsernameLength), false},
		{"over max length", strings.Repeat("a", MaxUsernameLength+1), true},

		// CASE 4: Invalid Case
		{"empty username", "", true},
		{"with newline", "user\nname", true},
		{"with tab", "user\tname", true},
		{"with null byte", "user\x00name", true},
		{"with carriage return", "user\rname", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestStruct{
				Platform: domain.PlatformTwitch,
				Username: tt.username,
				Quantity: 10,
			}

			err := v.ValidateStruct(input)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidator_QuantityValidation(t *testing.T) {
	InitValidator()
	v := GetValidator()

	tests := []struct {
		name     string
		quantity int
		wantErr  bool
	}{
		// CASE 1: Best Case
		{"valid quantity", 10, false},
		{"mid range", 5000, false},

		// CASE 2: Boundary Case
		{"negative (beyond lower)", -1, true},
		{"zero (on lower boundary)", 0, true},
		{"one (at min)", MinQuantity, false},
		{"max allowed", MaxQuantity, false},
		{"over max (beyond upper)", MaxQuantity + 1, true},

		// CASE 2: Worst Case - extremes
		{"very negative", -999999, true},
		{"very large", 999999, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestStruct{
				Platform: domain.PlatformTwitch,
				Username: "validuser",
				Quantity: tt.quantity,
			}

			err := v.ValidateStruct(input)

			if tt.wantErr {
				assert.Error(t, err, "Expected validation error for quantity=%d", tt.quantity)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_MultipleFieldErrors(t *testing.T) {
	InitValidator()
	v := GetValidator()

	t.Run("all fields invalid", func(t *testing.T) {
		input := TestStruct{
			Platform: "invalid",
			Username: "", // Required field
			Quantity: 0,  // Below minimum
		}

		err := v.ValidateStruct(input)

		require.Error(t, err)
		// Should have errors for all three fields
		assert.Contains(t, err.Error(), "Platform")
		assert.Contains(t, err.Error(), "Username")
		assert.Contains(t, err.Error(), "Quantity")
	})
}
