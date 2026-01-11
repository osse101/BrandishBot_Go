package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/features"
	"github.com/stretchr/testify/assert"
)

func TestHandleGetInfo(t *testing.T) {
	configDir := "../../configs/info"
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Fatalf("Config directory not found at %s. Ensure you are running tests from project root or correct relative path.", configDir)
	}

	loader := features.NewLoader(configDir)
	handler := HandleGetInfo(loader)

	tests := []struct {
		name           string
		queryPlatform  string
		queryFeature   string
		expectedStatus int
		// We verify the FORMATTING added by the code, not the file content
		expectedPrefix string
		expectedSubstrings []string
		expectLink     bool
		expectError    bool
	}{
		{
			name:           "Missing platform",
			queryPlatform:  "",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Twitch - General",
			queryPlatform:  domain.PlatformTwitch,
			expectedStatus: http.StatusOK,
			expectedPrefix: "BrandishBot Features",
			expectLink:     true,
		},
		{
			name:           "Twitch - Specific Feature",
			queryPlatform:  domain.PlatformTwitch,
			queryFeature:   "crafting",
			expectedStatus: http.StatusOK,
			expectedPrefix: "[CRAFTING]", // Code adds [NAME]
			expectLink:     false,
		},
		{
			name:           "Discord - General",
			queryPlatform:  domain.PlatformDiscord,
			expectedStatus: http.StatusOK,
			expectedPrefix: "**BrandishBot Features**", // Code adds bold title
			expectLink:     true,
		},
		{
			name:           "Discord - Specific Feature",
			queryPlatform:  domain.PlatformDiscord,
			queryFeature:   "crafting",
			expectedStatus: http.StatusOK,
			expectedPrefix: "# CRAFTING", // Code adds Header 1
			expectedSubstrings: []string{
				"> ",           // Code adds blockquote for description
				"**Commands**", // Code adds Commands header
				"• `",          // Code adds bullet points for commands
			},
			expectLink:     false,
		},
		{
			name:           "Unknown Feature",
			queryPlatform:  domain.PlatformDiscord,
			queryFeature:   "nonexistent",
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/v1/info?platform=" + tt.queryPlatform
			if tt.queryFeature != "" {
				url += "&feature=" + tt.queryFeature
			}

			req, err := http.NewRequest("GET", url, nil)
			assert.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectError {
				var resp ErrorResponse
				err = json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.NotEmpty(t, resp.Error)
			} else {
				var resp InfoResponse
				err = json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.NoError(t, err)
				
				if tt.expectedPrefix != "" {
					assert.Contains(t, resp.Description, tt.expectedPrefix, "Description should start with/contain expected prefix/header")
				}
				
				for _, sub := range tt.expectedSubstrings {
					assert.Contains(t, resp.Description, sub, "Description should contain formatting element")
				}
				
				if tt.expectLink {
					assert.NotEmpty(t, resp.Link, "Expected link to be present")
				} else {
					assert.Empty(t, resp.Link, "Expected link to be empty")
				}
				assert.Equal(t, tt.queryPlatform, resp.Platform)
			}
		})
	}
}
