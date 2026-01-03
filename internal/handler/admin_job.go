package handler

import (
	"encoding/json"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// AdminAwardXPRequest is the request body for admin XP awards
type AdminAwardXPRequest struct {
	Platform string `json:"platform"` // discord, twitch, youtube
	Username string `json:"username"` // Platform username
	JobKey   string `json:"job_key"`  // explorer, blacksmith, etc.
	Amount   int    `json:"amount"`   //  XP amount to award
}

//  AdminJobHandler handles admin job operations
type AdminJobHandler struct {
	jobService  job.Service
	userService user.Service
}

// NewAdminJobHandler creates a new admin job handler
func NewAdminJobHandler(jobService job.Service, userService user.Service) *AdminJobHandler {
	return &AdminJobHandler{
		jobService:  jobService,
		userService: userService,
	}
}

// HandleAdminAwardXP awards XP to a user identified by platform and username
// POST /admin/job/award-xp
func (h *AdminJobHandler) HandleAdminAwardXP(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	
	var req AdminAwardXPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body",  http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Platform == "" || req.Username == "" || req.JobKey == "" {
		http.Error(w, "platform, username, and job_key are required", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		http.Error(w, "amount must be positive", http.StatusBadRequest)
		return
	}

	if req.Amount > 10000 {
		http.Error(w, "amount exceeds maximum (10000)", http.StatusBadRequest)
		return
	}

	log.Info("Admin XP award requested",
		"platform", req.Platform,
		"username", req.Username,
		"job_key", req.JobKey,
		"amount", req.Amount)

	//  Resolve user by platform and username
	user, err := h.userService.GetUserByPlatformUsername(r.Context(), req.Platform, req.Username)
	if err != nil {
		log.Warn("User not found for admin XP award",
			"error", err,
			"platform", req.Platform,
			"username", req.Username)
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// Award XP using the job service
	result, err := h.jobService.AwardXP(
		r.Context(),
		user.ID,
		req.JobKey,
		req.Amount,
		"admin_award",
		map[string]interface{}{
			"platform": req.Platform,
			"username": req.Username,
		},
	)

	if err != nil {
		log.Error("Failed to award XP",
			"error", err,
			"user_id", user.ID,
			"job_key", req.JobKey)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Info("Admin XP awarded successfully",
		"user_id", user.ID,
		"username", req.Username,
		"job_key", req.JobKey,
		"amount", req.Amount,
		"leveled_up", result.LeveledUp,
		"new_level", result.NewLevel)

	// Return success response
	response := map[string]interface{}{
		"success":    true,
		"user_id":    user.ID,
		"username":   user.Username,
		"job_key":    req.JobKey,
		"xp_awarded": req.Amount,
		"result":     result,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("Failed to encode response", "error", err)
	}
}
