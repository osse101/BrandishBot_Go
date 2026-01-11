# RESOLVED

# API Versioning Framework

**Priority:** MEDIUM  
**Complexity:** 6/10  
**Estimated Effort:** 4-5 hours  
**Created:** 2026-01-03

## Problem

The API currently has no versioning mechanism, which creates risks for:
- **Breaking changes:** No way to evolve API without breaking existing clients
- **Client compatibility:** Can't track which client versions are in use
- **Deprecation:** No strategy for sunsetting old endpoints
- **Future growth:** Need foundation for v2, v3, etc.

## Proposed Solution

Implement URL-based API versioning with immediate migration of all endpoints to `/api/v1/`.

**Strategy:**
- URL paths: `/api/v1/`, `/api/v2/`, etc.
- Latest version available at `/api/` (currently redirect to v1)
- Client version tracking via `X-Client-Version` header
- Version info endpoint at `/api/version`

**Migration:** All endpoints move to `/api/v1/` immediately (not gradual)

## Implementation

### 1. Versioning Infrastructure

Create [`internal/handler/versioning.go`](file:///home/osse1/projects/BrandishBot_Go/internal/handler/versioning.go):

```go
package handler

import (
    "net/http"
    "strings"
)

// APIVersion represents an API version
type APIVersion string

const (
    V1     APIVersion = "v1"
    V2     APIVersion = "v2"  // Future
    Latest APIVersion = V1     // Update as new versions added
)

// VersionedHandler wraps handlers with version routing
type VersionedHandler struct {
    handlers map[APIVersion]http.HandlerFunc
    latest   APIVersion
}

func NewVersionedHandler(latest APIVersion) *VersionedHandler {
    return &VersionedHandler{
        handlers: make(map[APIVersion]http.HandlerFunc),
        latest:   latest,
    }
}

func (vh *VersionedHandler) Register(version APIVersion, handler http.HandlerFunc) {
    vh.handlers[version] = handler
}

func (vh *VersionedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Extract version from URL
    version := extractVersion(r.URL.Path)
    
    // Use latest if no version specified
    if version == "" {
        version = string(vh.latest)
    }
    
    handler, ok := vh.handlers[APIVersion(version)]
    if !ok {
        http.Error(w, "Unsupported API version", http.StatusNotFound)
        return
    }
    
    // Add version to response header
    w.Header().Set("X-API-Version", version)
    handler(w, r)
}

func extractVersion(path string) string {
    parts := strings.Split(strings.TrimPrefix(path, "/api/"), "/")
    if len(parts) > 0 && strings.HasPrefix(parts[0], "v") {
        return parts[0]
    }
    return ""
}
```

### 2. Client Version Middleware

Add to [`internal/handler/middleware.go`](file:///home/osse1/projects/BrandishBot_Go/internal/handler/middleware.go):

```go
func ClientVersionMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        clientVersion := r.Header.Get("X-Client-Version")
        if clientVersion != "" {
            logger.FromContext(r.Context()).Debug("Client version",
                "version", clientVersion,
                "endpoint", r.URL.Path,
                "ip", r.RemoteAddr)
        }
        next.ServeHTTP(w, r)
    })
}
```

### 3. Version Info Endpoint

Update [`internal/handler/version.go`](file:///home/osse1/projects/BrandishBot_Go/internal/handler/version.go):

```go
// GET /api/version
func HandleGetAPIVersions(w http.ResponseWriter, r *http.Request) {
    response := map[string]interface{}{
        "latest":          "v1",
        "supported":       []string{"v1"},
        "deprecated":      []string{},
        "backend_version": version.Version,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

### 4. Router Setup

Update [`cmd/app/main.go`](file:///home/osse1/projects/BrandishBot_Go/cmd/app/main.go):

```go
// Apply client version middleware globally
mux := http.NewServeMux()
handler := handler.ClientVersionMiddleware(mux)

// Register version endpoint
mux.HandleFunc("/api/version", handler.HandleGetAPIVersions)

// Example versioned endpoint
inventoryV1 := handler.NewVersionedHandler(handler.V1)
inventoryV1.Register(handler.V1, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    handler.HandleGetInventory(w, r, userService, namingResolver)
}))

mux.Handle("/api/v1/inventory", inventoryV1)
mux.Handle("/api/inventory", inventoryV1)  // Latest (v1)
```

### 5. Update C# Client

Update `tools/BrandishBotClient.cs`:

```csharp
public class BrandishBotClient
{
    private const string ClientVersion = "1.0.0";  // Update with releases
    
    private HttpRequestMessage CreateRequest(string endpoint, HttpMethod method)
    {
        var request = new HttpRequestMessage(method, endpoint);
        request.Headers.Add("X-Client-Version", ClientVersion);
        return request;
    }
}
```

## Implementation Checklist

### Phase 1: Infrastructure
- [ ] Create `internal/handler/versioning.go`
- [ ] Create `internal/handler/versioning_test.go`
  - [ ] Test version extraction from URLs
  - [ ] Test version routing
  - [ ] Test fallback to latest
  - [ ] Test unsupported version returns 404
- [ ] Add `ClientVersionMiddleware` to `internal/handler/middleware.go`
- [ ] Update `HandleGetAPIVersions` in `internal/handler/version.go`

### Phase 2: Migration
- [ ] Migrate all endpoints to `/api/v1/` pattern:
  - [ ] `/api/v1/inventory`
  - [ ] `/api/v1/sell`
  - [ ] `/api/v1/buy`
  - [ ] `/api/v1/search`
  - [ ] `/api/v1/upgrade`
  - [ ] `/api/v1/disassemble`
  - [ ] `/api/v1/crafting/recipes`
  - [ ] `/api/v1/jobs/award-xp`
  - [ ] `/api/v1/progression/*`
  - [ ] `/api/v1/admin/*`
- [ ] Keep unversioned aliases pointing to v1
- [ ] Update route registration in `cmd/app/main.go`

### Phase 3: Client Updates
- [ ] Update C# client to send `X-Client-Version` header
- [ ] Update C# client to use `/api/v1/` endpoints
- [ ] Test C# client compatibility

### Phase 4: Documentation
- [ ] Create `docs/api_versioning.md`
  - [ ] Versioning strategy
  - [ ] How to add breaking changes (future v2)
  - [ ] Deprecation policy (6 month minimum)
  - [ ] Client migration guide
- [ ] Update API documentation with version info
- [ ] Add version examples to README

### Phase 5: Testing
- [ ] Test version routing:
  ```bash
  curl http://localhost:8080/api/v1/version  # v1 explicitly
  curl http://localhost:8080/api/version     # Latest (v1)
  curl http://localhost:8080/api/v99/version # 404 Not Found
  ```
- [ ] Test version header in responses:
  ```bash
  curl -I http://localhost:8080/api/inventory
  # Should include: X-API-Version: v1
  ```
- [ ] Integration test for client version tracking
- [ ] Test all endpoints work at both `/api/v1/*` and `/api/*`

## Affected Files

- [NEW] `internal/handler/versioning.go`
- [NEW] `internal/handler/versioning_test.go`
- [MODIFY] `internal/handler/middleware.go` (add ClientVersionMiddleware)
- [MODIFY] [`internal/handler/version.go`](file:///home/osse1/projects/BrandishBot_Go/internal/handler/version.go)
- [MODIFY] [`cmd/app/main.go`](file:///home/osse1/projects/BrandishBot_Go/cmd/app/main.go)
- [MODIFY] `tools/BrandishBotClient.cs`
- [NEW] `docs/api_versioning.md`

## URL Structure Examples

```
/api/v1/inventory      # Version 1
/api/v2/inventory      # Version 2 (future)
/api/inventory         # Latest (routes to v1)
/api/version           # Version info endpoint
```

## Success Criteria

- ✅ All endpoints support `/api/v1/` prefix
- ✅ Unversioned endpoints route to latest
- ✅ Version info endpoint available at `/api/version`
- ✅ Client version tracking implemented and logging
- ✅ `X-API-Version` header in all responses
- ✅ C# client updated with version header
- ✅ Documentation for API versioning complete
- ✅ All tests pass
- ✅ No breaking changes for existing clients (aliases maintained)

## Future Phases (Not in this issue)

**Phase 2: Breaking Changes (when needed)**
- Create `/api/v2/` handlers
- Deprecation notices in v1 responses
- Client migration guide

**Phase 3: Sunset Old Versions**
- Remove deprecated versions after migration period
- Minimum 6 months support window

## Monitoring Client Versions

Query logs for client version distribution:

```bash
# Get version counts
grep "Client version" app.log | jq -r '.version' | sort | uniq -c | sort -rn

# Example output:
#   150 v1.0.0
#    45 v0.9.5
#     5 v0.9.0
```

## Related Issues

- Implementation plan: [implementation_plan.md](file:///home/osse1/.gemini/antigravity/brain/db319d15-571c-413e-a190-ece6fbdbc1e5/implementation_plan.md)
