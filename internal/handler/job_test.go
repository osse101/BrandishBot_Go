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

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHandleGetUserJobs_Cases(t *testing.T) {
	tests := []struct {
		name           string
		queryURL       string
		setupMock      func(*mocks.MockJobService, *mocks.MockRepositoryUser)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:     "Best Case: Valid platform and platform_id",
			queryURL: "/jobs?platform=twitch&platform_id=u1",
			setupMock: func(svc *mocks.MockJobService, userRepo *mocks.MockRepositoryUser) {
				userJobs := []domain.UserJobInfo{{JobKey: job.JobKeyBlacksmith, Level: 5}}
				primaryJob := &domain.UserJobInfo{JobKey: job.JobKeyBlacksmith, Level: 5}
				svc.On("GetUserJobsByPlatform", mock.Anything, domain.PlatformTwitch, "u1").Return(userJobs, nil)
				svc.On("GetPrimaryJob", mock.Anything, domain.PlatformTwitch, "u1").Return(primaryJob, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "u1", // We'll assert on platform_id
		},
		{
			name:     "Best Case: Valid platform and username",
			queryURL: "/jobs?platform=twitch&username=testuser",
			setupMock: func(svc *mocks.MockJobService, userRepo *mocks.MockRepositoryUser) {
				user := &domain.User{
					Username: "testuser",
					TwitchID: "u2",
				}
				userRepo.On("GetUserByPlatformUsername", mock.Anything, domain.PlatformTwitch, "testuser").Return(user, nil)

				userJobs := []domain.UserJobInfo{{JobKey: job.JobKeyBlacksmith, Level: 3}}
				primaryJob := &domain.UserJobInfo{JobKey: job.JobKeyBlacksmith, Level: 3}
				svc.On("GetUserJobsByPlatform", mock.Anything, domain.PlatformTwitch, "u2").Return(userJobs, nil)
				svc.On("GetPrimaryJob", mock.Anything, domain.PlatformTwitch, "u2").Return(primaryJob, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "u2", // We'll assert on platform_id
		},
		{
			name:           "Invalid Case: Missing both platform_id and username",
			queryURL:       "/jobs?platform=twitch",
			setupMock:      func(svc *mocks.MockJobService, userRepo *mocks.MockRepositoryUser) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Either platform_id or username is required",
		},
		{
			name:     "Invalid Case: Username provided but user not found",
			queryURL: "/jobs?platform=twitch&username=nonexistent",
			setupMock: func(svc *mocks.MockJobService, userRepo *mocks.MockRepositoryUser) {
				userRepo.On("GetUserByPlatformUsername", mock.Anything, domain.PlatformTwitch, "nonexistent").Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "User not found",
		},
		{
			name:     "Invalid Case: Username provided but no platform ID (edge case)",
			queryURL: "/jobs?platform=twitch&username=testuser",
			setupMock: func(svc *mocks.MockJobService, userRepo *mocks.MockRepositoryUser) {
				user := &domain.User{
					Username: "testuser",
					TwitchID: "", // Missing Twitch ID
				}
				userRepo.On("GetUserByPlatformUsername", mock.Anything, domain.PlatformTwitch, "testuser").Return(user, nil)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "User not found on platform",
		},
		{
			name:     "Edge Case: User has no primary job",
			queryURL: "/jobs?platform=twitch&platform_id=u1",
			setupMock: func(svc *mocks.MockJobService, userRepo *mocks.MockRepositoryUser) {
				userJobs := []domain.UserJobInfo{{JobKey: job.JobKeyBlacksmith, Level: 5}}
				svc.On("GetUserJobsByPlatform", mock.Anything, domain.PlatformTwitch, "u1").Return(userJobs, nil)
				svc.On("GetPrimaryJob", mock.Anything, domain.PlatformTwitch, "u1").Return(nil, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "no_primary_job", // Custom check flag
		},
		{
			name:     "Error Case: Service returns an error",
			queryURL: "/jobs?platform=twitch&platform_id=u1",
			setupMock: func(svc *mocks.MockJobService, userRepo *mocks.MockRepositoryUser) {
				svc.On("GetUserJobsByPlatform", mock.Anything, domain.PlatformTwitch, "u1").Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := mocks.NewMockJobService(t)
			userRepo := mocks.NewMockRepositoryUser(t)
			tt.setupMock(svc, userRepo)

			h := NewJobHandler(svc, userRepo)

			req := httptest.NewRequest("GET", tt.queryURL, nil)
			w := httptest.NewRecorder()

			h.HandleGetUserJobs(w, req)

			resp := w.Result()
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var result map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&result)
				assert.NoError(t, err)

				if tt.expectedBody == "no_primary_job" {
					assert.Nil(t, result["primary_job"])
				} else if tt.expectedBody != "" {
					assert.Equal(t, tt.expectedBody, result["platform_id"])
				}
			} else if tt.expectedBody != "" {
				// For error cases, standard error response format assumes {"error": "message"}
				var errResp handlerErrorResponse
				json.NewDecoder(resp.Body).Decode(&errResp)
				assert.Contains(t, errResp.Error, tt.expectedBody)
			}
		})
	}
}

// helper struct for parsing error responses
type handlerErrorResponse struct {
	Error string `json:"error"`
}

func TestHandleAwardXP_Cases(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mocks.MockJobService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Best Case: Valid request with all required fields",
			requestBody: AwardXPRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "u1",
				JobKey:     job.JobKeyBlacksmith,
				XPAmount:   100,
				Source:     "test",
			},
			setupMock: func(svc *mocks.MockJobService) {
				awardResult := &domain.XPAwardResult{
					JobKey:   job.JobKeyBlacksmith,
					XPGained: 100,
					NewLevel: 1,
				}
				svc.On("AwardXPByPlatform", mock.Anything, domain.PlatformTwitch, "u1", job.JobKeyBlacksmith, 100, "test", domain.JobXPMetadata{}).Return(awardResult, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "100", // Check XPGained
		},
		{
			name: "Boundary/Edge Case: Awarding XP with valid metadata",
			requestBody: AwardXPRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "u1",
				JobKey:     job.JobKeyBlacksmith,
				XPAmount:   50,
				Source:     "upgrade",
				Metadata: domain.JobXPMetadata{
					Extras: map[string]interface{}{
						"item_quality": "rare",
						"recipe_id":    123.0,
					},
				},
			},
			setupMock: func(svc *mocks.MockJobService) {
				awardResult := &domain.XPAwardResult{
					JobKey:   job.JobKeyBlacksmith,
					XPGained: 50,
					NewLevel: 2,
				}
				svc.On("AwardXPByPlatform", mock.Anything, domain.PlatformTwitch, "u1", job.JobKeyBlacksmith, 50, "upgrade", mock.MatchedBy(func(m domain.JobXPMetadata) bool {
					return m.Extras["item_quality"] == "rare"
				})).Return(awardResult, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "50", // Check XPGained
		},
		{
			name: "Invalid Case: Missing user ID",
			requestBody: AwardXPRequest{
				JobKey:   job.JobKeyExplorer,
				XPAmount: 100,
				Source:   "test",
			},
			setupMock:      func(svc *mocks.MockJobService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   ErrMsgMissingRequiredFields,
		},
		{
			name: "Invalid Case: Missing job key",
			requestBody: AwardXPRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "u1",
				XPAmount:   100,
				Source:     "test",
			},
			setupMock:      func(svc *mocks.MockJobService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   ErrMsgMissingRequiredFields,
		},
		{
			name: "Boundary Case: Zero XP",
			requestBody: AwardXPRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "u1",
				JobKey:     job.JobKeyBlacksmith,
				XPAmount:   0,
				Source:     "test",
			},
			setupMock:      func(svc *mocks.MockJobService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   ErrMsgMissingRequiredFields,
		},
		{
			name: "Boundary Case: Negative XP",
			requestBody: AwardXPRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "u1",
				JobKey:     job.JobKeyBlacksmith,
				XPAmount:   -50,
				Source:     "test",
			},
			setupMock:      func(svc *mocks.MockJobService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   ErrMsgMissingRequiredFields,
		},
		{
			name: "Error Case: Job not found",
			requestBody: AwardXPRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "u1",
				JobKey:     "invalid_job",
				XPAmount:   100,
				Source:     "test",
			},
			setupMock: func(svc *mocks.MockJobService) {
				svc.On("AwardXPByPlatform", mock.Anything, domain.PlatformTwitch, "u1", "invalid_job", 100, "test", domain.JobXPMetadata{}).Return(nil, errors.New("job not found"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "job not found",
		},
		{
			name: "Error Case: Feature locked",
			requestBody: AwardXPRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "u1",
				JobKey:     job.JobKeyBlacksmith,
				XPAmount:   100,
				Source:     "test",
			},
			setupMock: func(svc *mocks.MockJobService) {
				svc.On("AwardXPByPlatform", mock.Anything, domain.PlatformTwitch, "u1", job.JobKeyBlacksmith, 100, "test", domain.JobXPMetadata{}).Return(nil, errors.New("jobs XP system not unlocked"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "jobs XP system not unlocked",
		},
		{
			name: "Error Case: Daily cap reached",
			requestBody: AwardXPRequest{
				Platform:   domain.PlatformTwitch,
				PlatformID: "u1",
				JobKey:     job.JobKeyBlacksmith,
				XPAmount:   100,
				Source:     "test",
			},
			setupMock: func(svc *mocks.MockJobService) {
				svc.On("AwardXPByPlatform", mock.Anything, domain.PlatformTwitch, "u1", job.JobKeyBlacksmith, 100, "test", domain.JobXPMetadata{}).Return(nil, errors.New("daily XP cap reached for blacksmith"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "daily XP cap reached for blacksmith",
		},
		{
			name:           "Invalid Case: Invalid JSON body",
			requestBody:    `{invalid json`,
			setupMock:      func(svc *mocks.MockJobService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := mocks.NewMockJobService(t)
			userRepo := mocks.NewMockRepositoryUser(t)
			tt.setupMock(svc)

			h := NewJobHandler(svc, userRepo)

			var bodyReader *bytes.Reader
			if s, ok := tt.requestBody.(string); ok {
				bodyReader = bytes.NewReader([]byte(s))
			} else {
				body, _ := json.Marshal(tt.requestBody)
				bodyReader = bytes.NewReader(body)
			}

			req := httptest.NewRequest("POST", "/jobs/award-xp", bodyReader)
			w := httptest.NewRecorder()

			h.HandleAwardXP(w, req)

			resp := w.Result()
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				var result domain.XPAwardResult
				err := json.NewDecoder(resp.Body).Decode(&result)
				assert.NoError(t, err)

				if tt.expectedBody == "100" {
					assert.Equal(t, 100, result.XPGained)
				} else if tt.expectedBody == "50" {
					assert.Equal(t, 50, result.XPGained)
				}
			} else if tt.expectedBody != "" {
				var errResp handlerErrorResponse
				json.NewDecoder(resp.Body).Decode(&errResp)
				if errResp.Error != "" {
					assert.Contains(t, errResp.Error, tt.expectedBody)
				} else {
					// Fallback for non-JSON responses (e.g., standard HTTP errors)
					assert.Contains(t, w.Body.String(), tt.expectedBody)
				}
			}
		})
	}
}
