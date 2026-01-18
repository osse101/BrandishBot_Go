package handler

import (
	"fmt"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/economy"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/middleware"
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
type RemoveItemResponse struct {
	Removed int `json:"removed"`
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

type GiveItemRequest struct {
	OwnerPlatform      string `json:"owner_platform" validate:"required,platform"`
	OwnerPlatformID    string `json:"owner_platform_id" validate:"required"`
	Owner              string `json:"owner" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ReceiverPlatform   string `json:"receiver_platform" validate:"required,platform"`
	Receiver           string `json:"receiver" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ItemName           string `json:"item_name" validate:"required,max=100"`
	Quantity           int    `json:"quantity" validate:"min=1,max=10000"`
}

// HandleGiveItem handles transferring items between users
// @Summary Give item to another user
// @Description Transfer an item from one user to another
// @Tags inventory
// @Accept json
// @Produce json
// @Param request body GiveItemRequest true "Transfer details"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user/item/give [post]
func HandleGiveItem(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req GiveItemRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Give item"); err != nil {
			return
		}

		log := logger.FromContext(r.Context())

		if err := svc.GiveItem(r.Context(), req.OwnerPlatform, req.OwnerPlatformID, req.Owner, req.ReceiverPlatform, req.Receiver, req.ItemName, req.Quantity); err != nil {
			log.Error("Failed to give item", "error", err, "owner", req.Owner, "receiver", req.Receiver, "item", req.ItemName)
			statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
			return
		}

		log.Info("Item transferred successfully", "owner", req.Owner, "receiver", req.Receiver, "item", req.ItemName, "quantity", req.Quantity)

		respondJSON(w, http.StatusOK, SuccessResponse{Message: MsgItemTransferredSuccess})
	}
}

type SellItemRequest struct {
	Platform   string `json:"platform" validate:"required,platform"`
	PlatformID string `json:"platform_id" validate:"required"`
	Username   string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ItemName   string `json:"item_name" validate:"required,max=100"`
	Quantity   int    `json:"quantity" validate:"min=1,max=10000"`
}

type SellItemResponse struct {
	MoneyGained int `json:"money_gained"`
	ItemsSold   int `json:"items_sold"`
}

// HandleSellItem handles selling items for currency
// @Summary Sell item
// @Description Sell an item for currency
// @Tags economy
// @Accept json
// @Produce json
// @Param request body SellItemRequest true "Sell details"
// @Success 200 {object} SellItemResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse "Feature locked"
// @Failure 500 {object} ErrorResponse
// @Router /user/item/sell [post]
func HandleSellItem(svc economy.Service, progressionSvc progression.Service, eventBus event.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if sell feature is unlocked
		if CheckFeatureLocked(w, r, progressionSvc, progression.FeatureSell) {
			return
		}

		var req SellItemRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Sell item"); err != nil {
			return
		}

		log := logger.FromContext(r.Context())

		moneyGained, itemsSold, err := svc.SellItem(r.Context(), req.Platform, req.PlatformID, req.Username, req.ItemName, req.Quantity)
		if err != nil {
			log.Error("Failed to sell item", "error", err, "username", req.Username, "item", req.ItemName)
			statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
			return
		}

		log.Info("Item sold successfully",
			"username", req.Username,
			"item", req.ItemName,
			"items_sold", itemsSold,
			"money_gained", moneyGained)

		// Track engagement for selling
		middleware.TrackEngagementFromContext(
			middleware.WithUserID(r.Context(), req.Username),
			eventBus,
			"item_sold",
			itemsSold,
		)

		// Publish item.sold event
		if err := PublishEvent(r.Context(), eventBus, "item.sold", map[string]interface{}{
			"user_id":      req.Username,
			"item_name":    req.ItemName,
			"quantity":     itemsSold,
			"money_gained": moneyGained,
		}); err != nil {
			_ = err // Error already logged in PublishEvent
		}

		respondJSON(w, http.StatusOK, SellItemResponse{
			MoneyGained: moneyGained,
			ItemsSold:   itemsSold,
		})
	}
}

type BuyItemRequest struct {
	Platform   string `json:"platform" validate:"required,platform"`
	PlatformID string `json:"platform_id" validate:"required"`
	Username   string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ItemName   string `json:"item_name" validate:"required,max=100"`
	Quantity   int    `json:"quantity" validate:"min=1,max=10000"`
}

type BuyItemResponse struct {
	ItemsBought int `json:"items_bought"`
}

// HandleBuyItem handles buying items with currency
// @Summary Buy item
// @Description Buy an item with currency
// @Tags economy
// @Accept json
// @Produce json
// @Param request body BuyItemRequest true "Buy details"
// @Success 200 {object} BuyItemResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse "Feature locked"
// @Failure 500 {object} ErrorResponse
// @Router /user/item/buy [post]
func HandleBuyItem(svc economy.Service, progressionSvc progression.Service, eventBus event.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if buy feature is unlocked
		if CheckFeatureLocked(w, r, progressionSvc, progression.FeatureBuy) {
			return
		}

		var req BuyItemRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Buy item"); err != nil {
			return
		}

		log := logger.FromContext(r.Context())

		bought, err := svc.BuyItem(r.Context(), req.Platform, req.PlatformID, req.Username, req.ItemName, req.Quantity)
		if err != nil {
			log.Error("Failed to buy item", "error", err, "username", req.Username, "item", req.ItemName)
			statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
			return
		}

		log.Info("Item purchased successfully",
			"username", req.Username,
			"item", req.ItemName,
			"items_bought", bought)

		// Track engagement for buying
		middleware.TrackEngagementFromContext(
			middleware.WithUserID(r.Context(), req.Username),
			eventBus,
			"item_bought",
			bought,
		)

		// Publish item.bought event
		// Note: We don't have the exact cost here, would need to modify economy.Service to return it
		if err := PublishEvent(r.Context(), eventBus, "item.bought", map[string]interface{}{
			"user_id":   req.Username,
			"item_name": req.ItemName,
			"quantity":  bought,
		}); err != nil {
			_ = err // Error already logged in PublishEvent
		}

		respondJSON(w, http.StatusOK, BuyItemResponse{
			ItemsBought: bought,
		})
	}
}

type UseItemRequest struct {
	Platform   string `json:"platform" validate:"required,platform"`
	PlatformID string `json:"platform_id" validate:"required"`
	Username   string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ItemName   string `json:"item_name" validate:"required,max=100"`
	Quantity   int    `json:"quantity" validate:"min=1,max=10000"`
	TargetUser string `json:"target_user,omitempty" validate:"omitempty,max=100,excludesall=\x00\n\r\t"`
}

type UseItemResponse struct {
	Message string `json:"message"`
}

// HandleUseItem handles using an item
// @Summary Use item
// @Description Use an item from inventory
// @Tags inventory
// @Accept json
// @Produce json
// @Param request body UseItemRequest true "Usage details"
// @Success 200 {object} UseItemResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user/item/use [post]
func HandleUseItem(svc user.Service, eventBus event.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req UseItemRequest
		if err := DecodeAndValidateRequest(r, w, &req, "Use item"); err != nil {
			return
		}

		// Default quantity to 1 if not provided (0)
		if req.Quantity <= 0 {
			req.Quantity = 1
		}

		log := logger.FromContext(r.Context())

		message, err := svc.UseItem(r.Context(), req.Platform, req.PlatformID, req.Username, req.ItemName, req.Quantity, req.TargetUser)
		if err != nil {
			log.Error("Failed to use item", "error", err, "username", req.Username, "item", req.ItemName)
			statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
			return
		}

		log.Info("Item used successfully",
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
// @Router /user/inventory [get]
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
				statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
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
			statusCode, userMsg := mapServiceErrorToUserMessage(err); respondError(w, statusCode, userMsg)
			return
		}

		log.Info("Inventory retrieved", "username", username, "item_count", len(items))

		respondJSON(w, http.StatusOK, GetInventoryResponse{
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

