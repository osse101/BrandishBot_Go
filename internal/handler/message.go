package handler

import (
	"encoding/json"
	"net/http"

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
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req HandleMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Platform == "" || req.PlatformID == "" || req.Username == "" {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		user, err := userService.HandleIncomingMessage(req.Platform, req.PlatformID, req.Username)
		if err != nil {
			http.Error(w, "Failed to handle message", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)
	}
}
