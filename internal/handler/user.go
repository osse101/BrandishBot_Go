package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// RegisterUserRequest represents the request to register or link a user.
type RegisterUserRequest struct {
	Username        string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	KnownPlatform   string `json:"known_platform" validate:"required,platform"`
	KnownPlatformID string `json:"known_platform_id" validate:"required"`
	NewPlatform     string `json:"new_platform" validate:"required,platform"`
	NewPlatformID   string `json:"new_platform_id" validate:"required"`
}

// HandleRegisterUser handles user registration and account linking.
// @Summary Register or link a user
// @Description Register a new user or link an existing user to a new platform
// @Tags user
// @Accept json
// @Produce json
// @Param request body RegisterUserRequest true "Registration details"
// @Success 200 {object} domain.User "User updated"
// @Success 201 {object} domain.User "User created"
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user/register [post]
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

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			log.Warn("Invalid request", "error", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
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
		respondJSON(w, http.StatusOK, updatedUser)
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

// HandleGetTimeout returns the remaining timeout duration for a user
// @Summary Get user timeout
// @Description Get the remaining timeout duration for a user
// @Tags user
// @Produce json
// @Param username query string true "Username to check"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user/timeout [get]
func HandleGetTimeout(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		username := r.URL.Query().Get("username")
		if username == "" {
			http.Error(w, "Missing username parameter", http.StatusBadRequest)
			return
		}

		duration, err := svc.GetTimeout(r.Context(), username)
		if err != nil {
			log.Error("Failed to get timeout", "error", err, "username", username)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"username":          username,
			"is_timed_out":      duration > 0,
			"remaining_seconds": duration.Seconds(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
