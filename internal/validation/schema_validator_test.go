package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchemaValidator_ValidateFile(t *testing.T) {
	validator := NewSchemaValidator()

	// Create temp directory for test files
	tmpDir := t.TempDir()

	// Create a simple test schema
	schemaPath := filepath.Join(tmpDir, "test.schema.json")
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"name": {
				"type": "string"
			},
			"age": {
				"type": "integer",
				"minimum": 0
			}
		},
		"required": ["name"]
	}`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	tests := []struct {
		name      string
		data      string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid data",
			data:      `{"name": "John", "age": 30}`,
			wantError: false,
		},
		{
			name:      "valid data without optional field",
			data:      `{"name": "Jane"}`,
			wantError: false,
		},
		{
			name:      "missing required field",
			data:      `{"age": 25}`,
			wantError: true,
			errorMsg:  "required",
		},
		{
			name:      "wrong type for field",
			data:      `{"name": "John", "age": "thirty"}`,
			wantError: true,
			errorMsg:  "age",
		},
		{
			name:      "constraint violation",
			data:      `{"name": "John", "age": -5}`,
			wantError: true,
			errorMsg:  "age",
		},
		{
			name:      "invalid JSON",
			data:      `{"name": "John", "age": }`,
			wantError: true,
			errorMsg:  "parse JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write test data to file
			dataPath := filepath.Join(tmpDir, "test_data.json")
			if err := os.WriteFile(dataPath, []byte(tt.data), 0644); err != nil {
				t.Fatalf("Failed to write data file: %v", err)
			}

			err := validator.ValidateFile(dataPath, schemaPath)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSchemaValidator_ValidateBytes(t *testing.T) {
	validator := NewSchemaValidator()

	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "test.schema.json")
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "array",
		"items": {
			"type": "object",
			"properties": {
				"id": {"type": "integer"},
				"name": {"type": "string"}
			},
			"required": ["id", "name"]
		}
	}`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	tests := []struct {
		name      string
		data      []byte
		wantError bool
	}{
		{
			name:      "valid array",
			data:      []byte(`[{"id": 1, "name": "Item1"}, {"id": 2, "name": "Item2"}]`),
			wantError: false,
		},
		{
			name:      "empty array",
			data:      []byte(`[]`),
			wantError: false,
		},
		{
			name:      "invalid item in array",
			data:      []byte(`[{"id": 1, "name": "Item1"}, {"id": "two", "name": "Item2"}]`),
			wantError: true,
		},
		{
			name:      "missing required field",
			data:      []byte(`[{"id": 1}]`),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateBytes(tt.data, schemaPath)

			if tt.wantError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestSchemaValidator_InvalidSchemaFile(t *testing.T) {
	validator := NewSchemaValidator()

	tmpDir := t.TempDir()
	dataPath := filepath.Join(tmpDir, "data.json")
	if err := os.WriteFile(dataPath, []byte(`{}`), 0644); err != nil {
		t.Fatalf("Failed to write data file: %v", err)
	}

	// Test with non-existent schema file
	err := validator.ValidateFile(dataPath, "nonexistent.schema.json")
	if err == nil {
		t.Error("Expected error for non-existent schema file")
	}
	if !strings.Contains(err.Error(), "failed to load schema") {
		t.Errorf("Expected 'failed to load schema' error, got: %v", err)
	}
}

func TestSchemaValidator_InvalidDataFile(t *testing.T) {
	validator := NewSchemaValidator()

	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "test.schema.json")
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object"
	}`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	// Test with non-existent data file
	err := validator.ValidateFile("nonexistent.json", schemaPath)
	if err == nil {
		t.Error("Expected error for non-existent data file")
	}
	if !strings.Contains(err.Error(), "failed to read data file") {
		t.Errorf("Expected 'failed to read data file' error, got: %v", err)
	}
}

func TestSchemaValidator_CachesCompiledSchemas(t *testing.T) {
	v := NewSchemaValidator().(*validator)

	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "test.schema.json")
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object"
	}`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	// First validation should compile and cache the schema
	data := []byte(`{"test": "value"}`)
	if err := v.ValidateBytes(data, schemaPath); err != nil {
		t.Fatalf("First validation failed: %v", err)
	}

	if len(v.schemas) != 1 {
		t.Errorf("Expected 1 cached schema, got %d", len(v.schemas))
	}

	// Second validation should use cached schema
	if err := v.ValidateBytes(data, schemaPath); err != nil {
		t.Fatalf("Second validation failed: %v", err)
	}

	if len(v.schemas) != 1 {
		t.Errorf("Expected 1 cached schema after second validation, got %d", len(v.schemas))
	}
}

func TestSchemaValidator_EnumValidation(t *testing.T) {
	validator := NewSchemaValidator()

	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "enum.schema.json")
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"status": {
				"type": "string",
				"enum": ["active", "inactive", "pending"]
			}
		},
		"required": ["status"]
	}`
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	tests := []struct {
		name      string
		data      string
		wantError bool
	}{
		{
			name:      "valid enum value",
			data:      `{"status": "active"}`,
			wantError: false,
		},
		{
			name:      "invalid enum value",
			data:      `{"status": "invalid"}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateBytes([]byte(tt.data), schemaPath)

			if tt.wantError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
