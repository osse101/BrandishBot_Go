package handler

import (
	"encoding/json"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

type JobHandler struct {
	service job.Service
}

func NewJobHandler(service job.Service) *JobHandler {
	return &JobHandler{
		service: service,
	}
}

// HandleGetAllJobs returns all job definitions with unlock status
func (h *JobHandler) HandleGetAllJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.service.GetAllJobs(r.Context())
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to get jobs", "error", err)
		http.Error(w, "Failed to retrieve jobs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jobs": jobs,
	})
}

// HandleGetUserJobs returns a user's job progress
func (h *JobHandler) HandleGetUserJobs(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Missing user_id parameter", http.StatusBadRequest)
		return
	}

	userJobs, err := h.service.GetUserJobs(r.Context(), userID)
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to get user jobs", "error", err, "user_id", userID)
		http.Error(w, "Failed to retrieve user jobs", http.StatusInternalServerError)
		return
	}

	primaryJob, _ := h.service.GetPrimaryJob(r.Context(), userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":     userID,
		"primary_job": primaryJob,
		"jobs":        userJobs,
	})
}

// AwardXPRequest is the request body for awarding XP
type AwardXPRequest struct {
	UserID   string                 `json:"user_id"`
	JobKey   string                 `json:"job_key"`
	XPAmount int                    `json:"xp_amount"`
	Source   string                 `json:"source"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// HandleAwardXP awards XP to a user's job (internal/bot use)
func (h *JobHandler) HandleAwardXP(w http.ResponseWriter, r *http.Request) {
	var req AwardXPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" || req.JobKey == "" || req.XPAmount <= 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	result, err := h.service.AwardXP(r.Context(), req.UserID, req.JobKey, req.XPAmount, req.Source, req.Metadata)
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to award XP",
			"error", err,
			"user_id", req.UserID,
			"job_key", req.JobKey,
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleGetJobBonus returns the active bonus for a specific job and bonus type
func (h *JobHandler) HandleGetJobBonus(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	jobKey := r.URL.Query().Get("job_key")
	bonusType := r.URL.Query().Get("bonus_type")

	if userID == "" || jobKey == "" || bonusType == "" {
		http.Error(w, "Missing required parameters (user_id, job_key, bonus_type)", http.StatusBadRequest)
		return
	}

	bonus, err := h.service.GetJobBonus(r.Context(), userID, jobKey, bonusType)
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to get job bonus",
			"error", err,
			"user_id", userID,
			"job_key", jobKey,
			"bonus_type", bonusType,
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":    userID,
		"job_key":    jobKey,
		"bonus_type": bonusType,
		"bonus_val":  bonus,
	})
}
