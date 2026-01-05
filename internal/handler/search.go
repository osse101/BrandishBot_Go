package handler

import (
	"encoding/json"
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
func HandleSearch(svc user.Service, progressionSvc progression.Service, eventBus event.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		// Check if search feature is unlocked
		// Check if search feature is unlocked
		if CheckFeatureLocked(w, r, progressionSvc, progression.FeatureSearch) {
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
			validationErrors := FormatValidationError(err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "Validation failed",
				"details": validationErrors,
			}); err != nil {
				log.Error("Failed to encode response", "error", err)
			}
			return
		}

		message, err := svc.HandleSearch(r.Context(), req.Platform, req.PlatformID, req.Username)
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
			eventBus,
			"search_performed",
			1,
		)

		// Publish search.performed event
		if err := eventBus.Publish(r.Context(), event.Event{
			Version: "1.0",
			Type: "search.performed",
			Payload: map[string]interface{}{
				"user_id":  req.Username,
				"platform": req.Platform,
				"message":  message,
			},
		}); err != nil {
			log.Error("Failed to publish search.performed event", "error", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		respondJSON(w, http.StatusOK, SearchResponse{
			Message: message,
		})
	}
}
