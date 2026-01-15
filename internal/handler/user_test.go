package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHandleRegisterUser(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success - New User",
			requestBody: RegisterUserRequest{
				Username:        "newuser",
				KnownPlatform:   domain.PlatformTwitch,
				KnownPlatformID: "12345",
				NewPlatform:     domain.PlatformDiscord,
				NewPlatformID:   "67890",
			},
			setupMock: func(m *mocks.MockUserService) {
				// FindUserByPlatformID returns error -> user not found
				m.On("FindUserByPlatformID", mock.Anything, domain.PlatformTwitch, "12345").Return(nil, errors.New("not found"))

				// Expect RegisterUser with new user data
				m.On("RegisterUser", mock.Anything, mock.MatchedBy(func(u domain.User) bool {
					return u.Username == "newuser" && u.TwitchID == "12345" && u.DiscordID == "67890"
				})).Return(domain.User{ID: "new-id", Username: "newuser"}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `"username":"newuser"`,
		},
		{
			name: "Success - Existing User",
			requestBody: RegisterUserRequest{
				Username:        "existinguser",
				KnownPlatform:   domain.PlatformTwitch,
				KnownPlatformID: "12345",
				NewPlatform:     domain.PlatformDiscord,
				NewPlatformID:   "67890",
			},
			setupMock: func(m *mocks.MockUserService) {
				// FindUserByPlatformID returns existing user
				existingUser := &domain.User{ID: "existing-id", Username: "existinguser", TwitchID: "12345"}
				m.On("FindUserByPlatformID", mock.Anything, domain.PlatformTwitch, "12345").Return(existingUser, nil)

				// Expect RegisterUser with updated user data
				m.On("RegisterUser", mock.Anything, mock.MatchedBy(func(u domain.User) bool {
					return u.ID == "existing-id" && u.TwitchID == "12345" && u.DiscordID == "67890"
				})).Return(*existingUser, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"username":"existinguser"`,
		},
		{
			name: "Invalid Request - Missing Fields",
			requestBody: RegisterUserRequest{
				Username: "badrequest",
			},
			setupMock:      func(m *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Service Error - Register Failed",
			requestBody: RegisterUserRequest{
				Username:        "erroruser",
				KnownPlatform:   domain.PlatformTwitch,
				KnownPlatformID: "12345",
				NewPlatform:     domain.PlatformDiscord,
				NewPlatformID:   "67890",
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("FindUserByPlatformID", mock.Anything, domain.PlatformTwitch, "12345").Return(nil, errors.New("not found"))
				m.On("RegisterUser", mock.Anything, mock.Anything).Return(domain.User{}, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to register user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := mocks.NewMockUserService(t)
			tt.setupMock(mockSvc)

			handler := HandleRegisterUser(mockSvc)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/user/register", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestHandleGetTimeout(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    map[string]string
		setupMock      func(*mocks.MockUserService)
		expectedStatus int
		verifyBody     func(*testing.T, string)
	}{
		{
			name:        "Success - Is Timed Out",
			queryParams: map[string]string{"username": "baduser"},
			setupMock: func(m *mocks.MockUserService) {
				m.On("GetTimeout", mock.Anything, "baduser").Return(time.Duration(60)*time.Second, nil) // 1 minute
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body string) {
				var resp map[string]interface{}
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Equal(t, "baduser", resp["username"])
				assert.Equal(t, true, resp["is_timed_out"])
				assert.Equal(t, 60.0, resp["remaining_seconds"])
			},
		},
		{
			name:        "Success - Not Timed Out",
			queryParams: map[string]string{"username": "gooduser"},
			setupMock: func(m *mocks.MockUserService) {
				m.On("GetTimeout", mock.Anything, "gooduser").Return(time.Duration(0), nil)
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body string) {
				var resp map[string]interface{}
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Equal(t, "gooduser", resp["username"])
				assert.Equal(t, false, resp["is_timed_out"])
				assert.Equal(t, 0.0, resp["remaining_seconds"])
			},
		},
		{
			name:           "Missing Username",
			queryParams:    map[string]string{},
			setupMock:      func(m *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			verifyBody:     func(t *testing.T, body string) {},
		},
		{
			name:        "Service Error",
			queryParams: map[string]string{"username": "erroruser"},
			setupMock: func(m *mocks.MockUserService) {
				m.On("GetTimeout", mock.Anything, "erroruser").Return(time.Duration(0), errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			verifyBody:     func(t *testing.T, body string) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := mocks.NewMockUserService(t)
			tt.setupMock(mockSvc)

			handler := HandleGetTimeout(mockSvc)

			req := httptest.NewRequest("GET", "/user/timeout", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			rec := httptest.NewRecorder()

			handler(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.verifyBody(t, rec.Body.String())
		})
	}
}
