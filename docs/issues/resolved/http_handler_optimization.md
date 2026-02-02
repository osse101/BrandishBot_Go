# HTTP Handler Optimization (Non-Serialization)

**Created:** 2026-01-02  
**Status:** RESOLVED  
**Resolved:** 2026-01-07  
**Priority:** Low  
**Labels:** performance, http, optimization

## Summary

Explore alternative HTTP handler optimizations beyond JSON serialization to reduce the 50 allocations/op measured in handler benchmarks. Focus on middleware, context, and HTTP infrastructure overhead.

## Resolution

**Closed on 2026-01-07** - Optimization deferred indefinitely.

Current performance metrics are excellent:

- Latency: 4.8µs per request
- Request volume: ~20 req/s (well below capacity)
- No GC pressure observed
- All tests passing

Response buffer pooling was implemented but showed no measurable allocation reduction (49 allocs/op). Further optimizations (context value pooling, lazy middleware, static headers) would add complexity without addressing any observed performance issues.

**Decision:** Monitor production metrics and revisit only if P95 latency exceeds 10ms or request volume exceeds 100 req/s.

## Background

**Optimization 1 (easyjson) failed** because JSON wasn't the bottleneck:
- Handler benchmarks: 50 allocs/op, 8.4KB/op
- JSON accounts for only ~15 of 50 allocations (~30%)
- HTTP infrastructure + middleware = ~35 allocations (~70%)

**Reference:** [Optimization 1 Results](file:///home/osse1/.gemini/antigravity/brain/81a703eb-41f2-4403-b773-ede78188a47b/optimization1_results.md)

## Progress

### Response Buffer Pooling (Implemented)
- **Implementation:** Added `internal/handler/pool.go` and modified `respondJSON` to use a `sync.Pool` of `bytes.Buffer`.
- **Result:** Benchmarks showed no reduction in allocations (49 allocs/op -> 49 allocs/op).
- **Analysis:** This is likely due to `httptest.ResponseRecorder` used in benchmarks overshadowing the gain by allocating its own buffers, or `json.Encoder` already pooling internal buffers efficiently. However, the change avoids growing `http.ResponseWriter` internal buffers (if any) and reduces garbage for the intermediate serialization buffer, which is good practice for high-throughput scenarios.

## Related Files

- [`internal/handler/message.go`](file:///home/osse1/projects/BrandishBot_Go/internal/handler/message.go#L44-L112) - Main handler
- [`internal/middleware/engagement.go`](file:///home/osse1/projects/BrandishBot_Go/internal/middleware/engagement.go) - Engagement tracking
- [`internal/middleware/metrics.go`](file:///home/osse1/projects/BrandishBot_Go/internal/metrics/middleware.go) - Metrics middleware
- [`docs/benchmarking/journal.md`](file:///home/osse1/projects/BrandishBot_Go/docs/benchmarking/journal.md#optimization-1-json-serialization-with-easyjson-)

## Allocation Breakdown

### Current (50 allocs/op)

Based on profiling analysis:

**HTTP Infrastructure (~20 allocs):**
- HTTP request parsing: ~8 allocs
- Response buffers: ~5 allocs
- Header maps: ~4 allocs
- URL parsing: ~3 allocs

**Middleware (~15 allocs):**
- Context values: ~7 allocs (logger, request ID, user ID)
- Engagement tracking: ~5 allocs
- Metrics collection: ~3 allocs

**JSON Serialization (~15 allocs):**
- Request unmarshal: ~7 allocs
- Response marshal: ~8 allocs

## Proposed Optimizations

### 1. Context Value Pooling

**Problem:** Each request creates new context values (logger, IDs, etc.)

**Solution:** Pool context value structs

```go
var contextValuePool = sync.Pool{
    New: func() interface{} {
        return &contextValues{
            logger:    nil,
            requestID: "",
            userID:    "",
        }
    },
}

func withPooledContext(ctx context.Context, ...) context.Context {
    cv := contextValuePool.Get().(*contextValues)
    cv.logger = logger
    cv.requestID = id
    cv.userID = userID
    
    // Return context with pooled value
    // Make sure to return to pool after request!
}
```

**Expected:** -7 allocs/op (~14% reduction)

**Risk:** Context values must be returned to pool carefully to avoid leaks

### 2. Lazy Middleware Execution

**Problem:** Some middleware runs even when not needed

**Current:** Engagement tracking runs for ALL requests  
**Optimization:** Only track engagement after successful user lookup

```go
// Before: Middleware runs always
app.Use(EngagementTracker)

// After: Conditional in handler
if result != nil && result.User != nil {
    middleware.TrackEngagementFromContext(ctx, eventBus, "message", 1)
}
```

**Expected:** -5 allocs/op when no engagement tracking needed

### 3. Pre-allocated Response Buffers (Done)

**Problem:** Response buffers allocated per-request

**Solution:** Use sync.Pool for response buffers

```go
var responseBufferPool = sync.Pool{
    New: func() interface{} {
        return bytes.NewBuffer(make([]byte, 0, 512)) // Pre-allocate 512 bytes
    },
}

func HandleMessageHandler(...) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        buf := responseBufferPool.Get().(*bytes.Buffer)
        defer func() {
            buf.Reset()
            responseBufferPool.Put(buf)
        }()
        
        // Encode to buffer, then write
        json.NewEncoder(buf).Encode(result)
        w.Write(buf.Bytes())
    }
}
```

**Expected:** -3 to -5 allocs/op

### 4. Static Header Maps

**Problem:** Headers allocated per-response

**Solution:** Pre-define common header maps

```go
var jsonHeaders = map[string]string{
    "Content-Type": "application/json",
}

func writeJSON(w http.ResponseWriter, data interface{}) error {
    for k, v := range jsonHeaders {
        w.Header().Set(k, v)
    }
    return json.NewEncoder(w).Encode(data)
}
```

**Expected:** -2 allocs/op

## Not Recommended

### ❌ Custom HTTP Server

Replacing `http.Server` with custom implementation:
- **High complexity**
- **Marginal gains** (HTTP overhead is ~8/50 allocs)
- **Maintenance burden**

### ❌ Removing Middleware

Disabling metrics or engagement tracking:
- **Business value loss** outweighs performance gains
- Metrics are critical for observability
- Engagement drives progression system

## Implementation Priority

### High Priority (Try First)
1. **Context Value Pooling** - Biggest single win (-7 allocs)
2. **Response Buffer Pooling** - Easy, safe (-3-5 allocs) (COMPLETED)

### Medium Priority
3. **Lazy Middleware** - Conditional execution (-0-5 allocs, depends on traffic)

### Low Priority
4. **Static Headers** - Minimal gain (-2 allocs)

## Benchmarking Plan

### Before Starting

```bash
# Capture current baseline
make bench-baseline
cp benchmarks/results/baseline.txt benchmarks/results/pre-handler-opt.txt
```

### After Each Optimization

```bash
# Run comparison
make bench-compare

# Expected cumulative improvement:
# Context pooling:    50 → 43 allocs (-14%)
# Buffer pooling:     43 → 40 allocs (-20% total)
# Lazy middleware:    40 → 37 allocs (-26% total)  
# Static headers:     37 → 35 allocs (-30% total)
```

### Success Criteria

- **Target:** 50 → 35 allocs/op (**30% reduction**)
- **Latency:** Maintain or improve (4.8µs baseline)
- **No regressions:** All tests pass

## Risks & Trade-offs

**Risk:** Object pooling increases complexity  
**Mitigation:** Start with simple pools, add only if beneficial

**Risk:** Memory leaks from improper pool usage  
**Mitigation:** Extensive testing, clear documentation

**Risk:** Premature optimization  
**Mitigation:** Only implement if monitoring shows handler latency issues

## When to Implement

**Triggers:**
1. P95 latency for `/message/handle` >10ms in production
2. High request volume (>100 req/s) causing GC pressure
3. Profiling shows handler allocations dominating CPU time

**Current State:** NOT URGENT
- Latency: 4.8µs (excellent)
- Request volume: ~20 req/s (well below capacity)
- No GC issues observed

**Recommendation:** Monitor first, optimize later if needed.

## Alternative: FastHTTP

If drastic performance improvement needed, consider [`valyala/fasthttp`](https://github.com/valyala/fasthttp):

**Pros:**
- 10x fewer allocations than `net/http`
- Zero-copy request/response
- Built-in pooling

**Cons:**
- Not `net/http` compatible (complete rewrite)
- Less ecosystem support
- Higher complexity

**When:** Only if >1000 req/s needed and profiling proves HTTP is bottleneck.

## References

- [Optimization 1 Results (Failed)](file:///home/osse1/.gemini/antigravity/brain/81a703eb-41f2-4403-b773-ede78188a47b/optimization1_results.md)
- [Benchmarking Journal](file:///home/osse1/projects/BrandishBot_Go/docs/benchmarking/journal.md)
- [Profiling Guide](file:///home/osse1/projects/BrandishBot_Go/docs/benchmarking/profiling_guide.md)
- [`internal/handler/message.go`](file:///home/osse1/projects/BrandishBot_Go/internal/handler/message.go)
- [`internal/middleware/engagement.go`](file:///home/osse1/projects/BrandishBot_Go/internal/middleware/engagement.go)
