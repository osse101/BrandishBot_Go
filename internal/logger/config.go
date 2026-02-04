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
		Level:       LogLevelInfo,
		Format:      LogFormatJSON,
		ServiceName: DefaultServiceName,
		Version:     ProductionVersion,
		Environment: EnvironmentProduction,
		AddSource:   false,
	}
}

// DevelopmentConfig returns development-friendly defaults
func DevelopmentConfig() Config {
	return Config{
		Level:       LogLevelDebug,
		Format:      LogFormatText,
		ServiceName: DefaultServiceName,
		Version:     DefaultVersion,
		Environment: EnvironmentDev,
		AddSource:   true,
	}
}

// LogLevel converts string level to slog.Level
func (c Config) LogLevel() slog.Level {
	switch strings.ToLower(c.Level) {
	case LogLevelDebug:
		return slog.LevelDebug
	case LogLevelInfo:
		return slog.LevelInfo
	case LogLevelWarn, LogLevelWarning:
		return slog.LevelWarn
	case LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// IsJSON returns true if format is JSON
func (c Config) IsJSON() bool {
	return strings.ToLower(c.Format) == LogFormatJSON
}

// BaseAttributes returns common attributes to add to all logs
func (c Config) BaseAttributes() []slog.Attr {
	return []slog.Attr{
		slog.String(AttrKeyService, c.ServiceName),
		slog.String(AttrKeyVersion, c.Version),
		slog.String(AttrKeyEnvironment, c.Environment),
	}
}

// DefaultConfig returns defaults (fallback when no config provided)
// Prefer using NewConfig with explicit values from your app config
func DefaultConfig() Config {
	return Config{
		Level:       LogLevelInfo,
		Format:      LogFormatText,
		ServiceName: DefaultServiceName,
		Version:     DefaultVersion,
		Environment: EnvironmentDev,
		AddSource:   false,
	}
}
