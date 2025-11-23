package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	Port       int
	LogLevel   string
	DBUser     string
	DBPassword string
	DBHost     string
	DBPort     string
	DBName     string
	LogDir     string
	APIKey     string // API key for authentication
}

// Load loads the configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists, but don't fail if it doesn't (could be real env vars)
	_ = godotenv.Load()

	cfg := &Config{
		LogLevel:   getEnv("LOG_LEVEL", "INFO"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "brandishbot"),
		LogDir:     getEnv("LOG_DIR", "logs"),
		APIKey:     getEnv("API_KEY", ""),
	}

	portStr := getEnv("PORT", "8080")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid PORT value: %w", err)
	}
	cfg.Port = port

	// Validate API key is set
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API_KEY environment variable must be set for security")
	}

	return cfg, nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// GetDBConnString returns the PostgreSQL connection string
func (c *Config) GetDBConnString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
	)
}
