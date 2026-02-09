package handler

import (
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// AdminUserHandler handles admin user operations
type AdminUserHandler struct {
	userRepo    repository.User
	userService user.Service
}

// NewAdminUserHandler creates a new admin user handler
func NewAdminUserHandler(userRepo repository.User, userService user.Service) *AdminUserHandler {
	return &AdminUserHandler{
		userRepo:    userRepo,
		userService: userService,
	}
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

// HandleGetRecentUsers returns a list of recently active users
// GET /api/v1/admin/users/recent
func (h *AdminUserHandler) HandleGetRecentUsers(w http.ResponseWriter, r *http.Request) {
	limit := 10 // Default limit

	users, err := h.userRepo.GetRecentlyActiveUsers(r.Context(), limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get recent users: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, users)
}

// HandleGetItems returns all items for autocomplete
// GET /api/v1/admin/items
func (h *AdminUserHandler) HandleGetItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.userRepo.GetAllItems(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get items: "+err.Error())
		return
	}
	respondJSON(w, http.StatusOK, items)
}

// HandleGetJobs returns all jobs for autocomplete
// GET /api/v1/admin/jobs
func (h *AdminUserHandler) HandleGetJobs(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, job.AllJobs)
}

// HandleGetActiveChatters returns a list of users who recently sent messages
// GET /api/v1/admin/users/active
func (h *AdminUserHandler) HandleGetActiveChatters(w http.ResponseWriter, r *http.Request) {
	chatters := h.userService.GetActiveChatters()
	respondJSON(w, http.StatusOK, chatters)
}
