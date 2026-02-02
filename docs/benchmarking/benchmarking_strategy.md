# Benchmarking Strategy for BrandishBot_Go

## Executive Summary

This document outlines a comprehensive benchmarking strategy for BrandishBot_Go to identify performance hotspots, validate optimization decisions, and establish performance baselines. The strategy focuses on the critical execution path: **HTTP Handler → Middleware → Service Layer → Database Operations**, with special attention to `HandleMessageHandler` and the middleware chain.

## Architecture Analysis

### Request Flow

```
HTTP Request
    ↓
[Metrics Middleware] ← Instrument req/response times
    ↓
[Handler Layer] ← Validation, routing
    ↓
[Service Layer] ← Business logic, cache lookups
    ↓
[Repository Layer] ← Database transactions
    ↓
[Database] ← PostgreSQL queries
```

### Identified Hotspots

Based on code analysis, the following areas are performance-critical:

1. **Message Handling Path** (`/message/handle`)
   - `HandleMessageHandler` - Entry point with validation
   - `userService.HandleIncomingMessage` - User lookup/creation
   - `TrackEngagementFromContext` - Event bus publishing
   - Database operations (user lookup, inventory fetch)

2. **Inventory Operations**
   - `AddItem` / `RemoveItem` - Transaction management
   - `GetInventory` - Batch item lookups with caching
   - Inventory slot linear scans vs map lookups (already has benchmarks!)

3. **Database Layer**
   - User platform ID lookups
   - Inventory JSONB operations
   - Item metadata queries (with caching)
   - Transaction commit/rollback overhead

4. **Middleware Chain**
   - Metrics collection overhead
   - Engagement tracking event publishing

## Requirements (User-Specified)

> [!NOTE]
> These requirements guide the benchmarking implementation:

### Performance Goals
- **Primary Focus:** Latency (endpoint-to-response time)
- **Target Throughput:** ~20 requests/second (current expected load)
- **Scope:** Hot path benchmarking (critical endpoints), efficient but lazy benchmarking for others

### Benchmarking Approach
- **Database:** Mock DB for benchmarks (faster, more reproducible)
- **Middleware:** Include middleware overhead (full endpoint-to-response measurement)
- **Individual Components:** Also measure individual steps on the path separately

### Environment
- **Primary:** Dev environment for regular testing
- **Secondary:** Staging for reference measurements
- **Production:** Profiling only (debugging purposes)

### Deliverables
- **Makefile Commands:** Simple, easy-to-use benchmark targets
- **Profiling Guide:** Documentation for the team
- **CI/CD:** Nightly benchmarks against master (non-blocking)
- **No Deployment Gates:** Benchmarks inform but don't block deploys

## Recommended Framework & Tools

### Core: Go's Built-in Testing Package

**Rationale:** Go's `testing.B` is the standard, well-integrated, and requires no external dependencies.

```go
func BenchmarkHandleMessage(b *testing.B) {
    // Setup
    for i := 0; i < b.N; i++ {
        // Code to benchmark
    }
}
```

**Run with:** `go test -bench=. -benchmem -cpuprofile=cpu.prof`

### Supplementary Libraries

1. **[benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)** - Statistical comparison
   ```bash
   benchstat old.txt new.txt
   ```

2. **[pprof](https://pkg.go.dev/net/http/pprof)** - CPU/Memory profiling
   - Already instrumented via `import _ "net/http/pprof"`
   - Access at `/debug/pprof/` when server running

3. **[vegeta](https://github.com/tsenart/vegeta)** - HTTP load testing
   ```bash
   echo "POST http://localhost:8080/message/handle" | vegeta attack -rate=100 -duration=30s | vegeta report
   ```

4. **[k6](https://k6.io/)** - Modern load testing (optional, for integration tests)

### Database Benchmarking

- **testcontainers-go** (already in use!) - Spin up real PostgreSQL for integration benchmarks
- **pgx native benchmarks** - Profile specific queries

## Proposed Benchmarking Structure

### Directory Layout

```
BrandishBot_Go/
├── internal/
│   ├── handler/
│   │   ├── message_test.go         # Unit tests
│   │   └── message_bench_test.go   # NEW: Handler benchmarks
│   ├── user/
│   │   ├── service_test.go
│   │   └── service_bench_test.go   # NEW: Service layer benchmarks
│   ├── database/postgres/
│   │   └── user_bench_test.go      # NEW: Database operation benchmarks
│   └── utils/
│       └── inventory_test.go       # EXISTING: Has benchmarks!
├── benchmarks/                      # NEW: Integration benchmarks
│   ├── e2e_bench_test.go           # Full request lifecycle
│   ├── fixtures/                   # Test data
│   └── README.md                   # How to run benchmarks
└── docs/
    └── planning/
        └── benchmarking_strategy.md # This document
```

## Benchmarking Categories

### 1. Unit Benchmarks (Fast, No I/O)

**Target:** Individual functions with mocked dependencies

**Examples:**
- Validation logic
- Cache hit/miss paths
- Item lookup optimization (already exists in `utils/inventory_test.go`)

```go
// Example: Benchmark cache performance
func BenchmarkGetItemByNameCached(b *testing.B) {
    s := setupServiceWithCache()
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        s.getItemByNameCached(ctx, "lootbox0")
    }
}
```

### 2. Integration Benchmarks (Realistic, With DB)

**Target:** Full service operations with real PostgreSQL

**Examples:**
- `HandleIncomingMessage` with user creation
- `AddItem` transaction performance
- `GetInventory` with large inventories

```go
func BenchmarkHandleIncomingMessage_NewUser(b *testing.B) {
    db := setupTestDB(b) // testcontainers
    service := user.NewService(...)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        service.HandleIncomingMessage(ctx, "twitch", fmt.Sprintf("user_%d", i), "testuser", "hi")
    }
}
```

### 3. HTTP Handler Benchmarks

**Target:** Handler layer including middleware

```go
func BenchmarkHandleMessageHandler(b *testing.B) {
    handler := HandleMessageHandler(mockService, mockProgression, mockBus)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        req := httptest.NewRequest("POST", "/message/handle", body)
        w := httptest.NewRecorder()
        handler.ServeHTTP(w, req)
    }
}
```

### 4. End-to-End Load Tests (Vegeta/k6)

**Target:** Entire application under realistic load

```bash
# Vegeta: Sustained load
vegeta attack -rate=50/s -duration=60s -targets=targets.txt | vegeta report

# Profiling while load testing
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
```

## Key Metrics to Track

### Per-Operation Metrics
- **Latency** (p50, p95, p99)
- **Throughput** (ops/sec)
- **Memory allocations** (`-benchmem`)
- **Bytes allocated per operation**

### System Metrics (During Load Tests)
- **Database connection pool utilization**
- **Goroutine count**
- **Heap allocations**
- **GC pressure**

## Implementation Guidelines

### 1. Benchmark Naming Convention

```go
// Pattern: Benchmark{Component}_{Operation}_{Condition}
BenchmarkHandler_HandleMessage_NewUser
BenchmarkService_AddItem_CacheHit
BenchmarkRepo_GetInventory_Large
```

### 2. Benchmark Best Practices

```go
func BenchmarkExample(b *testing.B) {
    // Setup (not timed)
    ctx := context.Background()
    service := setupService()
    
    // Reset timer to exclude setup
    b.ResetTimer()
    
    // Run benchmark
    for i := 0; i < b.N; i++ {
        b.StopTimer()  // Pause for expensive setup if needed
        input := generateInput(i)
        b.StartTimer()
        
        result, err := service.Operation(ctx, input)
        if err != nil {
            b.Fatal(err)
        }
        
        // Prevent compiler optimization
        _ = result
    }
}
```

### 3. Table-Driven Benchmarks

```go
func BenchmarkInventoryOperations(b *testing.B) {
    cases := []struct {
        name     string
        slotSize int
    }{
        {"Small", 10},
        {"Medium", 100},
        {"Large", 1000},
    }
    
    for _, tc := range cases {
        b.Run(tc.name, func(b *testing.B) {
            // Benchmark with tc.slotSize
        })
    }
}
```

## Profiling Workflow

### 1. Identify Hotspots with CPU Profiling

```bash
# Run benchmarks with CPU profiling
go test -bench=BenchmarkHandleMessage -cpuprofile=cpu.prof ./internal/handler

# Analyze profile
go tool pprof -http=:8080 cpu.prof
```

### 2. Memory Profiling

```bash
# Memory allocations
go test -bench=. -memprofile=mem.prof -benchmem

# Analyze
go tool pprof -http=:8080 mem.prof
```

### 3. Continuous Profiling (Production)

```go
// Already available at /debug/pprof/ endpoints:
// - /debug/pprof/heap
// - /debug/pprof/goroutine
// - /debug/pprof/profile?seconds=30
```

## CI/CD Integration

### Makefile Targets (Simple Commands)

```makefile
.PHONY: bench bench-hot bench-save bench-compare bench-profile

# Run all benchmarks
bench:
	@echo "Running all benchmarks..."
	@go test -bench=. -benchmem -benchtime=2s ./...

# Run only hot path benchmarks (core critical paths)
bench-hot:
	@echo "Running hot path benchmarks..."
	@go test -bench=BenchmarkHandler_HandleMessage -benchmem ./internal/handler
	@go test -bench=BenchmarkService_HandleIncomingMessage -benchmem ./internal/user
	@go test -bench=BenchmarkService_AddItem -benchmem ./internal/user

# Run benchmarks and save timestamped results
bench-save:
	@echo "Running benchmarks and saving results..."
	@mkdir -p benchmarks/results
	@go test -bench=. -benchmem -benchtime=2s ./... | tee benchmarks/results/$(shell date +%Y%m%d-%H%M%S).txt
	@echo "Results saved to benchmarks/results/"

# Compare current benchmarks against baseline
bench-compare:
	@if [ ! -f benchmarks/results/baseline.txt ]; then \
		echo "Error: No baseline found. Run 'make bench-baseline' first."; \
		exit 1; \
	fi
	@echo "Running benchmarks and comparing to baseline..."
	@go test -bench=. -benchmem -benchtime=2s ./... > benchmarks/results/current.txt
	@benchstat benchmarks/results/baseline.txt benchmarks/results/current.txt

# Set current results as baseline
bench-baseline:
	@echo "Setting new baseline..."
	@mkdir -p benchmarks/results
	@go test -bench=. -benchmem -benchtime=2s ./... | tee benchmarks/results/baseline.txt
	@echo "Baseline set: benchmarks/results/baseline.txt"

# Profile hot paths (CPU + memory)
bench-profile:
	@echo "Profiling hot paths..."
	@mkdir -p benchmarks/profiles
	@echo "Running CPU profile..."
	@go test -bench=BenchmarkHandler_HandleMessage -cpuprofile=benchmarks/profiles/cpu.prof ./internal/handler
	@echo "Running memory profile..."
	@go test -bench=BenchmarkHandler_HandleMessage -memprofile=benchmarks/profiles/mem.prof ./internal/handler
	@echo "Profiles saved to benchmarks/profiles/"
	@echo "View with: go tool pprof -http=:8080 benchmarks/profiles/cpu.prof"
```

### GitHub Actions (Nightly Benchmarks)

```yaml
# .github/workflows/nightly-benchmark.yml
name: Nightly Benchmarks

on:
  schedule:
    - cron: '0 2 * * *'  # 2 AM UTC daily
  workflow_dispatch:  # Allow manual trigger

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run benchmarks
        run: make bench-save
      
      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: benchmark-results
          path: benchmarks/results/
      
      - name: Compare with previous
        continue-on-error: true
        run: |
          # Download previous benchmark if exists
          # Compare and post results
          echo "Benchmark comparison goes here"
```

## Success Metrics

### Phase 1: Baseline (First 2 weeks)
- [ ] Benchmarks for all handler endpoints
- [ ] Service layer operation benchmarks
- [ ] Database query benchmarks
- [ ] Document baseline results

### Phase 2: Optimization (Ongoing)
- [ ] Identify operations with >10ms p95 latency
- [ ] Reduce memory allocations by 20%
- [ ] Achieve <5ms p99 for cached operations

### Phase 3: Monitoring (Long-term)
- [ ] Automated benchmark comparison on PRs
- [ ] Performance regression alerts
- [ ] Quarterly performance reviews

## Verification Plan

### 1. Validate Benchmark Infrastructure
- Run existing benchmarks: `go test -bench=. ./internal/utils`
- Verify profiling endpoints: `curl http://localhost:8080/debug/pprof/`

### 2. Create Sample Benchmarks
- Add handler benchmark for `HandleMessageHandler`
- Add service benchmark for `HandleIncomingMessage`
- Run and document results

### 3. Load Testing
- Create vegeta target file for `/message/handle`
- Run 60s load test at 50 req/sec
- Collect CPU profile during load test
- Analyze pprof flame graph

## Next Steps

1. **User reviews this plan** and answers clarification questions
2. **Implement sample benchmarks** for critical paths
3. **Run baseline measurements** and document results
4. **Identify optimization opportunities** from profiles
5. **Establish CI/CD integration** for regression detection
