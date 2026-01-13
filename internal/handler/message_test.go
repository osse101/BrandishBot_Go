package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHandleMessageHandler(t *testing.T) {
	// Initialize validator
	InitValidator()

	tests := []struct {
		name           string
		method         string
		body           interface{}
		setupMocks     func(*mocks.MockUserService, *mocks.MockProgressionService, *mocks.MockEventBus)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "Success",
			method: http.MethodPost,
			body: HandleMessageRequest{
				Platform:   "twitch",
				PlatformID: "123",
				Username:   "testuser",
				Message:    "hello",
			},
			setupMocks: func(mu *mocks.MockUserService, mp *mocks.MockProgressionService, me *mocks.MockEventBus) {
				mu.On("HandleIncomingMessage", mock.Anything, "twitch", "123", "testuser", "hello").
					Return(&domain.MessageResult{
						User: domain.User{ID: "user-123", Username: "testuser"},
					}, nil)

				// Expect engagement tracking event
				me.On("Publish", mock.Anything, mock.MatchedBy(func(evt event.Event) bool {
					if evt.Type != "engagement" {
						return false
					}
					payload, ok := evt.Payload.(*domain.EngagementMetric)
					return ok && payload.MetricType == "message"
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"user":{"internal_id":"user-123"`,
		},
		{
			name:           "Invalid Method",
			method:         http.MethodGet,
			body:           nil,
			setupMocks:     func(mu *mocks.MockUserService, mp *mocks.MockProgressionService, me *mocks.MockEventBus) {},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
		{
			name:           "Invalid Body (Malformed JSON)",
			method:         http.MethodPost,
			body:           "invalid-json", // passing string to fail decode
			setupMocks:     func(mu *mocks.MockUserService, mp *mocks.MockProgressionService, me *mocks.MockEventBus) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name:   "Validation Failure (Missing Fields)",
			method: http.MethodPost,
			body: HandleMessageRequest{
				Platform: "", // Missing required
			},
			setupMocks:     func(mu *mocks.MockUserService, mp *mocks.MockProgressionService, me *mocks.MockEventBus) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Validation failed",
		},
		{
			name:   "Service Error",
			method: http.MethodPost,
			body: HandleMessageRequest{
				Platform:   "twitch",
				PlatformID: "123",
				Username:   "testuser",
				Message:    "error",
			},
			setupMocks: func(mu *mocks.MockUserService, mp *mocks.MockProgressionService, me *mocks.MockEventBus) {
				mu.On("HandleIncomingMessage", mock.Anything, "twitch", "123", "testuser", "error").
					Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to handle message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUser := mocks.NewMockUserService(t)
			mockProgression := mocks.NewMockProgressionService(t)
			mockEvent := mocks.NewMockEventBus(t)

			tt.setupMocks(mockUser, mockProgression, mockEvent)

			handler := HandleMessageHandler(mockUser, mockProgression, mockEvent)

			var reqBody []byte
			if str, ok := tt.body.(string); ok && str == "invalid-json" {
				reqBody = []byte("invalid-json")
			} else if tt.body != nil {
				var err error
				reqBody, err = json.Marshal(tt.body)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(tt.method, "/message/handle", bytes.NewReader(reqBody))
			rec := httptest.NewRecorder()

			// Add basic logger to context to prevent nil pointer if handler uses it
			// (Assuming logger.FromContext handles nil gracefully or we might need to inject one)
			// In this codebase, it seems logger.FromContext retrieves from context or returns default.

			handler(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, rec.Body.String(), tt.expectedBody)
			}

			mockUser.AssertExpectations(t)
			mockProgression.AssertExpectations(t)
			mockEvent.AssertExpectations(t)
		})
	}
}
