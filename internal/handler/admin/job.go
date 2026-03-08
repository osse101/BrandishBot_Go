package admin

import (
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/handler"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// AwardXPRequest is the request body for admin XP awards
type AwardXPRequest struct {
	Platform string `json:"platform"` // discord, twitch, youtube
	Username string `json:"username"` // Platform username
	JobKey   string `json:"job_key"`  // explorer, blacksmith, etc.
	Amount   int    `json:"amount"`   //  XP amount to award
}

// JobHandler handles admin job operations
type JobHandler struct {
	jobService  job.Service
	userService user.Service
}

// NewJobHandler creates a new admin job handler
func NewJobHandler(jobService job.Service, userService user.Service) *JobHandler {
	return &JobHandler{
		jobService:  jobService,
		userService: userService,
	}
}

// HandleAwardXP awards XP to a user identified by platform and username
// POST /admin/job/award-xp
func (h *JobHandler) HandleAwardXP(w http.ResponseWriter, r *http.Request) {
	var req AwardXPRequest
	if err := handler.DecodeAndValidateRequest(r, w, &req, "Admin award XP"); err != nil {
		return
	}

	log := logger.FromContext(r.Context())

	// Validate required fields
	if req.Platform == "" || req.Username == "" || req.JobKey == "" {
		handler.RespondError(w, http.StatusBadRequest, handler.ErrMsgPlatformUsernameJobRequired)
		return
	}

	if req.Amount <= 0 {
		handler.RespondError(w, http.StatusBadRequest, handler.ErrMsgAmountMustBePositive)
		return
	}

	if req.Amount > 10000 {
		handler.RespondError(w, http.StatusBadRequest, handler.ErrMsgAmountExceedsMax)
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
		handler.RespondError(w, http.StatusNotFound, handler.ErrMsgUserNotFoundHTTP)
		return
	}

	// Award XP using the job service
	result, err := h.jobService.AwardXP(
		r.Context(),
		user.ID,
		req.JobKey,
		req.Amount,
		"admin_award",
		domain.JobXPMetadata{
			Platform: req.Platform,
			Username: req.Username,
		},
	)

	if err != nil {
		log.Error("Failed to award XP",
			"error", err,
			"user_id", user.ID,
			"job_key", req.JobKey)
		statusCode, userMsg := handler.MapServiceErrorToUserMessage(err)
		handler.RespondError(w, statusCode, userMsg)
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

	handler.RespondJSON(w, http.StatusOK, response)
}
