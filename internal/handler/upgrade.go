package handler

import (
	"encoding/json"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/middleware"
	"github.com/osse101/BrandishBot_Go/internal/progression"
)

type UpgradeItemRequest struct {
	Username string `json:"username"`
	Platform string `json:"platform"`
	Item     string `json:"item"`
	Quantity int    `json:"quantity"`
}

type UpgradeItemResponse struct {
	NewItem          string `json:"new_item"`
	QuantityUpgraded int    `json:"quantity_upgraded"`
}

func HandleUpgradeItem(svc crafting.Service, progressionSvc progression.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		// Check if upgrade feature is unlocked
		unlocked, err := progressionSvc.IsFeatureUnlocked(r.Context(), progression.FeatureUpgrade)
		if err != nil {
			log.Error("Failed to check feature unlock status", "error", err)
			http.Error(w, "Failed to check feature availability", http.StatusInternalServerError)
			return
		}
		if !unlocked {
			log.Warn("Upgrade feature is locked")
			http.Error(w, "Upgrade feature is not yet unlocked", http.StatusForbidden)
			return
		}

		var req UpgradeItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode upgrade item request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Debug("Upgrade item request",
			"username", req.Username,
			"item", req.Item,
			"quantity", req.Quantity)

		if req.Username == "" || req.Item == "" || req.Quantity <= 0 {
			log.Warn("Invalid upgrade item request")
			http.Error(w, "Missing required fields or invalid quantity", http.StatusBadRequest)
			return
		}

		newItem, quantityUpgraded, err := svc.UpgradeItem(r.Context(), req.Username, req.Platform, req.Item, req.Quantity)
		if err != nil {
			log.Error("Failed to upgrade item", "error", err, "username", req.Username, "item", req.Item)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("Item upgraded successfully",
			"username", req.Username,
			"item", req.Item,
			"quantity_upgraded", quantityUpgraded)

		// Track engagement for crafting
		middleware.TrackEngagementFromContext(
			middleware.WithUserID(r.Context(), req.Username),
			progressionSvc,
			"item_crafted",
			quantityUpgraded,
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(UpgradeItemResponse{
			NewItem:          newItem,
			QuantityUpgraded: quantityUpgraded,
		})
	}
}

// HandleGetRecipes returns recipe information based on query parameters
// ?item=X - returns base recipe for item X
// ?item=X&user=Y - returns recipe for item X with lock status for user Y
// ?user=Y - returns all unlocked recipes for user Y
func HandleGetRecipes(svc crafting.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		itemName := r.URL.Query().Get("item")
		username := r.URL.Query().Get("user")

		log.Debug("Get recipes request", "item", itemName, "user", username)

		// Case 1: Only user provided - return unlocked recipes
		if username != "" && itemName == "" {
			recipes, err := svc.GetUnlockedRecipes(r.Context(), username)
			if err != nil {
				log.Error("Failed to get unlocked recipes", "error", err, "username", username)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			log.Info("Unlocked recipes retrieved", "username", username, "count", len(recipes))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"recipes": recipes,
			})
			return
		}

		// Case 2 & 3: Item provided (with or without user) - return recipe info
		if itemName != "" {
			recipe, err := svc.GetRecipe(r.Context(), itemName, username)
			if err != nil {
				log.Error("Failed to get recipe", "error", err, "item", itemName)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			log.Info("Recipe retrieved", "item", itemName, "user", username)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(recipe)
			return
		}

		// No valid parameters provided
		log.Warn("Invalid recipe query", "item", itemName, "user", username)
		http.Error(w, "Must provide either 'item' or 'user' query parameter", http.StatusBadRequest)
	}
}
