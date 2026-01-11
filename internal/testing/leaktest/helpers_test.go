package leaktest

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestGoroutineChecker_NoLeak(t *testing.T) {
	checker := NewGoroutineChecker(t)

	// Do nothing - no goroutines leaked

	checker.Check(0)
}

func TestGoroutineChecker_WithTolerance(t *testing.T) {
	checker := NewGoroutineChecker(t)

	// Intentionally leak a small number of goroutines within tolerance
	done := make(chan struct{})
	go func() {
		<-done
	}()

	time.Sleep(20 * time.Millisecond)

	// Check with tolerance of 2 - should pass
	checker.Check(2)

	// Cleanup
	close(done)
}

func TestMemoryChecker_SmallAllocation(t *testing.T) {
	checker := NewMemoryChecker(t)

	// Allocate small amount that should be GC'd
	_ = make([]byte, 1024)

	checker.Check(1.0) // Allow 1MB growth
}

func TestCheckNoGoroutineLeak_Success(t *testing.T) {
	CheckNoGoroutineLeak(t, func() {
		// Simple operation that doesn't leak
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(1 * time.Millisecond)
		}()
		wg.Wait()
	})
}

func TestCheckNoMemoryLeak_Success(t *testing.T) {
	CheckNoMemoryLeak(t, 1.0, func() {
		// Temporary allocation
		data := make([]byte, 1024)
		_ = data
	})
}

func TestWaitForGoroutines_Success(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(5)

	before := runtime.NumGoroutine()

	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
		}()
	}

	wg.Wait()

	// Wait for goroutines to actually terminate
	WaitForGoroutines(t, before, 1*time.Second)
}

// TestGoroutineChecker_Integration demonstrates real-world usage
func TestGoroutineChecker_Integration(t *testing.T) {
	t.Run("goroutines properly cleaned up", func(t *testing.T) {
		checker := NewGoroutineChecker(t)

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				time.Sleep(5 * time.Millisecond)
			}()
		}

		wg.Wait()
		checker.Check(0)
	})
}
