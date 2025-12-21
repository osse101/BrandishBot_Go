## 2025-12-14 - Timing Attacks on API Key Verification
**Vulnerability:** The `AuthMiddleware` in `internal/server/security.go` used standard string comparison (`!=`) for API key validation.
**Learning:** This allows attackers to potentially guess the API key by measuring response times (timing attack), as the comparison returns early upon the first mismatching character.
**Prevention:** Use `crypto/subtle.ConstantTimeCompare` for all secret comparisons. This ensures the comparison takes the same amount of time regardless of content.

## 2025-05-23 - Unbounded Security Event Tracking
**Vulnerability:** The `SuspiciousActivityDetector` tracked failed authentication attempts in an unbounded map without eviction or reset logic in `RecordFailedAuth`.
**Learning:** Security monitoring tools themselves can become a DoS vector if they allocate memory for every attacker IP without limits or cleanup.
**Prevention:** Always implement periodic reset or LRU eviction for security event counters to prevent memory exhaustion attacks.

## 2025-12-18 - Predictable Gambling Outcomes
**Vulnerability:** The gambling service (`internal/gamble/service.go`) used `math/rand` seeded with `time.Now()` to determine tie-break winners, making outcomes predictable.
**Learning:** `math/rand` is not cryptographically secure and its seed can often be guessed or manipulated, especially when based on system time.
**Prevention:** Use `crypto/rand` for all security-sensitive random number generation. Added `SecureRandomInt` helper to `internal/utils/math.go`.

## 2025-12-21 - IP Spoofing via Unvalidated X-Forwarded-For
**Vulnerability:** The application blindly trusted the `X-Forwarded-For` header in `extractIP` (`internal/server/security.go`), allowing attackers to spoof their IP address to bypass rate limiting and hide their identity.
**Learning:** `X-Forwarded-For` is user-controlled unless verified. Trusting it by default exposes the system to IP spoofing. Additionally, when behind a trusted proxy, the correct client IP is typically the *last* IP in the list (since proxies append), not the first.
**Prevention:** Only trust `X-Forwarded-For` if the request comes from a configured Trusted Proxy. When parsing, ensure robust address handling (e.g., `net.SplitHostPort` for IPv6) and understand the proxy's appending behavior (taking the last IP added by the trusted peer).
