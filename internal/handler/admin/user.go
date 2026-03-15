package admin

import (
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/handler"
	"github.com/osse101/BrandishBot_Go/internal/job"
	"github.com/osse101/BrandishBot_Go/internal/repository"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// UserHandler handles admin user operations
type UserHandler struct {
	userRepo    repository.User
	userService user.Service
}

// NewUserHandler creates a new admin user handler
func NewUserHandler(userRepo repository.User, userService user.Service) *UserHandler {
	return &UserHandler{
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
func (h *UserHandler) HandleUserLookup(w http.ResponseWriter, r *http.Request) {
	platform := r.URL.Query().Get("platform")
	username := r.URL.Query().Get("username")

	if platform == "" || username == "" {
		handler.RespondError(w, http.StatusBadRequest, "platform and username are required")
		return
	}

	user, err := h.userRepo.GetUserByPlatformUsername(r.Context(), platform, username)
	if err != nil {
		status, msg := handler.MapServiceErrorToUserMessage(err)
		handler.RespondError(w, status, msg)
		return
	}

	// Determine platform ID based on platform
	var platformID string
	switch platform {
	case domain.PlatformTwitch:
		platformID = user.TwitchID
	case domain.PlatformDiscord:
		platformID = user.DiscordID
	case domain.PlatformYoutube:
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

	handler.RespondJSON(w, http.StatusOK, resp)
}

// HandleGetRecentUsers returns a list of recently active users
// GET /api/v1/admin/users/recent
func (h *UserHandler) HandleGetRecentUsers(w http.ResponseWriter, r *http.Request) {
	limit := 10 // Default limit

	users, err := h.userRepo.GetRecentlyActiveUsers(r.Context(), limit)
	if err != nil {
		handler.RespondError(w, http.StatusInternalServerError, "failed to get recent users: "+err.Error())
		return
	}

	handler.RespondJSON(w, http.StatusOK, users)
}

// HandleGetItems returns all items for autocomplete
// GET /api/v1/admin/items
func (h *UserHandler) HandleGetItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.userRepo.GetAllItems(r.Context())
	if err != nil {
		handler.RespondError(w, http.StatusInternalServerError, "failed to get items: "+err.Error())
		return
	}
	handler.RespondJSON(w, http.StatusOK, items)
}

// HandleGetJobs returns all jobs for autocomplete
// GET /api/v1/admin/jobs
func (h *UserHandler) HandleGetJobs(w http.ResponseWriter, r *http.Request) {
	handler.RespondJSON(w, http.StatusOK, job.AllJobs)
}

// HandleGetActiveChatters returns a list of users who recently sent messages
// GET /api/v1/admin/users/active
func (h *UserHandler) HandleGetActiveChatters(w http.ResponseWriter, r *http.Request) {
	chatters := h.userService.GetActiveChatters()
	// chatters is []user.ActiveChatter, which is aliased to activechatter.Chatter
	handler.RespondJSON(w, http.StatusOK, chatters)
}
