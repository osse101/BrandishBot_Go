package handler

import (
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/middleware"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

type SearchRequest struct {
	Platform   string `json:"platform" validate:"required,platform"`
	PlatformID string `json:"platform_id" validate:"required"`
	Username   string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
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
func HandleSearch(svc user.Service, progressionSvc progression.Service, eventBus event.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if search feature is unlocked
		if CheckFeatureLocked(w, r, progressionSvc, progression.FeatureSearch) {
			return
		}

		var req SearchRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Search"); err != nil {
			return
		}

		// Perform search through user service
		resultMessage, err := svc.HandleSearch(r.Context(), req.Platform, req.PlatformID, req.Username)
		if err != nil {
			logger.FromContext(r.Context()).Error("Search failed", "error", err, "username", req.Username)
			statusCode, userMsg := mapServiceErrorToUserMessage(err)
			respondError(w, statusCode, userMsg)
			return
		}

		// Track engagement
		middleware.TrackEngagementFromContext(
			middleware.WithUserID(r.Context(), req.Username),
			eventBus,
			"search",
			1,
		)

		respondJSON(w, http.StatusOK, SearchResponse{
			Message: resultMessage,
		})
	}
}
