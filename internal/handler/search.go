package handler

import (
	"encoding/json"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/middleware"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

type SearchRequest struct {
	Username string `json:"username"`
	Platform string `json:"platform"`
}

type SearchResponse struct {
	Message string `json:"message"`
}

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

		// Validate username
		if err := ValidateUsername(req.Username); err != nil {
			log.Warn("Invalid username", "error", err)
			http.Error(w, "Invalid username", http.StatusBadRequest)
			return
		}

		// Validate platform (allow empty as service handles default)
		if req.Platform != "" {
			if err := ValidatePlatform(req.Platform); err != nil {
				log.Warn("Invalid platform", "platform", req.Platform)
				http.Error(w, "Invalid platform", http.StatusBadRequest)
				return
			}
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
		json.NewEncoder(w).Encode(SearchResponse{
			Message: message,
		})
	}
}
