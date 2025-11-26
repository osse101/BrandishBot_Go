package logger

import (
	"log/slog"
	"strings"
)

// Config represents logger configuration
type Config struct {
	Level       string // "debug", "info", "warn", "error"
	Format      string // "json", "text"
	ServiceName string
	Version     string
	Environment string // "dev", "staging", "prod"
	AddSource   bool   // Include source file/line in logs
}

// NewConfig creates a config from explicit values (recommended)
func NewConfig(level, format, serviceName, version, environment string, addSource bool) Config {
	return Config{
		Level:       level,
		Format:      format,
		ServiceName: serviceName,
		Version:     version,
		Environment: environment,
		AddSource:   addSource,
	}
}

// ProductionConfig returns production-ready defaults
func ProductionConfig() Config {
	return Config{
		Level:       "info",
		Format:      "json",
		ServiceName: "brandish-bot",
		Version:     "1.0.0",
		Environment: "prod",
		AddSource:   false,
	}
}

// DevelopmentConfig returns development-friendly defaults
func DevelopmentConfig() Config {
	return Config{
		Level:       "debug",
		Format:      "text",
		ServiceName: "brandish-bot",
		Version:     "dev",
		Environment: "dev",
		AddSource:   true,
	}
}

// LogLevel converts string level to slog.Level
func (c Config) LogLevel() slog.Level {
	switch strings.ToLower(c.Level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// IsJSON returns true if format is JSON
func (c Config) IsJSON() bool {
	return strings.ToLower(c.Format) == "json"
}

// BaseAttributes returns common attributes to add to all logs
func (c Config) BaseAttributes() []slog.Attr {
	return []slog.Attr{
		slog.String("service", c.ServiceName),
		slog.String("version", c.Version),
		slog.String("environment", c.Environment),
	}
}

// DefaultConfig returns defaults (fallback when no config provided)
// Prefer using NewConfig with explicit values from your app config
func DefaultConfig() Config {
	return Config{
		Level:       "info",
		Format:      "text",
		ServiceName: "brandish-bot",
		Version:     "dev",
		Environment: "dev",
		AddSource:   false,
	}
}
