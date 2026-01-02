# Additional Caching Opportunities

**Created:** 2026-01-02  
**Status:** Proposed  
**Priority:** Medium  
**Labels:** performance, caching, enhancement

## Summary

Identify and implement additional caching opportunities beyond user lookups to reduce database load and improve latency. Focus on frequently accessed, rarely changing data: item metadata, inventory, and progression tree.

## Background

**Existing Caching:**
1. ✅ **Item metadata** - Already cached in [`service.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/service.go#L114-L122)
2. ✅ **User lookups** - Recently implemented ([`cache.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/cache.go))

**Opportunities:** Inventory, progression tree, leaderboards, stats

## Related Files

- [`internal/user/service_helpers.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/service_helpers.go#L9-L29) - Item cache (existing)
- [`internal/user/cache.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/cache.go) - User cache (existing)
- [`internal/user/service.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/service.go#L607-L665) - GetInventory
- [`internal/progression/service.go`](file:///home/osse1/projects/BrandishBot_Go/internal/progression/service.go) - Tree queries
- [`internal/stats/service.go`](file:///home/osse1/projects/BrandishBot_Go/internal/stats/service.go) - Leaderboards

## Proposed Caching Strategies

### 1. Inventory Caching (HIGH PRIORITY)

**Problem:** `GetInventory()` called frequently, inventory changes infrequently

**Access Pattern:**
- `/inventory` endpoint (read-heavy)
- Discord `/inventory` command (very frequent)
- Quest completion checks

**Current:** Every request hits database

**Proposed:** LRU cache with write-through invalidation

```go
type inventoryCache struct {
    lru *expirable.LRU[string, *domain.Inventory]  // key: userID
}

func (s *service) GetInventory(ctx context.Context, platform, platformID, username, filter string) ([]UserInventoryItem, error) {
    user, err := s.getUserOrRegister(ctx, platform, platformID, username)
    if err != nil {
        return nil, err
    }
    
    // Try cache first
    if inv, ok := s.inventoryCache.Get(user.ID); ok {
        return s.filterInventory(ctx, inv, filter)
    }
    
    // Cache miss - fetch from DB
    inventory, err := s.repo.GetInventory(ctx, user.ID)
    if err != nil {
        return nil, err
    }
    
    // Cache for 2 minutes (shorter TTL due to frequent updates)
    s.inventoryCache.Set(user.ID, inventory)
    
    return s.filterInventory(ctx, inventory, filter)
}
```

**Invalidation Points:**
- `AddItem()` / `AddItems()`
- `RemoveItem()`
- `UseItem()`
- `GiveItem()`

```go
func (s *service) AddItem(...) error {
    // ... existing logic ...
    
    // Invalidate cache after inventory update
    s.inventoryCache.Invalidate(user.ID)
    
    return nil
}
```

**Expected Impact:**
- **Read-heavy endpoints:** 80% cache hit rate
- **DB query reduction:** 70-80% for inventory queries
- **Latency:** GetInventory: 10ms → <1ms

**Configuration:**
- Size: 500 users
- TTL: 2 minutes (shorter due to frequent updates)

---

### 2. Progression Tree Caching (MEDIUM PRIORITY)

**Problem:** Progression tree queried on every vote/unlock check

**Access Pattern:**
- Tree structure rarely changes (only on admin updates)
- Heavy read traffic (every `/progression/*` endpoint)

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

---

### 3. Leaderboard Caching (LOW PRIORITY)

**Problem:** Leaderboard computation expensive, but data changes slowly

**Current:** Computed on every request

**Proposed:** Background refresh with stale-while-revalidate

```go
type leaderboardCache struct {
    data      []domain.LeaderboardEntry
    mu        sync.RWMutex
    refreshAt time.Time
}

// Background goroutine refreshes every 30 seconds
func (s *service) startLeaderboardRefresh(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    go func() {
        for {
            select {
            case <-ticker.C:
                lb, err := s.repo.GetLeaderboard(ctx)
                if err == nil {
                    s.leaderboardCacheMu.Lock()
                    s.leaderboardCache = lb
                    s.leaderboardCacheMu.Unlock()
                }
            case <-ctx.Done():
                return
            }
        }
    }()
}

func (s *service) GetLeaderboard(ctx context.Context) ([]domain.LeaderboardEntry, error) {
    s.leaderboardCacheMu.RLock()
    defer s.leaderboardCacheMu.RUnlock()
    
    // Always return cached (stale-while-revalidate)
    return s.leaderboardCache, nil
}
```

**Expected Impact:**
- **Freshness:** Max 30 seconds stale (acceptable for leaderboards)
- **Latency:** 50ms → <1ms
- **DB load:** Constant (1 query/30s) vs per-request

---

### 4. System Stats Caching (LOW PRIORITY)

**Problem:** System-wide stats computed on every request

**Similar to leaderboards:** Background refresh strategy

```go
// Refresh system stats every 1 minute
func (s *service) refreshSystemStats(ctx context.Context) {
    // Similar pattern to leaderboard caching
}
```

## Implementation Priority

### Phase 1: Inventory Caching (Highest ROI)
- **Why:** Most frequently accessed, easy invalidation points
- **Effort:** 4 hours
- **Impact:** 70-80% DB query reduction for inventory

### Phase 2: Progression Tree Caching
- **Why:** Queried on every progression endpoint, rarely changes
- **Effort:** 3 hours
- **Impact:** ~99% query reduction for tree

### Phase 3: Leaderboard Background Refresh
- **Why:** Expensive computation, acceptable staleness
- **Effort:** 2 hours
- **Impact:** Constant DB load vs per-request

### Phase 4: System Stats
- **Why:** Similar to leaderboards
- **Effort:** 1 hour
- **Impact:** Reduced per-request computation

**Total Estimate:** 10 hours across 4 sprints

## Cache Configuration Matrix

| Cache Type | Size | TTL | Invalidation | Priority |
|------------|------|-----|--------------|----------|
| User | 1000 | 5min | On update | ✅ Done |
| Item Metadata | Unlimited | Forever | On restart | ✅ Done |
| Inventory | 500 | 2min | On mutation | 🔴 High |
| Progression Tree | 1 | 10min | Manual | 🟡 Medium |
| Leaderboard | 1 | 30s | Background | 🟢 Low |
| System Stats | 1 | 1min | Background | 🟢 Low |

## Success Criteria

**After Phase 1 (Inventory):**
- ✅ 70%+ cache hit rate for `GetInventory`
- ✅ P95 latency <2ms (down from ~10ms)
- ✅ No stale inventory bugs

**After Phase 2 (Tree):**
- ✅ 99%+ cache hit rate for tree queries
- ✅ Progression endpoints <5ms P95

**After Phase 3-4:**
- ✅ Leaderboard/stats always <1ms
- ✅ Acceptable freshness (30s/1min)

## Monitoring

Add to Grafana dashboard:

**Panels:**
1. Cache hit rates (all caches)
2. DB query rate by type
3. Cache memory usage
4. P95 latency by endpoint

**Alerts:**
- Inventory cache hit rate <50%
- Memory usage >500MB for caches

## Risks & Mitigation

**Risk:** Stale inventory after mutations  
**Mitigation:** Write-through invalidation at all mutation points

**Risk:** Memory usage growth  
**Mitigation:** LRU eviction, size limits, monitoring

**Risk:** Cache inconsistency across instances  
**Mitigation:** Accept (single instance deployment), or use Redis if scaling

## Future: Redis for Multi-Instance

If scaling horizontally:

1. Replace all in-memory caches with Redis
2. Use Redis pub/sub for invalidation across instances
3. Centralized cache = no inconsistency

**When**: Running >2 instances

## References

- [User Cache Implementation](file:///home/osse1/projects/BrandishBot_Go/internal/user/cache.go)
- [Item Cache (Existing)](file:///home/osse1/projects/BrandishBot_Go/internal/user/service_helpers.go#L9-L29)
- [Benchmarking Journal](file:///home/osse1/projects/BrandishBot_Go/docs/benchmarking/journal.md)
- [User Cache Enhancements](file:///home/osse1/projects/BrandishBot_Go/docs/issues/user_cache_enhancements.md)
