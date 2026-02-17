package handler

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/repository"
)

// CraftingActionRequest represents a common request for crafting operations (upgrade, disassemble, etc.)
type CraftingActionRequest struct {
	Platform   string `json:"platform" validate:"required,platform"`
	PlatformID string `json:"platform_id" validate:"required"`
	Username   string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	Item       string `json:"item" validate:"required,max=100"`
	Quantity   int    `json:"quantity" validate:"min=1,max=10000"`
}

// decodeCraftingRequest decodes and validates a crafting action request from the HTTP request body
func decodeCraftingRequest(r *http.Request, w http.ResponseWriter, actionName string) (*CraftingActionRequest, error) {
	var req CraftingActionRequest
	if err := DecodeAndValidateRequest(r, w, &req, actionName); err != nil {
		return nil, err
	}

	return &req, nil
}

// CraftingHandler handles crafting-related requests
type CraftingHandler struct {
	service        crafting.Service
	userRepo       repository.User
	progressionSvc progression.Service
	eventBus       event.Bus
}

// NewCraftingHandler creates a new CraftingHandler
func NewCraftingHandler(service crafting.Service, userRepo repository.User, progressionSvc progression.Service, eventBus event.Bus) *CraftingHandler {
	return &CraftingHandler{
		service:        service,
		userRepo:       userRepo,
		progressionSvc: progressionSvc,
		eventBus:       eventBus,
	}
}

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
func (h *CraftingHandler) HandleUpgradeItem(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	// Check if upgrade feature is unlocked
	if CheckFeatureLocked(w, r, h.progressionSvc, progression.FeatureUpgrade) {
		return
	}

	req, err := decodeCraftingRequest(r, w, "Upgrade item")
	if err != nil {
		return
	}

	result, err := h.service.UpgradeItem(r.Context(), req.Platform, req.PlatformID, req.Username, req.Item, req.Quantity)
	if err != nil {
		log.Error("Failed to upgrade item", "error", err, "username", req.Username, "item", req.Item)
		statusCode, userMsg := mapServiceErrorToUserMessage(err)
		respondError(w, statusCode, userMsg)
		return
	}

	log.Info("Item upgraded successfully",
		"username", req.Username,
		"item", req.Item,
		"quantity_upgraded", result.Quantity,
		"masterwork", result.IsMasterwork)

	// Track engagement for crafting
	TrackEngagement(r.Context(), h.eventBus, req.Username, "item_crafted", result.Quantity)

	// Record contribution for crafting
	if err := h.progressionSvc.RecordEngagement(r.Context(), req.Username, "item_crafted", result.Quantity); err != nil {
		log.Error("Failed to record upgrade engagement", "error", err)
		// Don't fail the request
	}

	// Construct user message
	message := fmt.Sprintf("Successfully upgraded to %dx %s", result.Quantity, result.ItemName)
	if result.IsMasterwork {
		message = fmt.Sprintf("MASTERWORK! Critical success! You received %dx %s (Bonus: +%d)", result.Quantity, result.ItemName, result.BonusQuantity)
	}

	respondJSON(w, http.StatusOK, UpgradeItemResponse{
		Message:          message,
		NewItem:          result.ItemName,
		QuantityUpgraded: result.Quantity,
		IsMasterwork:     result.IsMasterwork,
		BonusQuantity:    result.BonusQuantity,
	})
}

// HandleGetRecipes returns recipe information based on query parameters
// @Summary Get recipes
// @Description Get recipe information. Can filter by item or get all unlocked recipes for a user.
// @Tags crafting
// @Produce json
// @Param item query string false "Item name to get recipe for"
// @Param user query string false "Username to get unlocked recipes for"
// @Param platform query string false "Platform (required if user provided)"
// @Param platform_id query string false "Platform ID (self-mode, optional for target-mode)"
// @Success 200 {object} map[string]interface{} "Recipes or single recipe"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /recipes [get]
func (h *CraftingHandler) HandleGetRecipes(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	itemName := r.URL.Query().Get("item")
	username := r.URL.Query().Get("user")

	log.Debug("Get recipes request", "item", itemName, "user", username)

	// Case 1: Only user provided - return unlocked recipes (dual-mode support)
	if username != "" && itemName == "" {
		platform, ok := GetQueryParam(r, w, "platform")
		if !ok {
			return
		}

		platformID := r.URL.Query().Get("platform_id")

		// Target-mode: resolve user by username if platform_id not provided
		if platformID == "" {
			user, err := h.userRepo.GetUserByPlatformUsername(r.Context(), platform, username)
			if err != nil {
				log.Error("Failed to find user by username", "error", err, "platform", platform, "username", username)
				respondError(w, http.StatusNotFound, "User not found")
				return
			}
			platformID = getPlatformID(user, platform)
			if platformID == "" {
				log.Error("User found but no platform ID", "username", username, "platform", platform)
				respondError(w, http.StatusNotFound, "User not found on platform")
				return
			}
			log.Debug("Resolved username to platform_id", "username", username, "platform_id", platformID)
		}

		recipes, err := h.service.GetUnlockedRecipes(r.Context(), platform, platformID, username)
		if err != nil {
			log.Error("Failed to get unlocked recipes", "error", err, "username", username)
			statusCode, userMsg := mapServiceErrorToUserMessage(err)
			respondError(w, statusCode, userMsg)
			return
		}

		log.Info("Unlocked recipes retrieved", "username", username, "count", len(recipes))

		respondJSON(w, http.StatusOK, UnlockedRecipesResponse{
			Recipes: recipes,
		})
		return
	}

	// Case 2: Item provided (with or without user) - return recipe info
	if itemName != "" {
		platform := r.URL.Query().Get("platform")
		platformID := r.URL.Query().Get("platform_id")

		recipe, err := h.service.GetRecipe(r.Context(), itemName, platform, platformID, username)
		if err != nil {
			log.Error("Failed to get recipe", "error", err, "item", itemName)
			statusCode, userMsg := mapServiceErrorToUserMessage(err)
			respondError(w, statusCode, userMsg)
			return
		}

		log.Info("Recipe retrieved", "item", itemName, "user", username)

		respondJSON(w, http.StatusOK, recipe)
		return
	}

	recipes, err := h.service.GetAllRecipes(r.Context())
	if err != nil {
		log.Error("Failed to get all recipes", "error", err)
		statusCode, userMsg := mapServiceErrorToUserMessage(err)
		respondError(w, statusCode, userMsg)
		return
	}

	respondJSON(w, http.StatusOK, AllRecipesResponse{
		Recipes: recipes,
	})
}

// UnlockedRecipesResponse defines response for unlocked recipes
type UnlockedRecipesResponse struct {
	Recipes []repository.UnlockedRecipeInfo `json:"recipes"`
}

// AllRecipesResponse defines response for all recipes
type AllRecipesResponse struct {
	Recipes []repository.RecipeListItem `json:"recipes"`
}

type DisassembleItemResponse struct {
	Message           string         `json:"message"`
	Outputs           map[string]int `json:"outputs"`
	QuantityProcessed int            `json:"quantity_processed"`
	IsPerfectSalvage  bool           `json:"is_perfect_salvage"`
	Multiplier        float64        `json:"multiplier"`
}

// HandleDisassembleItem handles disassembling items
// @Summary Disassemble item
// @Description Disassemble an item into materials
// @Tags crafting
// @Accept json
// @Produce json
// @Param request body CraftingActionRequest true "Disassemble details"
// @Success 200 {object} DisassembleItemResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse "Feature locked"
// @Failure 500 {object} ErrorResponse
// @Router /user/item/disassemble [post]
func (h *CraftingHandler) HandleDisassembleItem(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	// Check if disassemble feature is unlocked
	if CheckFeatureLocked(w, r, h.progressionSvc, progression.FeatureDisassemble) {
		return
	}

	req, err := decodeCraftingRequest(r, w, "Disassemble item")
	if err != nil {
		return
	}

	result, err := h.service.DisassembleItem(r.Context(), req.Platform, req.PlatformID, req.Username, req.Item, req.Quantity)
	if err != nil {
		log.Error("Failed to disassemble item", "error", err, "username", req.Username, "item", req.Item)
		statusCode, userMsg := mapServiceErrorToUserMessage(err)
		respondError(w, statusCode, userMsg)
		return
	}

	log.Info("Item disassembled successfully",
		"username", req.Username,
		"item", req.Item,
		"quantity_processed", result.QuantityProcessed,
		"outputs", result.Outputs)

	// Track engagement for disassembling
	TrackEngagement(r.Context(), h.eventBus, req.Username, "item_disassembled", result.QuantityProcessed)

	// Construct user message
	// Optimization: Use strings.Builder and avoid fmt.Sprintf in loop
	var sb strings.Builder

	// Sort keys for deterministic output
	keys := make([]string, 0, len(result.Outputs))
	for k := range result.Outputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		if i > 0 {
			sb.WriteString(", ")
		}
		v := result.Outputs[k]
		sb.WriteString(strconv.Itoa(v))
		sb.WriteString("x ")
		sb.WriteString(k)
	}
	outputStr := sb.String()

	message := fmt.Sprintf("Disassembled %d items into: %s", result.QuantityProcessed, outputStr)
	if result.IsPerfectSalvage {
		message = fmt.Sprintf("PERFECT SALVAGE! You efficiently recovered more materials! (+50%% Bonus): %s", outputStr)
	}

	respondJSON(w, http.StatusOK, DisassembleItemResponse{
		Message:           message,
		Outputs:           result.Outputs,
		QuantityProcessed: result.QuantityProcessed,
		IsPerfectSalvage:  result.IsPerfectSalvage,
		Multiplier:        result.Multiplier,
	})
}
