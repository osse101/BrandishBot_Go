package logger

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

type ctxKey string

const requestIDKey ctxKey = "requestID"

// GenerateRequestID creates a new UUID for tracing requests.
func GenerateRequestID() string {
    return uuid.NewString()
}

// WithRequestID returns a new context containing the request ID.
func WithRequestID(ctx context.Context, requestID string) context.Context {
    return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext extracts the request ID from the context, if present.
func RequestIDFromContext(ctx context.Context) (string, bool) {
    v := ctx.Value(requestIDKey)
    if v == nil {
        return "", false
    }
    if id, ok := v.(string); ok {
        return id, true
    }
    return "", false
}

// FromContext returns a logger that includes the request_id attribute when present.
func FromContext(ctx context.Context) *slog.Logger {
    if id, ok := RequestIDFromContext(ctx); ok {
        return slog.Default().With("request_id", id)
    }
    return slog.Default()
}
