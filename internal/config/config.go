package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	// Server
	Port   int
	APIKey string // API key for authentication

	// Logging
	LogLevel    string
	LogFormat   string // "json" or "text"
	LogDir      string
	ServiceName string
	Version     string
	Environment string // "dev", "staging", "prod"

	// Discord Configuration
	DiscordToken        string `mapstructure:"DISCORD_TOKEN"`
	DiscordAppID        string `mapstructure:"DISCORD_APP_ID"`
	DiscordDevChannelID string `mapstructure:"DISCORD_DEV_CHANNEL_ID"`
	DiscordDiggingGameChannelID string `mapstructure:"DISCORD_DIGGING_GAME_CHANNEL_ID"`
	DiscordWebhookPort  string `mapstructure:"DISCORD_WEBHOOK_PORT"`

	// GitHub Configuration
	GithubToken     string `mapstructure:"GITHUB_TOKEN"`
	GithubOwnerRepo string `mapstructure:"GITHUB_OWNER_REPO"`

	// Database
	DBUser     string
	DBPassword string
	DBHost     string
	DBPort     string
	DBName     string

	// Gamble configuration
	GambleJoinDuration time.Duration // Duration for users to join a gamble
}

// Load loads the configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists, but don't fail if it doesn't (could be real env vars)
	_ = godotenv.Load()

	cfg := &Config{
		// Logging config
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		LogFormat:   getEnv("LOG_FORMAT", "text"),
		LogDir:      getEnv("LOG_DIR", "logs"),
		ServiceName: getEnv("SERVICE_NAME", "brandish-bot"),
		Version:     getEnv("VERSION", "dev"),
		Environment: getEnv("ENVIRONMENT", "dev"),

		// Database config
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "brandishbot"),

		// Server config
		APIKey: getEnv("API_KEY", ""),

		// Discord config
		DiscordToken:        getEnv("DISCORD_TOKEN", ""),
		DiscordAppID:        getEnv("DISCORD_APP_ID", ""),
		DiscordDevChannelID: getEnv("DISCORD_DEV_CHANNEL_ID", ""),
		DiscordDiggingGameChannelID: getEnv("DISCORD_DIGGING_GAME_CHANNEL_ID", ""),
		DiscordWebhookPort:  getEnv("DISCORD_WEBHOOK_PORT", "8082"),

		// GitHub config
		GithubToken:     getEnv("GITHUB_TOKEN", ""),
		GithubOwnerRepo: getEnv("GITHUB_OWNER_REPO", "osse101/BrandishBot_Go"),
	}

	portStr := getEnv("PORT", "8080")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid PORT value: %w", err)
	}
	cfg.Port = port

	// Gamble config
	gambleJoinStr := getEnv("GAMBLE_JOIN_DURATION_MINUTES", "2")
	gambleJoinMins, err := strconv.Atoi(gambleJoinStr)
	if err != nil {
		return nil, fmt.Errorf("invalid GAMBLE_JOIN_DURATION_MINUTES value: %w", err)
	}
	cfg.GambleJoinDuration = time.Duration(gambleJoinMins) * time.Minute

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
