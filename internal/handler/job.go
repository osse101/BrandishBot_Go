package handler

import (
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

type JobHandler struct {
	service  job.Service
	userRepo repository.User
}

func NewJobHandler(service job.Service, userRepo repository.User) *JobHandler {
	return &JobHandler{
		service:  service,
		userRepo: userRepo,
	}
}

// HandleGetUserJobs returns a user's job progress
// Supports dual-mode: platform+platform_id (self-mode) or platform+username (target-mode)
func (h *JobHandler) HandleGetUserJobs(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	platform, ok := GetQueryParam(r, w, "platform")
	if !ok {
		return
	}

	platformID := r.URL.Query().Get("platform_id")
	username := r.URL.Query().Get("username")

	// Require either platform_id or username
	if platformID == "" && username == "" {
		log.Warn("Missing required parameter: either platform_id or username required")
		respondError(w, http.StatusBadRequest, "Either platform_id or username is required")
		return
	}

	// Target-mode: resolve user by username
	if platformID == "" && username != "" {
		user, err := h.userRepo.GetUserByPlatformUsername(r.Context(), platform, username)
		if err != nil {
			log.Error("Failed to find user by username", "error", err, "platform", platform, "username", username)
			respondError(w, http.StatusNotFound, "User not found")
			return
		}
		platformID = getPlatformID(user, platform)
		if platformID == "" {
			log.Error("User found but no platform ID", "username", username, "platform", platform)
			respondError(w, http.StatusNotFound, "User not found on platform")
			return
		}
		log.Debug("Resolved username to platform_id", "username", username, "platform_id", platformID)
	}

	userJobs, err := h.service.GetUserJobsByPlatform(r.Context(), platform, platformID)
	if err != nil {
		log.Error("Failed to get user jobs", "error", err, "platform", platform, "platform_id", platformID)
		statusCode, userMsg := mapServiceErrorToUserMessage(err)
		respondError(w, statusCode, userMsg)
		return
	}

	primaryJob, _ := h.service.GetPrimaryJob(r.Context(), platform, platformID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"platform":    platform,
		"platform_id": platformID,
		"primary_job": primaryJob,
		"jobs":        userJobs,
	})
}

// AwardXPRequest is the request body for awarding XP
type AwardXPRequest struct {
	Platform   string               `json:"platform"`
	PlatformID string               `json:"platform_id"`
	JobKey     string               `json:"job_key"`
	XPAmount   int                  `json:"xp_amount"`
	Source     string               `json:"source"`
	Metadata   domain.JobXPMetadata `json:"metadata,omitempty"`
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
		statusCode, userMsg := mapServiceErrorToUserMessage(err)
		respondError(w, statusCode, userMsg)
		return
	}

	respondJSON(w, http.StatusOK, result)
}
