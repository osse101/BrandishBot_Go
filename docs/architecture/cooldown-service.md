# Cooldown Service Architecture Design

**Design Doc ID:** `ARCH-002`  
**Author:** System  
**Created:** 2025-12-22  
**Status:** Proposed  
**Related Issues:** [`RACE-001`](file:///home/osse1/projects/BrandishBot_Go/docs/issues/RACE-001-handlesearch-cooldown.md)

---

## Table of Contents
1. [Problem Statement](#problem-statement)
2. [Goals & Non-Goals](#goals--non-goals)
3. [Architecture Overview](#architecture-overview)
4. [Interface Design](#interface-design)
5. [Implementation Options](#implementation-options)
6. [Migration Strategy](#migration-strategy)
7. [Testing](#testing)
8. [Performance Considerations](#performance-considerations)
9. [Future Enhancements](#future-enhancements)

---

## Problem Statement

### Current State

Cooldown logic is currently **scattered** across the codebase:

```go
// In user/service.go (HandleSearch)
lastUsed := repo.GetLastCooldown(ctx, user.ID, "search")
if onCooldown(lastUsed) {
    return "cooldown active"
}
// ... process action ...
repo.UpdateCooldown(ctx, user.ID, "search", now)
```

**Problems:**

1. **Race Conditions:** Non-atomic check/update allows concurrent bypass
2. **Code Duplication:** Each feature reimplements cooldown logic
3. **Inconsistent Patterns:** Dev mode handling varies
4. **Hard to Scale:** Tight coupling to database implementation
5. **Testing Complexity:** Need to mock repository methods

### Desired State

Centralized cooldown service with:
- ✅ Built-in race prevention
- ✅ Consistent API across all features
- ✅ Pluggable backend (Postgres, Redis, hybrid)
- ✅ Easy testing and mocking

---

## Goals & Non-Goals

### Goals

- **Atomicity:** Eliminate race conditions via proper locking
- **Consistency:** Single source of truth for all cooldowns
- **Flexibility:** Support multiple backend implementations
- **Performance:** Optimize for common case (requests on cooldown)
- **Testability:** Easy to mock and test

### Non-Goals

- **Distributed Locking:** Single-instance focus (can add later)
- **Complex Scheduling:** Not a cron/job scheduler
- **Historical Tracking:** Don't store cooldown history (use stats service)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                    Application Layer                     │
│  (User Service, Economy Service, Combat Service, etc.)  │
└────────────────┬────────────────────────────────────────┘
                 │
                 │ calls EnforceCooldown()
                 │
┌────────────────▼────────────────────────────────────────┐
│              Cooldown Service Interface                  │
│  - CheckCooldown()                                       │
│  - EnforceCooldown()                                     │
│  - ResetCooldown()                                       │
└────────────────┬────────────────────────────────────────┘
                 │
         ┌───────┴───────────┐
         │                   │
    ┌────▼─────┐      ┌─────▼──────┐
    │ Postgres │      │   Redis    │
    │ Backend  │      │  Backend   │
    └──────────┘      └────────────┘
```

### Key Components

1. **Cooldown Service** - High-level interface
2. **Backend Implementations** - Postgres, Redis, or Hybrid
3. **Action Registry** - Define cooldowns per action type

---

## Interface Design

### Core Interface

```go
package cooldown

import (
    "context"
    "time"
)

// Service manages action cooldowns for users
type Service interface {
    // CheckCooldown checks if a user's action is on cooldown
    // Returns: (onCooldown bool, remaining time.Duration, error)
    CheckCooldown(ctx context.Context, userID, action string) (bool, time.Duration, error)
    
    // EnforceCooldown atomically checks cooldown and executes action if allowed
    // This is the primary method - prevents race conditions
    EnforceCooldown(ctx context.Context, userID, action string, fn func() error) error
    
    // ResetCooldown manually resets a cooldown (admin/testing)
    ResetCooldown(ctx context.Context, userID, action string) error
    
    // GetLastUsed returns when action was last performed (for UI display)
    GetLastUsed(ctx context.Context, userID, action string) (*time.Time, error)
}

// Action defines a cooldown configuration
type Action struct {
    Name     string        // "search", "daily_claim", "pvp_attack"
    Duration time.Duration // How long to wait between uses
}

// ErrOnCooldown is returned when action is still on cooldown
type ErrOnCooldown struct {
    Action    string
    Remaining time.Duration
}

func (e ErrOnCooldown) Error() string {
    return fmt.Sprintf("action '%s' on cooldown for %v", e.Action, e.Remaining)
}
```

### Usage Example

```go
// In user service
func (s *service) HandleSearch(ctx context.Context, platform, platformID, username string) (string, error) {
    user := s.getUserOrRegister(ctx, platform, platformID, username)
    
    // Single call handles everything atomically
    err := s.cooldownService.EnforceCooldown(ctx, user.ID, "search", func() error {
        // This only executes if cooldown passed
        return s.executeSearch(ctx, user)
    })
    
    if errors.Is(err, cooldown.ErrOnCooldown{}) {
        remaining := err.(cooldown.ErrOnCooldown).Remaining
        return fmt.Sprintf("Search on cooldown: %v remaining", remaining), nil
    }
    
    return result, err
}
```

---

## Implementation Options

### Option 1: Postgres Backend (Default)

**Pros:**
- ✅ No new infrastructure
- ✅ ACID guarantees
- ✅ Works with existing schema

**Implementation:**

```go
type postgresBackend struct {
    db *pgxpool.Pool
}

func (b *postgresBackend) EnforceCooldown(ctx, userID, action, fn) error {
    // Start transaction
    tx, err := b.db.Begin(ctx)
    defer tx.Rollback(ctx)
    
    // LOCKED check (SELECT FOR UPDATE)
    lastUsed, err := b.getLastUsedLocked(tx, userID, action)
    if stillOnCooldown(lastUsed) {
        return ErrOnCooldown{Remaining: remaining}
    }
    
    // Execute user function
    if err := fn(); err != nil {
        return err
    }
    
    // Update cooldown
    b.updateCooldownTx(tx, userID, action, time.Now())
    
    return tx.Commit()
}
```

**Performance:**
- **Fast path (on cooldown):** 1-2ms (rejected without transaction)
- **Slow path (allowed):** 5-10ms (full transaction)

---

### Option 2: Redis Backend (Scalable)

**Pros:**
- ✅ Sub-millisecond latency
- ✅ Naturally atomic (SET NX)
- ✅ Horizontal scaling

**Cons:**
- ⚠️ Requires Redis infrastructure
- ⚠️ Data split between systems

**Implementation:**

```go
type redisBackend struct {
    client *redis.Client
}

func (b *redisBackend) EnforceCooldown(ctx, userID, action, fn) error {
    key := fmt.Sprintf("cooldown:%s:%s", action, userID)
    
    // Atomic check: try to set key with NX (only if not exists)
    ok, err := b.client.SetNX(ctx, key, "1", duration).Result()
    if err != nil {
        return err
    }
    
    if !ok {
        // Key exists = on cooldown
        ttl, _ := b.client.TTL(ctx, key).Result()
        return ErrOnCooldown{Remaining: ttl}
    }
    
    // Key was set = we acquired the cooldown lock
    return fn()
}
```

**Performance:**
- **Check + set:** <1ms
- **TTL query:** <1ms

---

### Option 3: Hybrid Backend (Best of Both)

**Strategy:** Redis for speed, Postgres for durability

```go
type hybridBackend struct {
    redis    *redisBackend
    postgres *postgresBackend
}

func (b *hybridBackend) EnforceCooldown(ctx, userID, action, fn) error {
    // Fast path: Try Redis first
    err := b.redis.EnforceCooldown(ctx, userID, action, fn)
    
    if err == nil {
        // Success! Async persist to Postgres for durability
        go b.postgres.updateCooldownAsync(userID, action, time.Now())
        return nil
    }
    
    // Redis failed (down/timeout) - fallback to Postgres
    if isRedisError(err) {
        return b.postgres.EnforceCooldown(ctx, userID, action, fn)
    }
    
    // On cooldown or other error
    return err
}
```

---

## Migration Strategy

### Phase 1: Create Service (Non-Breaking)

```go
// 1. Create new package
package cooldown

// 2. Implement Postgres backend (reuse existing tables)
type postgresBackend struct { /* ... */ }

// 3. Add to main.go (optional dependency)
cooldownSvc := cooldown.NewPostgresService(db, cooldown.Config{
    DevMode: cfg.DevMode,
})
```

No existing code changes required yet.

### Phase 2: Migrate HandleSearch

```go
// Update user service constructor
func NewService(..., cooldownSvc cooldown.Service) *service {
    return &service{
        cooldownService: cooldownSvc,
        // ...
    }
}

// Refactor HandleSearch
func (s *service) HandleSearch(...) (string, error) {
    // Old code (40 lines) → New code (5 lines)
    err := s.cooldownService.EnforceCooldown(ctx, user.ID, "search", func() error {
        return s.executeSearch(ctx, user)
    })
    // Handle err...
}
```

### Phase 3: Migrate Other Features

Gradually migrate other cooldown uses:
- Economy daily claims
- PvP attack cooldowns
- Crafting cooldowns (if any)

### Phase 4: Remove Old Code

Once all migrated:
- Remove `GetLastCooldown`, `UpdateCooldown` from user repository
- Clean up `user_cooldowns` table schema if needed

---

## Testing

### Unit Tests

```go
func TestCooldownService_EnforceCooldown_Success(t *testing.T) {
    svc := cooldown.NewPostgresService(testDB, cooldown.Config{})
    
    executed := false
    err := svc.EnforceCooldown(ctx, "user123", "search", func() error {
        executed = true
        return nil
    })
    
    assert.NoError(t, err)
    assert.True(t, executed)
}

func TestCooldownService_EnforceCooldown_OnCooldown(t *testing.T) {
    svc := cooldown.NewPostgresService(testDB, cooldown.Config{})
    
    // First call succeeds
    svc.EnforceCooldown(ctx, "user123", "search", func() error { return nil })
    
    // Second call immediately after should fail
    executed := false
    err := svc.EnforceCooldown(ctx, "user123", "search", func() error {
        executed = true
        return nil
    })
    
    assert.Error(t, err)
    assert.ErrorIs(t, err, cooldown.ErrOnCooldown{})
    assert.False(t, executed) // Function not called
}
```

### Integration Tests

```go
func TestCooldownService_ConcurrentRequests(t *testing.T) {
    svc := cooldown.NewPostgresService(realDB, cooldown.Config{})
    
    // Fire 10 concurrent requests
    var successCount atomic.Int32
    var wg sync.WaitGroup
    
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            err := svc.EnforceCooldown(ctx, "user123", "search", func() error {
                successCount.Add(1)
                return nil
            })
        }()
    }
    
    wg.Wait()
    
    // Only ONE should succeed
    assert.Equal(t, int32(1), successCount.Load())
}
```

### Mock for Testing

```go
type MockCooldownService struct {
    AlwaysAllow bool
    Calls       []string
}

func (m *MockCooldownService) EnforceCooldown(ctx, userID, action, fn) error {
    m.Calls = append(m.Calls, action)
    
    if m.AlwaysAllow {
        return fn()
    }
    return cooldown.ErrOnCooldown{Action: action, Remaining: time.Minute}
}
```

---

## Performance Considerations

### Check-Then-Lock Optimization

```go
func (b *postgresBackend) EnforceCooldown(ctx, userID, action, fn) error {
    // PHASE 1: Cheap unlocked check (rejects ~90% fast)
    lastUsed, _ := b.getLastUsed(ctx, userID, action) // No lock
    if stillOnCooldown(lastUsed) {
        return ErrOnCooldown{} // 1ms, no transaction
    }
    
    // PHASE 2: Expensive locked check (only if passed Phase 1)
    tx := b.db.Begin(ctx)
    lastUsed, _ = b.getLastUsedLocked(tx, userID, action) // FOR UPDATE
    if stillOnCooldown(lastUsed) {
        return ErrOnCooldown{} // Race detected
    }
    
    // Execute and update
    fn()
    b.updateCooldownTx(tx, userID, action, now)
    tx.Commit()
}
```

**Latency Profile:**
- Requests on cooldown: **1-2ms** (fast rejection)
- Requests off cooldown: **5-10ms** (full transaction)

### Caching (Optional Enhancement)

For read-heavy workloads:

```go
type cachedBackend struct {
    backend Backend
    cache   *lru.Cache // In-memory cache
}

func (b *cachedBackend) CheckCooldown(ctx, userID, action) (bool, time.Duration, error) {
    // Check cache first
    if cached, ok := b.cache.Get(userID + ":" + action); ok {
        return calculateRemaining(cached.(time.Time))
    }
    
    // Miss - query backend
    lastUsed, _ := b.backend.GetLastUsed(ctx, userID, action)
    b.cache.Set(userID+":"+action, lastUsed, 10*time.Second)
    return calculateRemaining(lastUsed)
}
```

---

## Configuration

### Action Registry

```go
// In domain/actions.go
package domain

const (
    ActionSearch      = "search"
    ActionDailyClaim  = "daily_claim"
    ActionPvPAttack   = "pvp_attack"
)

var ActionCooldowns = map[string]time.Duration{
    ActionSearch:     5 * time.Minute,
    ActionDailyClaim: 24 * time.Hour,
    ActionPvPAttack:  30 * time.Second,
}
```

### Service Configuration

```go
type Config struct {
    DevMode     bool              // Bypass all cooldowns
    Backend     string            // "postgres", "redis", "hybrid"
    RedisURL    string            // If using Redis backend
    Cooldowns   map[string]time.Duration // Override defaults
}
```

---

## Future Enhancements

### 1. Variable Cooldowns

Support cooldowns that vary by user level:

```go
func (s *service) EnforceCooldownWithDuration(ctx, userID, action string, duration time.Duration, fn func() error) error
```

### 2. Cooldown Groups

Share cooldowns across related actions:

```go
// Any skill1/skill2/skill3 starts cooldown for all
group := cooldown.NewGroup("combat_skills", 1*time.Minute)
group.Add("skill1", "skill2", "skill3")
```

### 3. Distributed Locking

For multi-instance deployments with Redis:

```go
type redisBackend struct {
    client   *redis.Client
    redsync  *redsync.Redsync // Distributed locks
}
```

### 4. Metrics & Monitoring

```go
// Track cooldown bypass attempts, hit rates, etc.
type MetricsBackend struct {
    backend Backend
    metrics prometheus.Collector
}
```

---

## Decision Matrix

| Feature | Postgres | Redis | Hybrid |
|---------|----------|-------|--------|
| No new infra | ✅ | ❌ | ❌ |
| Performance | Good (5-10ms) | Excellent (<1ms) | Excellent |
| Horizontal scaling | Limited | ✅ | ✅ |
| ACID guarantees | ✅ | ❌ | ✅ (Postgres fallback) |
| Complexity | Low | Medium | High |
| **Recommendation** | **Start here** | When scaling | Future optimization |

---

## Implementation Timeline

### Minimal (1-2 hours)
- Create `internal/cooldown` package
- Implement Postgres backend
- Unit tests
- Wire into main.go

### Full Migration (4-6 hours)
- Migrate HandleSearch
- Add integration tests
- Performance benchmarks
- Documentation

### Redis Support (8-12 hours)
- Redis backend implementation
- Hybrid backend
- Distributed lock testing
- Deployment setup

---

## Acceptance Criteria

✅ **Must Have:**
- [ ] Service interface defined
- [ ] Postgres backend implemented
- [ ] Race conditions eliminated
- [ ] Unit tests (90%+ coverage)
- [ ] Integration test (concurrent requests)
- [ ] HandleSearch migrated successfully

✅ **Should Have:**
- [ ] Config-based cooldown durations
- [ ] Dev mode bypass
- [ ] Admin reset capability
- [ ] Performance benchmarks

🎯 **Nice to Have:**
- [ ] Redis backend
- [ ] Metrics/monitoring
- [ ] Cooldown groups

---

## References

- Related Issue: [`RACE-001`](file:///home/osse1/projects/BrandishBot_Go/docs/issues/RACE-001-handlesearch-cooldown.md)
- Pattern: Check-then-lock (Optimistic Concurrency Control)
- Example: Stripe's rate limiting service
- Database: Existing `user_cooldowns` table schema

---

## Questions & Discussions

1. **Should cooldown durations be configurable per-environment?**
   - Shorter cooldowns in dev/staging?

2. **Should we track cooldown history for analytics?**
   - Or rely on stats service for event tracking?

3. **Priority of Redis support?**
   - Worth the infrastructure complexity now?

4. **Naming: "Cooldown" vs "RateLimit" service?**
   - Cooldown = game-specific
   - RateLimit = more generic
