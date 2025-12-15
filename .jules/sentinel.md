## 2025-12-14 - Timing Attacks on API Key Verification
**Vulnerability:** The `AuthMiddleware` in `internal/server/security.go` used standard string comparison (`!=`) for API key validation.
**Learning:** This allows attackers to potentially guess the API key by measuring response times (timing attack), as the comparison returns early upon the first mismatching character.
**Prevention:** Use `crypto/subtle.ConstantTimeCompare` for all secret comparisons. This ensures the comparison takes the same amount of time regardless of content.
