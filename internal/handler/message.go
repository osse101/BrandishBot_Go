package handler

import (
	"net/http"
	"time"

	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/middleware"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// HandleMessageRequest represents the request to handle an incoming message.
type HandleMessageRequest struct {
	Platform   string `json:"platform" validate:"required,platform"`
	PlatformID string `json:"platform_id" validate:"required"`
	Username   string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	Message    string `json:"message"`
}

// HandleMessageHandler handles the incoming message flow.
// @Summary Handle chat message
// @Description Process a chat message for potential commands or triggers
// @Tags message
// @Accept json
// @Produce json
// @Param request body HandleMessageRequest true "Message details"
// @Success 200 {object} domain.User
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /message/handle [post]
// HandleMessageHandler handles the incoming message flow.
// @Summary Handle chat message
// @Description Process a chat message for potential commands or triggers
// @Tags message
// @Accept json
// @Produce json
// @Param request body HandleMessageRequest true "Message details"
// @Success 200 {object} domain.MessageResult
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /message/handle [post]
func HandleMessageHandler(userService user.Service, progressionSvc progression.Service, eventBus event.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log := logger.FromContext(r.Context())

		if r.Method != http.MethodPost {
			log.Warn("Method not allowed", "method", r.Method)
			http.Error(w, ErrMsgMethodNotAllowed, http.StatusMethodNotAllowed)
			return
		}

		var req HandleMessageRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Handle message"); err != nil {
			return
		}

		result, err := userService.HandleIncomingMessage(r.Context(), req.Platform, req.PlatformID, req.Username, req.Message)
		if err != nil {
			log.Error("Failed to handle message",
				"error", err,
				"platform", req.Platform,
				"platform_id", req.PlatformID,
				"username", req.Username)
			statusCode, userMsg := mapServiceErrorToUserMessage(err)
			respondError(w, statusCode, userMsg)
			return
		}

		// Inject user context into logger for downstream operations
		ctx := logger.WithUser(r.Context(), result.User.ID, result.User.Username)
		r = r.WithContext(ctx)

		// Track engagement for message
		middleware.TrackEngagementFromContext(
			middleware.WithUserID(ctx, result.User.ID),
			eventBus,
			"message",
			1,
		)

		// Single consolidated summary log
		duration := time.Since(start)
		log.Info("Message processed",
			"username", req.Username,
			"platform", req.Platform,
			"duration_ms", duration.Milliseconds(),
			"matches_found", len(result.Matches))

		respondJSON(w, http.StatusOK, result)
	}
}
