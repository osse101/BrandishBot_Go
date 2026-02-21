package admin

import (
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/handler"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// ClearTimeoutRequest represents the request to clear a user timeout
type ClearTimeoutRequest struct {
	Platform string `json:"platform" validate:"required,platform"`
	Username string `json:"username" validate:"required,max=100"`
}

// HandleClearTimeout clears a user's timeout (admin action)
// @Summary Clear user timeout
// @Description Remove a user's timeout (admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Param request body ClearTimeoutRequest true "Clear timeout request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/timeout/clear [post]
func HandleClearTimeout(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		var req ClearTimeoutRequest
		if err := handler.DecodeAndValidateRequest(r, w, &req, "Admin clear timeout"); err != nil {
			return
		}

		// Validate platform
		if !handler.IsValidPlatform(req.Platform) {
			handler.RespondError(w, http.StatusBadRequest, "Invalid platform")
			return
		}

		if err := svc.ClearTimeout(r.Context(), req.Platform, req.Username); err != nil {
			log.Error("Failed to clear timeout", "error", err, "platform", req.Platform, "username", req.Username)
			statusCode, userMsg := handler.MapServiceErrorToUserMessage(err)
			handler.RespondError(w, statusCode, userMsg)
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

		handler.RespondJSON(w, http.StatusOK, response)
	}
}
