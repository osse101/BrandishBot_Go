package admin

import (
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

func TestNewDailyResetHandler(t *testing.T) {
	t.Parallel()
	svc := mocks.NewMockJobService(t)
	h := NewDailyResetHandler(svc)
	assert.NotNil(t, h)
	assert.Equal(t, svc, h.jobService)
}

func TestHandleManualReset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupMock      func(*mocks.MockJobService)
		expectedStatus int
		verifyBody     func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "Best Case - Success",
			setupMock: func(svc *mocks.MockJobService) {
				svc.On("ResetDailyJobXP", mock.Anything).Return(int64(10), nil)
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, true, body["success"])
				assert.Equal(t, float64(10), body["records_affected"]) // JSON numbers are float64
				assert.Equal(t, "Daily XP reset completed", body["message"])
			},
		},
		{
			name: "Boundary Case - 0 Records Affected",
			setupMock: func(svc *mocks.MockJobService) {
				svc.On("ResetDailyJobXP", mock.Anything).Return(int64(0), nil)
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, true, body["success"])
				assert.Equal(t, float64(0), body["records_affected"])
				assert.Equal(t, "Daily XP reset completed", body["message"])
			},
		},
		{
			name: "Invalid Case - Service Error",
			setupMock: func(svc *mocks.MockJobService) {
				svc.On("ResetDailyJobXP", mock.Anything).Return(int64(0), errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			verifyBody: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body["error"], "Failed to reset daily XP")
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := mocks.NewMockJobService(t)
			if tc.setupMock != nil {
				tc.setupMock(svc)
			}

			h := NewDailyResetHandler(svc)
			req := httptest.NewRequest("POST", "/admin/jobs/reset-daily-xp", nil)
			w := httptest.NewRecorder()

			h.HandleManualReset(w, req)

			resp := w.Result()
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			var body map[string]interface{}
			err := json.NewDecoder(w.Body).Decode(&body)
			require.NoError(t, err)

			if tc.verifyBody != nil {
				tc.verifyBody(t, body)
			}
		})
	}
}

func TestHandleGetResetStatus(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2023, 10, 27, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		setupMock      func(*mocks.MockJobService)
		expectedStatus int
		verifyBody     func(t *testing.T, respBytes []byte)
	}{
		{
			name: "Best Case - Success",
			setupMock: func(svc *mocks.MockJobService) {
				expectedStatus := &domain.DailyResetStatus{
					LastResetTime:   fixedTime.Add(-1 * time.Hour),
					NextResetTime:   fixedTime.Add(23 * time.Hour),
					RecordsAffected: 5,
				}
				svc.On("GetDailyResetStatus", mock.Anything).Return(expectedStatus, nil)
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, respBytes []byte) {
				var status domain.DailyResetStatus
				err := json.Unmarshal(respBytes, &status)
				require.NoError(t, err)

				assert.Equal(t, fixedTime.Add(-1*time.Hour).UTC(), status.LastResetTime.UTC())
				assert.Equal(t, fixedTime.Add(23*time.Hour).UTC(), status.NextResetTime.UTC())
				assert.Equal(t, int64(5), status.RecordsAffected)
			},
		},
		{
			name: "Boundary Case - Zero Time",
			setupMock: func(svc *mocks.MockJobService) {
				expectedStatus := &domain.DailyResetStatus{
					LastResetTime:   time.Time{},
					NextResetTime:   fixedTime,
					RecordsAffected: 0,
				}
				svc.On("GetDailyResetStatus", mock.Anything).Return(expectedStatus, nil)
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, respBytes []byte) {
				var status domain.DailyResetStatus
				err := json.Unmarshal(respBytes, &status)
				require.NoError(t, err)

				assert.True(t, status.LastResetTime.IsZero())
				assert.Equal(t, fixedTime.UTC(), status.NextResetTime.UTC())
				assert.Equal(t, int64(0), status.RecordsAffected)
			},
		},
		{
			name: "Invalid Case - Service Error",
			setupMock: func(svc *mocks.MockJobService) {
				svc.On("GetDailyResetStatus", mock.Anything).Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			verifyBody: func(t *testing.T, respBytes []byte) {
				var errorResp map[string]interface{}
				err := json.Unmarshal(respBytes, &errorResp)
				require.NoError(t, err)
				assert.Contains(t, errorResp["error"], "Failed to get reset status")
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := mocks.NewMockJobService(t)
			if tc.setupMock != nil {
				tc.setupMock(svc)
			}

			h := NewDailyResetHandler(svc)
			req := httptest.NewRequest("GET", "/admin/jobs/reset-status", nil)
			w := httptest.NewRecorder()

			h.HandleGetResetStatus(w, req)

			resp := w.Result()
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			if tc.verifyBody != nil {
				tc.verifyBody(t, w.Body.Bytes())
			}
		})
	}
}
