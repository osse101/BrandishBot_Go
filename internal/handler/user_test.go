package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleRegisterUser(t *testing.T) {
	InitValidator()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success - New User",
			requestBody: RegisterUserRequest{
				Username:        "newuser",
				KnownPlatform:   "twitch",
				KnownPlatformID: "12345",
				NewPlatform:     "discord",
				NewPlatformID:   "67890",
			},
			setupMock: func(m *MockUserService) {
				// FindUserByPlatformID returns error -> user not found
				m.On("FindUserByPlatformID", mock.Anything, "twitch", "12345").Return(nil, errors.New("not found"))

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
				KnownPlatform:   "twitch",
				KnownPlatformID: "12345",
				NewPlatform:     "discord",
				NewPlatformID:   "67890",
			},
			setupMock: func(m *MockUserService) {
				// FindUserByPlatformID returns existing user
				existingUser := &domain.User{ID: "existing-id", Username: "existinguser", TwitchID: "12345"}
				m.On("FindUserByPlatformID", mock.Anything, "twitch", "12345").Return(existingUser, nil)

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
			setupMock:      func(m *MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Service Error - Register Failed",
			requestBody: RegisterUserRequest{
				Username:        "erroruser",
				KnownPlatform:   "twitch",
				KnownPlatformID: "12345",
				NewPlatform:     "discord",
				NewPlatformID:   "67890",
			},
			setupMock: func(m *MockUserService) {
				m.On("FindUserByPlatformID", mock.Anything, "twitch", "12345").Return(nil, errors.New("not found"))
				m.On("RegisterUser", mock.Anything, mock.Anything).Return(domain.User{}, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to register user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockUserService{}
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
