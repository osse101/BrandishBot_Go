package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// TestRequest represents the request body for the test endpoint
type TestRequest struct {
	Username   string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	Platform   string `json:"platform" validate:"required,platform"`
	PlatformID string `json:"platform_id" validate:"required"`
}

// TestResponse represents the response body for the test endpoint
type TestResponse struct {
	Message string `json:"message"`
}

// TestHandler handles the /test endpoint
func HandleTest(userService user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		if r.Method != http.MethodPost {
			log.Warn("Method not allowed", "method", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req TestRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode test request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Debug("Decoded test request",
			"username", req.Username,
			"platform", req.Platform,
			"platform_id", req.PlatformID)

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			log.Warn("Invalid request", "error", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		// Check if user exists, create if not
		log.Info("HandleIncomingMessage called", "platform", req.Platform, "platformID", req.PlatformID, "username", req.Username)
		_, err := userService.HandleIncomingMessage(r.Context(), req.Platform, req.PlatformID, req.Username, "")
		if err != nil {
			log.Error("Failed to process user", "error", err, "username", req.Username)
			http.Error(w, "Failed to process user: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("Test request completed", "username", req.Username)

		resp := TestResponse{
			Message: fmt.Sprintf("Greetings, %s!", req.Username),
		}

		w.Header().Set("Content-Type", "application/json")
		respondJSON(w, http.StatusOK, resp)
	}
}
