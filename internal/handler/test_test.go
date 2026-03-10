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
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHandleTest(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		requestBody    interface{}
		setupMock      func(*mocks.MockUserService)
		expectedStatus int
		verifyBody     func(*testing.T, string)
	}{
		{
			name:   "Success",
			method: http.MethodPost,
			requestBody: TestRequest{
				Username:   "testuser",
				Platform:   domain.PlatformTwitch,
				PlatformID: "twitch-123",
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("HandleIncomingMessage", mock.Anything, domain.PlatformTwitch, "twitch-123", "testuser", "").Return(&domain.MessageResult{}, nil)
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body string) {
				var resp TestResponse
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Equal(t, "Greetings, testuser!", resp.Message)
			},
		},
		{
			name:           "Method Not Allowed",
			method:         http.MethodGet,
			requestBody:    nil,
			setupMock:      func(m *mocks.MockUserService) {},
			expectedStatus: http.StatusMethodNotAllowed,
			verifyBody: func(t *testing.T, body string) {
				assert.Contains(t, body, ErrMsgMethodNotAllowed)
			},
		},
		{
			name:   "Invalid Request - Missing Fields",
			method: http.MethodPost,
			requestBody: TestRequest{
				Username: "testuser",
			},
			setupMock:      func(m *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			verifyBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "Invalid request")
				assert.Contains(t, body, "This field is required")
			},
		},
		{
			name:   "Service Error",
			method: http.MethodPost,
			requestBody: TestRequest{
				Username:   "testuser",
				Platform:   domain.PlatformTwitch,
				PlatformID: "twitch-123",
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("HandleIncomingMessage", mock.Anything, domain.PlatformTwitch, "twitch-123", "testuser", "").Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			verifyBody: func(t *testing.T, body string) {
				assert.Contains(t, body, "Failed to process user: db error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := mocks.NewMockUserService(t)
			tt.setupMock(mockSvc)

			handler := HandleTest(mockSvc)

			var req *http.Request
			if tt.requestBody != nil {
				bodyBytes, _ := json.Marshal(tt.requestBody)
				req = httptest.NewRequest(tt.method, "/test", bytes.NewReader(bodyBytes))
			} else {
				req = httptest.NewRequest(tt.method, "/test", nil)
			}

			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.verifyBody(t, w.Body.String())
		})
	}
}
