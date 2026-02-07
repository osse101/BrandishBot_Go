package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/mocks"
)

func TestHandleManualReset_Success(t *testing.T) {
	svc := mocks.NewMockJobService(t)
	h := NewAdminDailyResetHandler(svc)

	svc.On("ResetDailyJobXP", mock.Anything).Return(int64(10), nil)

	req := httptest.NewRequest("POST", "/admin/jobs/reset-daily-xp", nil)
	w := httptest.NewRecorder()

	h.HandleManualReset(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, w.Body.String(), `"success":true`)
	assert.Contains(t, w.Body.String(), `"records_affected":10`)
}

func TestHandleManualReset_ServiceError(t *testing.T) {
	svc := mocks.NewMockJobService(t)
	h := NewAdminDailyResetHandler(svc)

	svc.On("ResetDailyJobXP", mock.Anything).Return(int64(0), errors.New("database error"))

	req := httptest.NewRequest("POST", "/admin/jobs/reset-daily-xp", nil)
	w := httptest.NewRecorder()

	h.HandleManualReset(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Contains(t, w.Body.String(), "Failed to reset daily XP")
}

func TestHandleGetResetStatus_Success(t *testing.T) {
	svc := mocks.NewMockJobService(t)
	h := NewAdminDailyResetHandler(svc)

	expectedStatus := &domain.DailyResetStatus{
		LastResetTime:   time.Now().Add(-1 * time.Hour),
		NextResetTime:   time.Now().Add(23 * time.Hour),
		RecordsAffected: 0,
	}

	svc.On("GetDailyResetStatus", mock.Anything).Return(expectedStatus, nil)

	req := httptest.NewRequest("GET", "/admin/jobs/reset-status", nil)
	w := httptest.NewRecorder()

	h.HandleGetResetStatus(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, w.Body.String(), "last_reset_time")
	assert.Contains(t, w.Body.String(), "next_reset_time")
}

func TestHandleGetResetStatus_ServiceError(t *testing.T) {
	svc := mocks.NewMockJobService(t)
	h := NewAdminDailyResetHandler(svc)

	svc.On("GetDailyResetStatus", mock.Anything).Return(nil, errors.New("database error"))

	req := httptest.NewRequest("GET", "/admin/jobs/reset-status", nil)
	w := httptest.NewRecorder()

	h.HandleGetResetStatus(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Contains(t, w.Body.String(), "Failed to get reset status")
}
