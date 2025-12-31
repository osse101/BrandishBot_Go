# Progression System Concurrency Issues

## Overview
This document tracks known concurrency and scalability issues in the progression system that affect performance under load.

## Critical Issues

### 1. Engagement Weight Cache Missing
**Severity:** High  
**Impact:** 100 concurrent messages = 100 DB queries for same data

**Current Behavior:**
```go
// progression/service.go:255
weights, err := s.repo.GetEngagementWeights(ctx)
```
Every `RecordEngagement` call fetches weights from DB, even though they rarely change.

**Solution:**
- Add in-memory cache with TTL (60s)
- Invalidate on weight updates
- Use sync.RWMutex for thread-safety

**Status:** ✅ Fixed in this session

---

### 2. Double-Fetch of Unlock Progress
**Severity:** Medium  
**Impact:** Unnecessary 100 extra DB queries under load

**Current Behavior:**
```go
// progression/voting_sessions.go:208
progress, err := s.repo.GetActiveUnlockProgress(ctx)
// ... do work ...
// Line 246 - fetch AGAIN
updatedProgress, err := s.repo.GetActiveUnlockProgress(ctx)
```

**Solution:**
Cache progress value from first fetch, only re-fetch if AddContribution succeeded.

**Status:** ✅ Fixed in this session

---

### 3. CheckAndUnlockNode Goroutine Stampede
**Severity:** High  
**Impact:** 50+ concurrent unlock attempts on threshold crossing

**Current Behavior:**
```go
// progression/voting_sessions.go:253
go s.CheckAndUnlockNode(context.Background())
```

Multiple concurrent `AddContribution` calls can all see threshold met and spawn goroutines.

**Solution Implemented:**
- Added semaphore (channel) limiting concurrent unlock checks to 1
- Non-blocking send - if unlock already in progress, skip

**Status:** ✅ Fixed in this session

---

### 4. Orphaned Background Contexts
**Severity:** Medium  
**Impact:** Unlock operations not cancelled on shutdown

**Current Behavior:**
```go
go s.CheckAndUnlockNode(context.Background())
```

Uses `context.Background()` which can't be cancelled. If app shuts down mid-unlock, operation continues.

**Solution:**
Use a long-lived service context that gets cancelled on shutdown.

**Status:** ⚠️ Documented, requires service lifecycle refactor

---

## Performance Issues

### 5. No Connection Pool Limits
**Severity:** Medium  
**Impact:** 1000 concurrent messages exhaust DB connections

**Current Behavior:**
Default PostgreSQL settings allow unlimited connection attempts. Under extreme load (1000+ concurrent), all connections consumed.

**Solution:**
- Set explicit `MaxOpenConns` and `MaxIdleConns` in DB config
- Add connection pool metrics
- Implement circuit breaker pattern

**Status:** ⚠️ Requires configuration change + middleware

---

### 6. Event Bus Unbounded Queue
**Severity:** Medium  
**Impact:** Memory growth under sustained high load

**Current Behavior:**
Event bus doesn't have backpressure mechanism. If subscribers are slower than publishers, queue grows.

**Solution:**
- Implement bounded queue with configurable size
- Drop events or block publishers when full
- Add queue depth metrics

**Status:** ⚠️ Requires event bus refactor

---

### 7. No Rate Limiting on Message Handler
**Severity:** Low  
**Impact:** Allows malicious burst traffic

**Current Behavior:**
`HandleMessage` endpoint has no rate limiting. User can spam messages.

**Solution:**
- Add per-user rate limiter (e.g., 10 messages/sec)
- Add global rate limiter (e.g., 100 messages/sec)
- Use token bucket or sliding window

**Status:** ⚠️ Requires middleware implementation

---

## Race Conditions (Theoretical)

### 8. Cache Invalidation During Read
**Severity:** Low  
**Impact:** Stale cache reads during voting session transitions

**Current Behavior:**
```go
// Read with RLock
s.mu.RLock()
cost := s.cachedTargetCost
s.mu.RUnlock()

// But EndVotingSession writes with Lock
s.mu.Lock()
s.cachedTargetCost = newCost
s.mu.Unlock()
```

RWMutex protects access, but there's a window where cache might not be set yet.

**Solution:**
Current implementation is actually safe. Document behavior: cache miss = 0, which means "check DB".

**Status:** ✅ Not a bug, working as intended

---

### 9. Duplicate Unlock Attempts
**Severity:** Medium  
**Impact:** Multiple unlocks of same node if semaphore fails

**Current Behavior:**
Even with semaphore, if first goroutine unlocks node but second goroutine already past the check, could attempt unlock twice.

**Solution:**
Database constraint on `progression_unlocks (node_id, level)` prevents duplicates. First wins, second fails gracefully.

**Verification Needed:**
Check that UNIQUE constraint exists in schema.

**Status:** ⚠️ Verify DB schema has constraint

---

## Load Testing Requirements

### Test Scenarios
1. **100 concurrent messages** - Should handle gracefully in <5s
2. **1000 concurrent messages** - Should degrade gracefully, no panic
3. **Sustained 10 msg/sec for 1 hour** - No memory leaks
4. **Unlock threshold crossing during load** - No duplicate unlocks

### Metrics to Track
- DB connection pool utilization
- Goroutine count
- Event queue depth
- Response time p50/p95/p99
- Error rate

---

## Migration Path

### Phase 1: Immediate Fixes (This Session)
- [x] Cache engagement weights
- [x] Remove double-fetch
- [x] Add unlock semaphore

### Phase 2: Configuration (Next Deploy)
- [ ] Set DB connection pool limits
- [ ] Add structured logging for concurrency events
- [ ] Add basic metrics/observability

### Phase 3: Architecture (Future)
- [ ] Implement rate limiting middleware
- [ ] Add circuit breaker for DB
- [ ] Bounded event bus queue
- [ ] Distributed lock for unlocks (Redis)
- [ ] Horizontal scaling with leader election

---

## References
- `internal/progression/service.go` - Core progression logic
- `internal/progression/voting_sessions.go` - Unlock threshold checking
- `internal/middleware/engagement.go` - Event publishing

**Last Updated:** 2025-12-28  
**Owner:** Progression System Team
