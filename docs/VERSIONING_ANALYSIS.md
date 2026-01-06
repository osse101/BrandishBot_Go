# I/O Versioning Analysis

**Date**: 2026-01-05  
**Purpose**: Identify I/O points where versioning would improve maintainability, compatibility, and debugging.

## Executive Summary

| I/O Point | Current State | Versioning Recommended | Priority |
| ----------- | --------------- | ------------------------ | ---------- |
| API Endpoints | ✅ Versioned (`/api/v1/`) | Maintain current approach | ✅ Done |
| Config Files (JSON) | ✅ Versioned (`progression_tree.json` has `"version": "2.0"`) | Add to other configs | 🟡 Medium |
| Database Migrations | ✅ Versioned (goose migrations) | Maintain current approach | ✅ Done |
| Event Schemas | ❌ No versioning | **Add schema versioning** | 🔴 High |
| Log Format | ❌ No versioning | Add format version field | 🟢 Low |
| Environment Config (.env) | ❌ No versioning | Add schema version checking | 🟡 Medium |
| Cache Data | ❌ No versioning | Add version to cached objects | 🟡 Medium |
| Dead-Letter Logs | ❌ No versioning | Add schema version | 🟢 Low |

---

## 1. API Endpoints ✅

### Current State

- **Location**: `/api/v1/*`
- **Status**: ✅ Already versioned
- **Implementation**: URL-based versioning in [server.go:L69](file:///home/osse1/projects/BrandishBot_Go/internal/server/server.go#L69)

```go
r.Route("/api/v1", func(r chi.Router) {
    // All API endpoints versioned
})
```

### Recommendation

✅ **No changes needed** - Continue using URL-based versioning for all endpoints.

### Future Considerations

- When breaking changes are needed, create `/api/v2/` routes
- Consider client version tracking via headers (see `docs/issues/api-versioning-framework.md`)

---

## 2. Configuration Files (JSON)

### Current State

#### ✅ `progression_tree.json` - Already Versioned

```json
{
  "version": "2.0",
  "description": "BrandishBot Progression Tree",
  "nodes": [...]
}
```

#### ❌ Other Config Files - Not Versioned

- `configs/items/aliases.json` - No version field
- `configs/items/themes.json` - No version field  
- `configs/loot_tables.json` - No version field

### Problems Without Versioning

1. **Silent Failures**: Old config format loaded into new parser → errors
2. **No Migration Path**: Can't detect when config needs updating
3. **Debugging Difficulty**: Can't tell which config format is deployed

### Recommendation

🟡 **Add version field to all JSON config files**

#### Implementation

```json
{
  "version": "1.0",
  "schema": "aliases",
  "last_updated": "2026-01-05",
  "data": {
    // actual config content
  }
}
```

#### Migration Strategy

1. Add version parser to config loaders
2. Support backward compatibility for v1.0
3. Log warnings for outdated configs
4. Provide migration tools for major version bumps

#### Files to Update

- [ ] `configs/items/aliases.json`
- [ ] `configs/items/themes.json`
- [ ] `configs/loot_tables.json`
- [ ] Update parsers in `internal/naming/` and related packages

---

## 3. Database Migrations ✅

### Current State

- **Tool**: `goose` migrations
- **Status**: ✅ Already versioned
- **Location**: `migrations/`

### Recommendation

✅ **No changes needed** - Migrations are inherently versioned by goose.

---

## 4. Event Schemas 🔴

### Current State

```go
// internal/event/event.go
type Event struct {
    Type     Type
    Payload  interface{}  // Untyped!
    Metadata map[string]interface{}
}
```

### Problems

1. **No Schema Validation**: `interface{}` payload = runtime errors
2. **No Version Tracking**: Can't evolve event schemas safely
3. **Breaking Changes Invisible**: Consumers break silently

### Recommendation

🔴 **Add event schema versioning (HIGH PRIORITY)**

#### Proposed Implementation

```go
// Versioned event structure
type Event struct {
    Version  string                 // e.g., "1.0", "2.0"
    Type     Type
    Payload  interface{}
    Metadata map[string]interface{}
}

// Typed event payloads
type EngagementEventV1 struct {
    UserID       int64  `json:"user_id"`
    PlatformID   int64  `json:"platform_id"`
    ActivityType string `json:"activity_type"`
    Timestamp    int64  `json:"timestamp"`
}

type ProgressionUnlockEventV1 struct {
    NodeKey   string `json:"node_key"`
    UserCount int    `json:"user_count"`
    Timestamp int64  `json:"timestamp"`
}
```

#### Benefits

- Type-safe event handling
- Clear schema evolution path
- Backward compatibility support
- Better debugging and logging

#### Migration Path

1. Add `Version` field to `Event` struct
2. Default to `"1.0"` for existing events
3. Create typed payload structs for each event type
4. Update event publishers to use typed constructors
5. Add schema validation middleware

#### Related Issue

See `docs/issues/CODE_REVIEW_ISSUES.md` - "document-event-system.md" (item #3)

---

## 5. Log Format

### Current State

- **Format**: slog JSON/Text (configurable via `LOG_FORMAT`)
- **No explicit version field**

```json
{
  "time": "2026-01-05T19:26:04Z",
  "level": "INFO",
  "msg": "Request started",
  "request_id": "abc123",
  // No version field
}
```

### Problems

1. **Parser Compatibility**: Log aggregators can't detect format changes
2. **Field Changes Break Dashboards**: Adding/removing fields breaks queries
3. **No Migration Tooling**: Can't update old logs to new format

### Recommendation

🟢 **Add log format version (LOW PRIORITY)**

#### Implementation

```go
// internal/logger/config.go
func (c Config) BaseAttributes() []slog.Attr {
    return []slog.Attr{
        slog.String("service", c.ServiceName),
        slog.String("environment", c.Environment),
        slog.String("log_version", "1.0"),  // Add this
    }
}
```

#### Benefits

- Log aggregators can filter by version
- Easier to track format changes over time
- Helps with log retention policies

---

## 6. Environment Configuration (.env)

### Current State

- **File**: `.env` (not tracked), `.env.example` (tracked)
- **No schema versioning**
- 68 lines, multiple configuration domains

### Problems

1. **Silent Deployment Failures**: Missing env vars → runtime errors
2. **No Validation**: Typos in variable names not caught until runtime
3. **Breaking Changes Invisible**: New required vars break old deployments

### Recommendation

🟡 **Add .env schema version checking (MEDIUM PRIORITY)**

#### Implementation

Add to `.env.example` and `.env`:

```bash
# Environment Schema Version
# Update this when adding/removing required variables
ENV_SCHEMA_VERSION=2.0

# Last Updated: 2026-01-05
# Breaking Changes: Added EVENT_MAX_RETRIES (required)
```

Add validation at startup:

```go
// cmd/app/main.go or internal/config/validator.go
func ValidateEnvSchema() error {
    schemaVersion := os.Getenv("ENV_SCHEMA_VERSION")
    if schemaVersion == "" {
        return errors.New("ENV_SCHEMA_VERSION not set - please update .env")
    }
    
    required := GetRequiredVarsForVersion(schemaVersion)
    for _, varName := range required {
        if os.Getenv(varName) == "" {
            return fmt.Errorf("required env var %s missing", varName)
        }
    }
    
    return nil
}
```

#### Benefits

- Early detection of configuration issues
- Clear upgrade paths for deployments
- Self-documenting configuration changes

#### Files to Update

- [ ] `.env.example` - Add `ENV_SCHEMA_VERSION`
- [ ] Create `internal/config/validator.go`
- [ ] Update `cmd/app/main.go` to validate on startup
- [ ] Add version to deployment documentation

---

## 7. Cache Data

### Current State

- User cache in `internal/user/cache.go`
- Progression engagement weights cache in `internal/progression/service.go`
- No version metadata on cached objects

### Problems

1. **Stale Cache Issues**: Code changes → cache format mismatch
2. **No Invalidation Strategy**: Can't detect outdated cache entries
3. **Silent Corruption**: Wrong data structure loaded from cache

### Recommendation
🟡 **Add version to cached objects (MEDIUM PRIORITY)**

#### Implementation

```go
// Versioned cache wrapper
type CachedData struct {
    Version   string      `json:"version"`
    UpdatedAt time.Time   `json:"updated_at"`
    Data      interface{} `json:"data"`
}

// Example for user cache
type CachedUser struct {
    Version string    `json:"version"` // "1.0"
    User    *User     `json:"user"`
    CachedAt time.Time `json:"cached_at"`
}

func (c *UserCache) Get(key string) (*User, bool) {
    cached, ok := c.data[key]
    if !ok {
        return nil, false
    }
    
    // Version check
    if cached.Version != CurrentCacheVersion {
        // Invalidate old version
        delete(c.data, key)
        return nil, false
    }
    
    return cached.User, true
}
```

#### Benefits

- Automatic invalidation on schema changes
- Prevents cache corruption bugs
- Easier cache debugging

#### Files to Update

- [ ] `internal/user/cache.go`
- [ ] Progression weight cache logic
- [ ] Add cache version constants

---

## 8. Dead-Letter Event Logs

### Current State

- **Path**: `logs/event_deadletter.jsonl` (configured in `.env`)
- **Format**: JSONL (JSON Lines)
- **No schema version**

### Recommendation

🟢 **Add schema version to dead-letter logs (LOW PRIORITY)**

#### Implementation

```json
{
  "schema_version": "1.0",
  "timestamp": "2026-01-05T19:26:04Z",
  "event_type": "engagement",
  "event_version": "1.0",
  "payload": {...},
  "error": "connection refused",
  "retry_count": 5
}
```

#### Benefits

- Easier log parsing and analysis
- Can update log format without breaking tools
- Better debugging of event failures

---

## 9. Discord Bot Commands

### Current State

- Interaction handlers in `internal/discord/`
- No explicit versioning

### Recommendation

✅ **No changes needed** - Discord interaction format is versioned by Discord API itself.

---

## Implementation Priority

### Phase 1: High Priority (Now)

1. 🔴 **Event Schema Versioning** - Prevents production bugs
   - Add `Version` field to events
   - Create typed event payloads
   - Estimated effort: 4-6 hours

### Phase 2: Medium Priority (Next Sprint)

1. 🟡 **Config File Versioning** - Improves deployment safety
   - Add version to JSON configs
   - Update parsers with validation
   - Estimated effort: 2-3 hours

1. 🟡 **Environment Schema Validation** - Catches deployment issues early
   - Add ENV_SCHEMA_VERSION checking
   - Create validator
   - Estimated effort: 2-3 hours

1. 🟡 **Cache Versioning** - Prevents cache corruption
   - Add version wrappers to caches
   - Implement auto-invalidation
   - Estimated effort: 2-3 hours

### Phase 3: Low Priority (Future)

1. 🟢 **Log Format Version** - Nice to have for log aggregation
   - Add `log_version` field
   - Estimated effort: 30 minutes

1. 🟢 **Dead-Letter Log Schema** - Improves debugging
   - Add `schema_version` field
   - Estimated effort: 30 minutes

---

## Related Issues

- `docs/issues/CODE_REVIEW_ISSUES.md` - Event system documentation
- `docs/issues/api-versioning-framework.md` - API versioning improvements
- `docs/VERSION_DETECTION.md` - Version detection guide

## References

- [Semantic Versioning](https://semver.org/) - Version numbering standard
- [JSON Schema](https://json-schema.org/) - Schema validation
- [Event Sourcing Patterns](https://martinfowler.com/eaaDev/EventSourcing.html) - Event versioning strategies
