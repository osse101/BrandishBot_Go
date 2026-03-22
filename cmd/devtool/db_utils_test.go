package main

import (
	"os"
	"path/filepath"
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

func TestRotateBackups(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "backup_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some dummy backup files
	// Naming: backup_env_YYYYMMDD_HHMMSS.sql
	backups := []string{
		"backup_staging_20240101_120000.sql",
		"backup_staging_20240102_120000.sql",
		"backup_staging_20240103_120000.sql",
		"backup_staging_20240104_120000.sql",
		"backup_staging_20240105_120000.sql",
		"backup_staging_20240106_120000.sql", // 6th backup, oldest should be removed
	}

	for _, b := range backups {
		err := os.WriteFile(filepath.Join(tmpDir, b), []byte("test"), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Rotate backups, keep 5
	err = RotateBackups(tmpDir, "backup_", 5)
	assert.NoError(t, err)

	// Verify files
	files, err := os.ReadDir(tmpDir)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(files))

	// Oldest (20240101) should be gone
	_, err = os.Stat(filepath.Join(tmpDir, "backup_staging_20240101_120000.sql"))
	assert.True(t, os.IsNotExist(err))

	// Newest (20240106) should remain
	_, err = os.Stat(filepath.Join(tmpDir, "backup_staging_20240106_120000.sql"))
	assert.NoError(t, err)
}
