package handler

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

func TestNewAdminDailyResetHandler(t *testing.T) {
	t.Parallel()
	svc := mocks.NewMockJobService(t)
	h := NewAdminDailyResetHandler(svc)
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
			name: "Success",
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
			name: "ServiceError",
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

			h := NewAdminDailyResetHandler(svc)
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
		verifyBody     func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "Success",
			setupMock: func(svc *mocks.MockJobService) {
				expectedStatus := &domain.DailyResetStatus{
					LastResetTime:   fixedTime.Add(-1 * time.Hour),
					NextResetTime:   fixedTime.Add(23 * time.Hour),
					RecordsAffected: 5,
				}
				svc.On("GetDailyResetStatus", mock.Anything).Return(expectedStatus, nil)
			},
			expectedStatus: http.StatusOK,
			verifyBody: func(t *testing.T, body map[string]interface{}) {
				// Check that fields exist. Exact time parsing might be tricky due to float64 vs string depending on how it's marshaled,
				// but usually time.Time marshals to RFC3339 string.
				assert.NotEmpty(t, body["last_reset_time"])
				assert.NotEmpty(t, body["next_reset_time"])
				assert.Equal(t, float64(5), body["records_affected"])
			},
		},
		{
			name: "ServiceError",
			setupMock: func(svc *mocks.MockJobService) {
				svc.On("GetDailyResetStatus", mock.Anything).Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			verifyBody: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body["error"], "Failed to get reset status")
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

			h := NewAdminDailyResetHandler(svc)
			req := httptest.NewRequest("GET", "/admin/jobs/reset-status", nil)
			w := httptest.NewRecorder()

			h.HandleGetResetStatus(w, req)

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
