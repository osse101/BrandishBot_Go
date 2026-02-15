package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateEnv_MissingVersion(t *testing.T) {
	// Unset ENV_SCHEMA_VERSION
	os.Unsetenv("ENV_SCHEMA_VERSION")

	err := ValidateEnv()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ENV_SCHEMA_VERSION is not set")
}

func TestValidateEnv_VersionMismatch(t *testing.T) {
	os.Setenv("ENV_SCHEMA_VERSION", "0.9")
	defer os.Unsetenv("ENV_SCHEMA_VERSION")

	err := ValidateEnv()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ENV_SCHEMA_VERSION mismatch")
	assert.Contains(t, err.Error(), "expected 1.0, got 0.9")
}

func TestValidateEnv_MissingRequired(t *testing.T) {
	// Set version but leave others unset
	os.Setenv("ENV_SCHEMA_VERSION", ExpectedEnvSchemaVersion)
	defer os.Unsetenv("ENV_SCHEMA_VERSION")

	err := ValidateEnv()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required environment variables")
}

func TestValidateEnvWithWarnings_InsecureDefaults(t *testing.T) {
	// Set all required including insecure defaults
	os.Setenv("ENV_SCHEMA_VERSION", ExpectedEnvSchemaVersion)
	os.Setenv("DB_PASSWORD", "change_this_secure_password")
	os.Setenv("API_KEY", "generate_with_openssl_rand_hex_32")
	// Set other DB parts so ValidateEnv passes
	os.Setenv("DB_USER", "user")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_NAME", "db")

	defer func() {
		os.Unsetenv("ENV_SCHEMA_VERSION")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("API_KEY")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_NAME")
	}()

	for _, envVar := range RequiredEnvVars {
		if envVar != "ENV_SCHEMA_VERSION" && envVar != "DB_PASSWORD" && envVar != "API_KEY" {
			os.Setenv(envVar, "test_value")
			defer os.Unsetenv(envVar)
		}
	}

	warnings, err := ValidateEnvWithWarnings()
	require.NoError(t, err, "Should not error even with warnings")
	assert.Len(t, warnings, 2, "Should have 2 warnings")
	if len(warnings) >= 2 {
		assert.Contains(t, warnings[0], "DB_PASSWORD")
		assert.Contains(t, warnings[1], "API_KEY")
	}
}
