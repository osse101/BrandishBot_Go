package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONLogging(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	config := Config{
		Level:       "info",
		Format:      "json",
		ServiceName: "test-service",
		Version:     "1.0.0",
		Environment: "test",
		AddSource:   false,
	}

	InitLoggerWithWriter(config, &buf)

	// Log a test message
	Info("test message", "key", "value", "number", 42)

	// Parse JSON output
	var logEntry map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err, "Failed to parse JSON log")

	// Verify base attributes
	assert.Equal(t, "test-service", logEntry["service"])
	assert.Equal(t, "1.0.0", logEntry["version"])
	assert.Equal(t, "test", logEntry["environment"])

	// Verify message
	assert.Equal(t, "test message", logEntry["msg"])

	// Verify level
	assert.Equal(t, "INFO", logEntry["level"])

	// Verify custom attributes
	assert.Equal(t, "value", logEntry["key"])
	assert.Equal(t, float64(42), logEntry["number"])
}

func TestRequestIDContext(t *testing.T) {
	t.Parallel()

	ctx := WithRequestID(context.Background(), "test-req-123")

	requestID := GetRequestID(ctx)
	assert.Equal(t, "test-req-123", requestID)

	// Test with logger
	log := FromContext(ctx)
	assert.NotNil(t, log)
}

func TestConfigDefaults(t *testing.T) {
	t.Parallel()

	config := DefaultConfig()

	assert.NotEmpty(t, config.ServiceName)
	assert.NotEmpty(t, config.Level)
	assert.NotEmpty(t, config.Format)
}

func TestProductionConfig(t *testing.T) {
	t.Parallel()

	config := ProductionConfig()

	assert.Equal(t, "json", config.Format)
	assert.Equal(t, "info", config.Level)
	assert.Equal(t, "prod", config.Environment)
	assert.False(t, config.AddSource)
}

func TestDevelopmentConfig(t *testing.T) {
	t.Parallel()

	config := DevelopmentConfig()

	assert.Equal(t, "text", config.Format)
	assert.Equal(t, "debug", config.Level)
	assert.True(t, config.AddSource)
}

func TestGenerateRequestID(t *testing.T) {
	t.Parallel()

	id := GenerateRequestID()
	assert.NotEmpty(t, id)
	// format is UUIDFormatPattern
	// Should be 36 characters long e.g. "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
	assert.Len(t, id, 36)

	// Test uniqueness
	id2 := GenerateRequestID()
	assert.NotEqual(t, id, id2)
}

func TestWithUser(t *testing.T) {
	t.Parallel()

	ctx := WithUser(context.Background(), "user123", "testuser")

	userID, username := GetUser(ctx)
	assert.Equal(t, "user123", userID)
	assert.Equal(t, "testuser", username)

	// Test with logger
	log := FromContext(ctx)
	assert.NotNil(t, log)
}
