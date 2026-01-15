package handler

import (
	"context"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/event"
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

// trackCraftingEngagement publishes engagement tracking for a crafting action
// Deprecated: Use TrackEngagement from event_helpers.go instead
func trackCraftingEngagement(ctx context.Context, eventBus event.Bus, username, eventType string, quantity int) {
	TrackEngagement(ctx, eventBus, username, eventType, quantity)
}

// publishCraftingEvent publishes a crafting event to the event bus
// Deprecated: Use PublishEvent from event_helpers.go instead
func publishCraftingEvent(ctx context.Context, eventBus event.Bus, eventType event.Type, payload map[string]interface{}) error {
	return PublishEvent(ctx, eventBus, eventType, payload)
}
