## 2024-05-23 - [Manual Mocks Maintenance]
**Learning:** Manual mocks (e.g., `MockRepository`) in Go tests can easily become outdated when interfaces change, leading to "missing method" build errors. This is especially true when multiple test files define their own mocks for the same interface.
**Action:** When updating a repository interface, grep for all implementations (including mocks in `_test.go` files) and update them simultaneously to avoid breaking the build.

## 2025-12-18 - [Map Iteration Non-Determinism]
**Learning:** Iterating over maps in Go is non-deterministic. When using a map to aggregate data that will be appended to a list, you MUST sort the keys first if order stability matters (e.g., for test reproducibility or consistent UI presentation).
**Action:** Always extract map keys to a slice and sort them before iterating to generate ordered lists from map data.
