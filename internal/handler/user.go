package handler

import (
	"encoding/json"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
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

// HandleRegisterUser handles user registration and account linking.
func HandleRegisterUser(userService user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		if r.Method != http.MethodPost {
			log.Warn("Method not allowed", "method", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req RegisterUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode register user request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Debug("Register user request",
			"username", req.Username,
			"known_platform", req.KnownPlatform,
			"new_platform", req.NewPlatform)

		// Validate platforms
		if err := ValidatePlatform(req.KnownPlatform); err != nil {
			log.Warn("Invalid known platform", "platform", req.KnownPlatform)
			http.Error(w, "Invalid platform", http.StatusBadRequest)
			return
		}
		if err := ValidatePlatform(req.NewPlatform); err != nil {
			log.Warn("Invalid new platform", "platform", req.NewPlatform)
			http.Error(w, "Invalid platform", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.KnownPlatformID == "" || req.NewPlatformID == "" {
			log.Warn("Missing required fields")
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		// Find user by the known platform ID.
		user, err := userService.FindUserByPlatformID(r.Context(), req.KnownPlatform, req.KnownPlatformID)
		isNewUser := false
		if err != nil {
			log.Debug("User not found by platform ID, will create new user", "platform", req.KnownPlatform)
			if req.Username == "" {
				log.Warn("Username required for new user")
				http.Error(w, "Username is required for new users", http.StatusBadRequest)
				return
			}
			// Validate username
			if err := ValidateUsername(req.Username); err != nil {
				log.Warn("Invalid username for new user", "error", err)
				http.Error(w, "Invalid username", http.StatusBadRequest)
				return
			}
			isNewUser = true
			// If user does not exist, create a new one.
			user = &domain.User{
				Username: req.Username,
			}
			// Set the known platform ID on the new user.
			updatePlatformID(user, req.KnownPlatform, req.KnownPlatformID)
		} else {
			log.Debug("Found existing user", "user_id", user.ID, "username", user.Username)
		}

		// Link the new platform ID.
		updatePlatformID(user, req.NewPlatform, req.NewPlatformID)
		log.Debug("Linking new platform", "platform", req.NewPlatform)

		updatedUser, err := userService.RegisterUser(r.Context(), *user)
		if err != nil {
			log.Error("Failed to register user", "error", err, "username", req.Username)
			http.Error(w, "Failed to register user", http.StatusInternalServerError)
			return
		}

		log.Info("User registered successfully",
			"user_id", updatedUser.ID,
			"username", updatedUser.Username,
			"is_new", isNewUser)

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
