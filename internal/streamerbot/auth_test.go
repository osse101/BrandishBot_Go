package streamerbot

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

// expectedAuthHash is a helper to verify the output matches the expected algorithm directly
func expectedAuthHash(password, salt, challenge string) string {
	firstHash := sha256.Sum256([]byte(password + salt))
	combined := append(firstHash[:], []byte(challenge)...)
	secondHash := sha256.Sum256(combined)
	return base64.StdEncoding.EncodeToString(secondHash[:])
}

func TestGenerateAuthHash(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		salt      string
		challenge string
	}{
		{
			name:      "Best Case - Standard Input",
			password:  "mysecretpassword",
			salt:      "randomsalt123",
			challenge: "challenge456",
		},
		{
			name:      "Boundary Case - Empty Password",
			password:  "",
			salt:      "somesalt",
			challenge: "somechallenge",
		},
		{
			name:      "Boundary Case - Empty Salt",
			password:  "somepassword",
			salt:      "",
			challenge: "somechallenge",
		},
		{
			name:      "Boundary Case - Empty Challenge",
			password:  "somepassword",
			salt:      "somesalt",
			challenge: "",
		},
		{
			name:      "Edge Case - All Empty Strings",
			password:  "",
			salt:      "",
			challenge: "",
		},
		{
			name:      "Invalid/Hostile Case - Special Characters and Unicode",
			password:  "!@#$%^&*()_+{}|:<>?",
			salt:      "🤷‍♂️🤦‍♀️",
			challenge: "こんにちは",
		},
		{
			name:      "Edge Case - Long Strings",
			password:  "a_very_long_password_that_exceeds_normal_lengths_1234567890",
			salt:      "a_very_long_salt_that_exceeds_normal_lengths_0987654321",
			challenge: "a_very_long_challenge_that_exceeds_normal_lengths_abcdefg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateAuthHash(tt.password, tt.salt, tt.challenge)
			expected := expectedAuthHash(tt.password, tt.salt, tt.challenge)

			assert.Equal(t, expected, got, "GenerateAuthHash did not produce the expected base64 encoded hash")

			// Verify it's a valid base64 string
			_, err := base64.StdEncoding.DecodeString(got)
			assert.NoError(t, err, "GenerateAuthHash did not produce a valid base64 string")
		})
	}
}
