package handler

import (
	"encoding/json"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// RegisterUserRequest represents the body of the register user request
type RegisterUserRequest struct {
	InternalID string `json:"internal_id"`
	Username   string `json:"username"`
	PlatformID string `json:"platform_id"`
	Platform   string `json:"platform"`
}

// RegisterUserHandler handles the /user/register endpoint
func RegisterUserHandler(userService user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req RegisterUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.InternalID == "" || req.Username == "" {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		newUser := domain.User{
			ID:         req.InternalID,
			Username:   req.Username,
			PlatformID: req.PlatformID,
			Platform:   req.Platform,
		}

		if err := userService.RegisterUser(newUser); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	}
}
