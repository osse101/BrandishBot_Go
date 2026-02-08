package handler

import (
	"net/http"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// SetTimeoutRequest represents the request to set/add a user timeout
type SetTimeoutRequest struct {
	Platform        string `json:"platform" validate:"required,platform"`
	Username        string `json:"username" validate:"required,max=100"`
	DurationSeconds int    `json:"duration_seconds" validate:"required,min=1,max=86400"`
	Reason          string `json:"reason" validate:"max=255"`
}

// AdminClearTimeoutRequest represents the request to clear a user timeout
type AdminClearTimeoutRequest struct {
	Platform string `json:"platform" validate:"required,platform"`
	Username string `json:"username" validate:"required,max=100"`
}

// HandleSetTimeout applies or extends a timeout for a user
// @Summary Set user timeout
// @Description Apply or extend a timeout for a user (accumulates with existing timeout)
// @Tags user
// @Accept json
// @Produce json
// @Param request body SetTimeoutRequest true "Timeout details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user/timeout [put]
func HandleSetTimeout(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		var req SetTimeoutRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Set timeout"); err != nil {
			return
		}

		// Validate platform
		if !isValidPlatform(req.Platform) {
			respondError(w, http.StatusBadRequest, "Invalid platform")
			return
		}

		duration := time.Duration(req.DurationSeconds) * time.Second

		if err := svc.AddTimeout(r.Context(), req.Platform, req.Username, duration, req.Reason); err != nil {
			log.Error("Failed to set timeout", "error", err, "platform", req.Platform, "username", req.Username)
			statusCode, userMsg := mapServiceErrorToUserMessage(err)
			respondError(w, statusCode, userMsg)
			return
		}

		// Get the new total remaining timeout
		remaining, _ := svc.GetTimeoutPlatform(r.Context(), req.Platform, req.Username)

		log.Info("Timeout set successfully",
			"platform", req.Platform,
			"username", req.Username,
			"added_duration", req.DurationSeconds,
			"total_remaining", remaining.Seconds())

		response := map[string]interface{}{
			"message":                 "Timeout applied successfully",
			"platform":                req.Platform,
			"username":                req.Username,
			"added_duration_seconds":  req.DurationSeconds,
			"total_remaining_seconds": remaining.Seconds(),
		}

		respondJSON(w, http.StatusOK, response)
	}
}

// HandleAdminClearTimeout clears a user's timeout (admin action)
// @Summary Clear user timeout
// @Description Remove a user's timeout (admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Param request body AdminClearTimeoutRequest true "Clear timeout request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/timeout/clear [post]
func HandleAdminClearTimeout(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		var req AdminClearTimeoutRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Admin clear timeout"); err != nil {
			return
		}

		// Validate platform
		if !isValidPlatform(req.Platform) {
			respondError(w, http.StatusBadRequest, "Invalid platform")
			return
		}

		if err := svc.ClearTimeout(r.Context(), req.Platform, req.Username); err != nil {
			log.Error("Failed to clear timeout", "error", err, "platform", req.Platform, "username", req.Username)
			statusCode, userMsg := mapServiceErrorToUserMessage(err)
			respondError(w, statusCode, userMsg)
			return
		}

		log.Info("Timeout cleared by admin",
			"platform", req.Platform,
			"username", req.Username)

		response := map[string]interface{}{
			"message":  "Timeout cleared successfully",
			"platform": req.Platform,
			"username": req.Username,
		}

		respondJSON(w, http.StatusOK, response)
	}
}

// isValidPlatform checks if the platform is valid
func isValidPlatform(platform string) bool {
	switch platform {
	case domain.PlatformTwitch, domain.PlatformYoutube, domain.PlatformDiscord:
		return true
	default:
		return false
	}
}
