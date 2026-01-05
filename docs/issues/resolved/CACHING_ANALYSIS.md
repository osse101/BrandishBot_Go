RESOLVED

# Caching Analysis Report

**Resolution Date:** 2026-01-05  
**Resolution Summary:** All recommendations have been implemented. The engagement weight cache TTL was already increased to 5 minutes, and the admin command `/admin-reload-weights` has been added to allow manual cache invalidation.

**Implementation Details:**
- Discord Command: `/admin-reload-weights` added in `cmd_progression_extended.go`
- API Endpoint: `POST /api/admin/progression/reload-weights` added in `progression.go`
- Route registered in `server.go`
- Command registered in `cmd/discord/main.go`

---

## Executive Summary

Analysis of caching strategies in the progression system, focusing on engagement weights and unlock threshold caching.


---

## Cache #1: Engagement Weights

### Purpose
Cache database-stored engagement metric weights to reduce DB load during high message volume.

### Implementation
```go
// service.go
type service struct {
    weightsMu     sync.RWMutex
    cachedWeights map[string]float64
    weightsExpiry time.Time
}
```

### Current Configuration
- **TTL**: 60 seconds
- **Invalidation**: Time-based expiry only
- **Thread Safety**: Yes (RWMutex)

### Usage Pattern
```
RecordEngagement("message", 1)
  ↓
getCachedWeight("message")
  ↓
IF expired OR not in cache:
    DB query → reload all weights → cache for 60s
ELSE:
    Return cached value (no DB query)
```

### Performance Impact

**Without Cache (theoretical):**
- 100 messages/minute = 100 DB queries for weights
- 1000 messages/minute = 1000 DB queries

**With Cache (actual):**
- 100 messages/minute = ~2 DB queries (1 initial + 1 refresh)
- 1000 messages/minute = ~17 DB queries (1 per 60s)
- **Reduction: 98-99% fewer queries**

### Question: Does 60s TTL Make Sense?

**Analysis:**
- Engagement weights rarely change (only on admin update)
- If weights change mid-cache, contributions will use old weight for up to 60s
- **Impact of stale weights**: Minimal
  - Example: Weight changes 1.0 → 2.0
  - Up to 60 messages might get scored at 1x instead of 2x
  - This is ~60 points out of thousands needed for unlock
  - Error margin: < 0.5%

**Recommendation: TTL can be longer**

| TTL | Pros | Cons |
|-----|------|------|
| **60s (current)** | Safe, responsive to changes | 17 DB queries/1000 messages |
| **5 minutes** | Better performance (3 queries/1000) | 5 min lag on weight updates |
| **15 minutes** | Excellent performance (1 query/1000) | 15 min lag on updates |
| **Until manual invalidate** | Best performance (1 query ever) | Requires restart or admin command |

**Suggested Change:** 
- Increase to **5 minutes** (300s)
- Add admin command: `/progression reload-weights` for manual invalidate
- 99.7% reduction in DB queries vs no cache
- Acceptable lag for a value that changes maybe once per month

---

## Cache #2: Unlock Threshold Cost

### Purpose
Enable instant unlock detection without DB query per contribution.

### Implementation
```go
type service struct {
    mu               sync.RWMutex
    cachedTargetCost int  // unlock_cost of target node
    cachedProgressID int  // current unlock progress ID
}
```

### Current Configuration
- **TTL**: None (manual invalidation only)
- **Invalidation**: On voting session end (target changes)
- **Thread Safety**: Yes (RWMutex)

### Usage Pattern
```
AddContribution(amount)
  ↓
DB: UPDATE contributions_accumulated += amount
  ↓
Check cache:
  IF cachedCost > 0 AND cachedProgressID matches:
    Re-fetch progress from DB
    IF accumulated >= cachedCost:
      Trigger unlock
```

### Performance Impact

**Cost of Check:**
- 1 extra DB query per contribution (re-fetch progress)
- But enables instant unlock detection
- Alternative: Poll every N seconds (worse)

### Question: Does This Need Expiry?

**Analysis:**
- Cache is set when voting ends and target is chosen
- Cache is valid until that specific target unlocks
- Once unlocked, new progress row created (different ID)
- Cache becomes stale but safe (ID won't match)

**Scenarios Where Cache is Wrong:**

1. **Admin changes target manually**
   - Old cachedProgressID won't match new progress
   - ✅ Safe: Check fails, no false unlock

2. **Database manually edited**
   - Unlock cost changed while target active
   - ❌ Risk: Could unlock early or late
   - **Reality**: This should never happen in production

3. **Multiple app instances** (future)
   - Instance A doesn't know Instance B set new target
   - ❌ Risk: Unlock might not trigger on Instance A
   - **Solution**: Distributed cache (Redis) or event broadcast

**Recommendation: No expiry needed**

This cache is **event-driven**, not time-driven. It should be invalidated by:
- Unlock completion (new progress created)
- Admin target override (set cache to new values)
- Never by time

**Suggested Changes:**
- Remove any time-based expiry logic (there is none currently ✓)
- Add cache invalidation to admin override commands
- Document that cache is session-scoped

---

## Cache #3: Proposed - Available Unlocks

### Current Behavior
Every call to `GetAvailableUnlocks()` queries:
- All nodes
- All current unlocks
- Filters based on prerequisites

This is called:
- At voting session start
- By admin commands
- Potentially by UI (future)

### Proposal
Cache the list of available nodes with short TTL (30s).

**Benefits:**
- Reduces load during admin tree browsing
- Allows rapid voting session restarts
- Supports future UI without DB spam

**TTL Rationale:**
- Available nodes change only when unlock happens
- Unlocks are infrequent (minutes to hours apart)
- 30s lag is acceptable

---

## Comparison Table

| Cache | Current TTL | Purpose | Query Reduction | Stale Data Risk | Recommended TTL |
|-------|------------|---------|-----------------|-----------------|----------------|
| **Engagement Weights** | 60s | Reduce weight lookups | 98% | Very Low (weights rarely change) | **5 min** |
| **Unlock Threshold** | None | Instant unlock detection | N/A (enables feature) | Low (ID mismatch safe) | **None (event-driven)** ✓ |
| **Available Unlocks** | None | - | - | - | **30s (proposed)** |

---

## Recommendations Summary

### Immediate
1. ✅ Keep unlock threshold cache with no expiry (correct as-is)
2. 🔧 Increase engagement weight TTL to 5 minutes
3. 🔧 Add `/progression reload-weights` admin command

### Future
4. 📋 Implement available unlocks cache (30s TTL)
5. 📋 Add metrics/monitoring for cache hit rates
6. 📋 Consider Redis for multi-instance deployments

---

## Code Changes

### Increase Weight Cache TTL

```go
// service.go:575
func (s *service) cacheWeights(weights map[string]float64) {
    s.weightsMu.Lock()
    defer s.weightsMu.Unlock()
    
    s.cachedWeights = weights
    s.weightsExpiry = time.Now().Add(5 * time.Minute) // was 60s
}
```

### Add Manual Invalidation

```go
// service.go
func (s *service) InvalidateWeightCache() {
    s.weightsMu.Lock()
    defer s.weightsMu.Unlock()
    
    s.cachedWeights = nil
    s.weightsExpiry = time.Time{} // Zero time = expired
}
```

### Admin Command

```go
// admin.go
func (s *service) AdminReloadWeights(ctx context.Context) error {
    s.InvalidateWeightCache()
    log.Info("Engagement weight cache invalidated")
    return nil
}
```

---

**Date:** 2025-12-28  
**Status:** Recommendations pending review
