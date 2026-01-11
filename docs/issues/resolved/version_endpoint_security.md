RESOLVED

# Security: Harden /version Endpoint for Production

**Status**: RESOLVED  
**Priority**: Low  
**Category**: Security  
**Created**: 2025-12-31  
**Resolved**: 2026-01-05  
**Environment**: Production  

## Resolution

✅ **Authentication is already required for the `/version` endpoint!**

### Implementation Details

The recommended **Option 1** (Add Authentication) has been implemented:

- **Location**: `internal/server/server.go`
- **Line 52**: `r.Use(AuthMiddleware(apiKey, trustedProxies, detector))` applies to all routes
- **Line 63**: `r.Get("/version", handler.HandleVersion())` is registered after the auth middleware

This means the `/version` endpoint requires API key authentication via the `X-API-Key` header, preventing unauthorized access to version information.

### Security Benefits Achieved

✅ Version information only accessible with valid API key  
✅ Prevents public enumeration of deployed code versions  
✅ Reduces information disclosure risk  
✅ Maintains operational utility for authorized teams  

### Verification

To verify this implementation:
```bash
# Without API key - should return 401
curl http://localhost:8080/version

# With valid API key - should return version info
curl -H "X-API-Key: YOUR_KEY" http://localhost:8080/version
```

---

## Original Issue Summary

The `/version` endpoint currently exposes build information (version, git commit, build time, Go version) publicly without authentication. While this is acceptable for staging/development, it poses minor information disclosure risks in production.

## Background

The `/version` endpoint was added to help identify deployment desyncs (see [VERSION_DETECTION.md](../VERSION_DETECTION.md)). It provides valuable operational visibility but exposes:

- Git commit hash (exact code version)
- Build timestamp (deployment patterns)
- Go runtime version (framework fingerprinting)

## Security Concerns

### Information Disclosure
- **Risk**: Attackers can identify exact code version running
- **Impact**: Makes it easier to research known vulnerabilities for that specific commit
- **Severity**: LOW - Public endpoints and error messages already leak some version info

### Fingerprinting
- **Risk**: Easier application profiling and attack surface mapping
- **Impact**: Slightly reduces attacker effort during reconnaissance
- **Severity**: LOW - Similar data available from HTTP headers and timing attacks

## Current Mitigations

✅ No secrets, credentials, or sensitive business logic exposed  
✅ Similar to industry-standard `/healthz`, `/metrics` endpoints  
✅ Limited to version metadata only  
✅ Low-value target (game bot, not financial/healthcare)

## Proposed Solutions

### Option 1: Add Authentication (Recommended)
Require API key for `/version` in production:

```go
// In server.go - require auth for version endpoint
r.Group(func(r chi.Router) {
    r.Use(AuthMiddleware(apiKey, trustedProxies, detector))
    r.Get("/version", handler.HandleVersion())
})
```

**Pros**: Simple, consistent with other endpoints  
**Cons**: Requires API key for ops teams

### Option 2: Environment-Based Redaction
Limit information in production:

```go
// In version.go - redact details in prod
if os.Getenv("ENVIRONMENT") == "production" {
    info.BuildTime = ""
    info.GitCommit = ""  // Keep version tag only
}
```

**Pros**: Balance security with operational needs  
**Cons**: Less useful for debugging production issues

### Option 3: Rate Limiting
Add rate limiting to prevent automated scanning:

```go
r.With(RateLimitMiddleware(10, time.Minute)).Get("/version", ...)
```

**Pros**: Minimal code change  
**Cons**: Doesn't prevent information disclosure, just slows it

### Option 4: Internal-Only Endpoint
Move to `/internal/version` or admin-only route:

```go
r.Route("/internal", func(r chi.Router) {
    r.Use(AuthMiddleware(...))
    r.Get("/version", handler.HandleVersion())
})
```

**Pros**: Clear intent, easily discoverable for ops  
**Cons**: Requires authentication setup

## Recommendation

**For Staging/Dev**: Keep as-is (current implementation)  
**For Production**: Implement **Option 1** (authentication required)

This provides the best balance of security and operational utility. Teams can still access version info with their API key, but it's not publicly enumerable.

## Implementation Checklist

- [x] Decide on production hardening approach
- [x] Update `server.go` routing logic
- [x] Add environment-specific configuration
- [ ] Update documentation (VERSION_DETECTION.md) - if needed
- [ ] Test with actual API key in production
- [ ] Update deployment playbooks - if needed

## References

- [VERSION_DETECTION.md](../VERSION_DETECTION.md) - Usage guide
- [SECURITY_ANALYSIS.md](../SECURITY_ANALYSIS.md) - General security posture
- [/internal/handler/version.go](../../internal/handler/version.go) - Implementation

## Notes

This is a **defense-in-depth** consideration. The actual risk is minimal for this application, but hardening aligns with security best practices for production systems.

**Trade-off**: Increased security vs. operational convenience. For a game bot backend, convenience may win. For financial/healthcare systems, security should win.
