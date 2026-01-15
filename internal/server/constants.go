package server

// HTTP error messages for middleware responses
const (
	ErrMsgUnauthorized    = "Unauthorized"
	ErrMsgTooManyRequests = "Too Many Requests"
)

// Security alert message templates
const (
	SecurityAlertFailedAuth = "⚠️ SECURITY ALERT: Multiple failed authentication attempts"
	SecurityAlertHighRate   = "⚠️ SECURITY ALERT: Blocking high request rate"
)

// Log messages for server lifecycle and request handling
const (
	LogMsgServerStarting    = "Server starting"
	LogMsgRequestStarted    = "Request started"
	LogMsgRequestCompleted  = "Request completed"
	LogMsgRequestHeaders    = "Request headers"
	LogMsgAuthFailed        = "Authentication failed"
)

// HTTP header names
const (
	HeaderAPIKey         = "X-API-Key"
	HeaderAuthorization  = "Authorization"
	HeaderForwardedFor   = "X-Forwarded-For"
	HeaderContentType    = "X-Content-Type-Options"
	HeaderFrameOptions   = "X-Frame-Options"
	HeaderXSSProtection  = "X-XSS-Protection"
	HeaderReferrerPolicy = "Referrer-Policy"
)

// Security header values
const (
	HeaderValueNoSniff                = "nosniff"
	HeaderValueSameOrigin             = "SAMEORIGIN"
	HeaderValueXSSBlock               = "1; mode=block"
	HeaderValueReferrerStrictOrigin   = "strict-origin-when-cross-origin"
)

// Public path prefixes that bypass authentication
var PublicPaths = []string{
	"/swagger/",
	"/healthz",
	"/readyz",
	"/metrics",
}

// Header redaction marker
const (
	RedactedValue = "[REDACTED]"
)
