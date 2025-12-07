package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockJobService
type MockJobService struct {
	mock.Mock
}

func (m *MockJobService) GetAllJobs(ctx context.Context) ([]domain.Job, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Job), args.Error(1)
}

func (m *MockJobService) GetUserJobs(ctx context.Context, userID string) ([]domain.UserJobInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.UserJobInfo), args.Error(1)
}

func (m *MockJobService) GetPrimaryJob(ctx context.Context, userID string) (*domain.UserJobInfo, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserJobInfo), args.Error(1)
}

func (m *MockJobService) AwardXP(ctx context.Context, userID, jobKey string, baseAmount int, source string, metadata map[string]interface{}) (*domain.XPAwardResult, error) {
	args := m.Called(ctx, userID, jobKey, baseAmount, source, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.XPAwardResult), args.Error(1)
}

func (m *MockJobService) GetJobLevel(ctx context.Context, userID, jobKey string) (int, error) {
	args := m.Called(ctx, userID, jobKey)
	return args.Int(0), args.Error(1)
}

func (m *MockJobService) GetJobBonus(ctx context.Context, userID, jobKey, bonusType string) (float64, error) {
	args := m.Called(ctx, userID, jobKey, bonusType)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockJobService) CalculateLevel(totalXP int64) int {
	args := m.Called(totalXP)
	return args.Int(0)
}

func (m *MockJobService) GetXPForLevel(level int) int64 {
	args := m.Called(level)
	return args.Get(0).(int64)
}

func (m *MockJobService) GetXPProgress(currentXP int64) (int, int64) {
	args := m.Called(currentXP)
	return args.Int(0), args.Get(1).(int64)
}

// Tests

func TestHandleGetAllJobs(t *testing.T) {
	svc := new(MockJobService)
	h := NewJobHandler(svc)

	jobs := []domain.Job{
		{ID: 1, JobKey: "j1", DisplayName: "Job 1"},
	}

	svc.On("GetAllJobs", mock.Anything).Return(jobs, nil)

	req := httptest.NewRequest("GET", "/jobs", nil)
	w := httptest.NewRecorder()

	h.HandleGetAllJobs(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string][]domain.Job
	json.NewDecoder(resp.Body).Decode(&result)
	
	assert.Len(t, result["jobs"], 1)
	assert.Equal(t, "j1", result["jobs"][0].JobKey)
}

func TestHandleGetAllJobs_Error(t *testing.T) {
	svc := new(MockJobService)
	h := NewJobHandler(svc)

	svc.On("GetAllJobs", mock.Anything).Return(nil, errors.New("db error"))

	req := httptest.NewRequest("GET", "/jobs", nil)
	w := httptest.NewRecorder()

	h.HandleGetAllJobs(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestHandleGetUserJobs(t *testing.T) {
	svc := new(MockJobService)
	h := NewJobHandler(svc)

	userID := "u1"
	userJobs := []domain.UserJobInfo{
		{JobKey: "j1", Level: 5},
	}
	primaryJob := &domain.UserJobInfo{JobKey: "j1", Level: 5}

	svc.On("GetUserJobs", mock.Anything, userID).Return(userJobs, nil)
	svc.On("GetPrimaryJob", mock.Anything, userID).Return(primaryJob, nil)

	req := httptest.NewRequest("GET", "/jobs?user_id=u1", nil)
	w := httptest.NewRecorder()

	h.HandleGetUserJobs(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, userID, result["user_id"])
	// JSON unmarshaling numbers makes them float64
	jobs := result["jobs"].([]interface{})
	assert.Len(t, jobs, 1)
}

func TestHandleGetUserJobs_MissingUser(t *testing.T) {
	svc := new(MockJobService)
	h := NewJobHandler(svc)

	req := httptest.NewRequest("GET", "/jobs", nil)
	w := httptest.NewRecorder()

	h.HandleGetUserJobs(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestHandleAwardXP(t *testing.T) {
	svc := new(MockJobService)
	h := NewJobHandler(svc)

	reqBody := AwardXPRequest{
		UserID: "u1",
		JobKey: "j1",
		XPAmount: 100,
		Source: "test",
	}
	body, _ := json.Marshal(reqBody)

	awardResult := &domain.XPAwardResult{
		JobKey: "j1",
		XPGained: 100,
		NewLevel: 1,
	}

	svc.On("AwardXP", mock.Anything, "u1", "j1", 100, "test", mock.Anything).Return(awardResult, nil)

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result domain.XPAwardResult
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, 100, result.XPGained)
}

func TestHandleAwardXP_InvalidRequest(t *testing.T) {
	svc := new(MockJobService)
	h := NewJobHandler(svc)

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader([]byte(`{}`))) // Empty body maps to default values (User/Key empty)
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

// Additional Handler Tests - Error Scenarios

func TestHandleGetUserJobs_ServiceError(t *testing.T) {
	svc := new(MockJobService)
	h := NewJobHandler(svc)

	svc.On("GetUserJobs", mock.Anything, "u1").Return(nil, errors.New("database error"))

	req := httptest.NewRequest("GET", "/jobs?user_id=u1", nil)
	w := httptest.NewRecorder()

	h.HandleGetUserJobs(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestHandleGetUserJobs_NoPrimaryJob(t *testing.T) {
	svc := new(MockJobService)
	h := NewJobHandler(svc)

	userJobs := []domain.UserJobInfo{
		{JobKey: "j1", Level: 5},
	}

	svc.On("GetUserJobs", mock.Anything, "u1").Return(userJobs, nil)
	svc.On("GetPrimaryJob", mock.Anything, "u1").Return(nil, nil) // No primary (edge case)

	req := httptest.NewRequest("GET", "/jobs?user_id=u1", nil)
	w := httptest.NewRecorder()

	h.HandleGetUserJobs(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Nil(t, result["primary_job"]) // Should be nil in response
}

func TestHandleAwardXP_ServiceError_DailyCap(t *testing.T) {
	svc := new(MockJobService)
	h := NewJobHandler(svc)

	reqBody := AwardXPRequest{
		UserID:   "u1",
		JobKey:   "blacksmith",
		XPAmount: 100,
		Source:   "test",
	}
	body, _ := json.Marshal(reqBody)

	svc.On("AwardXP", mock.Anything, "u1", "blacksmith", 100, "test", mock.Anything).Return(nil, errors.New("daily XP cap reached for blacksmith"))

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	assert.Contains(t, w.Body.String(), "daily XP cap reached")
}

func TestHandleAwardXP_NegativeXP(t *testing.T) {
	svc := new(MockJobService)
	h := NewJobHandler(svc)

	reqBody := AwardXPRequest{
		UserID:   "u1",
		JobKey:   "j1",
		XPAmount: -50, // Negative XP
		Source:   "test",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	// Handler validates XPAmount <= 0
	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestHandleAwardXP_ZeroXP(t *testing.T) {
	svc := new(MockJobService)
	h := NewJobHandler(svc)

	reqBody := AwardXPRequest{
		UserID:   "u1",
		JobKey:   "j1",
		XPAmount: 0, // Zero XP
		Source:   "test",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	// Handler validates XPAmount <= 0
	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestHandleAwardXP_InvalidJSON(t *testing.T) {
	svc := new(MockJobService)
	h := NewJobHandler(svc)

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader([]byte(`{invalid json`)))
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestHandleAwardXP_MissingUserID(t *testing.T) {
	svc := new(MockJobService)
	h := NewJobHandler(svc)

	reqBody := AwardXPRequest{
		// UserID missing
		JobKey:   "j1",
		XPAmount: 100,
		Source:   "test",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestHandleAwardXP_MissingJobKey(t *testing.T) {
	svc := new(MockJobService)
	h := NewJobHandler(svc)

	reqBody := AwardXPRequest{
		UserID: "u1",
		// JobKey missing
		XPAmount: 100,
		Source:   "test",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestHandleAwardXP_ServiceError_JobNotFound(t *testing.T) {
	svc := new(MockJobService)
	h := NewJobHandler(svc)

	reqBody := AwardXPRequest{
		UserID:   "u1",
		JobKey:   "invalid_job",
		XPAmount: 100,
		Source:   "test",
	}
	body, _ := json.Marshal(reqBody)

	svc.On("AwardXP", mock.Anything, "u1", "invalid_job", 100, "test", mock.Anything).Return(nil, errors.New("job not found: invalid_job"))

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	assert.Contains(t, w.Body.String(), "job not found")
}

func TestHandleAwardXP_ServiceError_FeatureLocked(t *testing.T) {
	svc := new(MockJobService)
	h := NewJobHandler(svc)

	reqBody := AwardXPRequest{
		UserID:   "u1",
		JobKey:   "blacksmith",
		XPAmount: 100,
		Source:   "test",
	}
	body, _ := json.Marshal(reqBody)

	svc.On("AwardXP", mock.Anything, "u1", "blacksmith", 100, "test", mock.Anything).Return(nil, errors.New("jobs XP system not unlocked"))

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	assert.Contains(t, w.Body.String(), "not unlocked")
}

func TestHandleAwardXP_WithMetadata(t *testing.T) {
	svc := new(MockJobService)
	h := NewJobHandler(svc)

	reqBody := AwardXPRequest{
		UserID:   "u1",
		JobKey:   "blacksmith",
		XPAmount: 50,
		Source:   "upgrade",
		Metadata: map[string]interface{}{
			"item_quality": "rare",
			"recipe_id":    123,
		},
	}
	body, _ := json.Marshal(reqBody)

	awardResult := &domain.XPAwardResult{
		JobKey:   "blacksmith",
		XPGained: 50,
		NewLevel: 2,
	}

	svc.On("AwardXP", mock.Anything, "u1", "blacksmith", 50, "upgrade", mock.MatchedBy(func(m map[string]interface{}) bool {
		return m["item_quality"] == "rare"
	})).Return(awardResult, nil)

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}
