package logger

// UUID Generation Constants
const (
	UUIDBytesLength   = 16
	FallbackUUID      = "00000000-0000-0000-0000-000000000000"
	UUIDFormatPattern = "%x-%x-%x-%x-%x"
)

// Context Keys
const (
	ContextKeyRequestID = "request_id"
)

// Log Level String Values
const (
	LogLevelDebug   = "debug"
	LogLevelInfo    = "info"
	LogLevelWarn    = "warn"
	LogLevelWarning = "warning"
	LogLevelError   = "error"
)

// Log Format String Values
const (
	LogFormatJSON = "json"
	LogFormatText = "text"
)

// Service Configuration Values
const (
	DefaultServiceName = "brandish-bot"
	DefaultVersion     = "dev"
	ProductionVersion  = "1.0.0"
)

// Environment String Values
const (
	EnvironmentDev        = "dev"
	EnvironmentStaging    = "staging"
	EnvironmentProduction = "prod"
	EnvironmentTest       = "test"
)

// Log Attribute Keys
const (
	AttrKeyService     = "service"
	AttrKeyVersion     = "version"
	AttrKeyEnvironment = "environment"
	AttrKeyRequestID   = "request_id"
)
