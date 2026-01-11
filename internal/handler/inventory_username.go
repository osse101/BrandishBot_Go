package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/middleware"
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
// @Description Add an item to a user's inventory using only platform and username
// @Tags inventory
// @Accept json
// @Produce json
// @Param request body AddItemByUsernameRequest true "Item details"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user/item/add-by-username [post]
func HandleAddItemByUsername(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		var req AddItemByUsernameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode add item by username request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Debug("Add item by username request",
			"username", req.Username,
			"item", req.ItemName,
			"quantity", req.Quantity)

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			log.Warn("Invalid request", "error", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		if err := svc.AddItemByUsername(r.Context(), req.Platform, req.Username, req.ItemName, req.Quantity); err != nil {
			log.Error("Failed to add item by username", "error", err, "username", req.Username, "item", req.ItemName)
			http.Error(w, ErrMsgAddItemFailed, http.StatusInternalServerError)
			return
		}

		log.Info("Item added successfully by username", "username", req.Username, "item", req.ItemName, "quantity", req.Quantity)

		respondJSON(w, http.StatusOK, SuccessResponse{Message: "Item added successfully"})
	}
}

type RemoveItemByUsernameRequest struct {
	Platform string `json:"platform" validate:"required,platform"`
	Username string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ItemName string `json:"item_name" validate:"required,max=100"`
	Quantity int    `json:"quantity" validate:"min=1,max=10000"`
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
// @Router /user/item/remove-by-username [post]
func HandleRemoveItemByUsername(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		var req RemoveItemByUsernameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode remove item by username request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Debug("Remove item by username request", "username", req.Username, "item", req.ItemName, "quantity", req.Quantity)

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			log.Warn("Invalid request", "error", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		removed, err := svc.RemoveItemByUsername(r.Context(), req.Platform, req.Username, req.ItemName, req.Quantity)
		if err != nil {
			log.Error("Failed to remove item by username", "error", err, "username", req.Username, "item", req.ItemName)
			http.Error(w, ErrMsgRemoveItemFailed, http.StatusInternalServerError)
			return
		}

		log.Info("Item removed successfully by username", "username", req.Username, "item", req.ItemName, "removed", removed)

		respondJSON(w, http.StatusOK, RemoveItemResponse{Removed: removed})
	}
}

type UseItemByUsernameRequest struct {
	Platform   string `json:"platform" validate:"required,platform"`
	Username   string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ItemName   string `json:"item_name" validate:"required,max=100"`
	Quantity   int    `json:"quantity" validate:"min=1,max=10000"`
	TargetUser string `json:"target_user,omitempty" validate:"omitempty,max=100,excludesall=\x00\n\r\t"`
}

// HandleUseItemByUsername handles using an item by username only
// @Summary Use item by username
// @Description Use an item from a user's inventory using only platform and username
// @Tags inventory
// @Accept json
// @Produce json
// @Param request body UseItemByUsernameRequest true "Usage details"
// @Success 200 {object} UseItemResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user/item/use-by-username [post]
func HandleUseItemByUsername(svc user.Service, eventBus event.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		var req UseItemByUsernameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode use item by username request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Default quantity to 1 if not provided (0)
		if req.Quantity <= 0 {
			req.Quantity = 1
		}

		log.Debug("Use item by username request",
			"username", req.Username,
			"item", req.ItemName,
			"quantity", req.Quantity,
			"target", req.TargetUser)

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			log.Warn("Invalid request", "error", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		message, err := svc.UseItemByUsername(r.Context(), req.Platform, req.Username, req.ItemName, req.Quantity, req.TargetUser)
		if err != nil {
			log.Error("Failed to use item by username", "error", err, "username", req.Username, "item", req.ItemName)
			http.Error(w, ErrMsgUseItemFailed, http.StatusInternalServerError)
			return
		}

		log.Info("Item used successfully by username",
			"username", req.Username,
			"item", req.ItemName,
			"quantity", req.Quantity,
			"message", message)

		// Track engagement for item usage
		middleware.TrackEngagementFromContext(
			middleware.WithUserID(r.Context(), req.Username),
			eventBus,
			"item_used",
			req.Quantity,
		)

		// Publish item.used event
		if err := eventBus.Publish(r.Context(), event.Event{
			Version: "1.0",
			Type:    "item.used",
			Payload: map[string]interface{}{
				"user_id":  req.Username,
				"item":     req.ItemName,
				"quantity": req.Quantity,
				"target":   req.TargetUser,
				"result":   message,
			},
		}); err != nil {
			log.Error("Failed to publish item.used event", "error", err)
		}

		respondJSON(w, http.StatusOK, UseItemResponse{
			Message: message,
		})
	}
}

type GiveItemByUsernameRequest struct {
	FromPlatform string `json:"from_platform" validate:"required,platform"`
	FromUsername string `json:"from_username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ToPlatform   string `json:"to_platform" validate:"required,platform"`
	ToUsername   string `json:"to_username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ItemName     string `json:"item_name" validate:"required,max=100"`
	Quantity     int    `json:"quantity" validate:"min=1,max=10000"`
}

// HandleGiveItemByUsername handles giving items by username only
// @Summary Give item by username
// @Description Transfer an item between users using only usernames
// @Tags inventory
// @Accept json
// @Produce json
// @Param request body GiveItemByUsernameRequest true "Transfer details"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user/item/give-by-username [post]
func HandleGiveItemByUsername(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		var req GiveItemByUsernameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode give item by username request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Debug("Give item by username request",
			"from", req.FromUsername,
			"to", req.ToUsername,
			"item", req.ItemName,
			"quantity", req.Quantity)

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			log.Warn("Invalid request", "error", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		message, err := svc.GiveItemByUsername(r.Context(), req.FromPlatform, req.FromUsername, req.ToPlatform, req.ToUsername, req.ItemName, req.Quantity)
		if err != nil {
			log.Error("Failed to give item by username", "error", err, "from", req.FromUsername, "to", req.ToUsername)
			http.Error(w, ErrMsgGiveItemFailed, http.StatusInternalServerError)
			return
		}

		log.Info("Item given successfully by username", "from", req.FromUsername, "to", req.ToUsername, "item", req.ItemName, "quantity", req.Quantity)

		respondJSON(w, http.StatusOK, SuccessResponse{Message: message})
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
// @Router /user/inventory-by-username [get]
func HandleGetInventoryByUsername(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		platform := r.URL.Query().Get("platform")
		if platform == "" {
			platform = "discord" // Default
		}
		username := r.URL.Query().Get("username")
		if username == "" {
			log.Warn("Missing username query parameter")
			http.Error(w, "Missing username query parameter", http.StatusBadRequest)
			return
		}
		filter := r.URL.Query().Get("filter")

		log.Debug("Get inventory by username request", "username", username, "filter", filter)

		items, err := svc.GetInventoryByUsername(r.Context(), platform, username, filter)
		if err != nil {
			log.Error("Failed to get inventory by username", "error", err, "username", username)
			http.Error(w, ErrMsgGetInventoryFailed, http.StatusInternalServerError)
			return
		}

		log.Info("Inventory retrieved by username", "username", username, "item_count", len(items))

		respondJSON(w, http.StatusOK, GetInventoryResponse{
			Items: items,
		})
	}
}
