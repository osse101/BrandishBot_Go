# Additional Caching Opportunities

**Created:** 2026-01-02  
**Status:** RESOLVED (2026-01-07)
**Priority:** ~~Medium~~ **Low** (Reduced based on findings)
**Labels:** performance, caching, enhancement

## Resolution Summary

**Status:** Most proposed caching is either **already implemented** or **not needed** based on actual usage patterns.

### What's Already Cached ✅

1. **User lookups** - LRU cache (1000 entries, 5min TTL) - [Done 2026-01-07]
2. **Item metadata** - In-memory cache (forever TTL, cleared on restart)
3. **Progression modifiers** - 30min TTL, invalidated on unlock/relock ([`cache.go`](file:///home/osse1/projects/BrandishBot_Go/internal/progression/cache.go))
4. **Engagement weights** - 5min TTL, manual invalidation

### What Doesn't Need Caching ❌


#### 1. Inventory (Phase 1 - REJECTED)

**Reason:** Usage pattern analysis shows caching provides minimal value.
- **Typical pattern:** Long break → `!inventory` → 1s → mutate inventory → 1s → `!inventory` → long break
- **Cache hit potential:** Very low due to immediate mutations after reads
- **User tolerance:** Slight delay is acceptable for inventory queries
- **Decision:** Don't implement - ROI too low

#### 2. Progression Tree (Phase 2 - LOW PRIORITY)
**Current implementation:** [`GetProgressionTree`](file:///home/osse1/projects/BrandishBot_Go/internal/progression/service.go#L113-L160)
- Queries: `GetAllNodes`, `GetAllUnlocks`, `GetDependents` (per node)
- **Used by:** Admin command `/admin-tree-status` only
- **Frequency:** Very low (admin-only operation)
- **Decision:** Not worth caching - low access frequency, admin can tolerate latency

#### 3. Leaderboards (Phase 3 - REJECTED)
**Reason:** Low frequency command, doesn't need optimization
- **Access:** Only via `/leaderboard` API and Discord commands
- **Frequency:** Low
- **Decision:** Current implementation is fine

#### 4. System Stats (Phase 4 - REJECTED)
**Reason:** Same as leaderboards - low frequency, admin/monitoring only
- **Decision:** Not needed

## Summary

**Before investigation:**
- Proposed 4 caching phases (inventory, progression tree, leaderboards, stats)
- Estimated 10 hours of work
- Expected significant performance gains

**After investigation:**
- ✅ Most valuable caching already done (users, items, modifiers, weights)
- ❌ Remaining opportunities have low ROI due to:
  - Unfavorable access patterns (inventory)
  - Low access frequency (tree, leaderboards, stats)
  - Admin-only operations (tree status)

**Final decision:** Close issue as RESOLVED. Core caching infrastructure is in place and optimized for actual usage patterns.

---

## Original Analysis (For Reference)

### Background

**Existing Caching:**
1. ✅ **Item metadata** - Already cached in [`service.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/service.go#L114-L122)
2. ✅ **User lookups** - Recently implemented ([`cache.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/cache.go))
3. ✅ **Progression modifiers** - Cached ([`progression/cache.go`](file:///home/osse1/projects/BrandishBot_Go/internal/progression/cache.go))
4. ✅ **Engagement weights** - Cached (5min TTL)

**Opportunities Initially Proposed:** Inventory, progression tree, leaderboards, stats

## Related Files

- [`internal/user/service_helpers.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/service_helpers.go#L9-L29) - Item cache (existing)
- [`internal/user/cache.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/cache.go) - User cache (existing)
- [`internal/progression/cache.go`](file:///home/osse1/projects/BrandishBot_Go/internal/progression/cache.go) - Modifier cache (existing)
- [`internal/user/service.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/service.go#L607-L665) - GetInventory
- [`internal/progression/service.go`](file:///home/osse1/projects/BrandishBot_Go/internal/progression/service.go) - Tree queries
- [`internal/stats/service.go`](file:///home/osse1/projects/BrandishBot_Go/internal/stats/service.go) - Leaderboards

## Originally Proposed Strategies (Not Implemented)

### ~~1. Inventory Caching (HIGH PRIORITY)~~ - REJECTED

**Problem:** `GetInventory()` called frequently, inventory changes infrequently

**REJECTED REASON:** User analysis shows inventory mutations happen immediately after reads, making cache ineffective.

**Access Pattern:**
- `/inventory` endpoint (read-heavy)
- Discord `/inventory` command (very frequent)
- Quest completion checks

**Current:** Every request hits database

**User feedback:** "inventory most common usage is Long break -> !inventory -> 1s -> mutate inventory -> 1s -> !inventory -> long break. So adding caching doesn't look useful/ not a priority. And its fine if there is slight delay."

---

### ~~2. Progression Tree Caching (MEDIUM PRIORITY)~~ - DEFERRED (Admin-only)

**Problem:** Progression tree queried on every vote/unlock check

**ANALYSIS:** `GetProgressionTree` is only called from `/admin-tree-status` Discord command (admin-only, low frequency).

**Access Pattern:**
- Tree structure rarely changes (only on admin updates)
- Heavy read traffic (every `/progression/*` endpoint)

**ACTUAL USAGE:** Admin command only - very low frequency

**Current:** Database query per request

**Proposed:** Application-wide cache with manual invalidation

```go
type progressionTreeCache struct {
    tree      *domain.ProgressionTree
    mu        sync.RWMutex
    expiresAt time.Time
}

func (s *service) GetTree(ctx context.Context) (*domain.ProgressionTree, error) {
    s.treeCacheMu.RLock()
    if s.treeCache != nil && time.Now().Before(s.treeCacheExpiry) {
        tree := s.treeCache
        s.treeCacheMu.RUnlock()
        return tree, nil
    }
    s.treeCacheMu.RUnlock()
    
    // Fetch from DB
    tree, err := s.repo.GetProgressionTree(ctx)
    if err != nil {
        return nil, err
    }
    
    // Cache for 10 minutes (tree changes rarely)
    s.treeCacheMu.Lock()
    s.treeCache = tree
    s.treeCacheExpiry = time.Now().Add(10 * time.Minute)
    s.treeCacheMu.Unlock()
    
    return tree, nil
}

// Explicit invalidation on tree updates
func (s *service) AdminUnlock(...) error {
    // ... unlock logic ...
    
    s.InvalidateTreeCache()
    return nil
}
```

**Invalidation:** Manual on admin operations (unlock, relock, instant unlock, JSON sync)

**Expected Impact:**
- **Cache hit rate:** ~99% (tree changes rarely)
- **DB query reduction:** ~99% for tree queries
- **P95 latency:** Progression endpoints: 15ms → 2ms

**DECISION:** Deferred - only admin uses this, ROI too low

---

### ~~3. Leaderboard Caching (LOW PRIORITY)~~ - REJECTED

**Problem:** Leaderboard computation expensive, but data changes slowly

**User feedback:** "Stats and leaderboard are low frequency commands and don't need to be made efficient"

---

### ~~4. System Stats Caching (LOW PRIORITY)~~ - REJECTED

**User feedback:** "Stats and leaderboard are low frequency commands and don't need to be made efficient"

---

## References

- [User Cache Implementation](file:///home/osse1/projects/BrandishBot_Go/internal/user/cache.go)
- [Progression Modifier Cache](file:///home/osse1/projects/BrandishBot_Go/internal/progression/cache.go)
- [Item Cache (Existing)](file:///home/osse1/projects/BrandishBot_Go/internal/user/service_helpers.go#L9-L29)
- [Benchmarking Journal](file:///home/osse1/projects/BrandishBot_Go/docs/benchmarking/journal.md)
- [User Cache Enhancements (Resolved)](file:///home/osse1/projects/BrandishBot_Go/docs/issues/resolved/user_cache_enhancements.md)
