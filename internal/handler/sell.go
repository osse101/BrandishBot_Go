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

type SellItemRequest struct {
	Platform   string `json:"platform" validate:"required,platform"`
	PlatformID string `json:"platform_id" validate:"required"`
	Username   string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ItemName   string `json:"item_name" validate:"required,max=100"`
	Quantity   int    `json:"quantity" validate:"min=1,max=10000"`
}

type SellItemResponse struct {
	Message     string `json:"message"`
	MoneyGained int    `json:"money_gained"`
	ItemsSold   int    `json:"items_sold"`
}

// HandleSellItem handles selling items for currency
// @Summary Sell item
// @Description Sell an item from inventory for currency. Requires Economy feature to be unlocked.
// @Tags economy
// @Accept json
// @Produce json
// @Param request body SellItemRequest true "Details of the item to sell and quantity"
// @Success 200 {object} SellItemResponse
// @Failure 400 {object} ErrorResponse "Item not found or not sellable"
// @Failure 403 {object} ErrorResponse "Economy feature locked"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/user/item/sell [post]
func HandleSellItem(svc economy.Service, userSvc user.ManagementService, progressionSvc progression.Service, eventBus event.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if sell feature is unlocked
		if CheckFeatureLocked(w, r, progressionSvc, progression.FeatureEconomy) {
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
			statusCode, userMsg := MapServiceErrorToUserMessage(err)
			RespondError(w, statusCode, userMsg)
			return
		}

		log.Info("Item sold successfully",
			"username", req.Username,
			"item", req.ItemName,
			"items_sold", itemsSold,
			"money_gained", moneyGained)

		// Attempt to resolve the correct UUID for metrics/events
		eventUserID := req.Username
		if userID, err := userSvc.GetUserIDByPlatformID(r.Context(), req.Platform, req.PlatformID); err == nil && userID != "" {
			eventUserID = userID
			middleware.TrackEngagementFromContext(
				middleware.WithUserID(r.Context(), userID),
				eventBus,
				domain.MetricTypeItemSold,
				itemsSold,
			)
		} else {
			log.Warn("Could not resolve UUID for item sold metrics, using username", "username", req.Username, "error", err)
		}

		// Publish item.sold event
		if err := PublishEvent(r.Context(), eventBus, domain.EventTypeItemSold, map[string]interface{}{
			"user_id":      eventUserID,
			"item_name":    req.ItemName,
			"quantity":     itemsSold,
			"money_gained": moneyGained,
		}); err != nil {
			_ = err // Error already logged in PublishEvent
		}

		RespondJSON(w, http.StatusOK, SellItemResponse{
			Message:     fmt.Sprintf("Sold %dx %s for %d money", itemsSold, req.ItemName, moneyGained),
			MoneyGained: moneyGained,
			ItemsSold:   itemsSold,
		})
	}
}
