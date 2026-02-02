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
			for _, path := range PublicPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Validate API key for all other endpoints
			providedKey := r.Header.Get(HeaderAPIKey)

			// Use constant time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(providedKey), []byte(apiKey)) != 1 {
				ip := extractIP(r, trustedProxies)
				detector.RecordFailedAuth(ip)

				log := logger.FromContext(r.Context())
				log.Warn(LogMsgAuthFailed,
					"remote_addr", r.RemoteAddr,
					"path", r.URL.Path,
					"has_key", providedKey != "",
					"ip", ip)

				http.Error(w, ErrMsgUnauthorized, http.StatusUnauthorized)
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
		slog.Warn(SecurityAlertFailedAuth,
			"ip", ip,
			"count", s.failedAuthByIP[ip])
	}
}

// RecordRequest records a request for rate monitoring and returns false if rate limit exceeded
func (s *SuspiciousActivityDetector) RecordRequest(ip string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.resetCountsIfNeeded()
	s.requestCountByIP[ip]++

	// Block if request rate exceeds limit (1000 req / 5 min)
	if s.requestCountByIP[ip] > 1000 {
		if s.requestCountByIP[ip]%100 == 0 { // Log every 100 requests to avoid log spam
			slog.Warn(SecurityAlertHighRate,
				"ip", ip,
				"count_in_5min", s.requestCountByIP[ip])
		}
		return false
	}
	return true
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

// SecurityLoggingMiddleware enhances logging with security information and enforces rate limits
func SecurityLoggingMiddleware(trustedProxies []string, detector *SuspiciousActivityDetector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract IP address
			ip := extractIP(r, trustedProxies)

			// Record request and check rate limit
			if !detector.RecordRequest(ip) {
				http.Error(w, ErrMsgTooManyRequests, http.StatusTooManyRequests)
				return
			}

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
		forwarded := r.Header.Get(HeaderForwardedFor)
		if forwarded != "" {
			// For X-Forwarded-For: client, proxy1, proxy2
			// We want the rightmost IP (the one that connected to our trusted proxy)
			// since we trust the proxy to accurately report the previous hop.
			ips := strings.Split(forwarded, ",")
			return strings.TrimSpace(ips[len(ips)-1])
		}
	}

	return remoteIP
}

// SecurityHeadersMiddleware adds security headers to responses
func SecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent MIME sniffing
			w.Header().Set(HeaderContentType, HeaderValueNoSniff)
			// Prevent clickjacking
			w.Header().Set(HeaderFrameOptions, HeaderValueSameOrigin)
			// Enable XSS protection (for older browsers)
			w.Header().Set(HeaderXSSProtection, HeaderValueXSSBlock)
			// Control referrer information
			w.Header().Set(HeaderReferrerPolicy, HeaderValueReferrerStrictOrigin)

			next.ServeHTTP(w, r)
		})
	}
}
