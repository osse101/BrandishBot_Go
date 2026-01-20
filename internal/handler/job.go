package handler

import (
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

// HandleGetUserJobs returns a user's job progress
func (h *JobHandler) HandleGetUserJobs(w http.ResponseWriter, r *http.Request) {
	platform, ok := GetQueryParam(r, w, "platform")
	if !ok {
		return
	}
	platformID, ok := GetQueryParam(r, w, "platform_id")
	if !ok {
		return
	}

	userJobs, err := h.service.GetUserJobsByPlatform(r.Context(), platform, platformID)
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to get user jobs", "error", err, "platform", platform, "platform_id", platformID)
		statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
		return
	}

	primaryJob, _ := h.service.GetPrimaryJob(r.Context(), platform, platformID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"platform":      platform,
		"platform_id": platformID,
		"primary_job": primaryJob,
		"jobs":        userJobs,
	})
}

// AwardXPRequest is the request body for awarding XP
type AwardXPRequest struct {
	Platform   string                 `json:"platform"`
	PlatformID string                 `json:"platform_id"`
	JobKey     string                 `json:"job_key"`
	XPAmount   int                    `json:"xp_amount"`
	Source     string                 `json:"source"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// HandleAwardXP awards XP to a user's job (internal/bot use)
func (h *JobHandler) HandleAwardXP(w http.ResponseWriter, r *http.Request) {
	var req AwardXPRequest
	if err := DecodeAndValidateRequest(r, w, &req, "Award XP"); err != nil {
		return
	}

	if req.Platform == "" || req.PlatformID == "" || req.JobKey == "" || req.XPAmount <= 0 {
		http.Error(w, ErrMsgMissingRequiredFields, http.StatusBadRequest)
		return
	}

	result, err := h.service.AwardXPByPlatform(r.Context(), req.Platform, req.PlatformID, req.JobKey, req.XPAmount, req.Source, req.Metadata)
	if err != nil {
		logger.FromContext(r.Context()).Error("Failed to award XP",
			"error", err,
			"platform", req.Platform,
			"platform_id", req.PlatformID,
			"job_key", req.JobKey,
		)
		statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
		return
	}

	respondJSON(w, http.StatusOK, result)
}
