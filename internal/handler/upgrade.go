package handler

import (
	"fmt"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
)

type UpgradeItemResponse struct {
	Message          string `json:"message"`
	NewItem          string `json:"new_item"`
	QuantityUpgraded int    `json:"quantity_upgraded"`
	IsMasterwork     bool   `json:"is_masterwork"`
	BonusQuantity    int    `json:"bonus_quantity"`
}

// HandleUpgradeItem handles upgrading an item
// @Summary Upgrade item
// @Description Upgrade an item to a higher tier
// @Tags crafting
// @Accept json
// @Produce json
// @Param request body CraftingActionRequest true "Upgrade details"
// @Success 200 {object} UpgradeItemResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse "Feature locked"
// @Failure 500 {object} ErrorResponse
// @Router /user/item/upgrade [post]
func HandleUpgradeItem(svc crafting.Service, progressionSvc progression.Service, eventBus event.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		// Check if upgrade feature is unlocked
		if CheckFeatureLocked(w, r, progressionSvc, progression.FeatureUpgrade) {
			return
		}

		req, err := decodeCraftingRequest(r, "Upgrade item")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		result, err := svc.UpgradeItem(r.Context(), req.Platform, req.PlatformID, req.Username, req.Item, req.Quantity)
		if err != nil {
			log.Error("Failed to upgrade item", "error", err, "username", req.Username, "item", req.Item)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info("Item upgraded successfully",
			"username", req.Username,
			"item", req.Item,
			"quantity_upgraded", result.Quantity,
			"masterwork", result.IsMasterwork)

		// Track engagement for crafting
		trackCraftingEngagement(r.Context(), eventBus, req.Username, "item_crafted", result.Quantity)

		// Publish item.upgraded event
		if err := publishCraftingEvent(r.Context(), eventBus, "item.upgraded", map[string]interface{}{
			"user_id":           req.Username,
			"source_item":       req.Item,
			"result_item":       result.ItemName,
			"quantity_upgraded": result.Quantity,
			"is_masterwork":     result.IsMasterwork,
		}); err != nil {
			_ = err // Error already logged in publishCraftingEvent
		}

		// Construct user message
		message := fmt.Sprintf("Successfully upgraded to %dx %s", result.Quantity, result.ItemName)
		if result.IsMasterwork {
			message = fmt.Sprintf("MASTERWORK! Critical success! You received %dx %s (Bonus: +%d)", result.Quantity, result.ItemName, result.BonusQuantity)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		respondJSON(w, http.StatusOK, UpgradeItemResponse{
			Message:          message,
			NewItem:          result.ItemName,
			QuantityUpgraded: result.Quantity,
			IsMasterwork:     result.IsMasterwork,
			BonusQuantity:    result.BonusQuantity,
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

		// Case 2: Item provided (with or without user) - return recipe info
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

		recipes, err := svc.GetAllRecipes(r.Context())
		if err != nil {
			log.Error("Failed to get all recipes", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"recipes": recipes,
		})
	}
}
