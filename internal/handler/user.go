package handler

import (
	"encoding/json"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// RegisterUserRequest represents the request to register or link a user.
type RegisterUserRequest struct {
	Username        string `json:"username"`
	KnownPlatform   string `json:"known_platform"`
	KnownPlatformID string `json:"known_platform_id"`
	NewPlatform     string `json:"new_platform"`
	NewPlatformID   string `json:"new_platform_id"`
}

// RegisterUserHandler handles user registration and account linking.
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

		if req.KnownPlatform == "" || req.KnownPlatformID == "" || req.NewPlatform == "" || req.NewPlatformID == "" {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		// Find user by the known platform ID.
		user, err := userService.FindUserByPlatformID(r.Context(), req.KnownPlatform, req.KnownPlatformID)
		isNewUser := false
		if err != nil {
			if req.Username == "" {
				http.Error(w, "Username is required for new users", http.StatusBadRequest)
				return
			}
			isNewUser = true
			// If user does not exist, create a new one.
			user = &domain.User{
				Username: req.Username,
			}
			// Set the known platform ID on the new user.
			updatePlatformID(user, req.KnownPlatform, req.KnownPlatformID)
		}

		// Link the new platform ID.
		updatePlatformID(user, req.NewPlatform, req.NewPlatformID)

		updatedUser, err := userService.RegisterUser(r.Context(), *user)
		if err != nil {
			http.Error(w, "Failed to register user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if isNewUser {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		json.NewEncoder(w).Encode(updatedUser)
	}
}

func updatePlatformID(user *domain.User, platform, platformID string) {
	switch platform {
	case "twitch":
		user.TwitchID = platformID
	case "youtube":
		user.YoutubeID = platformID
	case "discord":
		user.DiscordID = platformID
	}
}
