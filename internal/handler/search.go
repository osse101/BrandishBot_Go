package handler

import (
	"errors"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/middleware"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/search"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

type SearchRequest struct {
	Platform   string `json:"platform" validate:"required,platform"`
	PlatformID string `json:"platform_id" validate:"required"`
	Username   string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ItemHint   string `json:"item_hint,omitempty" validate:"max=50"`
}

type SearchResponse struct {
	Message string `json:"message"`
}

// HandleSearch handles player searching for items in the current environment.
// @Summary Perform environment search
// @Description Allows players to search for loot boxes. Results depend on daily usage and character progression.
// @Tags user
// @Accept json
// @Produce json
// @Param request body SearchRequest true "User identification"
// @Success 200 {object} SearchResponse
// @Failure 401 {object} ErrorResponse "Invalid API Key"
// @Failure 429 {object} ErrorResponse "Action on cooldown"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/user/search [post]
func HandleSearch(searchSvc search.Service, userService user.Service, progressionSvc progression.Service, eventBus event.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if search feature is unlocked
		if CheckFeatureLocked(w, r, progressionSvc, progression.FeatureSearch) {
			return
		}

		var req SearchRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Search"); err != nil {
			return
		}

		// Perform search through search service directly
		resultMessage, err := searchSvc.HandleSearch(r.Context(), req.Platform, req.PlatformID, req.Username, req.ItemHint)
		if err != nil {
			log := logger.FromContext(r.Context())
			if errors.Is(err, domain.ErrOnCooldown) {
				log.Debug("Search attempted while on cooldown", "username", req.Username, "error", err)
			} else {
				log.Error("Search failed", "error", err, "username", req.Username)
			}
			statusCode, userMsg := MapServiceErrorToUserMessage(err)
			RespondError(w, statusCode, userMsg)
			return
		}

		// Track engagement
		if userID, err := userService.GetUserIDByPlatformID(r.Context(), req.Platform, req.PlatformID); err == nil && userID != "" {
			middleware.TrackEngagementFromContext(
				middleware.WithUserID(r.Context(), userID),
				eventBus,
				domain.MetricTypeSearch,
				1,
			)
		}

		RespondJSON(w, http.StatusOK, SearchResponse{
			Message: resultMessage,
		})
	}
}
