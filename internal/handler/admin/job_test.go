package admin_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/handler"
	"github.com/osse101/BrandishBot_Go/internal/handler/admin"
	"github.com/osse101/BrandishBot_Go/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestJobHandler_HandleAwardXP(t *testing.T) {
	tests := []struct {
		name           string
		reqBody        admin.AwardXPRequest
		setupMocks     func(userSvc *mocks.MockUserService, jobSvc *mocks.MockJobService)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Success_AwardXP",
			reqBody: admin.AwardXPRequest{
				Platform: "discord",
				Username: "testuser",
				JobKey:   "explorer",
				Amount:   100,
			},
			setupMocks: func(userSvc *mocks.MockUserService, jobSvc *mocks.MockJobService) {
				userSvc.On("GetUserByPlatformUsername", mock.Anything, "discord", "testuser").
					Return(&domain.User{ID: "user123", Username: "testuser"}, nil)

				jobSvc.On("AwardXP", mock.Anything, "user123", "explorer", 100, "admin_award", domain.JobXPMetadata{Platform: "discord", Username: "testuser"}).
					Return(&domain.XPAwardResult{
						LeveledUp: true,
						NewLevel:  2,
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Error_MissingPlatform",
			reqBody: admin.AwardXPRequest{
				Username: "testuser",
				JobKey:   "explorer",
				Amount:   100,
			},
			setupMocks:     func(userSvc *mocks.MockUserService, jobSvc *mocks.MockJobService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  handler.ErrMsgPlatformUsernameJobRequired,
		},
		{
			name: "Error_MissingUsername",
			reqBody: admin.AwardXPRequest{
				Platform: "discord",
				JobKey:   "explorer",
				Amount:   100,
			},
			setupMocks:     func(userSvc *mocks.MockUserService, jobSvc *mocks.MockJobService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  handler.ErrMsgPlatformUsernameJobRequired,
		},
		{
			name: "Error_MissingJobKey",
			reqBody: admin.AwardXPRequest{
				Platform: "discord",
				Username: "testuser",
				Amount:   100,
			},
			setupMocks:     func(userSvc *mocks.MockUserService, jobSvc *mocks.MockJobService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  handler.ErrMsgPlatformUsernameJobRequired,
		},
		{
			name: "Error_AmountZero",
			reqBody: admin.AwardXPRequest{
				Platform: "discord",
				Username: "testuser",
				JobKey:   "explorer",
				Amount:   0,
			},
			setupMocks:     func(userSvc *mocks.MockUserService, jobSvc *mocks.MockJobService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  handler.ErrMsgAmountMustBePositive,
		},
		{
			name: "Error_AmountNegative",
			reqBody: admin.AwardXPRequest{
				Platform: "discord",
				Username: "testuser",
				JobKey:   "explorer",
				Amount:   -10,
			},
			setupMocks:     func(userSvc *mocks.MockUserService, jobSvc *mocks.MockJobService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  handler.ErrMsgAmountMustBePositive,
		},
		{
			name: "Error_AmountExceedsMax",
			reqBody: admin.AwardXPRequest{
				Platform: "discord",
				Username: "testuser",
				JobKey:   "explorer",
				Amount:   10001,
			},
			setupMocks:     func(userSvc *mocks.MockUserService, jobSvc *mocks.MockJobService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  handler.ErrMsgAmountExceedsMax,
		},
		{
			name: "Error_UserNotFound",
			reqBody: admin.AwardXPRequest{
				Platform: "discord",
				Username: "testuser",
				JobKey:   "explorer",
				Amount:   100,
			},
			setupMocks: func(userSvc *mocks.MockUserService, jobSvc *mocks.MockJobService) {
				userSvc.On("GetUserByPlatformUsername", mock.Anything, "discord", "testuser").
					Return((*domain.User)(nil), errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  handler.ErrMsgUserNotFoundHTTP,
		},
		{
			name: "Error_JobServiceFails",
			reqBody: admin.AwardXPRequest{
				Platform: "discord",
				Username: "testuser",
				JobKey:   "explorer",
				Amount:   100,
			},
			setupMocks: func(userSvc *mocks.MockUserService, jobSvc *mocks.MockJobService) {
				userSvc.On("GetUserByPlatformUsername", mock.Anything, "discord", "testuser").
					Return(&domain.User{ID: "user123", Username: "testuser"}, nil)

				jobSvc.On("AwardXP", mock.Anything, "user123", "explorer", 100, "admin_award", domain.JobXPMetadata{Platform: "discord", Username: "testuser"}).
					Return((*domain.XPAwardResult)(nil), errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "service error",
		},
		{
			name:           "Error_InvalidJSON",
			reqBody:        admin.AwardXPRequest{}, // This will be manually bypassed in the loop
			setupMocks:     func(userSvc *mocks.MockUserService, jobSvc *mocks.MockJobService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  handler.ErrMsgInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserSvc := mocks.NewMockUserService(t)
			mockJobSvc := mocks.NewMockJobService(t)
			tt.setupMocks(mockUserSvc, mockJobSvc)

			jobHandler := admin.NewJobHandler(mockJobSvc, mockUserSvc)

			router := chi.NewRouter()
			router.Post("/admin/job/award-xp", jobHandler.HandleAwardXP)

			var req *http.Request
			if tt.name == "Error_InvalidJSON" {
				req = httptest.NewRequest(http.MethodPost, "/admin/job/award-xp", bytes.NewReader([]byte("{invalid json}")))
			} else {
				body, err := json.Marshal(tt.reqBody)
				require.NoError(t, err)
				req = httptest.NewRequest(http.MethodPost, "/admin/job/award-xp", bytes.NewReader(body))
			}
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.True(t, response["success"].(bool))
				assert.Equal(t, float64(tt.reqBody.Amount), response["xp_awarded"])
				assert.Equal(t, tt.reqBody.JobKey, response["job_key"])
				assert.Equal(t, "user123", response["user_id"])
				assert.Equal(t, "testuser", response["username"])

				resultMap := response["result"].(map[string]interface{})
				assert.True(t, resultMap["leveled_up"].(bool))
				assert.Equal(t, float64(2), resultMap["new_level"])
			} else {
				var errResp handler.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &errResp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedError, errResp.Error)
			}
		})
	}
}
