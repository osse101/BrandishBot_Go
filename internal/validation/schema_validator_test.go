package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaValidator_ValidateFile(t *testing.T) {
	t.Parallel()

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
	err := os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err, "Failed to write schema file")

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
		// Boundary cases for minimum: 0
		{
			name:      "age exactly minimum boundary",
			data:      `{"name": "John", "age": 0}`,
			wantError: false,
		},
		{
			name:      "age just inside minimum boundary",
			data:      `{"name": "John", "age": 1}`,
			wantError: false,
		},
		{
			name:      "age just outside minimum boundary",
			data:      `{"name": "John", "age": -1}`,
			wantError: true,
			errorMsg:  "age",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validator := NewSchemaValidator()
			// Write test data to file
			subDir := t.TempDir()
			dataPath := filepath.Join(subDir, "test_data.json")
			err := os.WriteFile(dataPath, []byte(tt.data), 0644)
			require.NoError(t, err, "Failed to write data file")

			err = validator.ValidateFile(dataPath, schemaPath)

			if tt.wantError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.ErrorContains(t, err, tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSchemaValidator_ValidateBytes(t *testing.T) {
	t.Parallel()

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
	err := os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err, "Failed to write schema file")

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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validator := NewSchemaValidator()
			err := validator.ValidateBytes(tt.data, schemaPath)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSchemaValidator_InvalidSchemaFile(t *testing.T) {
	t.Parallel()
	validator := NewSchemaValidator()

	tmpDir := t.TempDir()
	dataPath := filepath.Join(tmpDir, "data.json")
	err := os.WriteFile(dataPath, []byte(`{}`), 0644)
	require.NoError(t, err, "Failed to write data file")

	// Test with non-existent schema file
	err = validator.ValidateFile(dataPath, "nonexistent.schema.json")
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to load schema")
}

func TestSchemaValidator_InvalidDataFile(t *testing.T) {
	t.Parallel()
	validator := NewSchemaValidator()

	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "test.schema.json")
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object"
	}`
	err := os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err, "Failed to write schema file")

	// Test with non-existent data file
	err = validator.ValidateFile("nonexistent.json", schemaPath)
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to read data file")
}

func TestSchemaValidator_CachesCompiledSchemas(t *testing.T) {
	t.Parallel()
	v := NewSchemaValidator().(*validator)

	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "test.schema.json")
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object"
	}`
	err := os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err, "Failed to write schema file")

	// First validation should compile and cache the schema
	data := []byte(`{"test": "value"}`)
	err = v.ValidateBytes(data, schemaPath)
	require.NoError(t, err, "First validation failed")
	assert.Len(t, v.schemas, 1)

	// Second validation should use cached schema
	err = v.ValidateBytes(data, schemaPath)
	require.NoError(t, err, "Second validation failed")
	assert.Len(t, v.schemas, 1)
}

func TestSchemaValidator_EnumValidation(t *testing.T) {
	t.Parallel()

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
	err := os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err, "Failed to write schema file")

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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validator := NewSchemaValidator()
			err := validator.ValidateBytes([]byte(tt.data), schemaPath)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
