package server

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// AuthMiddleware validates API key
func AuthMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow public access to documentation and health check endpoints
			publicPaths := []string{"/swagger/", "/healthz", "/readyz"}
			for _, path := range publicPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Validate API key for all other endpoints
			providedKey := r.Header.Get("X-API-Key")

			if providedKey != apiKey {
				log := logger.FromContext(r.Context())
				log.Warn("Authentication failed",
					"remote_addr", r.RemoteAddr,
					"path", r.URL.Path,
					"has_key", providedKey != "")

				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestSizeLimitMiddleware limits request body size
func RequestSizeLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// SuspiciousActivityDetector tracks and alerts on suspicious patterns
type SuspiciousActivityDetector struct {
	mu               sync.Mutex
	failedAuthByIP   map[string]int
	requestCountByIP map[string]int
	lastResetTime    time.Time
}

func NewSuspiciousActivityDetector() *SuspiciousActivityDetector {
	return &SuspiciousActivityDetector{
		failedAuthByIP:   make(map[string]int),
		requestCountByIP: make(map[string]int),
		lastResetTime:    time.Now(),
	}
}

// RecordFailedAuth records a failed authentication attempt
func (s *SuspiciousActivityDetector) RecordFailedAuth(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failedAuthByIP[ip]++

	// Alert if threshold exceeded
	if s.failedAuthByIP[ip] >= 5 {
		slog.Warn("⚠️ SECURITY ALERT: Multiple failed authentication attempts",
			"ip", ip,
			"count", s.failedAuthByIP[ip])
	}
}

// RecordRequest records a request for rate monitoring
func (s *SuspiciousActivityDetector) RecordRequest(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Reset counters every 5 minutes
	if time.Since(s.lastResetTime) > 5*time.Minute {
		s.requestCountByIP = make(map[string]int)
		s.lastResetTime = time.Now()
	}

	s.requestCountByIP[ip]++

	// Alert on high request rate
	if s.requestCountByIP[ip] > 1000 {
		slog.Warn("⚠️ SECURITY ALERT: High request rate detected",
			"ip", ip,
			"count_in_5min", s.requestCountByIP[ip])
	}
}

// SecurityLoggingMiddleware enhances logging with security information
func SecurityLoggingMiddleware(detector *SuspiciousActivityDetector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract IP address
			ip := extractIP(r)

			// Record request
			detector.RecordRequest(ip)

			// Add IP to context for other handlers
			// log := logger.FromContext(r.Context())
			// log = log.With("client_ip", ip)

			// Continue with request
			next.ServeHTTP(w, r)
		})
	}
}

// extractIP gets the client IP address from request
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take first IP if comma-separated
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	return ip
}
