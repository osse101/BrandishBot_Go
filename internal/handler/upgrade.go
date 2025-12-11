package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
)

type UpgradeItemRequest struct {
	Platform   string `json:"platform" validate:"required,platform"`
	PlatformID string `json:"platform_id" validate:"required"`
	Username   string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	Item       string `json:"item" validate:"required,max=100"`
	Quantity   int    `json:"quantity" validate:"min=1,max=10000"`
}

type UpgradeItemResponse struct {
	NewItem          string `json:"new_item"`
	QuantityUpgraded int    `json:"quantity_upgraded"`
}

// HandleUpgradeItem handles upgrading an item
// @Summary Upgrade item
// @Description Upgrade an item to a higher tier
// @Tags crafting
// @Accept json
// @Produce json
// @Param request body UpgradeItemRequest true "Upgrade details"
// @Success 200 {object} UpgradeItemResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse "Feature locked"
// @Failure 500 {object} ErrorResponse
// @Router /user/item/upgrade [post]
// HandleUpgradeItem handles upgrading an item
// @Summary Upgrade item
// @Description Upgrade an item to a higher tier
// @Tags crafting
// @Accept json
// @Produce json
// @Param request body UpgradeItemRequest true "Upgrade details"
// @Success 200 {object} UpgradeItemResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse "Feature locked"
// @Failure 500 {object} ErrorResponse
// @Router /user/item/upgrade [post]
func HandleUpgradeItem(svc crafting.Service, progressionSvc progression.Service, eventBus event.Bus) http.HandlerFunc {
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

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			log.Warn("Invalid request", "error", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		newItem, quantityUpgraded, err := svc.UpgradeItem(r.Context(), req.Platform, req.PlatformID, req.Username, req.Item, req.Quantity)
		if err != nil {
			log.Error("Failed to upgrade item", "error", err, "username", req.Username, "item", req.Item)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("Item upgraded successfully",
			"username", req.Username,
			"item", req.Item,
			"quantity_upgraded", quantityUpgraded)

		// Add contribution points for crafting
		if err := progressionSvc.AddContribution(r.Context(), quantityUpgraded); err != nil {
			log.Warn("Failed to add contribution points", "error", err)
		}

		// Publish item.upgraded event
		if err := eventBus.Publish(r.Context(), event.Event{
			Type: "item.upgraded",
			Payload: map[string]interface{}{
				"user_id":           req.Username,
				"source_item":       req.Item,
				"result_item":       newItem,
				"quantity_upgraded": quantityUpgraded,
			},
		}); err != nil {
			log.Error("Failed to publish item.upgraded event", "error", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		respondJSON(w, http.StatusOK, UpgradeItemResponse{
			NewItem:          newItem,
			QuantityUpgraded: quantityUpgraded,
		})
	}
}

// HandleGetRecipes returns recipe information based on query parameters
// @Summary Get recipes
// @Description Get recipe information. Can filter by item or get all unlocked recipes for a user.
// @Tags crafting
// @Produce json
// @Param item query string false "Item name to get recipe for"
// @Param user query string false "Username to get unlocked recipes for"
// @Param platform query string false "Platform (required if user provided)"
// @Param platform_id query string false "Platform ID (required if user provided)"
// @Success 200 {object} map[string]interface{} "Recipes or single recipe"
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /recipes [get]
func HandleGetRecipes(svc crafting.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		itemName := r.URL.Query().Get("item")
		username := r.URL.Query().Get("user")

		log.Debug("Get recipes request", "item", itemName, "user", username)

		// Case 1: Only user provided - return unlocked recipes
		if username != "" && itemName == "" {
			platform := r.URL.Query().Get("platform")
			platformID := r.URL.Query().Get("platform_id")

			if platform == "" || platformID == "" {
				http.Error(w, "Missing platform or platform_id", http.StatusBadRequest)
				return
			}

			recipes, err := svc.GetUnlockedRecipes(r.Context(), platform, platformID, username)
			if err != nil {
				log.Error("Failed to get unlocked recipes", "error", err, "username", username)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			log.Info("Unlocked recipes retrieved", "username", username, "count", len(recipes))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"recipes": recipes,
			})
			return
		}

		// Case 2 & 3: Item provided (with or without user) - return recipe info
		if itemName != "" {
			platform := r.URL.Query().Get("platform")
			platformID := r.URL.Query().Get("platform_id")

			recipe, err := svc.GetRecipe(r.Context(), itemName, platform, platformID, username)
			if err != nil {
				log.Error("Failed to get recipe", "error", err, "item", itemName)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			log.Info("Recipe retrieved", "item", itemName, "user", username)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			respondJSON(w, http.StatusOK, recipe)
			return
		}

		// No valid parameters provided
		log.Warn("Invalid recipe query", "item", itemName, "user", username)
		http.Error(w, "Must provide either 'item' or 'user' query parameter", http.StatusBadRequest)
	}
}
