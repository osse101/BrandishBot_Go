package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func setupExpeditionTest(t *testing.T) (*ExpeditionHandler, *mocks.MockExpeditionService, *mocks.MockProgressionService) {
	mockExpSvc := mocks.NewMockExpeditionService(t)
	mockProgSvc := mocks.NewMockProgressionService(t)
	handler := NewExpeditionHandler(mockExpSvc, mockProgSvc)
	return handler, mockExpSvc, mockProgSvc
}

// createTestExpeditionRequest Helper function to create a new HTTP request with JSON body
func createTestExpeditionRequest(method, url string, body interface{}) (*http.Request, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func TestHandleStartExpedition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		requestBody   interface{}
		setupMocks    func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService)
		expectedCode  int
		expectedError string
	}{
		{
			name: "Feature Locked",
			requestBody: StartExpeditionRequest{
				Platform:       domain.PlatformDiscord,
				PlatformID:     "user1",
				Username:       "testuser",
				ExpeditionType: string(domain.ExpeditionTypeNormal),
			},
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(false, errors.New("progression_locked"))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedError: "progression_locked",
		},
		{
			name:        "Invalid JSON",
			requestBody: "invalid-json",
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
			},
			expectedCode:  http.StatusBadRequest,
			expectedError: "Invalid request body",
		},
		{
			name: "Service Error",
			requestBody: StartExpeditionRequest{
				Platform:       domain.PlatformDiscord,
				PlatformID:     "user1",
				Username:       "testuser",
				ExpeditionType: string(domain.ExpeditionTypeNormal),
			},
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
				mockExp.On("StartExpedition", mock.Anything, domain.PlatformDiscord, "user1", "testuser", domain.ExpeditionTypeNormal).
					Return(nil, errors.New("Something went wrong"))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedError: "Something went wrong",
		},
		{
			name: "Success",
			requestBody: StartExpeditionRequest{
				Platform:       domain.PlatformDiscord,
				PlatformID:     "user1",
				Username:       "testuser",
				ExpeditionType: string(domain.ExpeditionTypeNormal),
			},
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
				mockProg.On("RecordEngagement", mock.Anything, "testuser", "expedition_started", 2).
					Return(nil)

				expID := uuid.New()
				mockExp.On("StartExpedition", mock.Anything, domain.PlatformDiscord, "user1", "testuser", domain.ExpeditionTypeNormal).
					Return(&domain.Expedition{
						ID:             expID,
						ExpeditionType: domain.ExpeditionTypeNormal,
						JoinDeadline:   time.Now().Add(10 * time.Minute),
					}, nil)
			},
			expectedCode: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler, mockExp, mockProg := setupExpeditionTest(t)

			tt.setupMocks(mockExp, mockProg)

			req, err := createTestExpeditionRequest(http.MethodPost, "/expedition/start", tt.requestBody)
			require.NoError(t, err)

			// Break JSON for "Invalid JSON" test
			if tt.name == "Invalid JSON" {
				req, _ = http.NewRequest(http.MethodPost, "/expedition/start", bytes.NewBufferString("{invalid"))
				req.Header.Set("Content-Type", "application/json")
			}

			rr := httptest.NewRecorder()
			handler.HandleStart(rr, req)

			assert.Equal(t, tt.expectedCode, rr.Code)

			if tt.expectedError != "" {
				var resp ErrorResponse
				err = json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Contains(t, resp.Error, tt.expectedError)
			} else {
				var resp StartExpeditionResponse
				err = json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, "Expedition started! Others can join.", resp.Message)
				assert.NotEmpty(t, resp.ExpeditionID)
				assert.NotEmpty(t, resp.JoinDeadline)
			}

			mockExp.AssertExpectations(t)
			mockProg.AssertExpectations(t)
		})
	}
}

func TestHandleGetStatusExpedition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupMocks    func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService)
		expectedCode  int
		expectedError string
	}{
		{
			name: "Feature Locked",
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(false, errors.New("progression_locked"))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedError: "progression_locked",
		},
		{
			name: "Service Error",
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
				mockExp.On("GetStatus", mock.Anything).
					Return(nil, errors.New("Database error"))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedError: "Database error",
		},
		{
			name: "Success",
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
				mockExp.On("GetStatus", mock.Anything).
					Return(&domain.ExpeditionStatus{
						HasActive:  false,
						OnCooldown: false,
					}, nil)
			},
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler, mockExp, mockProg := setupExpeditionTest(t)

			tt.setupMocks(mockExp, mockProg)

			req, err := http.NewRequest(http.MethodGet, "/expedition/status", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.HandleGetStatus(rr, req)

			assert.Equal(t, tt.expectedCode, rr.Code)

			if tt.expectedError != "" {
				var resp ErrorResponse
				err = json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Contains(t, resp.Error, tt.expectedError)
			} else {
				var resp domain.ExpeditionStatus
				err = json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.False(t, resp.HasActive)
				assert.False(t, resp.OnCooldown)
			}

			mockExp.AssertExpectations(t)
			mockProg.AssertExpectations(t)
		})
	}
}

func TestHandleGetJournalExpedition(t *testing.T) {
	t.Parallel()

	expID := uuid.New()

	tests := []struct {
		name          string
		queryParams   string
		setupMocks    func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService)
		expectedCode  int
		expectedError string
	}{
		{
			name:        "Feature Locked",
			queryParams: "?id=" + expID.String(),
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(false, errors.New("progression_locked"))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedError: "progression_locked",
		},
		{
			name:        "Missing ID",
			queryParams: "",
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
			},
			expectedCode:  http.StatusBadRequest,
			expectedError: "Missing id query parameter",
		},
		{
			name:        "Invalid ID",
			queryParams: "?id=invalid-uuid",
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
			},
			expectedCode:  http.StatusBadRequest,
			expectedError: "Invalid expedition ID",
		},
		{
			name:        "Service Error",
			queryParams: "?id=" + expID.String(),
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
				mockExp.On("GetJournal", mock.Anything, expID).
					Return(nil, errors.New("Journal not found"))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedError: "Journal not found",
		},
		{
			name:        "Success",
			queryParams: "?id=" + expID.String(),
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
				mockExp.On("GetJournal", mock.Anything, expID).
					Return([]domain.ExpeditionJournalEntry{
						{
							ExpeditionID: expID,
							TurnNumber:   1,
							Narrative:    "Start of adventure",
						},
					}, nil)
			},
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler, mockExp, mockProg := setupExpeditionTest(t)

			tt.setupMocks(mockExp, mockProg)

			req, err := http.NewRequest(http.MethodGet, "/expedition/journal"+tt.queryParams, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.HandleGetJournal(rr, req)

			assert.Equal(t, tt.expectedCode, rr.Code)

			if tt.expectedError != "" {
				var resp ErrorResponse
				err = json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Contains(t, resp.Error, tt.expectedError)
			} else {
				var resp []domain.ExpeditionJournalEntry
				err = json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Len(t, resp, 1)
				assert.Equal(t, "Start of adventure", resp[0].Narrative)
			}

			mockExp.AssertExpectations(t)
			mockProg.AssertExpectations(t)
		})
	}
}

func TestHandleGetActiveExpedition(t *testing.T) {
	t.Parallel()

	expID := uuid.New()

	tests := []struct {
		name          string
		setupMocks    func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService)
		expectedCode  int
		expectedError string
	}{
		{
			name: "Feature Locked",
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(false, errors.New("progression_locked"))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedError: "progression_locked",
		},
		{
			name: "Service Error",
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
				mockExp.On("GetActiveExpedition", mock.Anything).
					Return(nil, errors.New("No active expedition"))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedError: "No active expedition",
		},
		{
			name: "Success",
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
				mockExp.On("GetActiveExpedition", mock.Anything).
					Return(&domain.ExpeditionDetails{
						Expedition: domain.Expedition{
							ID:             expID,
							ExpeditionType: domain.ExpeditionTypeNormal,
						},
					}, nil)
			},
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler, mockExp, mockProg := setupExpeditionTest(t)

			tt.setupMocks(mockExp, mockProg)

			req, err := http.NewRequest(http.MethodGet, "/expedition/active", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.HandleGetActive(rr, req)

			assert.Equal(t, tt.expectedCode, rr.Code)

			if tt.expectedError != "" {
				var resp ErrorResponse
				err = json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Contains(t, resp.Error, tt.expectedError)
			} else {
				var resp domain.ExpeditionDetails
				err = json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, expID, resp.Expedition.ID)
				assert.Equal(t, domain.ExpeditionTypeNormal, resp.Expedition.ExpeditionType)
			}

			mockExp.AssertExpectations(t)
			mockProg.AssertExpectations(t)
		})
	}
}

func TestHandleGetExpedition(t *testing.T) {
	t.Parallel()

	expID := uuid.New()

	tests := []struct {
		name          string
		queryParams   string
		setupMocks    func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService)
		expectedCode  int
		expectedError string
	}{
		{
			name:        "Feature Locked",
			queryParams: "?id=" + expID.String(),
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(false, errors.New("progression_locked"))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedError: "progression_locked",
		},
		{
			name:        "Missing ID",
			queryParams: "",
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
			},
			expectedCode:  http.StatusBadRequest,
			expectedError: "Missing id query parameter",
		},
		{
			name:        "Invalid ID",
			queryParams: "?id=invalid-uuid",
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
			},
			expectedCode:  http.StatusBadRequest,
			expectedError: "Invalid expedition ID",
		},
		{
			name:        "Service Error",
			queryParams: "?id=" + expID.String(),
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
				mockExp.On("GetExpedition", mock.Anything, expID).
					Return(nil, errors.New("Expedition not found"))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedError: "Expedition not found",
		},
		{
			name:        "Success",
			queryParams: "?id=" + expID.String(),
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
				mockExp.On("GetExpedition", mock.Anything, expID).
					Return(&domain.ExpeditionDetails{
						Expedition: domain.Expedition{
							ID:             expID,
							ExpeditionType: domain.ExpeditionTypeNormal,
						},
					}, nil)
			},
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler, mockExp, mockProg := setupExpeditionTest(t)

			tt.setupMocks(mockExp, mockProg)

			req, err := http.NewRequest(http.MethodGet, "/expedition"+tt.queryParams, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.HandleGet(rr, req)

			assert.Equal(t, tt.expectedCode, rr.Code)

			if tt.expectedError != "" {
				var resp ErrorResponse
				err = json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Contains(t, resp.Error, tt.expectedError)
			} else {
				var resp domain.ExpeditionDetails
				err = json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, expID, resp.Expedition.ID)
				assert.Equal(t, domain.ExpeditionTypeNormal, resp.Expedition.ExpeditionType)
			}

			mockExp.AssertExpectations(t)
			mockProg.AssertExpectations(t)
		})
	}
}

func TestHandleJoinExpedition(t *testing.T) {
	t.Parallel()

	expID := uuid.New()

	tests := []struct {
		name          string
		requestBody   interface{}
		queryParams   string
		setupMocks    func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService)
		expectedCode  int
		expectedError string
	}{
		{
			name: "Feature Locked",
			requestBody: JoinExpeditionRequest{
				Platform:   domain.PlatformDiscord,
				PlatformID: "user2",
				Username:   "testuser2",
			},
			queryParams: "?id=" + expID.String(),
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(false, errors.New("progression_locked"))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedError: "progression_locked",
		},
		{
			name: "Success (Active Default)",
			requestBody: JoinExpeditionRequest{
				Platform:   domain.PlatformDiscord,
				PlatformID: "user2",
				Username:   "testuser2",
			},
			queryParams: "",
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
				mockExp.On("JoinExpedition", mock.Anything, domain.PlatformDiscord, "user2", "testuser2", uuid.Nil).
					Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "Invalid ID",
			requestBody: JoinExpeditionRequest{
				Platform:   domain.PlatformDiscord,
				PlatformID: "user2",
				Username:   "testuser2",
			},
			queryParams: "?id=invalid-uuid",
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
			},
			expectedCode:  http.StatusBadRequest,
			expectedError: "Invalid expedition ID",
		},
		{
			name:        "Invalid JSON",
			requestBody: "invalid-json",
			queryParams: "?id=" + expID.String(),
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
			},
			expectedCode:  http.StatusBadRequest,
			expectedError: "Invalid request body",
		},
		{
			name: "Service Error",
			requestBody: JoinExpeditionRequest{
				Platform:   domain.PlatformDiscord,
				PlatformID: "user2",
				Username:   "testuser2",
			},
			queryParams: "?id=" + expID.String(),
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
				mockExp.On("JoinExpedition", mock.Anything, "discord", "user2", "testuser2", expID).
					Return(errors.New("Expedition already full"))
			},
			expectedCode:  http.StatusInternalServerError,
			expectedError: "Expedition already full",
		},
		{
			name: "Success",
			requestBody: JoinExpeditionRequest{
				Platform:   domain.PlatformDiscord,
				PlatformID: "user2",
				Username:   "testuser2",
			},
			queryParams: "?id=" + expID.String(),
			setupMocks: func(mockExp *mocks.MockExpeditionService, mockProg *mocks.MockProgressionService) {
				mockProg.On("IsFeatureUnlocked", mock.Anything, "feature_expedition").
					Return(true, nil)
				mockExp.On("JoinExpedition", mock.Anything, "discord", "user2", "testuser2", expID).
					Return(nil)
			},
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler, mockExp, mockProg := setupExpeditionTest(t)

			tt.setupMocks(mockExp, mockProg)

			req, err := createTestExpeditionRequest(http.MethodPost, "/expedition/join"+tt.queryParams, tt.requestBody)
			require.NoError(t, err)

			if tt.name == "Invalid JSON" {
				req, _ = http.NewRequest(http.MethodPost, "/expedition/join"+tt.queryParams, bytes.NewBufferString("{invalid"))
				req.Header.Set("Content-Type", "application/json")
			}

			rr := httptest.NewRecorder()
			handler.HandleJoin(rr, req)

			assert.Equal(t, tt.expectedCode, rr.Code)

			if tt.expectedError != "" {
				var resp ErrorResponse
				err = json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Contains(t, resp.Error, tt.expectedError)
			} else {
				var resp SuccessResponse
				err = json.Unmarshal(rr.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, "Joined expedition!", resp.Message)
			}

			mockExp.AssertExpectations(t)
			mockProg.AssertExpectations(t)
		})
	}
}
