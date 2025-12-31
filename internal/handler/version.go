package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"runtime"
)

// VersionInfo contains version and build information
type VersionInfo struct {
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
	BuildTime string `json:"build_time,omitempty"`
	GitCommit string `json:"git_commit,omitempty"`
}

// Build-time variables (injected via ldflags)
var (
	Version   = "dev"          // Set via -X flag at build time
	BuildTime = "unknown"      // Set via -X flag at build time
	GitCommit = "unset"        // Set via -X flag at build time
)

// HandleVersion returns version information about the application
// This makes it easy to verify which version is deployed
func HandleVersion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		info := VersionInfo{
			Version:   getVersionInfo(),
			GoVersion: runtime.Version(),
			BuildTime: BuildTime,
			GitCommit: GitCommit,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}
}

// getVersionInfo returns version from build-time variable or environment
func getVersionInfo() string {
	// Priority: build-time > environment > default
	if Version != "dev" && Version != "" {
		return Version
	}
	if envVersion := os.Getenv("VERSION"); envVersion != "" {
		return envVersion
	}
	return "dev"
}
