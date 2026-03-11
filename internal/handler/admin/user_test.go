package admin

import (
	"errors"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/user"
	"github.com/osse101/BrandishBot_Go/mocks"
	repomocks "github.com/osse101/BrandishBot_Go/internal/user/mocks"
)

func TestHandleUserLookup(t *testing.T) {
	tests := []struct {
		name           string
		platform       string
		username       string
		mockRepoSetup  func(*repomocks.MockRepository)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:           "missing platform",
			platform:       "",
			username:       "testuser",
			mockRepoSetup:  func(m *repomocks.MockRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   ErrorResponse{Error: "platform and username are required"},
		},
		{
			name:           "missing username",
			platform:       "twitch",
			username:       "",
			mockRepoSetup:  func(m *repomocks.MockRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   ErrorResponse{Error: "platform and username are required"},
		},
		{
			name:     "user not found",
			platform: "twitch",
			username: "unknown",
			mockRepoSetup: func(m *repomocks.MockRepository) {
				m.On("GetUserByPlatformUsername", mock.Anything, "twitch", "unknown").
					Return(nil, domain.ErrUserNotFound)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   ErrorResponse{Error: "User not found"},
		},
		{
			name:     "success - twitch",
			platform: "twitch",
			username: "twitchuser",
			mockRepoSetup: func(m *repomocks.MockRepository) {
				m.On("GetUserByPlatformUsername", mock.Anything, "twitch", "twitchuser").
					Return(&domain.User{
						ID:        "user-1",
						Username:  "twitchuser",
						TwitchID:  "t123",
						CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: UserLookupResponse{
				ID:         "user-1",
				Platform:   "twitch",
				PlatformID: "t123",
				Username:   "twitchuser",
				CreatedAt:  "2023-01-01T00:00:00Z",
			},
		},
		{
			name:     "success - discord",
			platform: "discord",
			username: "discorduser",
			mockRepoSetup: func(m *repomocks.MockRepository) {
				m.On("GetUserByPlatformUsername", mock.Anything, "discord", "discorduser").
					Return(&domain.User{
						ID:        "user-2",
						Username:  "discorduser",
						DiscordID: "d123",
						CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: UserLookupResponse{
				ID:         "user-2",
				Platform:   "discord",
				PlatformID: "d123",
				Username:   "discorduser",
				CreatedAt:  "2023-01-01T00:00:00Z",
			},
		},
		{
			name:     "success - youtube",
			platform: "youtube",
			username: "youtubeuser",
			mockRepoSetup: func(m *repomocks.MockRepository) {
				m.On("GetUserByPlatformUsername", mock.Anything, "youtube", "youtubeuser").
					Return(&domain.User{
						ID:        "user-3",
						Username:  "youtubeuser",
						YoutubeID: "y123",
						CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: UserLookupResponse{
				ID:         "user-3",
				Platform:   "youtube",
				PlatformID: "y123",
				Username:   "youtubeuser",
				CreatedAt:  "2023-01-01T00:00:00Z",
			},
		},
		{
			name:     "success - unknown platform defaults to empty id",
			platform: "unknown-platform",
			username: "someuser",
			mockRepoSetup: func(m *repomocks.MockRepository) {
				m.On("GetUserByPlatformUsername", mock.Anything, "unknown-platform", "someuser").
					Return(&domain.User{
						ID:        "user-4",
						Username:  "someuser",
						CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: UserLookupResponse{
				ID:         "user-4",
				Platform:   "unknown-platform",
				PlatformID: "",
				Username:   "someuser",
				CreatedAt:  "2023-01-01T00:00:00Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(repomocks.MockRepository)
			mockSvc := new(mocks.MockUserService)
			tt.mockRepoSetup(mockRepo)

			handler := NewUserHandler(mockRepo, mockSvc)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/user/lookup", nil)
			q := req.URL.Query()
			q.Add("platform", tt.platform)
			q.Add("username", tt.username)
			req.URL.RawQuery = q.Encode()

			rr := httptest.NewRecorder()
			handler.HandleUserLookup(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var resp UserLookupResponse
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBody, resp)
			} else {
				var resp ErrorResponse
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBody, resp)
			}

			mockRepo.AssertExpectations(t)
			mockSvc.AssertExpectations(t)
		})
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func TestHandleGetRecentUsers(t *testing.T) {
	tests := []struct {
		name           string
		mockRepoSetup  func(*repomocks.MockRepository)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name: "error getting recent users",
			mockRepoSetup: func(m *repomocks.MockRepository) {
				m.On("GetRecentlyActiveUsers", mock.Anything, 10).
					Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrorResponse{Error: "failed to get recent users: db error"},
		},
		{
			name: "success",
			mockRepoSetup: func(m *repomocks.MockRepository) {
				m.On("GetRecentlyActiveUsers", mock.Anything, 10).
					Return([]domain.User{
						{
							ID:       "user-1",
							Username: "user1",
						},
						{
							ID:       "user-2",
							Username: "user2",
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: []domain.User{
				{
					ID:       "user-1",
					Username: "user1",
				},
				{
					ID:       "user-2",
					Username: "user2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(repomocks.MockRepository)
			mockSvc := new(mocks.MockUserService)
			tt.mockRepoSetup(mockRepo)

			handler := NewUserHandler(mockRepo, mockSvc)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/recent", nil)
			rr := httptest.NewRecorder()
			handler.HandleGetRecentUsers(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var resp []domain.User
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)

				// json unmarshalling will empty time fields compared to uninitialized struct time fields. Let's just compare ID and Username
				assert.Len(t, resp, len(tt.expectedBody.([]domain.User)))
				for i, u := range resp {
					assert.Equal(t, tt.expectedBody.([]domain.User)[i].ID, u.ID)
					assert.Equal(t, tt.expectedBody.([]domain.User)[i].Username, u.Username)
				}
			} else {
				var resp ErrorResponse
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBody, resp)
			}

			mockRepo.AssertExpectations(t)
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestHandleGetItems(t *testing.T) {
	tests := []struct {
		name           string
		mockRepoSetup  func(*repomocks.MockRepository)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name: "error getting items",
			mockRepoSetup: func(m *repomocks.MockRepository) {
				m.On("GetAllItems", mock.Anything).
					Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   ErrorResponse{Error: "failed to get items: db error"},
		},
		{
			name: "success",
			mockRepoSetup: func(m *repomocks.MockRepository) {
				m.On("GetAllItems", mock.Anything).
					Return([]domain.Item{
						{
							ID:           1,
							InternalName: "sword",
						},
						{
							ID:           2,
							InternalName: "shield",
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: []domain.Item{
				{
					ID:           1,
					InternalName: "sword",
				},
				{
					ID:           2,
					InternalName: "shield",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(repomocks.MockRepository)
			mockSvc := new(mocks.MockUserService)
			tt.mockRepoSetup(mockRepo)

			handler := NewUserHandler(mockRepo, mockSvc)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/items", nil)
			rr := httptest.NewRecorder()
			handler.HandleGetItems(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var resp []domain.Item
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)

				assert.Len(t, resp, len(tt.expectedBody.([]domain.Item)))
				for i, item := range resp {
					assert.Equal(t, tt.expectedBody.([]domain.Item)[i].ID, item.ID)
					assert.Equal(t, tt.expectedBody.([]domain.Item)[i].InternalName, item.InternalName)
				}
			} else {
				var resp ErrorResponse
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBody, resp)
			}

			mockRepo.AssertExpectations(t)
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestHandleGetJobs(t *testing.T) {
	mockRepo := new(repomocks.MockRepository)
	mockSvc := new(mocks.MockUserService)
	handler := NewUserHandler(mockRepo, mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/jobs", nil)
	rr := httptest.NewRecorder()
	handler.HandleGetJobs(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp []job.Info
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Len(t, resp, len(job.AllJobs))
	for key, expectedJob := range job.AllJobs {
		assert.Equal(t, expectedJob.Key, resp[key].Key)
		assert.Equal(t, expectedJob.DisplayName, resp[key].DisplayName)
	}

	mockRepo.AssertExpectations(t)
	mockSvc.AssertExpectations(t)
}

func TestHandleGetActiveChatters(t *testing.T) {
	mockRepo := new(repomocks.MockRepository)
	mockSvc := new(mocks.MockUserService)
	handler := NewUserHandler(mockRepo, mockSvc)

	expectedChatters := []user.ActiveChatter{
		{
			UserID: "user-1",
			Platform: "twitch",
						Username: "user1",
			LastMessageAt: time.Now(),
		},
	}

	mockSvc.On("GetActiveChatters").Return(expectedChatters)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/active", nil)
	rr := httptest.NewRecorder()
	handler.HandleGetActiveChatters(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp []user.ActiveChatter
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Len(t, resp, 1)
	assert.Equal(t, expectedChatters[0].UserID, resp[0].UserID)
	assert.Equal(t, expectedChatters[0].Username, resp[0].Username)

	mockRepo.AssertExpectations(t)
	mockSvc.AssertExpectations(t)
}
