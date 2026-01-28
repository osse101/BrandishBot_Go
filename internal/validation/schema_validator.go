package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// SchemaValidator validates JSON data against JSON schemas
type SchemaValidator interface {
	ValidateFile(dataPath, schemaPath string) error
	ValidateBytes(data []byte, schemaPath string) error
}

type validator struct {
	compiler *jsonschema.Compiler
	schemas  map[string]*jsonschema.Schema
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() SchemaValidator {
	return &validator{
		compiler: jsonschema.NewCompiler(),
		schemas:  make(map[string]*jsonschema.Schema),
	}
}

// ValidateFile validates a JSON file against a schema file
func (v *validator) ValidateFile(dataPath, schemaPath string) error {
	// Read data file
	data, err := os.ReadFile(dataPath)
	if err != nil {
		return fmt.Errorf("failed to read data file %s: %w", dataPath, err)
	}

	return v.ValidateBytes(data, schemaPath)
}

// ValidateBytes validates JSON data bytes against a schema file
func (v *validator) ValidateBytes(data []byte, schemaPath string) error {
	// Load and compile schema (cached)
	schema, err := v.loadSchema(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to load schema %s: %w", schemaPath, err)
	}

	// Parse JSON data
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return fmt.Errorf("failed to parse JSON data: %w", err)
	}

	// Validate data against schema
	if err := schema.Validate(jsonData); err != nil {
		return formatValidationError(err)
	}

	return nil
}

// loadSchema loads and compiles a schema, caching the result
func (v *validator) loadSchema(schemaPath string) (*jsonschema.Schema, error) {
	// Check cache
	if schema, ok := v.schemas[schemaPath]; ok {
		return schema, nil
	}

	// Resolve schema path (handle both absolute and relative paths)
	resolvedPath, err := resolveSchemaPath(schemaPath)
	if err != nil {
		return nil, err
	}

	// Read schema file
	schemaData, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	// Parse schema
	var schemaJSON interface{}
	if err := json.Unmarshal(schemaData, &schemaJSON); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	// Compile schema
	if err := v.compiler.AddResource(schemaPath, schemaJSON); err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}

	schema, err := v.compiler.Compile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	// Cache compiled schema
	v.schemas[schemaPath] = schema

	return schema, nil
}

// formatValidationError formats validation errors to be user-friendly
func formatValidationError(err error) error {
	if validationErr, ok := err.(*jsonschema.ValidationError); ok {
		var errors []string
		collectErrors(validationErr, &errors)
		return fmt.Errorf("schema validation failed:\n%s", strings.Join(errors, "\n"))
	}
	return fmt.Errorf("validation error: %w", err)
}

// collectErrors recursively collects all validation errors
func collectErrors(err *jsonschema.ValidationError, errors *[]string) {
	// Format the error message
	msg := formatError(err)
	if msg != "" {
		*errors = append(*errors, msg)
	}

	// Recursively collect errors from causes
	for _, cause := range err.Causes {
		collectErrors(cause, errors)
	}
}

// formatError formats a single validation error
func formatError(err *jsonschema.ValidationError) string {
	// Get instance location (path to the invalid data)
	location := strings.Join(err.InstanceLocation, "/")
	if location == "" {
		location = "(root)"
	} else {
		location = "/" + location
	}

	// Get the keyword path to understand what validation failed
	keywords := ""
	if err.ErrorKind != nil {
		keywordPath := err.ErrorKind.KeywordPath()
		if len(keywordPath) > 0 {
			keywords = strings.Join(keywordPath, ".")
		}
	}

	// Build error message
	var msg string
	if keywords != "" {
		msg = fmt.Sprintf("  - at %s: %s validation failed", location, keywords)
	} else {
		msg = fmt.Sprintf("  - at %s: validation failed", location)
	}

	return msg
}

// resolveSchemaPath resolves a schema path, handling both absolute and relative paths
// For relative paths, it searches upward from the current directory to find the project root
func resolveSchemaPath(schemaPath string) (string, error) {
	// If absolute path, use as-is
	if filepath.IsAbs(schemaPath) {
		return schemaPath, nil
	}

	// Try current working directory first
	if _, err := os.Stat(schemaPath); err == nil {
		return schemaPath, nil
	}

	// Try to find project root by looking for go.mod
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Walk up the directory tree looking for go.mod
	dir := cwd
	for {
		// Check if schema exists relative to this directory
		testPath := filepath.Join(dir, schemaPath)
		if _, err := os.Stat(testPath); err == nil {
			return testPath, nil
		}

		// Check if we found go.mod (project root)
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// Found project root, use schema path from here
			rootPath := filepath.Join(dir, schemaPath)
			if _, err := os.Stat(rootPath); err == nil {
				return rootPath, nil
			}
			return "", fmt.Errorf("schema file not found: %s", schemaPath)
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("schema file not found: %s (searched from %s)", schemaPath, cwd)
}
