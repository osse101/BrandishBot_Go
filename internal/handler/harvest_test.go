package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/handler"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestMain(m *testing.M) {
	// Initialize validator once for all tests in this package to avoid race conditions
	handler.InitValidator()
	os.Exit(m.Run())
}

func TestHarvestHandler_Harvest(t *testing.T) {
	t.Parallel()

	// Define test cases
	tests := []struct {
		name           string
		method         string
		requestBody    interface{} // Use interface{} to allow invalid JSON/types
		setupMock      func(*mocks.MockHarvestService)
		expectedStatus int
		expectedError  string
		expectedBody   interface{}
	}{
		{
			name:   "Success",
			method: http.MethodPost,
			requestBody: handler.HarvestRewardsRequest{
				Username:   "testuser",
				Platform:   domain.PlatformDiscord,
				PlatformID: "12345",
			},
			setupMock: func(m *mocks.MockHarvestService) {
				m.On("Harvest", mock.Anything, domain.PlatformDiscord, "12345", "testuser").
					Return(&domain.HarvestResponse{
						ItemsGained:       map[string]int{"gold": 100},
						HoursSinceHarvest: 5.0,
						NextHarvestAt:     time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
						Message:           "Harvest successful!",
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: &domain.HarvestResponse{
				ItemsGained:       map[string]int{"gold": 100},
				HoursSinceHarvest: 5.0,
				NextHarvestAt:     time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
				Message:           "Harvest successful!",
			},
		},
		{
			name:   "Harvest Too Soon",
			method: http.MethodPost,
			requestBody: handler.HarvestRewardsRequest{
				Username:   "eageruser",
				Platform:   domain.PlatformDiscord,
				PlatformID: "12345",
			},
			setupMock: func(m *mocks.MockHarvestService) {
				m.On("Harvest", mock.Anything, domain.PlatformDiscord, "12345", "eageruser").
					Return(nil, domain.ErrHarvestTooSoon)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  domain.ErrMsgHarvestTooSoon,
		},
		{
			name:   "User Not Found",
			method: http.MethodPost,
			requestBody: handler.HarvestRewardsRequest{
				Username:   "unknown",
				Platform:   domain.PlatformDiscord,
				PlatformID: "99999",
			},
			setupMock: func(m *mocks.MockHarvestService) {
				m.On("Harvest", mock.Anything, domain.PlatformDiscord, "99999", "unknown").
					Return(nil, domain.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "User not found",
		},
		{
			name:   "Feature Locked",
			method: http.MethodPost,
			requestBody: handler.HarvestRewardsRequest{
				Username:   "newuser",
				Platform:   domain.PlatformDiscord,
				PlatformID: "12345",
			},
			setupMock: func(m *mocks.MockHarvestService) {
				m.On("Harvest", mock.Anything, domain.PlatformDiscord, "12345", "newuser").
					Return(nil, domain.ErrFeatureLocked)
			},
			expectedStatus: http.StatusForbidden,
			expectedError:  "feature is locked",
		},
		{
			name:   "Internal Server Error",
			method: http.MethodPost,
			requestBody: handler.HarvestRewardsRequest{
				Username:   "broken",
				Platform:   domain.PlatformDiscord,
				PlatformID: "12345",
			},
			setupMock: func(m *mocks.MockHarvestService) {
				m.On("Harvest", mock.Anything, domain.PlatformDiscord, "12345", "broken").
					Return(nil, domain.ErrDatabaseError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "something went wrong",
		},
		{
			name:   "Invalid Method",
			method: http.MethodGet,
			requestBody: handler.HarvestRewardsRequest{
				Username:   "testuser",
				Platform:   domain.PlatformDiscord,
				PlatformID: "12345",
			},
			setupMock:      func(m *mocks.MockHarvestService) {}, // No mock call expected
			expectedStatus: http.StatusMethodNotAllowed,
			expectedError:  "method not allowed",
		},
		{
			name:        "Invalid Body (Malformed JSON)",
			method:      http.MethodPost,
			requestBody: "invalid-json", // passing string which will be written as raw body
			setupMock:   func(m *mocks.MockHarvestService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
		},
		{
			name:   "Validation Error (Missing Fields)",
			method: http.MethodPost,
			requestBody: handler.HarvestRewardsRequest{
				Username:   "", // Required field missing
				Platform:   domain.PlatformDiscord,
				PlatformID: "12345",
			},
			setupMock:      func(m *mocks.MockHarvestService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request",
		},
		{
			name:   "Validation Error (Invalid Platform)",
			method: http.MethodPost,
			requestBody: handler.HarvestRewardsRequest{
				Username:   "testuser",
				Platform:   "invalid_platform",
				PlatformID: "12345",
			},
			setupMock:      func(m *mocks.MockHarvestService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request",
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Arrange
			mockSvc := mocks.NewMockHarvestService(t)
			if tt.setupMock != nil {
				tt.setupMock(mockSvc)
			}

			h := handler.NewHarvestHandler(mockSvc)

			var body []byte
			var err error
			if s, ok := tt.requestBody.(string); ok {
				body = []byte(s)
			} else {
				body, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest(tt.method, "/harvest", bytes.NewReader(body))
			w := httptest.NewRecorder()

			// Act
			h.Harvest(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				// Parse JSON error response if possible
				var errResp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &errResp); err == nil {
					// Check 'error' field first
					if msg, ok := errResp["error"].(string); ok {
						assert.Contains(t, strings.ToLower(msg), strings.ToLower(tt.expectedError))
					} else {
						// Validation errors might be in "fields" or elsewhere, but title might match
						// If struct of error is different, fall back to string check of the whole body
						assert.Contains(t, strings.ToLower(w.Body.String()), strings.ToLower(tt.expectedError))
					}
				} else {
					// Fallback if not valid JSON
					assert.Contains(t, strings.ToLower(w.Body.String()), strings.ToLower(tt.expectedError))
				}
			}

			if tt.expectedBody != nil {
				// Verify Content-Type
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

				var response domain.HarvestResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				// Compare fields individually or use Equal with pointer handling
				expected := tt.expectedBody.(*domain.HarvestResponse)
				assert.Equal(t, expected.ItemsGained, response.ItemsGained)
				assert.InDelta(t, expected.HoursSinceHarvest, response.HoursSinceHarvest, 0.001)
				assert.Equal(t, expected.Message, response.Message)
				// Time might differ due to serialization/deserialization precision
				assert.WithinDuration(t, expected.NextHarvestAt, response.NextHarvestAt, time.Second)
			}
		})
	}
}
