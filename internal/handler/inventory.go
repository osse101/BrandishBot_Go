package handler

import (
	"encoding/json"
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

type AddItemRequest struct {
	Platform   string `json:"platform" validate:"required,platform"`
	PlatformID string `json:"platform_id" validate:"required"`
	Username   string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ItemName   string `json:"item_name" validate:"required,max=100"`
	Quantity   int    `json:"quantity" validate:"min=1,max=10000"`
}

// HandleAddItem handles adding items to a user's inventory
// @Summary Add item to inventory
// @Description Add an item to a user's inventory (admin/system action)
// @Tags inventory
// @Accept json
// @Produce json
// @Param request body AddItemRequest true "Item details"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user/item/add [post]
func HandleAddItem(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		var req AddItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode add item request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Debug("Add item request",
			"username", req.Username,
			"item", req.ItemName,
			"quantity", req.Quantity)

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			log.Warn("Invalid request", "error", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		if err := svc.AddItem(r.Context(), req.Platform, req.PlatformID, req.Username, req.ItemName, req.Quantity); err != nil {
			log.Error("Failed to add item", "error", err, "username", req.Username, "item", req.ItemName)
			http.Error(w, ErrMsgAddItemFailed, http.StatusInternalServerError)
			return
		}

		log.Info("Item added successfully", "username", req.Username, "item", req.ItemName, "quantity", req.Quantity)

		respondJSON(w, http.StatusOK, SuccessResponse{Message: "Item added successfully"})
	}
}

type RemoveItemRequest struct {
	Platform   string `json:"platform" validate:"required,platform"`
	PlatformID string `json:"platform_id" validate:"required"`
	Username   string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ItemName   string `json:"item_name" validate:"required,max=100"`
	Quantity   int    `json:"quantity" validate:"min=1,max=10000"`
}

type RemoveItemResponse struct {
	Removed int `json:"removed"`
}

// HandleRemoveItem handles removing items from a user's inventory
// @Summary Remove item from inventory
// @Description Remove an item from a user's inventory
// @Tags inventory
// @Accept json
// @Produce json
// @Param request body RemoveItemRequest true "Item details"
// @Success 200 {object} RemoveItemResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /user/item/remove [post]
func HandleRemoveItem(svc user.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.FromContext(r.Context())

		var req RemoveItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode remove item request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Debug("Remove item request", "username", req.Username, "item", req.ItemName, "quantity", req.Quantity)

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			log.Warn("Invalid request", "error", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		removed, err := svc.RemoveItem(r.Context(), req.Platform, req.PlatformID, req.Username, req.ItemName, req.Quantity)
		if err != nil {
			log.Error("Failed to remove item", "error", err, "username", req.Username, "item", req.ItemName)
			http.Error(w, ErrMsgRemoveItemFailed, http.StatusInternalServerError)
			return
		}

		log.Info("Item removed successfully", "username", req.Username, "item", req.ItemName, "removed", removed)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		respondJSON(w, http.StatusOK, RemoveItemResponse{Removed: removed})
	}
}

type GiveItemRequest struct {
	OwnerPlatform      string `json:"owner_platform" validate:"required,platform"`
	OwnerPlatformID    string `json:"owner_platform_id" validate:"required"`
	Owner              string `json:"owner" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ReceiverPlatform   string `json:"receiver_platform" validate:"required,platform"`
	ReceiverPlatformID string `json:"receiver_platform_id" validate:"required"`
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
		log := logger.FromContext(r.Context())

		var req GiveItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode give item request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Debug("Give item request",
			"owner", req.Owner,
			"receiver", req.Receiver,
			"item", req.ItemName,
			"quantity", req.Quantity)

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			log.Warn("Invalid request", "error", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		if err := svc.GiveItem(r.Context(), req.OwnerPlatform, req.OwnerPlatformID, req.Owner, req.ReceiverPlatform, req.ReceiverPlatformID, req.Receiver, req.ItemName, req.Quantity); err != nil {
			log.Error("Failed to give item", "error", err, "owner", req.Owner, "receiver", req.Receiver, "item", req.ItemName)
			http.Error(w, ErrMsgGiveItemFailed, http.StatusInternalServerError)
			return
		}

		log.Info("Item transferred successfully", "owner", req.Owner, "receiver", req.Receiver, "item", req.ItemName, "quantity", req.Quantity)

		respondJSON(w, http.StatusOK, SuccessResponse{Message: "Item transferred successfully"})
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
		log := logger.FromContext(r.Context())

		// Check if sell feature is unlocked
		// Check if sell feature is unlocked
		if CheckFeatureLocked(w, r, progressionSvc, progression.FeatureSell) {
			return
		}

		var req SellItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode sell item request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Debug("Sell item request", "username", req.Username, "item", req.ItemName, "quantity", req.Quantity)

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			log.Warn("Invalid request", "error", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		moneyGained, itemsSold, err := svc.SellItem(r.Context(), req.Platform, req.PlatformID, req.Username, req.ItemName, req.Quantity)
		if err != nil {
			log.Error("Failed to sell item", "error", err, "username", req.Username, "item", req.ItemName)
			http.Error(w, ErrMsgSellItemFailed, http.StatusInternalServerError)
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
		if err := eventBus.Publish(r.Context(), event.Event{
			Type: "item.sold",
			Payload: map[string]interface{}{
				"user_id":      req.Username,
				"item_name":    req.ItemName,
				"quantity":     itemsSold,
				"money_gained": moneyGained,
			},
		}); err != nil {
			log.Error("Failed to publish item.sold event", "error", err)
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
		log := logger.FromContext(r.Context())

		// Check if buy feature is unlocked
		// Check if buy feature is unlocked
		if CheckFeatureLocked(w, r, progressionSvc, progression.FeatureBuy) {
			return
		}

		var req BuyItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode buy item request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		log.Debug("Buy item request", "username", req.Username, "item", req.ItemName, "quantity", req.Quantity)

		// Validate request
		if err := GetValidator().ValidateStruct(req); err != nil {
			log.Warn("Invalid request", "error", err)
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		bought, err := svc.BuyItem(r.Context(), req.Platform, req.PlatformID, req.Username, req.ItemName, req.Quantity)
		if err != nil {
			log.Error("Failed to buy item", "error", err, "username", req.Username, "item", req.ItemName)
			http.Error(w, ErrMsgBuyItemFailed, http.StatusInternalServerError)
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
		if err := eventBus.Publish(r.Context(), event.Event{
			Type: "item.bought",
			Payload: map[string]interface{}{
				"user_id":   req.Username,
				"item_name": req.ItemName,
				"quantity":  bought,
			},
		}); err != nil {
			log.Error("Failed to publish item.bought event", "error", err)
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
		log := logger.FromContext(r.Context())

		var req UseItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("Failed to decode use item request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Default quantity to 1 if not provided (0)
		if req.Quantity <= 0 {
			req.Quantity = 1
		}

		log.Debug("Use item request",
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

		message, err := svc.UseItem(r.Context(), req.Platform, req.PlatformID, req.Username, req.ItemName, req.Quantity, req.TargetUser)
		if err != nil {
			log.Error("Failed to use item", "error", err, "username", req.Username, "item", req.ItemName)
			http.Error(w, ErrMsgUseItemFailed, http.StatusInternalServerError)
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
		if err := eventBus.Publish(r.Context(), event.Event{
			Type: "item.used",
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

type GetInventoryResponse struct {
	Items []user.UserInventoryItem `json:"items"`
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

		platform := r.URL.Query().Get("platform") // optional
		if platform == "" {
			platform = "discord" // Default
		}
		platformID := r.URL.Query().Get("platform_id")
		if platformID == "" {
			log.Warn("Missing platform_id query parameter")
			http.Error(w, "Missing platform_id query parameter", http.StatusBadRequest)
			return
		}
		username := r.URL.Query().Get("username")
		if username == "" {
			log.Warn("Missing username query parameter")
			http.Error(w, "Missing username query parameter", http.StatusBadRequest)
			return
		}
		filter := r.URL.Query().Get("filter")

		// Check filter unlock status
		if filter != "" {
			featureKey := fmt.Sprintf("feature_filter_%s", filter)
			// We only check locks for the specific ones we added.
			if filter == domain.FilterTypeUpgrade || filter == domain.FilterTypeSellable || filter == domain.FilterTypeConsumable {
				unlocked, err := progSvc.IsFeatureUnlocked(r.Context(), featureKey)
				if err != nil {
					log.Error("Failed to check filter unlock", "error", err)
					http.Error(w, ErrMsgFeatureCheckFailed, http.StatusInternalServerError)
					return
				}
				if !unlocked {
					log.Warn("Filter locked", "filter", filter, "username", username)
					http.Error(w, fmt.Sprintf("Filter '%s' is locked. Unlock it in the progression tree.", filter), http.StatusForbidden)
					return
				}
			}
		}

		log.Debug("Get inventory request", "username", username, "filter", filter)

		items, err := svc.GetInventory(r.Context(), platform, platformID, username, filter)
		if err != nil {
			log.Error("Failed to get inventory", "error", err, "username", username)
			http.Error(w, ErrMsgGetInventoryFailed, http.StatusInternalServerError)
			return
		}

		log.Info("Inventory retrieved", "username", username, "item_count", len(items))

		respondJSON(w, http.StatusOK, GetInventoryResponse{
			Items: items,
		})
	}
}
