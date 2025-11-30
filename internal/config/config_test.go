package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoad tests configuration loading from environment
func TestLoad(t *testing.T) {
	t.Run("loads config with defaults when no env vars set", func(t *testing.T) {
		// Clear relevant env vars
		clearEnvVars(t)
		// Must set API_KEY or it fails validation
		t.Setenv("API_KEY", "test-key")

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, 8080, cfg.Port, "Should use default port")
		assert.Equal(t, "info", cfg.LogLevel)
		assert.Equal(t, "text", cfg.LogFormat)
		assert.Equal(t, "dev", cfg.Environment)
		assert.Equal(t, "postgres", cfg.DBUser)
		assert.Equal(t, "localhost", cfg.DBHost)
		assert.Equal(t, "test-key", cfg.APIKey)
	})

	t.Run("loads config from environment variables", func(t *testing.T) {
		clearEnvVars(t)

		// Set custom values
		t.Setenv("PORT", "3000")
		t.Setenv("API_KEY", "custom-api-key")
		t.Setenv("LOG_LEVEL", "debug")
		t.Setenv("LOG_FORMAT", "json")
		t.Setenv("ENVIRONMENT", "production")
		t.Setenv("DB_USER", "customuser")
		t.Setenv("DB_PASSWORD", "custompass")
		t.Setenv("DB_HOST", "db.example.com")
		t.Setenv("DB_PORT", "5433")
		t.Setenv("DB_NAME", "customdb")

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, 3000, cfg.Port)
		assert.Equal(t, "custom-api-key", cfg.APIKey)
		assert.Equal(t, "debug", cfg.LogLevel)
		assert.Equal(t, "json", cfg.LogFormat)
		assert.Equal(t, "production", cfg.Environment)
		assert.Equal(t, "customuser", cfg.DBUser)
		assert.Equal(t, "custompass", cfg.DBPassword)
		assert.Equal(t, "db.example.com", cfg.DBHost)
		assert.Equal(t, "5433", cfg.DBPort)
		assert.Equal(t, "customdb", cfg.DBName)
	})

	t.Run("returns error when API_KEY is missing", func(t *testing.T) {
		clearEnvVars(t)
		// Explicitly unset API_KEY
		os.Unsetenv("API_KEY")

		cfg, err := Load()

		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "API_KEY")
		assert.Contains(t, err.Error(), "must be set")
	})

	t.Run("returns error for invalid PORT", func(t *testing.T) {
		clearEnvVars(t)
		t.Setenv("API_KEY", "test-key")
		t.Setenv("PORT", "not-a-number")

		cfg, err := Load()

		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "invalid PORT")
	})

	t.Run("handles negative port number", func(t *testing.T) {
		clearEnvVars(t)
		t.Setenv("API_KEY", "test-key")
		t.Setenv("PORT", "-1")

		// Should load without error (validation happens at server startup)
		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, -1, cfg.Port)
	})

	t.Run("handles PORT edge cases", func(t *testing.T) {
		testCases := []struct {
			name        string
			portValue   string
			shouldError bool
		}{
			{"zero port", "0", false},
			{"max valid port", "65535", false},
			{"above max port", "65536", false}, // Loads but invalid for use
			{"float port", "8080.5", true},
			{"empty string", "", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				clearEnvVars(t)
				t.Setenv("API_KEY", "test-key")
				t.Setenv("PORT", tc.portValue)

				_, err := Load()

				if tc.shouldError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

// TestGetDBConnString verifies database connection string generation
func TestGetDBConnString(t *testing.T) {
	t.Run("generates correct connection string", func(t *testing.T) {
		cfg := &Config{
			DBUser:     "testuser",
			DBPassword: "testpass",
			DBHost:     "testhost",
			DBPort:     "5432",
			DBName:     "testdb",
		}

		connStr := cfg.GetDBConnString()

		expected := "postgres://testuser:testpass@testhost:5432/testdb?sslmode=disable"
		assert.Equal(t, expected, connStr)
	})

	t.Run("handles special characters in password", func(t *testing.T) {
		cfg := &Config{
			DBUser:     "user",
			DBPassword: "p@ss:word/with$pecial",
			DBHost:     "localhost",
			DBPort:     "5432",
			DBName:     "db",
		}

		connStr := cfg.GetDBConnString()

		// Should contain the password as-is (URL encoding handled by driver)
		assert.Contains(t, connStr, "p@ss:word/with$pecial")
	})

	t.Run("uses custom port", func(t *testing.T) {
		cfg := &Config{
			DBUser:     "user",
			DBPassword: "pass",
			DBHost:     "db.example.com",
			DBPort:     "5433",
			DBName:     "custom",
		}

		connStr := cfg.GetDBConnString()

		assert.Contains(t, connStr, ":5433/")
		assert.Contains(t, connStr, "db.example.com")
	})

	t.Run("includes sslmode=disable", func(t *testing.T) {
		cfg := &Config{
			DBUser:     "user",
			DBPassword: "pass",
			DBHost:     "host",
			DBPort:     "5432",
			DBName:     "db",
		}

		connStr := cfg.GetDBConnString()

		assert.Contains(t, connStr, "sslmode=disable",
			"Should disable SSL for local development")
	})
}

// TestConfig_RealWorldScenarios tests realistic configuration scenarios
func TestConfig_RealWorldScenarios(t *testing.T) {
	t.Run("typical development environment", func(t *testing.T) {
		clearEnvVars(t)
		t.Setenv("API_KEY", "dev-api-key-12345")
		t.Setenv("ENVIRONMENT", "dev")
		t.Setenv("LOG_LEVEL", "debug")

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, "dev", cfg.Environment)
		assert.Equal(t, "debug", cfg.LogLevel)
		assert.Equal(t, "localhost", cfg.DBHost, "Dev should use localhost")
	})

	t.Run("typical production environment", func(t *testing.T) {
		clearEnvVars(t)
		t.Setenv("API_KEY", "prod-secure-key")
		t.Setenv("ENVIRONMENT", "prod")
		t.Setenv("LOG_LEVEL", "warn")
		t.Setenv("LOG_FORMAT", "json")
		t.Setenv("DB_HOST", "prod-db.example.com")
		t.Setenv("DB_PASSWORD", "secure-prod-password")

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, "prod", cfg.Environment)
		assert.Equal(t, "warn", cfg.LogLevel)
		assert.Equal(t, "json", cfg.LogFormat, "Prod should use JSON logging")
		assert.Equal(t, "prod-db.example.com", cfg.DBHost)
	})

	t.Run("docker compose environment", func(t *testing.T) {
		clearEnvVars(t)
		t.Setenv("API_KEY", "docker-key")
		t.Setenv("DB_HOST", "db") // Docker service name
		t.Setenv("DB_USER", "postgres")
		t.Setenv("DB_PASSWORD", "postgres")

		cfg, err := Load()

		require.NoError(t, err)
		connStr := cfg.GetDBConnString()
		assert.Contains(t, connStr, "postgres://postgres:postgres@db:5432/")
	})
}

// Helper function to clear environment variables
func clearEnvVars(t *testing.T) {
	t.Helper()

	// Clear all config-related env vars to ensure clean test state
	envVars := []string{
		"PORT", "API_KEY", "LOG_LEVEL", "LOG_FORMAT", "LOG_DIR",
		"SERVICE_NAME", "VERSION", "ENVIRONMENT",
		"DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME",
	}

	for _, key := range envVars {
		os.Unsetenv(key)
	}
}
