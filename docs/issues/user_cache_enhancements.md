# User Cache Enhancements

**Created:** 2026-01-02  
**Status:** Proposed  
**Priority:** Medium  
**Labels:** performance, caching, enhancement

## Summary

Enhance the recently implemented user lookup cache with instrumentation, explicit invalidation, and production monitoring to maximize effectiveness and prevent stale data issues.

## Background

User lookup caching was implemented in Optimization 3 ([`internal/user/cache.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/cache.go)) with:
- LRU cache, 1000 user capacity
- 5-minute TTL
- Cache-first lookup in [`getUserOrRegister()`](file:///home/osse1/projects/BrandishBot_Go/internal/user/service.go#L1024-L1063)

**Expected Impact:** 60-80% reduction in DB queries for user lookups

**Current Limitation:** No metrics visibility or explicit invalidation on updates.

## Related Files

- [`internal/user/cache.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/cache.go) - Cache implementation
- [`internal/user/service.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/service.go#L100-L153) - Service integration
- [`docs/benchmarking/journal.md`](file:///home/osse1/projects/BrandishBot_Go/docs/benchmarking/journal.md) - Optimization history

## Proposed Enhancements

### 1. Cache Instrumentation

Add metrics to track cache effectiveness:

```go
type CacheStats struct {
    Hits     atomic.Int64
    Misses   atomic.Int64
    Evictions atomic.Int64
    Size     func() int
}

func (s *service) GetCacheStats() CacheStats {
    return CacheStats{
        Hits:   s.userCacheHits,
        Misses: s.userCacheMisses,
        Size:   s.userCache.Len,
    }
}
```

**Metrics to expose:**
- `user_cache_hit_total` (counter)
- `user_cache_miss_total` (counter)
- `user_cache_hit_rate` (gauge, computed)
- `user_cache_size` (gauge)

### 2. Explicit Cache Invalidation

Invalidate cache on user updates to prevent stale data:

**Update Points:**
- [`RegisterUser()`](file:///home/osse1/projects/BrandishBot_Go/internal/user/service.go#L167-L190) - Auto-register new user
- [`UpdateUser()`](file:///home/osse1/projects/BrandishBot_Go/internal/user/service.go#L194-L202) - Profile updates  
- [`MergeUsers()`](file:///home/osse1/projects/BrandishBot_Go/internal/user/linking.go#L11-L68) - Account linking

```go
func (s *service) RegisterUser(ctx context.Context, user domain.User) (domain.User, error) {
    registered, err := s.repo.UpsertUser(ctx, &user)
    if err != nil {
        return domain.User{}, err
    }
    
    // Cache the newly registered user
    platform, platformID := getPlatformFromUser(user)
    s.userCache.Set(platform, platformID, registered)
    
    return *registered, nil
}

func (s *service) UpdateUser(ctx context.Context, user domain.User) error {
    if err := s.repo.UpdateUser(ctx, user); err != nil {
        return err
    }
    
    // Invalidate cache to force refresh on next lookup
    platform, platformID := getPlatformFromUser(user)
    s.userCache.Invalidate(platform, platformID)
    
    return nil
}
```

### 3. Production Monitoring Dashboard

Create Grafana dashboard tracking:

**Panels:**
1. Cache hit rate (target: >60%)
2. Cache size over time
3. DB query reduction (compare to pre-cache baseline)
4. P95/P99 latency for `/message/handle`

**Alerts:**
- Cache hit rate <40% (investigate cache size/TTL)
- Anomalous cache miss spike (possible bug)

### 4. Configuration Tuning

Make cache parameters configurable via environment variables:

```go
type CacheConfig struct {
    Size int           // Default: 1000
    TTL  time.Duration // Default: 5min
}

func newUserCacheFromConfig(cfg CacheConfig) *userCache {
    return newUserCache(cfg.Size, cfg.TTL)
}
```

**Environment Variables:**
- `USER_CACHE_SIZE` (default: 1000)
- `USER_CACHE_TTL` (default: 5m)

Allows production tuning without code changes.

## Implementation Plan

1. **Phase 1: Instrumentation** (1 hour)
   - Add cache stats struct
   - Increment hit/miss counters in `Get()`
   - Expose metrics endpoint

2. **Phase 2: Explicit Invalidation** (2 hours)
   - Add invalidation to `RegisterUser()`, `UpdateUser()`, `MergeUsers()`
   - Add helper: `getPlatformFromUser()`
   - Write unit tests for cache invalidation

3. **Phase 3: Monitoring** (2 hours)
   - Create Grafana dashboard JSON
   - Set up alerts
   - Document in runbook

4. **Phase 4: Configuration** (1 hour)
   - Add environment variable parsing
   - Update `.env.example`
   - Document in deployment guide

**Total Estimate:** 6 hours

## Success Criteria

- ✅ Cache hit rate >60% in production after 1 hour
- ✅ DB queries for `GetUserByPlatformID` reduced by 60-80%
- ✅ P95 latency improvement of 20-30% for `/message/handle`
- ✅ No stale user data issues reported

## Risks & Mitigations

**Risk:** TTL too short → Low cache hit rate  
**Mitigation:** Monitor hit rate, adjust TTL via env var

**Risk:** Cache size too small → Thrashing  
**Mitigation:** Monitor evictions, increase size if needed

**Risk:** Stale data after updates  
**Mitigation:** Explicit invalidation in Phase 2

## Future Considerations

### Multi-Instance Cache (Redis)

If scaling horizontally (multiple app instances):

1. Replace in-memory cache with Redis
2. Use pub/sub for cross-instance invalidation
3. Centralized cache = consistent hit rates across instances

**When to implement:** When running >2 instances

### Cache Warming

Pre-populate cache on startup with most active users:

```sql
SELECT platform, platform_id, username
FROM users
WHERE last_active > NOW() - INTERVAL '1 hour'
ORDER BY message_count DESC
LIMIT 1000
```

**Benefit:** Immediate high cache hit rate on deployment

## References

- [Optimization 3 Results](file:///home/osse1/.gemini/antigravity/brain/81a703eb-41f2-4403-b773-ede78188a47b/optimization3_results.md)
- [Benchmarking Journal](file:///home/osse1/projects/BrandishBot_Go/docs/benchmarking/journal.md#optimization-3-response-caching-for-user-lookups)
- [`internal/user/cache.go`](file:///home/osse1/projects/BrandishBot_Go/internal/user/cache.go)
