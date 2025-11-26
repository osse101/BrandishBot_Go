package logger

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"os"
)

type contextKey string

const requestIDKey contextKey = "request_id"

var (
	// defaultLogger is the package-level logger instance
	defaultLogger *slog.Logger
)

func init() {
	// Initialize with default config on package load
	InitLogger(DefaultConfig())
}

// GenerateRequestID creates a new UUID for tracing requests
func GenerateRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	// Format as UUID-like string
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// InitLogger initializes the global logger with the given configuration
func InitLogger(config Config) {
	var handler slog.Handler
	
	opts := &slog.HandlerOptions{
		Level:     config.LogLevel(),
		AddSource: config.AddSource,
	}
	
	// Create appropriate handler based on format
	if config.IsJSON() {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	
	// Wrap with base attributes
	handler = handler.WithAttrs(config.BaseAttributes())
	
	// Set as default
	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

// InitLoggerWithWriter initializes logger with custom writer (for testing)
func InitLoggerWithWriter(config Config, w io.Writer) {
	var handler slog.Handler
	
	opts := &slog.HandlerOptions{
		Level:     config.LogLevel(),
		AddSource: config.AddSource,
	}
	
	if config.IsJSON() {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}
	
	handler = handler.WithAttrs(config.BaseAttributes())
	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// RequestIDFromContext returns the request ID from a context
// Deprecated: Use GetRequestID instead
func RequestIDFromContext(ctx context.Context) string {
	return GetRequestID(ctx)
}

// FromContext returns a logger with request ID from context if available
func FromContext(ctx context.Context) *slog.Logger {
	if id := GetRequestID(ctx); id != "" {
		return slog.Default().With("request_id", id)
	}
	return slog.Default()
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}
