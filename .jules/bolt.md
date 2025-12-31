## 2024-05-23 - [Manual Mocks Maintenance]
**Learning:** Manual mocks (e.g., `MockRepository`) in Go tests can easily become outdated when interfaces change, leading to "missing method" build errors. This is especially true when multiple test files define their own mocks for the same interface.
**Action:** When updating a repository interface, grep for all implementations (including mocks in `_test.go` files) and update them simultaneously to avoid breaking the build.

## 2025-12-18 - [Map Iteration Non-Determinism]
**Learning:** Iterating over maps in Go is non-deterministic. When using a map to aggregate data that will be appended to a list, you MUST sort the keys first if order stability matters (e.g., for test reproducibility or consistent UI presentation).
**Action:** Always extract map keys to a slice and sort them before iterating to generate ordered lists from map data.

## 2025-12-20 - [Outdated Test Structs]
**Learning:** Changes to core domain structs (like `domain.Item`) often don't propagate to test files immediately if they use struct literals, leading to compilation errors in unrelated packages during full test runs.
**Action:** When running tests for a specific package, be prepared to fix unrelated test compilation errors if shared domain structs have changed.

## 2025-12-21 - [String Concatenation Efficiency]
**Learning:** Inefficient string concatenation (`+=`) in loops was found in `internal/discord/client.go`, leading to O(N^2) complexity.
**Action:** Use `strings.Builder` with `fmt.Fprintf` for constructing strings in loops to ensure O(N) performance and reduce memory allocations.

## 2025-12-22 - [Map Allocation Overhead vs Linear Scan]
**Learning:** Benchmarking revealed that replacing a linear scan O(N) with a map lookup O(1) for finding items in an inventory is SLOWER for N=1000 due to map allocation overhead (~43µs vs ~17µs).
**Action:** Do not blindly replace O(N) loops with maps for "small" datasets (N < 5000) without benchmarking, especially if the map must be built from scratch for every operation.

## 2025-12-22 - [Regex Optimization]
**Learning:** Iteratively running multiple  calls (one per rule) is significantly slower than compiling a single regex with alternation `(p1|p2|...)` and using a hash map for rule lookup.
**Action:** When matching against many static keywords/patterns, compile them into a single optimized regex to achieve O(1) complexity relative to the number of rules.

## 2025-12-22 - [Map Allocation Overhead vs Linear Scan]
**Learning:** Benchmarking revealed that replacing a linear scan O(N) with a map lookup O(1) for finding items in an inventory is SLOWER for N=1000 due to map allocation overhead (~43µs vs ~17µs).
**Action:** Do not blindly replace O(N) loops with maps for "small" datasets (N < 5000) without benchmarking, especially if the map must be built from scratch for every operation.

## 2025-12-22 - [Regex Optimization]
**Learning:** Iteratively running multiple  calls (one per rule) is significantly slower than compiling a single regex with alternation `(p1|p2|...)` and using a hash map for rule lookup.
**Action:** When matching against many static keywords/patterns, compile them into a single optimized regex to achieve O(1) complexity relative to the number of rules.

## 2025-12-22 - [Map Allocation Overhead vs Linear Scan]
**Learning:** Benchmarking revealed that replacing a linear scan O(N) with a map lookup O(1) for finding items in an inventory is SLOWER for N=1000 due to map allocation overhead (~43µs vs ~17µs).
**Action:** Do not blindly replace O(N) loops with maps for "small" datasets (N < 5000) without benchmarking, especially if the map must be built from scratch for every operation.

## 2025-12-22 - [Regex Optimization]
**Learning:** Iteratively running multiple `regexp.MatchString` calls (one per rule) is significantly slower than compiling a single regex with alternation `(p1|p2|...)` and using a hash map for rule lookup.
**Action:** When matching against many static keywords/patterns, compile them into a single optimized regex to achieve O(1) complexity relative to the number of rules.
