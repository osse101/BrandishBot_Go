package handler

import (
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
// @Router /api/v1/user/register [post]
func HandleRegisterUser(userService user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		if r.Method != http.MethodPost {
			log.Warn("Method not allowed", "method", r.Method)
			http.Error(w, ErrMsgMethodNotAllowed, http.StatusMethodNotAllowed)
			return
		}

		var req RegisterUserRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Register user"); err != nil {
			return
		}

		// Find user by the known platform ID.
		user, err := userService.FindUserByPlatformID(r.Context(), req.KnownPlatform, req.KnownPlatformID)
		isNewUser := false
		if err != nil {
			log.Debug("User not found by platform ID, will create new user", "platform", req.KnownPlatform)
			if req.Username == "" {
				log.Warn("Username required for new user")
				http.Error(w, ErrMsgUsernameRequired, http.StatusBadRequest)
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
			statusCode, userMsg := mapServiceErrorToUserMessage(err)
			respondError(w, statusCode, userMsg)
			return
		}

		log.Info("User registered successfully",
			"user_id", updatedUser.ID,
			"username", updatedUser.Username,
			"is_new", isNewUser)

		statusCode := http.StatusOK
		if isNewUser {
			statusCode = http.StatusCreated
		}
		respondJSON(w, statusCode, updatedUser)
	}
}

func updatePlatformID(user *domain.User, platform, platformID string) {
	switch platform {
	case domain.PlatformTwitch:
		user.TwitchID = platformID
	case domain.PlatformYoutube:
		user.YoutubeID = platformID
	case domain.PlatformDiscord:
		user.DiscordID = platformID
	}
}

// getPlatformID gets the platform ID from a user for a given platform
func getPlatformID(user *domain.User, platform string) string {
	switch platform {
	case domain.PlatformTwitch:
		return user.TwitchID
	case domain.PlatformYoutube:
		return user.YoutubeID
	case domain.PlatformDiscord:
		return user.DiscordID
	default:
		return ""
	}
}

// HandleGetTimeout returns the remaining timeout duration for a user
// @Summary Get user timeout
// @Description Get the remaining timeout duration for a user
// @Tags user
// @Produce json
// @Param platform query string false "Platform (default: twitch)" Enums(twitch, youtube, discord)
// @Param username query string true "Username to check"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/user/timeout [get]
func HandleGetTimeout(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		username, ok := GetQueryParam(r, w, "username")
		if !ok {
			return
		}

		// Platform is optional, defaults to twitch for backward compatibility
		platform := r.URL.Query().Get("platform")
		if platform == "" {
			platform = domain.PlatformTwitch
		}

		duration, err := svc.GetTimeoutPlatform(r.Context(), platform, username)
		if err != nil {
			log.Error("Failed to get timeout", "error", err, "platform", platform, "username", username)
			statusCode, userMsg := mapServiceErrorToUserMessage(err)
			respondError(w, statusCode, userMsg)
			return
		}

		respondJSON(w, http.StatusOK, GetUserTimeoutResponse{
			Platform:         platform,
			Username:         username,
			IsTimedOut:       duration > 0,
			RemainingSeconds: duration.Seconds(),
		})
	}
}

// GetUserTimeoutResponse defines the response structure for GetUserTimeout
type GetUserTimeoutResponse struct {
	Platform         string  `json:"platform"`
	Username         string  `json:"username"`
	IsTimedOut       bool    `json:"is_timed_out"`
	RemainingSeconds float64 `json:"remaining_seconds"`
}
