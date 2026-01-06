package config

import (
	"fmt"
	"os"
	"strings"
)

// ExpectedEnvSchemaVersion is the schema version that the application expects
const ExpectedEnvSchemaVersion = "1.0"

// RequiredEnvVars lists all environment variables that must be set
var RequiredEnvVars = []string{
	"ENV_SCHEMA_VERSION",
	"DB_USER",
	"DB_PASSWORD",
	"DB_HOST",
	"DB_PORT",
	"DB_NAME",
	"API_KEY",
	"DISCORD_TOKEN",
	"DISCORD_APP_ID",
	"DISCORD_PUBLIC_KEY",
}

// ValidateEnv checks that all required environment variables are set
// and that the schema version matches expectations
func ValidateEnv() error {
	// Check schema version first
	schemaVersion := os.Getenv("ENV_SCHEMA_VERSION")
	if schemaVersion == "" {
		return fmt.Errorf("ENV_SCHEMA_VERSION is not set - please update your .env file to include this field (expected: %s)", ExpectedEnvSchemaVersion)
	}
	
	if schemaVersion != ExpectedEnvSchemaVersion {
		return fmt.Errorf("ENV_SCHEMA_VERSION mismatch: expected %s, got %s - your .env file may be outdated", ExpectedEnvSchemaVersion, schemaVersion)
	}

	// Check all required variables
	var missing []string
	for _, envVar := range RequiredEnvVars {
		if os.Getenv(envVar) == "" {
			missing = append(missing, envVar)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

// ValidateEnvWithWarnings checks environment variables and returns warnings
// for non-critical issues (like using default values)
func ValidateEnvWithWarnings() ([]string, error) {
	// First do the critical validation
	if err := ValidateEnv(); err != nil {
		return nil, err
	}

	var warnings []string

	// Check for potentially insecure default values
	if os.Getenv("DB_PASSWORD") == "change_this_secure_password" {
		warnings = append(warnings, "DB_PASSWORD appears to be using the example value - please use a secure password")
	}

	if os.Getenv("API_KEY") == "generate_with_openssl_rand_hex_32" {
		warnings = append(warnings, "API_KEY appears to be using the example value - generate a secure key with: openssl rand -hex 32")
	}

	return warnings, nil
}
