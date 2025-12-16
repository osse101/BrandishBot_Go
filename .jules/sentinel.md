## 2025-12-14 - Timing Attacks on API Key Verification
**Vulnerability:** The `AuthMiddleware` in `internal/server/security.go` used standard string comparison (`!=`) for API key validation.
**Learning:** This allows attackers to potentially guess the API key by measuring response times (timing attack), as the comparison returns early upon the first mismatching character.
**Prevention:** Use `crypto/subtle.ConstantTimeCompare` for all secret comparisons. This ensures the comparison takes the same amount of time regardless of content.

## 2025-05-23 - Unbounded Security Event Tracking
**Vulnerability:** The `SuspiciousActivityDetector` tracked failed authentication attempts in an unbounded map without eviction or reset logic in `RecordFailedAuth`.
**Learning:** Security monitoring tools themselves can become a DoS vector if they allocate memory for every attacker IP without limits or cleanup.
**Prevention:** Always implement periodic reset or LRU eviction for security event counters to prevent memory exhaustion attacks.
