package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// EngagementTracker is middleware that automatically tracks user engagement
type EngagementTracker struct {
	eventBus event.Bus
}

// NewEngagementTracker creates a new engagement tracking middleware
func NewEngagementTracker(eventBus event.Bus) *EngagementTracker {
	return &EngagementTracker{
		eventBus: eventBus,
	}
}

// Track wraps an HTTP handler to automatically track engagement
func (e *EngagementTracker) Track(metricType string, getValue func(*http.Request) int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Call the actual handler first
			next.ServeHTTP(w, r)

			// Track engagement after successful execution
			// Extract user ID from request context or body
			userID := extractUserID(r)
			if userID != EmptyUserID {
				value := DefaultMetricValue
				if getValue != nil {
					value = getValue(r)
				}

				// Publish engagement event
				evt := event.Event{
					Version: EventVersion,
					Type:    domain.EventTypeEngagement,
					Payload: &domain.EngagementMetric{
						UserID:      userID,
						MetricType:  metricType,
						MetricValue: value,
						RecordedAt:  time.Now(),
					},
				}

				if err := e.eventBus.Publish(context.Background(), evt); err != nil {
					log := logger.FromContext(r.Context())
					log.Error(LogMsgEngagementEventPublishFailed, "error", err, "user_id", userID, "metric", metricType)
				}
			}
		})
	}
}

// TrackCommand tracks command execution
func (e *EngagementTracker) TrackCommand(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Execute handler
		next.ServeHTTP(w, r)

		// Track command usage
		userID := extractUserID(r)
		if userID != EmptyUserID {
			metadata := map[string]interface{}{
				MetadataKeyEndpoint: r.URL.Path,
				MetadataKeyMethod:   r.Method,
			}

			metric := &domain.EngagementMetric{
				UserID:      userID,
				MetricType:  MetricTypeCommand,
				MetricValue: DefaultMetricValue,
				RecordedAt:  time.Now(),
				Metadata:    metadata,
			}

			// Publish engagement event
			evt := event.Event{
				Version: EventVersion,
				Type:    domain.EventTypeEngagement,
				Payload: metric,
			}

			if err := e.eventBus.Publish(context.Background(), evt); err != nil {
				log := logger.FromContext(r.Context())
				log.Error(LogMsgCommandEngagementEventPublishFailed, "error", err, "user_id", userID)
			}
		}
	})
}

// extractUserID extracts user ID from request
// Tries multiple sources: context, query params, common body fields
func extractUserID(r *http.Request) string {
	// Try to get from context (if set by earlier middleware) using the typed key
	if userID := r.Context().Value(UserIDKey); userID != nil {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	// Backward compatibility fallback for a legacy user ID key
	if userID := r.Context().Value(UserIDKey); userID != nil {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}

	// Try query parameter
	if userID := r.URL.Query().Get(QueryParamUsername); userID != EmptyUserID {
		return userID
	}

	// For POST requests, we'd need to parse the body
	// But that would consume the body, so we rely on handlers to track engagement
	// or use context to pass user_id

	return EmptyUserID
}

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// UserIDKey is the context key for user ID
	UserIDKey contextKey = "user_id"
	// EngagementMetricKey is the context key for engagement metric type
	EngagementMetricKey contextKey = "engagement_metric"
)

// WithUserID adds user ID to request context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// GetUserID retrieves user ID from context
func GetUserID(ctx context.Context) string {
	if userID := ctx.Value(UserIDKey); userID != nil {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	return EmptyUserID
}

// TrackEngagementFromContext records engagement using info from context
func TrackEngagementFromContext(ctx context.Context, eventBus event.Bus, metricType string, value int) {
	userID := GetUserID(ctx)
	if userID == EmptyUserID {
		return
	}

	metric := &domain.EngagementMetric{
		UserID:      userID,
		MetricType:  metricType,
		MetricValue: value,
		RecordedAt:  time.Now(),
	}

	evt := event.Event{
		Version: EventVersion,
		Type:    domain.EventTypeEngagement,
		Payload: metric,
	}

	if err := eventBus.Publish(context.Background(), evt); err != nil {
		log := logger.FromContext(ctx)
		log.Error(LogMsgEngagementEventFromContextFailed, "error", err, "user_id", userID, "metric", metricType)
	}
}
