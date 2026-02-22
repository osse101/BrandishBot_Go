package handler

import (
	"fmt"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/progression"
	"github.com/osse101/BrandishBot_Go/internal/user"
)

// Username-based inventory operations (no platformID required)
type AddItemByUsernameRequest struct {
	Platform string `json:"platform" validate:"required,platform"`
	Username string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ItemName string `json:"item_name" validate:"required,max=100"`
	Quantity int    `json:"quantity" validate:"min=1,max=10000"`
}

// HandleAddItemByUsername handles adding items by username only
// @Summary Add item by username
// @Description Add an item to a user's inventory using only platform and username. This is an admin/system action.
// @Tags inventory
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body AddItemByUsernameRequest true "Item details including platform, username, and quantity"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/user/item/add [post]
func HandleAddItemByUsername(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AddItemByUsernameRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Add item by username"); err != nil {
			return
		}

		log := logger.FromContext(r.Context())

		if err := svc.AddItemByUsername(r.Context(), req.Platform, req.Username, req.ItemName, req.Quantity); err != nil {
			log.Error("Failed to add item by username", "error", err, "username", req.Username, "item", req.ItemName)
			statusCode, userMsg := MapServiceErrorToUserMessage(err)
			RespondError(w, statusCode, userMsg)
			return
		}

		log.Info("Item added successfully by username", "username", req.Username, "item", req.ItemName, "quantity", req.Quantity)

		RespondJSON(w, http.StatusOK, SuccessResponse{Message: MsgItemAddedSuccess})
	}
}

type RemoveItemByUsernameRequest struct {
	Platform string `json:"platform" validate:"required,platform"`
	Username string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ItemName string `json:"item_name" validate:"required,max=100"`
	Quantity int    `json:"quantity" validate:"min=1,max=10000"`
}
type RemoveItemResponse struct {
	Message string `json:"message"`
	Removed int    `json:"removed"`
}

// HandleRemoveItemByUsername handles removing items by username only
// @Summary Remove item by username
// @Description Remove an item from a user's inventory using only platform and username
// @Tags inventory
// @Accept json
// @Produce json
// @Param request body RemoveItemByUsernameRequest true "Item details"
// @Success 200 {object} RemoveItemResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/user/item/remove [post]
func HandleRemoveItemByUsername(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RemoveItemByUsernameRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Remove item by username"); err != nil {
			return
		}

		log := logger.FromContext(r.Context())

		removed, err := svc.RemoveItemByUsername(r.Context(), req.Platform, req.Username, req.ItemName, req.Quantity)
		if err != nil {
			log.Error("Failed to remove item by username", "error", err, "username", req.Username, "item", req.ItemName)
			statusCode, userMsg := MapServiceErrorToUserMessage(err)
			RespondError(w, statusCode, userMsg)
			return
		}

		log.Info("Item removed successfully by username", "username", req.Username, "item", req.ItemName, "removed", removed)

		RespondJSON(w, http.StatusOK, RemoveItemResponse{
			Message: fmt.Sprintf("Removed %dx %s from %s", removed, req.ItemName, req.Username),
			Removed: removed,
		})
	}
}

type GetInventoryResponse struct {
	Items []user.InventoryItem `json:"items"`
}

// HandleGetInventory gets the user's inventory
// @Summary Get inventory
// @Description Get the user's inventory
// @Tags inventory
// @Accept json
// @Produce json
// @Param platform_id query string true "Platform ID"
// @Param username query string true "Username"
// @Param filter query string false "Filter by item type (upgrade, sellable, consumable)"
// @Success 200 {object} GetInventoryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/user/inventory [get]
func HandleGetInventory(svc user.Service, progSvc progression.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		platform, ok := GetQueryParam(r, w, "platform")
		if !ok {
			return
		}

		platformID, ok := GetQueryParam(r, w, "platform_id")
		if !ok {
			return
		}
		username, ok := GetQueryParam(r, w, "username")
		if !ok {
			return
		}
		filter := r.URL.Query().Get("filter")

		// Validate filter parameter
		if filter != "" && !domain.IsValidFilterType(filter) {
			log.Warn("Invalid filter parameter", "filter", filter)
			http.Error(w, fmt.Sprintf(ErrMsgInvalidFilterType, filter), http.StatusBadRequest)
			return
		}

		// Check filter unlock status
		if filter != "" {
			featureKey := fmt.Sprintf("feature_filter_%s", filter)
			// We only check locks for the specific ones we added.
			unlocked, err := progSvc.IsFeatureUnlocked(r.Context(), featureKey)
			if err != nil {
				log.Error("Failed to check filter unlock", "error", err)
				statusCode, userMsg := MapServiceErrorToUserMessage(err)
				RespondError(w, statusCode, userMsg)
				return
			}
			if !unlocked {
				log.Warn("Filter locked", "filter", filter, "username", username)
				http.Error(w, fmt.Sprintf(ErrMsgFilterLocked, filter), http.StatusForbidden)
				return
			}
		}

		log.Debug("Get inventory request", "username", username, "filter", filter)

		items, err := svc.GetInventory(r.Context(), platform, platformID, username, filter)
		if err != nil {
			log.Error("Failed to get inventory", "error", err, "username", username)
			statusCode, userMsg := MapServiceErrorToUserMessage(err)
			RespondError(w, statusCode, userMsg)
			return
		}

		log.Info("Inventory retrieved", "username", username, "item_count", len(items))

		RespondJSON(w, http.StatusOK, GetInventoryResponse{
			Items: items,
		})
	}
}

// HandleGetInventoryByUsername gets inventory by username only
// @Summary Get inventory by username
// @Description Get a user's inventory using only platform and username
// @Tags inventory
// @Accept json
// @Produce json
// @Param platform query string true "Platform"
// @Param username query string true "Username"
// @Param filter query string false "Filter by item type"
// @Success 200 {object} GetInventoryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/user/inventory-by-username [get]
func HandleGetInventoryByUsername(svc user.Service, progSvc progression.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		platform := GetOptionalQueryParam(r, "platform", "discord")

		username, ok := GetQueryParam(r, w, "username")
		if !ok {
			return
		}
		filter := r.URL.Query().Get("filter")

		// Validate filter parameter
		if filter != "" && !domain.IsValidFilterType(filter) {
			log.Warn("Invalid filter parameter", "filter", filter)
			http.Error(w, fmt.Sprintf(ErrMsgInvalidFilterType, filter), http.StatusBadRequest)
			return
		}

		// Check filter unlock status
		if filter != "" {
			featureKey := fmt.Sprintf("feature_filter_%s", filter)
			// We only check locks for the specific ones we added.
			unlocked, err := progSvc.IsFeatureUnlocked(r.Context(), featureKey)
			if err != nil {
				log.Error("Failed to check filter unlock", "error", err)
				statusCode, userMsg := MapServiceErrorToUserMessage(err)
				RespondError(w, statusCode, userMsg)
				return
			}
			if !unlocked {
				log.Warn("Filter locked", "filter", filter, "username", username)
				http.Error(w, fmt.Sprintf(ErrMsgFilterLocked, filter), http.StatusForbidden)
				return
			}
		}

		log.Debug("Get inventory by username request", "username", username, "filter", filter)

		items, err := svc.GetInventoryByUsername(r.Context(), platform, username, filter)
		if err != nil {
			log.Error("Failed to get inventory by username", "error", err, "username", username)
			statusCode, userMsg := MapServiceErrorToUserMessage(err)
			RespondError(w, statusCode, userMsg)
			return
		}

		log.Info("Inventory retrieved by username", "username", username, "item_count", len(items))

		RespondJSON(w, http.StatusOK, GetInventoryResponse{
			Items: items,
		})
	}
}
