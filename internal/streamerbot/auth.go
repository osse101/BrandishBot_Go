package streamerbot

import (
	"crypto/sha256"
	"encoding/base64"
)

// GenerateAuthHash generates the authentication hash for Streamer.bot
// The algorithm is: Base64(SHA256(SHA256(password + salt) + challenge))
func GenerateAuthHash(password, salt, challenge string) string {
	// Step 1: SHA256(password + salt)
	firstHash := sha256.Sum256([]byte(password + salt))

	// Step 2: SHA256(firstHash + challenge)
	// Note: firstHash is bytes, challenge is string
	combined := append(firstHash[:], []byte(challenge)...)
	secondHash := sha256.Sum256(combined)

	// Step 3: Base64 encode the result
	return base64.StdEncoding.EncodeToString(secondHash[:])
}
