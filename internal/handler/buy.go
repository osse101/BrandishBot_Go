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

type BuyItemRequest struct {
	Platform   string `json:"platform" validate:"required,platform"`
	PlatformID string `json:"platform_id" validate:"required"`
	Username   string `json:"username" validate:"required,max=100,excludesall=\x00\n\r\t"`
	ItemName   string `json:"item_name" validate:"required,max=100"`
	Quantity   int    `json:"quantity" validate:"min=1,max=10000"`
}

type BuyItemResponse struct {
	Message     string `json:"message"`
	ItemsBought int    `json:"items_bought"`
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
// @Router /api/v1/user/item/buy [post]
func HandleBuyItem(svc economy.Service, userSvc user.ManagementService, progressionSvc progression.Service, eventBus event.Bus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if buy feature is unlocked
		if CheckFeatureLocked(w, r, progressionSvc, progression.FeatureEconomy) {
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
			statusCode, userMsg := MapServiceErrorToUserMessage(err)
			RespondError(w, statusCode, userMsg)
			return
		}

		log.Info("Item purchased successfully",
			"username", req.Username,
			"item", req.ItemName,
			"items_bought", bought)

		// Attempt to resolve the correct UUID for metrics/events
		eventUserID := req.Username
		if userID, err := userSvc.GetUserIDByPlatformID(r.Context(), req.Platform, req.PlatformID); err == nil && userID != "" {
			eventUserID = userID
			middleware.TrackEngagementFromContext(
				middleware.WithUserID(r.Context(), userID),
				eventBus,
				domain.MetricTypeItemBought,
				bought,
			)
		} else {
			log.Warn("Could not resolve UUID for item bought metrics, using username", "username", req.Username, "error", err)
		}

		// Record contribution for buying
		if err := progressionSvc.RecordEngagement(r.Context(), eventUserID, domain.MetricTypeItemBought, bought); err != nil {
			log.Error("Failed to record buy engagement", "error", err, "user_id", eventUserID)
			// Don't fail the request
		}

		// Publish item.bought event
		// Note: We don't have the exact cost here, would need to modify economy.Service to return it
		if err := PublishEvent(r.Context(), eventBus, domain.EventTypeItemBought, map[string]interface{}{
			"user_id":   eventUserID,
			"item_name": req.ItemName,
			"quantity":  bought,
		}); err != nil {
			_ = err // Error already logged in PublishEvent
		}

		RespondJSON(w, http.StatusOK, BuyItemResponse{
			Message:     fmt.Sprintf("Purchased %dx %s", bought, req.ItemName),
			ItemsBought: bought,
		})
	}
}
