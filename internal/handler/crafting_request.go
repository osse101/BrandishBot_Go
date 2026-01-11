package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
	"github.com/osse101/BrandishBot_Go/internal/middleware"
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
func decodeCraftingRequest(r *http.Request, actionName string) (*CraftingActionRequest, error) {
	log := logger.FromContext(r.Context())

	var req CraftingActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error("Failed to decode request", "action", actionName, "error", err)
		return nil, fmt.Errorf("invalid request body: %w", err)
	}

	log.Debug("Processing request",
		"action", actionName,
		"username", req.Username,
		"item", req.Item,
		"quantity", req.Quantity)

	// Validate request
	if err := GetValidator().ValidateStruct(req); err != nil {
		log.Warn("Invalid request", "error", err)
		return nil, fmt.Errorf("Invalid request: %v", err)
	}

	return &req, nil
}

// trackCraftingEngagement publishes engagement tracking for a crafting action
func trackCraftingEngagement(ctx context.Context, eventBus event.Bus, username, eventType string, quantity int) {
	middleware.TrackEngagementFromContext(
		middleware.WithUserID(ctx, username),
		eventBus,
		eventType,
		quantity,
	)
}

// publishCraftingEvent publishes a crafting event to the event bus
func publishCraftingEvent(ctx context.Context, eventBus event.Bus, eventType event.Type, payload map[string]interface{}) error {
	log := logger.FromContext(ctx)

	if err := eventBus.Publish(ctx, event.Event{
		Version: "1.0",
		Type:    eventType,
		Payload: payload,
	}); err != nil {
		log.Error("Failed to publish event", "type", eventType, "error", err)
		return err
	}

	return nil
}
