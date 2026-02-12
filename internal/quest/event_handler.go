package quest

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/osse101/BrandishBot_Go/internal/crafting"
	"github.com/osse101/BrandishBot_Go/internal/domain"
	"github.com/osse101/BrandishBot_Go/internal/event"
	"github.com/osse101/BrandishBot_Go/internal/logger"
)

// EventHandler handles events related to quests
type EventHandler struct {
	service Service
}

// NewEventHandler creates a new quest event handler
func NewEventHandler(service Service) *EventHandler {
	return &EventHandler{
		service: service,
	}
}

// Register subscribes the handler to relevant events
func (h *EventHandler) Register(bus event.Bus) {
	bus.Subscribe(event.Type(domain.EventTypeItemUpgraded), h.HandleItemUpgraded)
	bus.Subscribe(event.Type(domain.EventTypeItemDisassembled), h.HandleItemDisassembled)
}

// HandleItemUpgraded handles item upgrade events to update quest progress
func (h *EventHandler) HandleItemUpgraded(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	// Decode payload
	var payload crafting.ItemUpgradedPayload
	if err := mapToStruct(evt.Payload, &payload); err != nil {
		return fmt.Errorf("failed to decode item upgraded payload: %w", err)
	}

	// Update quest progress
	// Use RecipeKey if available, otherwise fallback to ItemName (handled by service logic usually,
	// but service expects a key. crafting service passed item name as fallback if recipe key empty)
	key := payload.RecipeKey
	if key == "" {
		key = payload.ItemName
	}

	if err := h.service.OnRecipeCrafted(ctx, payload.UserID, key, payload.Quantity); err != nil {
		log.Warn("Failed to update quest progress for upgrade", "error", err, "user_id", payload.UserID)
		return nil
	}

	return nil
}

// HandleItemDisassembled handles item disassemble events to update quest progress
func (h *EventHandler) HandleItemDisassembled(ctx context.Context, evt event.Event) error {
	log := logger.FromContext(ctx)

	// Decode payload
	var payload crafting.ItemDisassembledPayload
	if err := mapToStruct(evt.Payload, &payload); err != nil {
		return fmt.Errorf("failed to decode item disassembled payload: %w", err)
	}

	// Update quest progress
	key := payload.RecipeKey
	if key == "" {
		key = payload.ItemName
	}

	if err := h.service.OnRecipeCrafted(ctx, payload.UserID, key, payload.Quantity); err != nil {
		log.Warn("Failed to update quest progress for disassemble", "error", err, "user_id", payload.UserID)
		return nil
	}

	return nil
}

// mapToStruct converts a map payload to a struct
func mapToStruct(input interface{}, output interface{}) error {
	switch v := input.(type) {
	case crafting.ItemUpgradedPayload:
		if out, ok := output.(*crafting.ItemUpgradedPayload); ok {
			*out = v
			return nil
		}
	case crafting.ItemDisassembledPayload:
		if out, ok := output.(*crafting.ItemDisassembledPayload); ok {
			*out = v
			return nil
		}
	}

	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, output)
}
