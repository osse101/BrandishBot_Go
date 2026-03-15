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
	t.Parallel()

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
				m.On("RegisterUser", mock.Anything, mock.Anything).Return(domain.User{}, errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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

func TestHandleSetTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success - Best Case",
			requestBody: SetTimeoutRequest{
				Platform:        domain.PlatformTwitch,
				Username:        "validuser",
				DurationSeconds: 60,
				Reason:          "spam",
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("AddTimeout", mock.Anything, domain.PlatformTwitch, "validuser", time.Duration(60)*time.Second, "spam").Return(nil)
				m.On("GetTimeoutPlatform", mock.Anything, domain.PlatformTwitch, "validuser").Return(time.Duration(120)*time.Second, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"total_remaining_seconds":120`,
		},
		{
			name: "Success - Boundary Case (Min Duration)",
			requestBody: SetTimeoutRequest{
				Platform:        domain.PlatformYoutube,
				Username:        "minuser",
				DurationSeconds: 1,
				Reason:          "min",
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("AddTimeout", mock.Anything, domain.PlatformYoutube, "minuser", time.Duration(1)*time.Second, "min").Return(nil)
				m.On("GetTimeoutPlatform", mock.Anything, domain.PlatformYoutube, "minuser").Return(time.Duration(1)*time.Second, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"total_remaining_seconds":1`,
		},
		{
			name: "Success - Boundary Case (Max Duration)",
			requestBody: SetTimeoutRequest{
				Platform:        domain.PlatformDiscord,
				Username:        "maxuser",
				DurationSeconds: 86400,
				Reason:          "max",
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("AddTimeout", mock.Anything, domain.PlatformDiscord, "maxuser", time.Duration(86400)*time.Second, "max").Return(nil)
				m.On("GetTimeoutPlatform", mock.Anything, domain.PlatformDiscord, "maxuser").Return(time.Duration(86400)*time.Second, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"total_remaining_seconds":86400`,
		},
		{
			name: "Invalid Case - Invalid Platform",
			requestBody: SetTimeoutRequest{
				Platform:        "invalidplatform",
				Username:        "user",
				DurationSeconds: 60,
				Reason:          "spam",
			},
			setupMock:      func(m *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Invalid Case - Missing Fields",
			requestBody: SetTimeoutRequest{
				Username: "badrequest",
			},
			setupMock:      func(m *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Invalid Case - Duration Below Min",
			requestBody: SetTimeoutRequest{
				Platform:        domain.PlatformTwitch,
				Username:        "user",
				DurationSeconds: 0,
				Reason:          "short",
			},
			setupMock:      func(m *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Invalid Case - Duration Above Max",
			requestBody: SetTimeoutRequest{
				Platform:        domain.PlatformTwitch,
				Username:        "user",
				DurationSeconds: 86401,
				Reason:          "long",
			},
			setupMock:      func(m *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request",
		},
		{
			name: "Service Error - AddTimeout Failed",
			requestBody: SetTimeoutRequest{
				Platform:        domain.PlatformTwitch,
				Username:        "erroruser",
				DurationSeconds: 60,
				Reason:          "spam",
			},
			setupMock: func(m *mocks.MockUserService) {
				m.On("AddTimeout", mock.Anything, domain.PlatformTwitch, "erroruser", time.Duration(60)*time.Second, "spam").Return(errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrMsgGenericServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockSvc := mocks.NewMockUserService(t)
			tt.setupMock(mockSvc)

			handler := HandleSetTimeout(mockSvc)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("PUT", "/user/timeout", bytes.NewBuffer(body))
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

func TestGetPlatformID(t *testing.T) {
	t.Parallel()

	user := &domain.User{
		TwitchID:  "twitch123",
		YoutubeID: "yt123",
		DiscordID: "disc123",
	}

	tests := []struct {
		name     string
		platform string
		expected string
	}{
		{
			name:     "Twitch Platform",
			platform: domain.PlatformTwitch,
			expected: "twitch123",
		},
		{
			name:     "Youtube Platform",
			platform: domain.PlatformYoutube,
			expected: "yt123",
		},
		{
			name:     "Discord Platform",
			platform: domain.PlatformDiscord,
			expected: "disc123",
		},
		{
			name:     "Unknown Platform",
			platform: "unknown",
			expected: "",
		},
		{
			name:     "Empty Platform",
			platform: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := getPlatformID(user, tt.platform)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdatePlatformID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		initialUser      domain.User
		platform         string
		platformID       string
		platformUsername string
		expectedUser     domain.User
	}{
		{
			name:             "Twitch Platform",
			initialUser:      domain.User{},
			platform:         domain.PlatformTwitch,
			platformID:       "twitch123",
			platformUsername: "twitch_user",
			expectedUser: domain.User{
				TwitchID:          "twitch123",
				PlatformUsernames: map[string]string{domain.PlatformTwitch: "twitch_user"},
			},
		},
		{
			name:             "Youtube Platform",
			initialUser:      domain.User{},
			platform:         domain.PlatformYoutube,
			platformID:       "yt123",
			platformUsername: "yt_user",
			expectedUser: domain.User{
				YoutubeID:         "yt123",
				PlatformUsernames: map[string]string{domain.PlatformYoutube: "yt_user"},
			},
		},
		{
			name:             "Discord Platform",
			initialUser:      domain.User{},
			platform:         domain.PlatformDiscord,
			platformID:       "disc123",
			platformUsername: "disc_user",
			expectedUser: domain.User{
				DiscordID:         "disc123",
				PlatformUsernames: map[string]string{domain.PlatformDiscord: "disc_user"},
			},
		},
		{
			name:             "Unknown Platform",
			initialUser:      domain.User{},
			platform:         "unknown",
			platformID:       "unk123",
			platformUsername: "unk_user",
			expectedUser: domain.User{
				PlatformUsernames: map[string]string{"unknown": "unk_user"},
			},
		},
		{
			name:             "Empty Username",
			initialUser:      domain.User{},
			platform:         domain.PlatformTwitch,
			platformID:       "twitch123",
			platformUsername: "",
			expectedUser: domain.User{
				TwitchID:          "twitch123",
				PlatformUsernames: map[string]string{},
			},
		},
		{
			name: "Existing PlatformUsernames",
			initialUser: domain.User{
				PlatformUsernames: map[string]string{domain.PlatformTwitch: "existing_twitch"},
			},
			platform:         domain.PlatformDiscord,
			platformID:       "disc123",
			platformUsername: "disc_user",
			expectedUser: domain.User{
				DiscordID: "disc123",
				PlatformUsernames: map[string]string{
					domain.PlatformTwitch:  "existing_twitch",
					domain.PlatformDiscord: "disc_user",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			user := tt.initialUser
			updatePlatformID(&user, tt.platform, tt.platformID, tt.platformUsername)
			assert.Equal(t, tt.expectedUser, user)
		})
	}
}

func TestHandleGetTimeout(t *testing.T) {
	t.Parallel()
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
				m.On("GetTimeoutPlatform", mock.Anything, domain.PlatformTwitch, "baduser").Return(time.Duration(60)*time.Second, nil) // 1 minute
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body string) {
				var resp map[string]interface{}
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Equal(t, domain.PlatformTwitch, resp["platform"])
				assert.Equal(t, "baduser", resp["username"])
				assert.Equal(t, true, resp["is_timed_out"])
				assert.Equal(t, 60.0, resp["remaining_seconds"])
			},
		},
		{
			name:        "Success - Not Timed Out",
			queryParams: map[string]string{"username": "gooduser"},
			setupMock: func(m *mocks.MockUserService) {
				m.On("GetTimeoutPlatform", mock.Anything, domain.PlatformTwitch, "gooduser").Return(time.Duration(0), nil)
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body string) {
				var resp map[string]interface{}
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Equal(t, domain.PlatformTwitch, resp["platform"])
				assert.Equal(t, "gooduser", resp["username"])
				assert.Equal(t, false, resp["is_timed_out"])
				assert.Equal(t, 0.0, resp["remaining_seconds"])
			},
		},
		{
			name:        "Success - With Explicit Platform",
			queryParams: map[string]string{"username": "discorduser", "platform": domain.PlatformDiscord},
			setupMock: func(m *mocks.MockUserService) {
				m.On("GetTimeoutPlatform", mock.Anything, domain.PlatformDiscord, "discorduser").Return(time.Duration(30)*time.Second, nil)
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body string) {
				var resp map[string]interface{}
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Equal(t, domain.PlatformDiscord, resp["platform"])
				assert.Equal(t, "discorduser", resp["username"])
				assert.Equal(t, true, resp["is_timed_out"])
				assert.Equal(t, 30.0, resp["remaining_seconds"])
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
				m.On("GetTimeoutPlatform", mock.Anything, domain.PlatformTwitch, "erroruser").Return(time.Duration(0), errors.New(ErrMsgGenericServerError))
			},
			expectedStatus: http.StatusInternalServerError,
			verifyBody:     func(t *testing.T, body string) {},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
