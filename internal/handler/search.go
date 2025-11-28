package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/middleware"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

type SearchRequest struct {
	Username string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	Platform string `json:"platform" validate:"omitempty,platform"`
}

type SearchResponse struct {
	Message string `json:"message"`
}

// HandleSearch handles searching for items
// @Summary Search for items
// @Description Search for items (lootbox mechanic)
// @Tags user
// @Accept json
// @Produce json
// @Param request body SearchRequest true "Search details"
// @Success 200 {object} SearchResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse "Feature locked"
// @Failure 429 {object} ErrorResponse "Cooldown"
// @Failure 500 {object} ErrorResponse
// @Router /user/search [post]
func HandleSearch(svc user.Service, progressionSvc progression.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		// Check if search feature is unlocked
		unlocked, err := progressionSvc.IsFeatureUnlocked(r.Context(), progression.FeatureSearch)
		if err != nil {
			log.Error("Failed to check feature unlock status", "error", err)
			http.Error(w, "Failed to check feature availability", http.StatusInternalServerError)
			return
		}
		if !unlocked {
			log.Warn("Search feature is locked")
			http.Error(w, "Search feature is not yet unlocked", http.StatusForbidden)
			return
		}

		var req SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode search request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Debug("Search request", "username", req.Username, "platform", req.Platform)

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			log.Warn("Invalid request", "error", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		message, err := svc.HandleSearch(r.Context(), req.Username, req.Platform)
		if err != nil {
			log.Error("Failed to handle search", "error", err, "username", req.Username)
			http.Error(w, "Failed to perform search", http.StatusInternalServerError)
			return
		}

		log.Info("Search completed successfully",
			"username", req.Username,
			"message", message)

		// Track engagement for search
		middleware.TrackEngagementFromContext(
			middleware.WithUserID(r.Context(), req.Username),
			progressionSvc,
			"search_performed",
			1,
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		respondJSON(w, http.StatusOK, SearchResponse{
			Message: message,
		})
	}
}
