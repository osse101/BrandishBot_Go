package leaktest

import (
	"runtime"
	"testing"
	"time"
)

// GoroutineChecker helps detect goroutine leaks
type GoroutineChecker struct {
	before int
	t      testing.TB
}

// NewGoroutineChecker creates a new checker and records the current goroutine count
func NewGoroutineChecker(t testing.TB) *GoroutineChecker {
	t.Helper()

	// Allow time for background goroutines to stabilize
	runtime.Gosched()
	time.Sleep(10 * time.Millisecond)

	return &GoroutineChecker{
		before: runtime.NumGoroutine(),
		t:      t,
	}
}

// Check verifies that goroutine count hasn't increased significantly
func (g *GoroutineChecker) Check(tolerance int) {
	g.t.Helper()

	// Allow time for goroutines to finish
	runtime.Gosched()
	time.Sleep(50 * time.Millisecond)
	runtime.GC()
	time.Sleep(50 * time.Millisecond)

	after := runtime.NumGoroutine()
	leaked := after - g.before

	if leaked > tolerance {
		g.t.Errorf("Potential goroutine leak: before=%d, after=%d, leaked=%d (tolerance=%d)",
			g.before, after, leaked, tolerance)
	}
}

// MemoryChecker helps detect memory leaks
type MemoryChecker struct {
	before runtime.MemStats
	t      testing.TB
}

// NewMemoryChecker creates a new checker and records current memory stats
func NewMemoryChecker(t testing.TB) *MemoryChecker {
	t.Helper()

	runtime.GC()
	time.Sleep(10 * time.Millisecond)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &MemoryChecker{
		before: m,
		t:      t,
	}
}

// Check verifies memory hasn't grown beyond threshold
func (m *MemoryChecker) Check(maxGrowthMB float64) {
	m.t.Helper()

	runtime.GC()
	time.Sleep(50 * time.Millisecond)

	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	beforeMB := float64(m.before.Alloc) / 1024 / 1024
	afterMB := float64(after.Alloc) / 1024 / 1024
	growthMB := afterMB - beforeMB

	if growthMB > maxGrowthMB {
		m.t.Errorf("Potential memory leak: before=%.2fMB, after=%.2fMB, growth=%.2fMB (max=%.2fMB)",
			beforeMB, afterMB, growthMB, maxGrowthMB)
	}
}

// CheckNoGoroutineLeak is a convenience function for simple leak checks
func CheckNoGoroutineLeak(t *testing.T, fn func()) {
	t.Helper()

	checker := NewGoroutineChecker(t)
	fn()
	checker.Check(0) // No tolerance for simple cases
}

// CheckNoMemoryLeak is a convenience function for memory leak checks
func CheckNoMemoryLeak(t *testing.T, maxGrowthMB float64, fn func()) {
	t.Helper()

	checker := NewMemoryChecker(t)
	fn()
	checker.Check(maxGrowthMB)
}

// WaitForGoroutines waits for goroutines to finish or times out
func WaitForGoroutines(t *testing.T, target int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		runtime.Gosched()
		if runtime.NumGoroutine() <= target {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Errorf("Timeout waiting for goroutines to complete: current=%d, target=%d",
		runtime.NumGoroutine(), target)
}
