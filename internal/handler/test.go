package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/user"
)

// TestRequest represents the request body for the test endpoint
type TestRequest struct {
	Username   string `json:"username"`
	Platform   string `json:"platform"`
	PlatformID string `json:"platform_id"`
}

// TestResponse represents the response body for the test endpoint
type TestResponse struct {
	Message string `json:"message"`
}

// TestHandler handles the /test endpoint
func TestHandler(userService user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req TestRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Username == "" || req.Platform == "" || req.PlatformID == "" {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		// Check if user exists, create if not
		_, err := userService.HandleIncomingMessage(r.Context(), req.Platform, req.PlatformID, req.Username)
		if err != nil {
			http.Error(w, "Failed to process user: "+err.Error(), http.StatusInternalServerError)
			return
		}

		resp := TestResponse{
			Message: fmt.Sprintf("Greetings, %s!", req.Username),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
