package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	// Server
	Port           int
	APIKey         string   // API key for authentication
	TrustedProxies []string // List of trusted proxy IPs

	// Logging
	LogLevel    string
	LogFormat   string // "json" or "text"
	LogDir      string
	ServiceName string
	Version     string
	Environment string // "dev", "staging", "prod"

	// Discord Configuration
	DiscordToken                string `mapstructure:"DISCORD_TOKEN"`
	DiscordAppID                string `mapstructure:"DISCORD_APP_ID"`
	DiscordDevChannelID         string `mapstructure:"DISCORD_DEV_CHANNEL_ID"`
	DiscordDiggingGameChannelID string `mapstructure:"DISCORD_DIGGING_GAME_CHANNEL_ID"`
	DiscordWebhookPort          string `mapstructure:"DISCORD_WEBHOOK_PORT"`

	// GitHub Configuration
	GithubToken     string `mapstructure:"GITHUB_TOKEN"`
	GithubOwnerRepo string `mapstructure:"GITHUB_OWNER_REPO"`

	// Database
	DBUser     string
	DBPassword string
	DBHost     string
	DBPort     string
	DBName     string

	// Database Pool
	DBMaxConns        int
	DBMaxConnIdleTime time.Duration
	DBMaxConnLifetime time.Duration

	// Gamble configuration
	GambleJoinDuration time.Duration // Duration for users to join a gamble

	// Streamer.bot configuration
	StreamerbotEnabled    bool   // Enable WebSocket connection to Streamer.bot
	StreamerbotWebhookURL string // WebSocket URL for Streamer.bot (e.g., ws://127.0.0.1:8080/ or http://IP:PORT/streamerbot)

	// Development Settings
	DevMode bool // When true, bypasses cooldowns and enables test features

	// Event Publishing
	EventMaxRetries     int           // Max retries for event publishing (default: 5)
	EventRetryDelay     time.Duration // Base delay for exponential backoff (default: 2s)
	EventDeadLetterPath string        // Path to dead-letter log file (default: logs/event_deadletter.jsonl)
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

		// Database pool defaults
		DBMaxConns:        getEnvAsInt("DB_MAX_CONNS", 20),
		DBMaxConnIdleTime: getEnvAsDuration("DB_MAX_CONN_IDLE_TIME", 5*time.Minute),
		DBMaxConnLifetime: getEnvAsDuration("DB_MAX_CONN_LIFETIME", 30*time.Minute),

		// Server config
		APIKey: getEnv("API_KEY", ""),

		// Discord config
		DiscordToken:                getEnv("DISCORD_TOKEN", ""),
		DiscordAppID:                getEnv("DISCORD_APP_ID", ""),
		DiscordDevChannelID:         getEnv("DISCORD_DEV_CHANNEL_ID", ""),
		DiscordDiggingGameChannelID: getEnv("DISCORD_DIGGING_GAME_CHANNEL_ID", ""),
		DiscordWebhookPort:          getEnv("DISCORD_WEBHOOK_PORT", "8082"),

		// GitHub config
		GithubToken:     getEnv("GITHUB_TOKEN", ""),
		GithubOwnerRepo: getEnv("GITHUB_OWNER_REPO", "osse101/BrandishBot_Go"),

		// Streamer.bot config
		StreamerbotWebhookURL: getEnv("STREAMERBOT_WEBHOOK_URL", ""),

		// Event publishing config
		EventMaxRetries:     getEnvAsInt("EVENT_MAX_RETRIES", 5),
		EventRetryDelay:     getEnvAsDuration("EVENT_RETRY_DELAY", 2*time.Second),
		EventDeadLetterPath: getEnv("EVENT_DEADLETTER_PATH", "logs/event_deadletter.jsonl"),
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

	// Dev mode (bypasses cooldowns and enables test features)
	devModeStr := getEnv("DEV_MODE", "false")
	cfg.DevMode = devModeStr == "true" || devModeStr == "1"

	// Streamer.bot WebSocket enabled
	sbEnabledStr := getEnv("STREAMERBOT_ENABLED", "false")
	cfg.StreamerbotEnabled = sbEnabledStr == "true" || sbEnabledStr == "1"

	// Parse trusted proxies
	trustedProxiesStr := getEnv("TRUSTED_PROXIES", "")
	if trustedProxiesStr != "" {
		proxies := strings.Split(trustedProxiesStr, ",")
		for _, proxy := range proxies {
			trimmed := strings.TrimSpace(proxy)
			if trimmed != "" {
				cfg.TrustedProxies = append(cfg.TrustedProxies, trimmed)
			}
		}
	}

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

// getEnvAsInt retrieves an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// getEnvAsDuration retrieves an environment variable as a duration or returns a default value
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}
