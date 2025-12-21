package server

import (
	"crypto/subtle"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// AuthMiddleware validates API key
func AuthMiddleware(apiKey string, trustedProxies []string, detector *SuspiciousActivityDetector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow public access to documentation and health check endpoints
			publicPaths := []string{"/swagger/", "/healthz", "/readyz", "/metrics"}
			for _, path := range publicPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Validate API key for all other endpoints
			providedKey := r.Header.Get("X-API-Key")

			// Use constant time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(providedKey), []byte(apiKey)) != 1 {
				ip := extractIP(r, trustedProxies)
				detector.RecordFailedAuth(ip)

				log := logger.FromContext(r.Context())
				log.Warn("Authentication failed",
					"remote_addr", r.RemoteAddr,
					"path", r.URL.Path,
					"has_key", providedKey != "",
					"ip", ip)

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

	s.resetCountsIfNeeded()
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

	s.resetCountsIfNeeded()
	s.requestCountByIP[ip]++

	// Alert on high request rate
	if s.requestCountByIP[ip] > 1000 {
		slog.Warn("⚠️ SECURITY ALERT: High request rate detected",
			"ip", ip,
			"count_in_5min", s.requestCountByIP[ip])
	}
}

// resetCountsIfNeeded resets counters if the time window has passed
// Caller must hold the mutex
func (s *SuspiciousActivityDetector) resetCountsIfNeeded() {
	if time.Since(s.lastResetTime) > 5*time.Minute {
		s.requestCountByIP = make(map[string]int)
		s.failedAuthByIP = make(map[string]int)
		s.lastResetTime = time.Now()
	}
}

// SecurityLoggingMiddleware enhances logging with security information
func SecurityLoggingMiddleware(trustedProxies []string, detector *SuspiciousActivityDetector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract IP address
			ip := extractIP(r, trustedProxies)

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

// extractIP gets the client IP address from request.
// It only trusts X-Forwarded-For if the request comes from a trusted proxy.
func extractIP(r *http.Request, trustedProxies []string) string {
	// Get remote IP (direct connection)
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// Fallback if parsing fails (shouldn't happen for valid RemoteAddr)
		remoteIP = r.RemoteAddr
	}

	// Check if remote IP is a trusted proxy
	isTrusted := false
	for _, proxy := range trustedProxies {
		if proxy == remoteIP {
			isTrusted = true
			break
		}
	}

	// Only check X-Forwarded-For if trusted
	if isTrusted {
		forwarded := r.Header.Get("X-Forwarded-For")
		if forwarded != "" {
			// Standard behavior for proxies is to append the client IP to the list.
			// Therefore, if we trust the proxy, the *last* IP in the list is the one
			// that connected to the proxy.
			ips := strings.Split(forwarded, ",")
			return strings.TrimSpace(ips[len(ips)-1])
		}
	}

	return remoteIP
}
