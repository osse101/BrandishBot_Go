package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/info"
)

func TestHandleGetInfo(t *testing.T) {
	configDir := "../../configs/info"
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Fatalf("Config directory not found at %s. Ensure you are running tests from project root or correct relative path.", configDir)
	}

	loader := info.NewLoader(configDir)
	handler := HandleGetInfo(loader)

	tests := []struct {
		name               string
		queryPlatform      string
		queryFeature       string
		queryTopic         string
		expectedStatus     int
		expectedSubstrings []string
		expectLink         bool
		expectError        bool
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
			expectLink:     true,
			expectedSubstrings: []string{
				"BrandishBot Features",
			},
		},
		{
			name:           "Twitch - Specific Feature",
			queryPlatform:  domain.PlatformTwitch,
			queryFeature:   "crafting",
			expectedStatus: http.StatusOK,
			expectedSubstrings: []string{
				"!upgrade",
				"!disassemble",
				"!recipes",
			},
			expectLink: false,
		},
		{
			name:           "Discord - General",
			queryPlatform:  domain.PlatformDiscord,
			expectedStatus: http.StatusOK,
			expectLink:     true,
			expectedSubstrings: []string{
				"**BrandishBot Features**",
			},
		},
		{
			name:           "Discord - Specific Feature",
			queryPlatform:  domain.PlatformDiscord,
			queryFeature:   "crafting",
			expectedStatus: http.StatusOK,
			expectedSubstrings: []string{
				"Crafting",
				"**",
			},
			expectLink: false,
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
			if tt.queryTopic != "" {
				url += "&topic=" + tt.queryTopic
			}

			req, err := http.NewRequest("GET", url, nil)
			assert.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectError {
				// Verify we got an error response
				body := rr.Body.String()
				assert.NotEmpty(t, body)
				// If platform is missing, check for specific error message
				if tt.queryPlatform == "" {
					assert.Contains(t, body, "Missing platform query parameter")
				}
			} else {
				var resp InfoResponse
				err = json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.NoError(t, err)

				for _, sub := range tt.expectedSubstrings {
					assert.Contains(t, resp.Description, sub, "Description should contain expected substring")
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
