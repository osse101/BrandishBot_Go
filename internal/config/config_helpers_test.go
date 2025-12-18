package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetEnvAsInt tests the getEnvAsInt helper function
func TestGetEnvAsInt(t *testing.T) {
	t.Run("returns default value when env var not set", func(t *testing.T) {
		os.Unsetenv("TEST_INT_VAR")
		result := getEnvAsInt("TEST_INT_VAR", 42)
		assert.Equal(t, 42, result)
	})

	t.Run("parses valid integer from env var", func(t *testing.T) {
		t.Setenv("TEST_INT_VAR", "100")
		result := getEnvAsInt("TEST_INT_VAR", 42)
		assert.Equal(t, 100, result)
	})

	t.Run("returns default for invalid integer", func(t *testing.T) {
		t.Setenv("TEST_INT_VAR", "not-a-number")
		result := getEnvAsInt("TEST_INT_VAR", 42)
		assert.Equal(t, 42, result, "Should return default for invalid integer")
	})

	t.Run("parses negative integers", func(t *testing.T) {
		t.Setenv("TEST_INT_VAR", "-10")
		result := getEnvAsInt("TEST_INT_VAR", 42)
		assert.Equal(t, -10, result)
	})

	t.Run("parses zero", func(t *testing.T) {
		t.Setenv("TEST_INT_VAR", "0")
		result := getEnvAsInt("TEST_INT_VAR", 42)
		assert.Equal(t, 0, result)
	})

	t.Run("returns default for float values", func(t *testing.T) {
		t.Setenv("TEST_INT_VAR", "42.5")
		result := getEnvAsInt("TEST_INT_VAR", 10)
		assert.Equal(t, 10, result, "Should return default for float values")
	})

	t.Run("returns default for empty string", func(t *testing.T) {
		t.Setenv("TEST_INT_VAR", "")
		result := getEnvAsInt("TEST_INT_VAR", 42)
		assert.Equal(t, 42, result)
	})
}

// TestGetEnvAsDuration tests the getEnvAsDuration helper function
func TestGetEnvAsDuration(t *testing.T) {
	t.Run("returns default value when env var not set", func(t *testing.T) {
		os.Unsetenv("TEST_DURATION_VAR")
		result := getEnvAsDuration("TEST_DURATION_VAR", 5*time.Minute)
		assert.Equal(t, 5*time.Minute, result)
	})

	t.Run("parses valid duration from env var", func(t *testing.T) {
		t.Setenv("TEST_DURATION_VAR", "10m")
		result := getEnvAsDuration("TEST_DURATION_VAR", 5*time.Minute)
		assert.Equal(t, 10*time.Minute, result)
	})

	t.Run("parses seconds", func(t *testing.T) {
		t.Setenv("TEST_DURATION_VAR", "30s")
		result := getEnvAsDuration("TEST_DURATION_VAR", 5*time.Minute)
		assert.Equal(t, 30*time.Second, result)
	})

	t.Run("parses hours", func(t *testing.T) {
		t.Setenv("TEST_DURATION_VAR", "2h")
		result := getEnvAsDuration("TEST_DURATION_VAR", 5*time.Minute)
		assert.Equal(t, 2*time.Hour, result)
	})

	t.Run("parses complex duration", func(t *testing.T) {
		t.Setenv("TEST_DURATION_VAR", "1h30m45s")
		result := getEnvAsDuration("TEST_DURATION_VAR", 5*time.Minute)
		expected := 1*time.Hour + 30*time.Minute + 45*time.Second
		assert.Equal(t, expected, result)
	})

	t.Run("returns default for invalid duration", func(t *testing.T) {
		t.Setenv("TEST_DURATION_VAR", "not-a-duration")
		result := getEnvAsDuration("TEST_DURATION_VAR", 5*time.Minute)
		assert.Equal(t, 5*time.Minute, result, "Should return default for invalid duration")
	})

	t.Run("returns default for plain numbers without unit", func(t *testing.T) {
		t.Setenv("TEST_DURATION_VAR", "100")
		result := getEnvAsDuration("TEST_DURATION_VAR", 5*time.Minute)
		assert.Equal(t, 5*time.Minute, result, "Should return default for numbers without unit")
	})

	t.Run("returns default for empty string", func(t *testing.T) {
		t.Setenv("TEST_DURATION_VAR", "")
		result := getEnvAsDuration("TEST_DURATION_VAR", 5*time.Minute)
		assert.Equal(t, 5*time.Minute, result)
	})

	t.Run("parses nanoseconds", func(t *testing.T) {
		t.Setenv("TEST_DURATION_VAR", "500ns")
		result := getEnvAsDuration("TEST_DURATION_VAR", 5*time.Minute)
		assert.Equal(t, 500*time.Nanosecond, result)
	})

	t.Run("parses microseconds", func(t *testing.T) {
		t.Setenv("TEST_DURATION_VAR", "500us")
		result := getEnvAsDuration("TEST_DURATION_VAR", 5*time.Minute)
		assert.Equal(t, 500*time.Microsecond, result)
	})

	t.Run("parses milliseconds", func(t *testing.T) {
		t.Setenv("TEST_DURATION_VAR", "500ms")
		result := getEnvAsDuration("TEST_DURATION_VAR", 5*time.Minute)
		assert.Equal(t, 500*time.Millisecond, result)
	})
}

// TestLoad_DatabasePoolConfig tests that database pool configuration is loaded correctly
func TestLoad_DatabasePoolConfig(t *testing.T) {
	t.Run("loads default database pool configuration", func(t *testing.T) {
		clearEnvVars(t)
		t.Setenv("API_KEY", "test-key")

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, 20, cfg.DBMaxConns, "Should use default max connections")
		assert.Equal(t, 5*time.Minute, cfg.DBMaxConnIdleTime, "Should use default idle time")
		assert.Equal(t, 30*time.Minute, cfg.DBMaxConnLifetime, "Should use default lifetime")
	})

	t.Run("loads custom database pool configuration", func(t *testing.T) {
		clearEnvVars(t)
		t.Setenv("API_KEY", "test-key")
		t.Setenv("DB_MAX_CONNS", "50")
		t.Setenv("DB_MAX_CONN_IDLE_TIME", "10m")
		t.Setenv("DB_MAX_CONN_LIFETIME", "1h")

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, 50, cfg.DBMaxConns, "Should use custom max connections")
		assert.Equal(t, 10*time.Minute, cfg.DBMaxConnIdleTime, "Should use custom idle time")
		assert.Equal(t, 1*time.Hour, cfg.DBMaxConnLifetime, "Should use custom lifetime")
	})

	t.Run("uses defaults for invalid pool config values", func(t *testing.T) {
		clearEnvVars(t)
		t.Setenv("API_KEY", "test-key")
		t.Setenv("DB_MAX_CONNS", "not-a-number")
		t.Setenv("DB_MAX_CONN_IDLE_TIME", "invalid")
		t.Setenv("DB_MAX_CONN_LIFETIME", "bad-duration")

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, 20, cfg.DBMaxConns, "Should fallback to default for invalid max conns")
		assert.Equal(t, 5*time.Minute, cfg.DBMaxConnIdleTime, "Should fallback to default for invalid idle time")
		assert.Equal(t, 30*time.Minute, cfg.DBMaxConnLifetime, "Should fallback to default for invalid lifetime")
	})

	t.Run("production-like pool configuration", func(t *testing.T) {
		clearEnvVars(t)
		t.Setenv("API_KEY", "prod-key")
		t.Setenv("ENVIRONMENT", "prod")
		t.Setenv("DB_MAX_CONNS", "100")
		t.Setenv("DB_MAX_CONN_IDLE_TIME", "15m")
		t.Setenv("DB_MAX_CONN_LIFETIME", "2h")

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, 100, cfg.DBMaxConns, "Production should support more connections")
		assert.Equal(t, 15*time.Minute, cfg.DBMaxConnIdleTime)
		assert.Equal(t, 2*time.Hour, cfg.DBMaxConnLifetime)
	})
}
