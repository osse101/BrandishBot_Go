package handler

import (
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// AdminUserHandler handles admin user operations
type AdminUserHandler struct {
	userRepo repository.User
}

// NewAdminUserHandler creates a new admin user handler
func NewAdminUserHandler(userRepo repository.User) *AdminUserHandler {
	return &AdminUserHandler{userRepo: userRepo}
}

// UserLookupResponse contains user lookup result
type UserLookupResponse struct {
	ID         string `json:"id"`
	Platform   string `json:"platform"`
	PlatformID string `json:"platform_id"`
	Username   string `json:"username"`
	CreatedAt  string `json:"created_at"`
}

// HandleUserLookup looks up a user by platform and username
// GET /api/v1/admin/user/lookup?platform=twitch&username=foo
func (h *AdminUserHandler) HandleUserLookup(w http.ResponseWriter, r *http.Request) {
	platform := r.URL.Query().Get("platform")
	username := r.URL.Query().Get("username")

	if platform == "" || username == "" {
		respondError(w, http.StatusBadRequest, "platform and username are required")
		return
	}

	user, err := h.userRepo.GetUserByPlatformUsername(r.Context(), platform, username)
	if err != nil {
		status, msg := mapServiceErrorToUserMessage(err)
		respondError(w, status, msg)
		return
	}

	// Determine platform ID based on platform
	var platformID string
	switch platform {
	case "twitch":
		platformID = user.TwitchID
	case "discord":
		platformID = user.DiscordID
	case "youtube":
		platformID = user.YoutubeID
	default:
		platformID = ""
	}

	resp := UserLookupResponse{
		ID:         user.ID,
		Platform:   platform,
		PlatformID: platformID,
		Username:   user.Username,
		CreatedAt:  user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	respondJSON(w, http.StatusOK, resp)
}
