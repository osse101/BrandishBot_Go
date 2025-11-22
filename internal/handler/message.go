package handler

import (
	"encoding/json"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// HandleMessageRequest represents the request to handle an incoming message.
type HandleMessageRequest struct {
	Platform   string `json:"platform"`
	PlatformID string `json:"platform_id"`
	Username   string `json:"username"`
}

// HandleMessageHandler handles the incoming message flow.
func HandleMessageHandler(userService user.Service) http.HandlerFunc {
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

		if req.Platform == "" || req.PlatformID == "" || req.Username == "" {
			log.Warn("Missing required fields",
				"platform_empty", req.Platform == "",
				"platform_id_empty", req.PlatformID == "",
				"username_empty", req.Username == "")
			http.Error(w, "Missing required fields", http.StatusBadRequest)
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
		
		log.Info("Message handled successfully",
			"user_id", user.ID,
			"username", user.Username)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)
	}
}
