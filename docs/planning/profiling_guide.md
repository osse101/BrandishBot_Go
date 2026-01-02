# Profiling Guide

This guide explains how to profile BrandishBot_Go to identify and optimize performance bottlenecks.

## Quick Start

```bash
# Run existing benchmarks
make bench-hot

# Profile hot paths
make bench-profile

# View CPU profile in browser
go tool pprof -http=:8080 benchmarks/profiles/cpu.prof
```

## Understanding the Tools

### Benchmarks vs Profiling

| Tool | Purpose | When to Use |
|------|---------|-------------|
| **Benchmarks** | Measure performance quantitatively | Compare before/after changes, track trends |
| **CPU Profiling** | Identify where CPU time is spent | Find slow functions/algorithms |
| **Memory Profiling** | Find allocations and memory leaks | Reduce GC pressure, optimize memory |

## Benchmarking Workflow

### 1. Run Hot Path Benchmarks

```bash
make bench-hot
```

**Output:**
```
BenchmarkHandler_HandleMessage-8     5000    250000 ns/op    4096 B/op    45 allocs/op
BenchmarkService_HandleIncomingMsg-8 10000   150000 ns/op    2048 B/op    25 allocs/op
```

**What to look for:**
- `ns/op` - nanoseconds per operation (lower is better)
- `B/op` - bytes allocated per operation (lower is better)
- `allocs/op` - number of allocations (lower is better)

### 2. Set a Baseline

Before making changes, establish a baseline:

```bash
make bench-baseline
```

### 3. Make Changes & Compare

After optimization:

```bash
make bench-compare
```

**Example output (if benchstat installed):**
```
name                         old time/op  new time/op  delta
Handler_HandleMessage-8      250µs ± 2%   180µs ± 3%  -28.00%
Service_HandleIncomingMsg-8  150µs ± 1%   120µs ± 2%  -20.00%

name                         old alloc/op new alloc/op delta
Handler_HandleMessage-8      4.00kB ± 0%  2.50kB ± 0%  -37.50%
```

✅ **Green** = improvement | ❌ **Red** = regression

## CPU Profiling

### Step 1: Generate Profile

```bash
make bench-profile
```

This runs benchmarks with CPU profiling enabled and saves to `benchmarks/profiles/cpu.prof`.

### Step 2: Analyze with pprof

```bash
go tool pprof -http=:8080 benchmarks/profiles/cpu.prof
```

This opens an interactive web UI at `http://localhost:8080`.

### Step 3: Interpret the Flame Graph

**Flame Graph View:**
- **Width** = % of total CPU time
- **Height** = call stack depth
- **Hover** = function name and stats

**Look for:**
- **Wide flames** = hot functions (spend most time here)
- **Tall stacks** = deep call chains
- **Flat boxes** = functions doing actual work

**Example findings:**
```
json.Unmarshal           ████████░░ 40%  <-- Hot! Consider caching
database.Query           ████░░░░░░ 20%
inventory.FindSlot       █░░░░░░░░░  5%  <-- Already optimized (see utils/inventory_test.go)
```

### Step 4: Top Functions View

In pprof UI, click **"Top"** to see functions ordered by CPU time:

```
flat  flat%  sum%  cum   cum%   Function
200ms 40.00% 40.00% 250ms 50.00% encoding/json.Unmarshal
100ms 20.00% 60.00% 150ms 30.00% database/sql.(*DB).Query
 50ms 10.00% 70.00%  50ms 10.00% runtime.mallocgc
```

- **flat** = time spent in THIS function (excluding calls)
- **cum** = time spent in this function AND its callees

**Focus on high `cum` values first.**

## Memory Profiling

### Step 1: Generate Memory Profile

```bash
# Memory profile included in:
make bench-profile
```

### Step 2: Analyze Allocations

```bash
go tool pprof -http=:8080 benchmarks/profiles/mem.prof
```

### Step 3: Find Allocation Hot Spots

In pprof UI, use **"alloc_space"** view:

```
      flat  flat%   sum%        cum   cum%
    2.50GB 45.45% 45.45%     3.00GB 54.55%  user.(*service).GetInventory
    1.00GB 18.18% 63.64%     1.20GB 21.82%  handler.HandleMessageHandler
```

**Red flags:**
- Large allocations in tight loops
- JSON marshal/unmarshal in hot paths
- String concatenation (use `strings.Builder` instead)

### Common Optimizations

1. **Pre-allocate slices**
   ```go
   // Bad
   items := []Item{}
   for _, id := range ids {
       items = append(items, fetchItem(id))
   }
   
   // Good
   items := make([]Item, 0, len(ids))
   for _, id := range ids {
       items = append(items, fetchItem(id))
   }
   ```

2. **Reuse objects**
   ```go
   // Use sync.Pool for frequently allocated objects
   var bufferPool = sync.Pool{
       New: func() interface{} {
           return new(bytes.Buffer)
       },
   }
   ```

3. **Avoid unnecessary allocations**
   ```go
// Bad (allocates on every call)
   func formatUsername(username string) string {
       return "User: " + username
   }
   
   // Good (single allocation with builder)
   func formatUsername(username string) string {
       var b strings.Builder
       b.WriteString("User: ")
       b.WriteString(username)
       return b.String()
   }
   ```

## Production Profiling

### Continuous Profiling Endpoints

The app exposes pprof endpoints at `/debug/pprof/`:

```bash
# CPU profile (30 seconds)
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof

# Heap profile
curl http://localhost:8080/debug/pprof/heap > heap.prof

# Goroutine profile
curl http://localhost:8080/debug/pprof/goroutine > goroutine.prof
```

**Analyze:**
```bash
go tool pprof -http=:8080 cpu.prof
```

### Staging Profiling

```bash
# Profile staging during load
curl http://staging:8080/debug/pprof/profile?seconds=60 > staging_cpu.prof

# Compare staging vs local
go tool pprof -http=:8080 -diff_base=local_cpu.prof staging_cpu.prof
```

## Profiling Checklist

Before optimizing, ask:

- [ ] **Is this actually slow?** Measure first, optimize second
- [ ] **Is this a hot path?** Focus on code that runs frequently
- [ ] **What's the impact?** 10% improvement on cold code < 1% on hot code
- [ ] **Are there benchmarks?** Can't prove improvement without them

## Tips & Best Practices

### ✅ Do
- **Profile before optimizing** - "Premature optimization is the root of all evil"
- **Focus on hot paths** - 80% of time spent in 20% of code
- **Measure the impact** - Use benchstat to compare results
- **Keep baselines** - Track performance over time

### ❌ Don't
- **Don't guess** - Profile to find actual bottlenecks
- **Don't micro-optimize** - Big wins come from algorithm/architecture changes
- **Don't optimize cold code** - Focus on frequently-executed paths
- **Don't forget readability** - Maintainability > marginal gains

## Common Bottlenecks in BrandishBot

Based on architecture analysis:

1. **Database Queries**
   - **Symptom:** High latency on `/message/handle`
   - **Fix:** Add caching (already done for items), optimize JSONB queries

2. **JSON Serialization**
   - **Symptom:** High CPU in `encoding/json`
   - **Fix:** Consider `easyjson` or Protocol Buffers for hot paths

3. **Inventory Operations**
   - **Symptom:** Slow `GetInventory` with many items
   - **Fix:** Already optimized! See `internal/utils/inventory_test.go` benchmarks

4. **Event Bus Publishing**
   - **Symptom:** High latency after request processing
   - **Fix:** Make event publishing async (already using worker pool)

## Useful Commands Reference

```bash
# Benchmarks
make bench                 # Run all benchmarks
make bench-hot             # Run hot path benchmarks
make bench-baseline        # Set baseline
make bench-compare         # Compare to baseline
make bench-save            # Save timestamped results

# Profiling
make bench-profile         # Generate CPU + memory profiles
go tool pprof -http=:8080 benchmarks/profiles/cpu.prof    # View CPU profile
go tool pprof -http=:8080 benchmarks/profiles/mem.prof    # View memory profile

# Production
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof -http=:8080 cpu.prof
```

## Further Reading

- [Go Profiling Guide](https://go.dev/blog/pprof)
- [Benchmarking Best Practices](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)
- [pprof Tutorial](https://jvns.ca/blog/2017/09/24/profiling-go-with-pprof/)
- [Existing Inventory Benchmarks](file:///home/osse1/projects/BrandishBot_Go/internal/utils/inventory_test.go) - Great example to follow!
