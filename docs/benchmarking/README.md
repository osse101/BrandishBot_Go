# Benchmarking Documentation

This directory contains all benchmarking and performance optimization documentation for BrandishBot_Go.

## Quick Start

```bash
# Run hot path benchmarks
make bench-hot

# Set baseline
make bench-baseline

# Compare to baseline after changes
make bench-compare

# Generate profiles
make bench-profile
```

## Documentation

### [📔 Journal](./journal.md) **← START HERE**
Real-world optimization attempts, learnings, and results. Read this to understand what works and what doesn't.

### [📋 Benchmarking Strategy](./benchmarking_strategy.md)
Comprehensive benchmarking infrastructure, methodology, and guidelines for the project.

### [📊 Profiling Guide](./profiling_guide.md)
Practical guide to using pprof and analyzing performance bottlenecks.

## Key Results

### ✅ Successful Optimizations
- **Transaction Batching** (2026-01-02): 2.7-12x improvement for bulk operations
- **Inventory Linear Scan** (pre-existing): 3.2x faster than map lookup for small inventories

### ❌ Failed Optimizations
- **easyjson** (2026-01-02): No improvement for small payloads, reverted

## Performance Baselines

| Operation | Latency | Memory | Allocations |
|-----------|---------|--------|-------------|
| Handler (HandleMessage) | 4.8µs | 8.4KB | 50 |
| Service (AddItem single) | 420ns | 680B | 18 |
| Service (AddItems batch10) | 828ns | 1.2KB | 21 |
| Utils (Linear scan) | 5µs | 32KB | 1 |

## Benchmark Files

### Handler Benchmarks
- `internal/handler/message_bench_test.go`
  - `BenchmarkHandler_HandleMessage`
  - `BenchmarkHandler_HandleMessage_ExistingUser`
  - `BenchmarkHandler_HandleMessage_WithMatches`

### Service Benchmarks
- `internal/user/service_bench_test.go`
  - `BenchmarkService_HandleIncomingMessage`
  - `BenchmarkService_HandleIncomingMessage_WithMatches`
  - `BenchmarkService_AddItem`
  - `BenchmarkService_AddItem_NewItem`
  - `BenchmarkService_AddItems_Batch10` ⚡
  - `BenchmarkService_AddItems_Batch25` ⚡
  - `BenchmarkService_AddItem_Individual10`

### Utility Benchmarks
- `internal/utils/inventory_test.go`
  - `BenchmarkAddItemsLinearScan`
  - `BenchmarkAddItemsMapLookup`
  - `BenchmarkAddItemsWithPrebuiltMap`

## CI/CD Integration

Benchmarks run nightly via GitHub Actions:
- `.github/workflows/nightly-benchmark.yml`
- Results stored as artifacts
- Baseline comparison automated

## Tools

Required:
- `go test -bench` (built-in)
- `benchstat` - `go install golang.org/x/perf/cmd/benchstat@latest`

Optional:
- `pprof` - CPU/memory profiling
- `go tool trace` - Execution trace analysis

## Contributing

When adding new benchmarks:
1. Follow naming convention: `Benchmark<Component>_<Operation>`
2. Use `b.ResetTimer()` and `b.ReportAllocs()`
3. Add to appropriate `*_bench_test.go` file
4. Update this README
5. Document learnings in journal.md

---

*For questions or suggestions, update journal.md with your findings!*
