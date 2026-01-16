package handler

import (
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
		var req AddItemByUsernameRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Add item by username"); err != nil {
			return
		}

		log := logger.FromContext(r.Context())

		if err := svc.AddItemByUsername(r.Context(), req.Platform, req.Username, req.ItemName, req.Quantity); err != nil {
			log.Error("Failed to add item by username", "error", err, "username", req.Username, "item", req.ItemName)
			statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
			return
		}

		log.Info("Item added successfully by username", "username", req.Username, "item", req.ItemName, "quantity", req.Quantity)

		respondJSON(w, http.StatusOK, SuccessResponse{Message: MsgItemAddedSuccess})
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
		var req RemoveItemByUsernameRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Remove item by username"); err != nil {
			return
		}

		log := logger.FromContext(r.Context())

		removed, err := svc.RemoveItemByUsername(r.Context(), req.Platform, req.Username, req.ItemName, req.Quantity)
		if err != nil {
			log.Error("Failed to remove item by username", "error", err, "username", req.Username, "item", req.ItemName)
			statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
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
		var req UseItemByUsernameRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Use item by username"); err != nil {
			return
		}

		// Default quantity to 1 if not provided (0)
		if req.Quantity <= 0 {
			req.Quantity = 1
		}

		log := logger.FromContext(r.Context())

		message, err := svc.UseItemByUsername(r.Context(), req.Platform, req.Username, req.ItemName, req.Quantity, req.TargetUser)
		if err != nil {
			log.Error("Failed to use item by username", "error", err, "username", req.Username, "item", req.ItemName)
			statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
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
		if err := PublishEvent(r.Context(), eventBus, "item.used", map[string]interface{}{
			"user_id":  req.Username,
			"item":     req.ItemName,
			"quantity": req.Quantity,
			"target":   req.TargetUser,
			"result":   message,
		}); err != nil {
			_ = err // Error already logged in PublishEvent
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
		var req GiveItemByUsernameRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Give item by username"); err != nil {
			return
		}

		log := logger.FromContext(r.Context())

		message, err := svc.GiveItemByUsername(r.Context(), req.FromPlatform, req.FromUsername, req.ToPlatform, req.ToUsername, req.ItemName, req.Quantity)
		if err != nil {
			log.Error("Failed to give item by username", "error", err, "from", req.FromUsername, "to", req.ToUsername)
			statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
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

		platform := GetOptionalQueryParam(r, "platform", "discord")

		username, ok := GetQueryParam(r, w, "username")
		if !ok {
			return
		}
		filter := r.URL.Query().Get("filter")

		log.Debug("Get inventory by username request", "username", username, "filter", filter)

		items, err := svc.GetInventoryByUsername(r.Context(), platform, username, filter)
		if err != nil {
			log.Error("Failed to get inventory by username", "error", err, "username", username)
			statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
			return
		}

		log.Info("Inventory retrieved by username", "username", username, "item_count", len(items))

		respondJSON(w, http.StatusOK, GetInventoryResponse{
			Items: items,
		})
	}
}
