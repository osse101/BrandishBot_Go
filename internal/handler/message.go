package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

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
func HandleMessageHandler(userService user.Service, progressionSvc progression.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		if r.Method != http.MethodPost {
			log.Warn("Method not allowed", "method", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req HandleMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode request body", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Debug("Decoded request",
			"platform", req.Platform,
			"platform_id", req.PlatformID,
			"username", req.Username)

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			log.Warn("Invalid request", "error", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		user, err := userService.HandleIncomingMessage(r.Context(), req.Platform, req.PlatformID, req.Username)
		if err != nil {
			log.Error("Failed to handle message",
				"error", err,
				"platform", req.Platform,
				"platform_id", req.PlatformID,
				"username", req.Username)
			http.Error(w, "Failed to handle message", http.StatusInternalServerError)
			return
		}

		log.Info("Message processed", "username", req.Username)

		// Track engagement for message
		middleware.TrackEngagementFromContext(
			middleware.WithUserID(r.Context(), user.ID),
			progressionSvc,
			"message",
			1,
		)

		log.Info("Message handled successfully",
			"user_id", user.ID,
			"username", user.Username)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		respondJSON(w, http.StatusOK, user)
	}
}
