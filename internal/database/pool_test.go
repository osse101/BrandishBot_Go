package database

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/osse101/BrandishBot_Go/internal/testing/leaktest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPool_ConnectionsReleased verifies connections are returned to the pool
func TestPool_ConnectionsReleased(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	connString := getTestDBConnString(t)
	pool, err := NewPool(connString, 5, 1*time.Minute, 5*time.Minute)
	require.NoError(t, err)
	defer pool.Close()

	ctx := context.Background()

	// Acquire and release connections multiple times
	for i := 0; i < 10; i++ {
		conn, err := pool.Acquire(ctx)
		require.NoError(t, err, "Failed to acquire connection on iteration %d", i)
		
		// Do something with connection
		var result int
		err = conn.QueryRow(ctx, "SELECT 1").Scan(&result)
		assert.NoError(t, err)
		assert.Equal(t, 1, result)
		
		conn.Release()
	}

	// All connections should be released back to pool
	stats := pool.Stat()
	assert.Equal(t, int32(0), stats.AcquiredConns(), "All connections should be released")
}

// TestPool_MaxConnsEnforced verifies pool respects MaxConns limit
func TestPool_MaxConnsEnforced(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	connString := getTestDBConnString(t)
	maxConns := 3
	pool, err := NewPool(connString, maxConns, 1*time.Minute, 5*time.Minute)
	require.NoError(t, err)
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Acquire max connections
	conns := make([]*pgxpool.Conn, maxConns)
	for i := 0; i < maxConns; i++ {
		conn, err := pool.Acquire(ctx)
		require.NoError(t, err)
		conns[i] = conn
	}

	stats := pool.Stat()
	assert.Equal(t, int32(maxConns), stats.AcquiredConns())

	// Try to acquire one more - should block/timeout
	acquireDone := make(chan error, 1)
	go func() {
		shortCtx, shortCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer shortCancel()
		_, err := pool.Acquire(shortCtx)
		acquireDone <- err
	}()

	select {
	case err := <-acquireDone:
		assert.Error(t, err, "Should fail to acquire when pool is exhausted")
	case <-time.After(500 * time.Millisecond):
		t.Error("Acquire should have timed out")
	}

	// Release one connection
	conns[0].Release()

	// Now acquisition should succeed
	conn, err := pool.Acquire(ctx)
	assert.NoError(t, err)
	if conn != nil {
		conn.Release()
	}

	// Release remaining
	for i := 1; i < maxConns; i++ {
		conns[i].Release()
	}
}

// TestPool_NoConnectionLeakOnError verifies connections are released even on errors
func TestPool_NoConnectionLeakOnError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	connString := getTestDBConnString(t)
	pool, err := NewPool(connString, 5, 1*time.Minute, 5*time.Minute)
	require.NoError(t, err)
	defer pool.Close()

	ctx := context.Background()
	initialStats := pool.Stat()

	// Execute invalid queries that will error
	for i := 0; i < 5; i++ {
		conn, err := pool.Acquire(ctx)
		require.NoError(t, err)

		// Invalid SQL - should error
		_, err = conn.Query(ctx, "SELECT * FROM nonexistent_table_xyz")
		assert.Error(t, err, "Query should fail")

		conn.Release()
	}

	// Verify no connections leaked
	stats := pool.Stat()
	assert.Equal(t, initialStats.AcquiredConns(), stats.AcquiredConns(),
		"No connections should be leaked after errors")
}

// TestPool_ConcurrentAccess tests thread safety
func TestPool_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	connString := getTestDBConnString(t)
	pool, err := NewPool(connString, 10, 1*time.Minute, 5*time.Minute)
	require.NoError(t, err)
	defer pool.Close()

	checker := leaktest.NewGoroutineChecker(t)

	var wg sync.WaitGroup
	concurrency := 20

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx := context.Background()
			conn, err := pool.Acquire(ctx)
			if err != nil {
				t.Errorf("Worker %d failed to acquire connection: %v", id, err)
				return
			}
			defer conn.Release()

			// Do some work
			var result int
			err = conn.QueryRow(ctx, "SELECT $1::int", id).Scan(&result)
			if err != nil {
				t.Errorf("Worker %d query failed: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify no connections leaked
	stats := pool.Stat()
	assert.Equal(t, int32(0), stats.AcquiredConns(), "All connections should be released")

	// Check for goroutine leaks
	checker.Check(2) // Allow small tolerance for background workers
}

// getTestDBConnString returns test database connection string
// Skips test if required env vars not set
func getTestDBConnString(t *testing.T) string {
	t.Helper()

	// Use environment variables or skip
	dbUser := getEnvOrSkip(t, "DB_USER", "postgres")
	dbPassword := getEnvOrSkip(t, "DB_PASSWORD", "postgres")
	dbHost := getEnvOrSkip(t, "DB_HOST", "localhost")
	dbPort := getEnvOrSkip(t, "DB_PORT", "5432")
	dbName := getEnvOrSkip(t, "DB_NAME", "brandishbot")

	return "postgres://" + dbUser + ":" + dbPassword + "@" + dbHost + ":" + dbPort + "/" + dbName + "?sslmode=disable"
}

func getEnvOrSkip(t *testing.T, key, defaultValue string) string {
	t.Helper()
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
