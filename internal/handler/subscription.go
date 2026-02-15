package handler

import (
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/subscription"
)

// SubscriptionHandler handles subscription-related HTTP requests
type SubscriptionHandler struct {
	service subscription.Service
}

// NewSubscriptionHandler creates a new subscription handler
func NewSubscriptionHandler(service subscription.Service) *SubscriptionHandler {
	return &SubscriptionHandler{
		service: service,
	}
}

// HandleSubscriptionEvent handles incoming subscription events from Streamer.bot webhook
// @Summary Receive subscription event from Streamer.bot
// @Description Processes subscription lifecycle events (subscribed, renewed, upgraded, downgraded, cancelled)
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param event body domain.SubscriptionEvent true "Subscription event"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/subscriptions/event [post]
func (h *SubscriptionHandler) HandleSubscriptionEvent(w http.ResponseWriter, r *http.Request) {
	var evt domain.SubscriptionEvent
	if err := DecodeAndValidateRequest(r, w, &evt, "Subscription event"); err != nil {
		return
	}

	log := logger.FromContext(r.Context())

	if err := h.service.HandleSubscriptionEvent(r.Context(), evt); err != nil {
		log.Error("Failed to handle subscription event", "error", err, "platform", evt.Platform, "username", evt.Username)
		statusCode, userMsg := mapServiceErrorToUserMessage(err)
		respondError(w, statusCode, userMsg)
		return
	}

	log.Info("Subscription event processed",
		"platform", evt.Platform,
		"username", evt.Username,
		"tier", evt.TierName,
		"event_type", evt.EventType)

	respondJSON(w, http.StatusOK, SuccessResponse{
		Message: "Subscription event processed successfully",
	})
}

// HandleGetUserSubscription retrieves a user's subscription status by platform
// @Summary Get user subscription status
// @Description Retrieves subscription information for a user on a specific platform
// @Tags subscriptions
// @Produce json
// @Param platform query string true "Platform (twitch or youtube)"
// @Param platform_id query string true "Platform user ID"
// @Success 200 {object} domain.SubscriptionWithTier
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/subscriptions/user [get]
func (h *SubscriptionHandler) HandleGetUserSubscription(w http.ResponseWriter, r *http.Request) {
	platform, ok := GetQueryParam(r, w, "platform")
	if !ok {
		return
	}

	platformID, ok := GetQueryParam(r, w, "platform_id")
	if !ok {
		return
	}

	log := logger.FromContext(r.Context())

	// Note: This endpoint expects platform_id, but our service uses userID
	// We'll need to lookup the user first via userRepo in the service
	// For now, we'll assume the calling code provides the internal userID

	// This is a simplified implementation - in production, you'd want to:
	// 1. Add a method to subscription service that takes platform+platformID
	// 2. Or add userRepo to this handler to lookup userID first

	log.Warn("GetUserSubscription endpoint needs platform_id -> userID lookup",
		"platform", platform,
		"platform_id", platformID)

	// Return not implemented for now
	respondError(w, http.StatusNotImplemented, "User lookup by platform ID not yet implemented")
}
