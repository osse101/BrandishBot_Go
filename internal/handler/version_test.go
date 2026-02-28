package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleVersion(t *testing.T) {
	tests := []struct {
		name       string
		setupEnv   func()
		cleanupEnv func()
		verifyBody func(*testing.T, string)
	}{
		{
			name: "Success - Dev Version (Default)",
			setupEnv: func() {
				// Ensure Version is "dev"
				Version = "dev"
			},
			cleanupEnv: func() {},
			verifyBody: func(t *testing.T, body string) {
				var info VersionInfo
				err := json.Unmarshal([]byte(body), &info)
				require.NoError(t, err)

				assert.Equal(t, "dev", info.Version)
				assert.Equal(t, runtime.Version(), info.GoVersion)
				assert.Equal(t, BuildTime, info.BuildTime)
				assert.Equal(t, GitCommit, info.GitCommit)
			},
		},
		{
			name: "Success - Env Version",
			setupEnv: func() {
				Version = "dev"
				os.Setenv("VERSION", "1.2.3")
			},
			cleanupEnv: func() {
				os.Unsetenv("VERSION")
			},
			verifyBody: func(t *testing.T, body string) {
				var info VersionInfo
				err := json.Unmarshal([]byte(body), &info)
				require.NoError(t, err)

				assert.Equal(t, "1.2.3", info.Version)
				assert.Equal(t, runtime.Version(), info.GoVersion)
			},
		},
		{
			name: "Success - Build Version",
			setupEnv: func() {
				Version = "2.0.0"
				os.Setenv("VERSION", "1.2.3") // Build version should take precedence
			},
			cleanupEnv: func() {
				Version = "dev"
				os.Unsetenv("VERSION")
			},
			verifyBody: func(t *testing.T, body string) {
				var info VersionInfo
				err := json.Unmarshal([]byte(body), &info)
				require.NoError(t, err)

				assert.Equal(t, "2.0.0", info.Version)
				assert.Equal(t, runtime.Version(), info.GoVersion)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			req := httptest.NewRequest("GET", "/version", nil)
			w := httptest.NewRecorder()

			handler := HandleVersion()
			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			tt.verifyBody(t, w.Body.String())
		})
	}
}

func TestGetVersionInfo(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		env      string
		expected string
	}{
		{
			name:     "Default",
			version:  "dev",
			env:      "",
			expected: "dev",
		},
		{
			name:     "Build Version",
			version:  "1.0.0",
			env:      "",
			expected: "1.0.0",
		},
		{
			name:     "Env Version",
			version:  "dev",
			env:      "1.1.0",
			expected: "1.1.0",
		},
		{
			name:     "Build Version Precedence",
			version:  "1.2.0",
			env:      "1.1.0",
			expected: "1.2.0",
		},
		{
			name:     "Empty Build Version",
			version:  "",
			env:      "1.3.0",
			expected: "1.3.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.version
			if tt.env != "" {
				os.Setenv("VERSION", tt.env)
			} else {
				os.Unsetenv("VERSION")
			}

			result := getVersionInfo()
			assert.Equal(t, tt.expected, result)
		})
	}
	// Reset
	Version = "dev"
	os.Unsetenv("VERSION")
}
