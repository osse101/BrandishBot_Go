package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDBURL(t *testing.T) {
	// Do not use t.Parallel() because we are modifying global environment variables

	tests := []struct {
		name     string
		envVars  map[string]string
		expected string
	}{
		{
			name: "DB_URL is explicitly set",
			envVars: map[string]string{
				"DB_URL": "postgres://explicit:pass@myhost:1234/mydb?sslmode=require",
			},
			expected: "postgres://explicit:pass@myhost:1234/mydb?sslmode=require",
		},
		{
			name: "fallback to default env vars",
			envVars: map[string]string{
				"DB_USER":     "myuser",
				"DB_PASSWORD": "mypassword",
				"DB_HOST":     "dbhost",
				"DB_PORT":     "5433",
				"DB_NAME":     "testdb",
			},
			expected: "postgres://myuser:mypassword@dbhost:5433/testdb?sslmode=disable",
		},
		{
			name:     "fallback to defaults when no env vars set",
			envVars:  map[string]string{},
			expected: "postgres://dev:change_this_secure_password@localhost:5432/app?sslmode=disable",
		},
		{
			name: "partial env vars",
			envVars: map[string]string{
				"DB_HOST": "remote",
			},
			expected: "postgres://dev:change_this_secure_password@remote:5432/app?sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear relevant env vars before each test to ensure clean state
			t.Setenv("DB_URL", "")
			t.Setenv("DB_USER", "")
			t.Setenv("DB_PASSWORD", "")
			t.Setenv("DB_HOST", "")
			t.Setenv("DB_PORT", "")
			t.Setenv("DB_NAME", "")
			os.Unsetenv("DB_URL")
			os.Unsetenv("DB_USER")
			os.Unsetenv("DB_PASSWORD")
			os.Unsetenv("DB_HOST")
			os.Unsetenv("DB_PORT")
			os.Unsetenv("DB_NAME")

			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			result := GetDBURL()
			assert.Equal(t, tt.expected, result)
		})
	}
}
