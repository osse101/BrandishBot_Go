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

func TestHandleGetUserJobs(t *testing.T) {
	svc := mocks.NewMockJobService(t)
	userRepo := mocks.NewMockRepositoryUser(t)
	h := NewJobHandler(svc, userRepo)

	platform := domain.PlatformTwitch
	platformID := "u1"
	userJobs := []domain.UserJobInfo{
		{JobKey: job.JobKeyBlacksmith, Level: 5},
	}
	primaryJob := &domain.UserJobInfo{JobKey: job.JobKeyBlacksmith, Level: 5}

	svc.On("GetUserJobsByPlatform", mock.Anything, platform, platformID).Return(userJobs, nil)
	svc.On("GetPrimaryJob", mock.Anything, platform, platformID).Return(primaryJob, nil)

	req := httptest.NewRequest("GET", "/jobs?platform=twitch&platform_id=u1", nil)
	w := httptest.NewRecorder()

	h.HandleGetUserJobs(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Equal(t, platformID, result["platform_id"])
	// JSON unmarshaling numbers makes them float64
	jobs := result["jobs"].([]interface{})
	assert.Len(t, jobs, 1)
}

func TestHandleGetUserJobs_MissingUser(t *testing.T) {
	svc := mocks.NewMockJobService(t)
	userRepo := mocks.NewMockRepositoryUser(t)
	h := NewJobHandler(svc, userRepo)

	req := httptest.NewRequest("GET", "/jobs", nil)
	w := httptest.NewRecorder()

	h.HandleGetUserJobs(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestHandleAwardXP(t *testing.T) {
	svc := mocks.NewMockJobService(t)
	userRepo := mocks.NewMockRepositoryUser(t)
	h := NewJobHandler(svc, userRepo)

	reqBody := AwardXPRequest{
		Platform:   domain.PlatformTwitch,
		PlatformID: "u1",
		JobKey:     job.JobKeyBlacksmith,
		XPAmount:   100,
		Source:     "test",
	}
	body, _ := json.Marshal(reqBody)

	awardResult := &domain.XPAwardResult{
		JobKey:   job.JobKeyBlacksmith,
		XPGained: 100,
		NewLevel: 1,
	}

	svc.On("AwardXPByPlatform", mock.Anything, domain.PlatformTwitch, "u1", job.JobKeyBlacksmith, 100, "test", mock.Anything).Return(awardResult, nil)

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
	svc := mocks.NewMockJobService(t)
	userRepo := mocks.NewMockRepositoryUser(t)
	h := NewJobHandler(svc, userRepo)

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader([]byte(`{}`))) // Empty body maps to default values (User/Key empty)
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

// Additional Handler Tests - Error Scenarios

func TestHandleGetUserJobs_ServiceError(t *testing.T) {
	svc := mocks.NewMockJobService(t)
	userRepo := mocks.NewMockRepositoryUser(t)
	h := NewJobHandler(svc, userRepo)

	svc.On("GetUserJobsByPlatform", mock.Anything, "twitch", "u1").Return(nil, errors.New("database error"))

	req := httptest.NewRequest("GET", "/jobs?platform=twitch&platform_id=u1", nil)
	w := httptest.NewRecorder()

	h.HandleGetUserJobs(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
}

func TestHandleGetUserJobs_NoPrimaryJob(t *testing.T) {
	svc := mocks.NewMockJobService(t)
	userRepo := mocks.NewMockRepositoryUser(t)
	h := NewJobHandler(svc, userRepo)

	userJobs := []domain.UserJobInfo{
		{JobKey: job.JobKeyBlacksmith, Level: 5},
	}

	svc.On("GetUserJobsByPlatform", mock.Anything, "twitch", "u1").Return(userJobs, nil)
	svc.On("GetPrimaryJob", mock.Anything, "twitch", "u1").Return(nil, nil) // No primary (edge case)

	req := httptest.NewRequest("GET", "/jobs?platform=twitch&platform_id=u1", nil)
	w := httptest.NewRecorder()

	h.HandleGetUserJobs(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	assert.Nil(t, result["primary_job"]) // Should be nil in response
}

func TestHandleAwardXP_ServiceError_DailyCap(t *testing.T) {
	svc := mocks.NewMockJobService(t)
	userRepo := mocks.NewMockRepositoryUser(t)
	h := NewJobHandler(svc, userRepo)

	reqBody := AwardXPRequest{
		Platform:   domain.PlatformTwitch,
		PlatformID: "u1",
		JobKey:     job.JobKeyBlacksmith,
		XPAmount:   100,
		Source:     "test",
	}
	body, _ := json.Marshal(reqBody)

	svc.On("AwardXPByPlatform", mock.Anything, domain.PlatformTwitch, "u1", job.JobKeyBlacksmith, 100, "test", mock.Anything).Return(nil, errors.New("daily XP cap reached for blacksmith"))

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	assert.Contains(t, w.Body.String(), "daily XP cap reached")
}

func TestHandleAwardXP_NegativeXP(t *testing.T) {
	svc := mocks.NewMockJobService(t)
	userRepo := mocks.NewMockRepositoryUser(t)
	h := NewJobHandler(svc, userRepo)

	reqBody := AwardXPRequest{
		Platform:   domain.PlatformTwitch,
		PlatformID: "u1",
		JobKey:     job.JobKeyBlacksmith,
		XPAmount:   -50, // Negative XP
		Source:     "test",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	// Handler validates XPAmount <= 0
	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestHandleAwardXP_ZeroXP(t *testing.T) {
	svc := mocks.NewMockJobService(t)
	userRepo := mocks.NewMockRepositoryUser(t)
	h := NewJobHandler(svc, userRepo)

	reqBody := AwardXPRequest{
		Platform:   domain.PlatformTwitch,
		PlatformID: "u1",
		JobKey:     job.JobKeyBlacksmith,
		XPAmount:   0, // Zero XP
		Source:     "test",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	// Handler validates XPAmount <= 0
	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestHandleAwardXP_InvalidJSON(t *testing.T) {
	svc := mocks.NewMockJobService(t)
	userRepo := mocks.NewMockRepositoryUser(t)
	h := NewJobHandler(svc, userRepo)

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader([]byte(`{invalid json`)))
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestHandleAwardXP_MissingUserID(t *testing.T) {
	svc := mocks.NewMockJobService(t)
	userRepo := mocks.NewMockRepositoryUser(t)
	h := NewJobHandler(svc, userRepo)

	reqBody := AwardXPRequest{
		// UserID missing
		JobKey:   job.JobKeyExplorer,
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
	svc := mocks.NewMockJobService(t)
	userRepo := mocks.NewMockRepositoryUser(t)
	h := NewJobHandler(svc, userRepo)

	reqBody := AwardXPRequest{
		Platform:   domain.PlatformTwitch,
		PlatformID: "u1",
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
	svc := mocks.NewMockJobService(t)
	userRepo := mocks.NewMockRepositoryUser(t)
	h := NewJobHandler(svc, userRepo)

	reqBody := AwardXPRequest{
		Platform:   domain.PlatformTwitch,
		PlatformID: "u1",
		JobKey:     "invalid_job",
		XPAmount:   100,
		Source:     "test",
	}
	body, _ := json.Marshal(reqBody)

	svc.On("AwardXPByPlatform", mock.Anything, domain.PlatformTwitch, "u1", "invalid_job", 100, "test", mock.Anything).Return(nil, errors.New("job not found: invalid_job"))

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	assert.Contains(t, w.Body.String(), "job not found")
}

func TestHandleAwardXP_ServiceError_FeatureLocked(t *testing.T) {
	svc := mocks.NewMockJobService(t)
	userRepo := mocks.NewMockRepositoryUser(t)
	h := NewJobHandler(svc, userRepo)

	reqBody := AwardXPRequest{
		Platform:   domain.PlatformTwitch,
		PlatformID: "u1",
		JobKey:     job.JobKeyBlacksmith,
		XPAmount:   100,
		Source:     "test",
	}
	body, _ := json.Marshal(reqBody)

	svc.On("AwardXPByPlatform", mock.Anything, domain.PlatformTwitch, "u1", job.JobKeyBlacksmith, 100, "test", mock.Anything).Return(nil, errors.New("jobs XP system not unlocked"))

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Result().StatusCode)
	assert.Contains(t, w.Body.String(), "not unlocked")
}

func TestHandleAwardXP_WithMetadata(t *testing.T) {
	svc := mocks.NewMockJobService(t)
	userRepo := mocks.NewMockRepositoryUser(t)
	h := NewJobHandler(svc, userRepo)

	reqBody := AwardXPRequest{
		Platform:   domain.PlatformTwitch,
		PlatformID: "u1",
		JobKey:     job.JobKeyBlacksmith,
		XPAmount:   50,
		Source:     "upgrade",
		Metadata: map[string]interface{}{
			"item_quality": "rare",
			"recipe_id":    123,
		},
	}
	body, _ := json.Marshal(reqBody)

	awardResult := &domain.XPAwardResult{
		JobKey:   job.JobKeyBlacksmith,
		XPGained: 50,
		NewLevel: 2,
	}

	svc.On("AwardXPByPlatform", mock.Anything, domain.PlatformTwitch, "u1", job.JobKeyBlacksmith, 50, "upgrade", mock.MatchedBy(func(m map[string]interface{}) bool {
		return m["item_quality"] == "rare"
	})).Return(awardResult, nil)

	req := httptest.NewRequest("POST", "/jobs/award-xp", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleAwardXP(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}
