package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

func TestJSONLogging(t *testing.T) {
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
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON log: %v", err)
	}
	
	// Verify base attributes
	if logEntry["service"] != "test-service" {
		t.Errorf("Expected service=test-service, got %v", logEntry["service"])
	}
	
	if logEntry["version"] != "1.0.0" {
		t.Errorf("Expected version=1.0.0, got %v", logEntry["version"])
	}
	
	if logEntry["environment"] != "test" {
		t.Errorf("Expected environment=test, got %v", logEntry["environment"])
	}
	
	// Verify message
	if logEntry["msg"] != "test message" {
		t.Errorf("Expected msg='test message', got %v", logEntry["msg"])
	}
	
	// Verify level
	if logEntry["level"] != "INFO" {
		t.Errorf("Expected level=INFO, got %v", logEntry["level"])
	}
	
	// Verify custom attributes
	if logEntry["key"] != "value" {
		t.Errorf("Expected key=value, got %v", logEntry["key"])
	}
	
	if logEntry["number"] != float64(42) {
		t.Errorf("Expected number=42, got %v", logEntry["number"])
	}
}

func TestRequestIDContext(t *testing.T) {
	ctx := WithRequestID(context.Background(), "test-req-123")
	
	requestID := GetRequestID(ctx)
	if requestID != "test-req-123" {
		t.Errorf("Expected request_id=test-req-123, got %s", requestID)
	}
	
	// Test with logger
	log := FromContext(ctx)
	if log == nil {
		t.Error("Expected non-nil logger")
	}
}

func TestConfigDefaults(t *testing.T) {
	config := DefaultConfig()
	
	if config.ServiceName == "" {
		t.Error("Expected non-empty service name")
	}
	
	if config.Level == "" {
		t.Error("Expected non-empty log level")
	}
	
	if config.Format == "" {
		t.Error("Expected non-empty format")
	}
}

func TestProductionConfig(t *testing.T) {
	config := ProductionConfig()
	
	if config.Format != "json" {
		t.Errorf("Expected JSON format in prod, got %s", config.Format)
	}
	
	if config.Level != "info" {
		t.Errorf("Expected info level in prod, got %s", config.Level)
	}
	
	if config.Environment != "prod" {
		t.Errorf("Expected prod environment, got %s", config.Environment)
	}
	
	if config.AddSource {
		t.Error("Expected AddSource=false in production")
	}
}

func TestDevelopmentConfig(t *testing.T) {
	config := DevelopmentConfig()
	
	if config.Format != "text" {
		t.Errorf("Expected text format in dev, got %s", config.Format)
	}
	
	if config.Level != "debug" {
		t.Errorf("Expected debug level in dev, got %s", config.Level)
	}
	
	if !config.AddSource {
		t.Error("Expected AddSource=true in development")
	}
}
